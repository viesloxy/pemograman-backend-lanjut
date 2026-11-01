package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"api-students/app/model"
)

// courseColumns adalah daftar kolom tabel courses, dipakai ulang RETURNING.
const courseColumns = `id, code, name, credits, quota, enrolled_count, version, created_at`

// CourseRepository menyediakan bacaan kelas yang dibutuhkan layanan lain.
// Ulasan disimpan di MongoDB, tetapi "kelasnya ada" dan "mahasiswanya
// terdaftar" tetap harus ditanyakan ke PostgreSQL. Sejak Modul 11 ia juga
// menampung seluruh kebutuhan KRS: mendaftar, keluar, dan pindah kelas.
type CourseRepository interface {
	WithTx(tx DBTX) CourseRepository

	FindByID(ctx context.Context, id int) (model.Course, error)
	IsEnrolled(ctx context.Context, courseID, studentID int) (bool, error)

	// UpdateQuota mengubah kuota dengan optimistic locking (version) dan
	// menolak kuota yang lebih kecil dari jumlah peserta saat ini.
	UpdateQuota(ctx context.Context, id, quota, version int) (model.Course, error)

	// LockStudent mengunci baris mahasiswa: seluruh perubahan KRS milik
	// satu mahasiswa berjalan bergiliran, tidak pernah serentak.
	LockStudent(ctx context.Context, studentID int) error
	// LockCourses mengunci baris kelas dalam urutan id menaik agar dua
	// pindah kelas yang berlawanan arah tidak saling menunggu (deadlock).
	LockCourses(ctx context.Context, ids ...int) error

	IncrementEnrolled(ctx context.Context, courseID int) (model.Course, error)
	DecrementEnrolled(ctx context.Context, courseID int) (model.Course, error)
	InsertEnrollment(ctx context.Context, courseID, studentID int) error
	DeleteEnrollment(ctx context.Context, courseID, studentID int) error

	// FindParticipantIDs memuat seluruh peserta kelas — dipakai worker
	// untuk memecah event perubahan kuota menjadi notifikasi per peserta.
	FindParticipantIDs(ctx context.Context, courseID int) ([]int, error)
}

type coursePostgresRepository struct{ db DBTX }

func NewCourseRepository(db DBTX) CourseRepository {
	return &coursePostgresRepository{db: db}
}

func (r *coursePostgresRepository) WithTx(tx DBTX) CourseRepository {
	return &coursePostgresRepository{db: tx}
}

func scanCourse(row pgx.Row) (model.Course, error) {
	var c model.Course
	err := row.Scan(&c.ID, &c.Code, &c.Name, &c.Credits, &c.Quota,
		&c.EnrolledCount, &c.Version, &c.CreatedAt)
	return c, err
}

func (r *coursePostgresRepository) FindByID(ctx context.Context, id int) (model.Course, error) {
	c, err := scanCourse(r.db.QueryRow(ctx,
		`SELECT `+courseColumns+` FROM courses WHERE id = $1`, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Course{}, ErrNotFound
		}
		return model.Course{}, fmt.Errorf("mengambil kelas: %w", err)
	}
	return c, nil
}

// IsEnrolled dipakai layer lain — misalnya layanan ulasan di MongoDB — yang
// perlu tahu apakah seorang mahasiswa terdaftar di sebuah kelas.
func (r *coursePostgresRepository) IsEnrolled(ctx context.Context, courseID, studentID int) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx,
		"SELECT EXISTS (SELECT 1 FROM enrollments WHERE course_id = $1 AND student_id = $2)",
		courseID, studentID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("memeriksa pendaftaran: %w", err)
	}
	return exists, nil
}

func (r *coursePostgresRepository) UpdateQuota(ctx context.Context, id, quota, version int) (model.Course, error) {
	c, err := scanCourse(r.db.QueryRow(ctx,
		`UPDATE courses SET quota = $1, version = version + 1
		 WHERE id = $2 AND version = $3 AND $1 >= enrolled_count
		 RETURNING `+courseColumns,
		quota, id, version))
	if err == nil {
		return c, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return model.Course{}, fmt.Errorf("mengubah kuota kelas: %w", err)
	}

	current, findErr := r.FindByID(ctx, id)
	if findErr != nil {
		return model.Course{}, findErr
	}
	if current.Version != version {
		return model.Course{}, ErrVersionConflict
	}
	return model.Course{}, ErrQuotaBelowFilled
}

func (r *coursePostgresRepository) LockStudent(ctx context.Context, studentID int) error {
	var id int
	err := r.db.QueryRow(ctx,
		"SELECT id FROM students WHERE id = $1 FOR UPDATE", studentID).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("mengunci data mahasiswa: %w", err)
	}
	return nil
}

func (r *coursePostgresRepository) LockCourses(ctx context.Context, ids ...int) error {
	rows, err := r.db.Query(ctx,
		"SELECT id FROM courses WHERE id = ANY($1) ORDER BY id FOR UPDATE", ids)
	if err != nil {
		return fmt.Errorf("mengunci kelas: %w", err)
	}
	defer rows.Close()

	found := 0
	for rows.Next() {
		found++
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("mengunci kelas: %w", err)
	}
	if found != len(ids) {
		return ErrNotFound
	}
	return nil
}

// IncrementEnrolled mengambil satu kursi kelas secara ATOMIK.
//
// Pemeriksaan kuota dan penambahan jumlah peserta terjadi dalam satu
// pernyataan UPDATE bersyarat. PostgreSQL mengunci row kelas selama UPDATE
// berjalan, sehingga dua pendaftaran serentak tidak mungkin sama-sama
// melihat "masih ada kursi". Selama transaction belum commit, mahasiswa
// berikutnya menunggu di pernyataan yang sama — lock inilah yang membuat
// mengirim email di dalam transaction KRS begitu berbahaya (Modul 11).
func (r *coursePostgresRepository) IncrementEnrolled(ctx context.Context, courseID int) (model.Course, error) {
	c, err := scanCourse(r.db.QueryRow(ctx,
		`UPDATE courses SET enrolled_count = enrolled_count + 1
		 WHERE id = $1 AND enrolled_count < quota
		 RETURNING `+courseColumns, courseID))
	if err == nil {
		return c, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return model.Course{}, fmt.Errorf("menambah peserta kelas: %w", err)
	}

	// Tidak ada row yang berubah. Dua kemungkinan: kelasnya tidak ada,
	// atau kelasnya ada tetapi sudah penuh. Bedakan keduanya.
	if _, findErr := r.FindByID(ctx, courseID); findErr != nil {
		return model.Course{}, findErr
	}
	return model.Course{}, ErrCourseFull
}

// DecrementEnrolled melepas satu kursi kelas, selalu sebagai pasangan
// DeleteEnrollment di dalam transaction yang sama. Bila langkah berikutnya
// gagal, ROLLBACK mengembalikan nilai ini.
func (r *coursePostgresRepository) DecrementEnrolled(ctx context.Context, courseID int) (model.Course, error) {
	c, err := scanCourse(r.db.QueryRow(ctx,
		`UPDATE courses SET enrolled_count = enrolled_count - 1
		 WHERE id = $1
		 RETURNING `+courseColumns, courseID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Course{}, ErrNotFound
		}
		return model.Course{}, fmt.Errorf("mengurangi peserta kelas: %w", err)
	}
	return c, nil
}

func (r *coursePostgresRepository) InsertEnrollment(ctx context.Context, courseID, studentID int) error {
	_, err := r.db.Exec(ctx,
		"INSERT INTO enrollments (course_id, student_id) VALUES ($1, $2)",
		courseID, studentID)
	if err != nil {
		if pgErrorCode(err) == codeUniqueViolation {
			return ErrDuplicate
		}
		return fmt.Errorf("mencatat pendaftaran: %w", err)
	}
	return nil
}

func (r *coursePostgresRepository) DeleteEnrollment(ctx context.Context, courseID, studentID int) error {
	tag, err := r.db.Exec(ctx,
		"DELETE FROM enrollments WHERE course_id = $1 AND student_id = $2",
		courseID, studentID)
	if err != nil {
		return fmt.Errorf("menghapus pendaftaran: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotEnrolled
	}
	return nil
}

func (r *coursePostgresRepository) FindParticipantIDs(ctx context.Context, courseID int) ([]int, error) {
	rows, err := r.db.Query(ctx,
		"SELECT student_id FROM enrollments WHERE course_id = $1 ORDER BY student_id",
		courseID)
	if err != nil {
		return nil, fmt.Errorf("mengambil peserta kelas: %w", err)
	}
	defer rows.Close()

	participants := []int{}
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("membaca peserta kelas: %w", err)
		}
		participants = append(participants, id)
	}
	return participants, rows.Err()
}
