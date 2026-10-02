package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"siakad-mini/app/model"
)

type EnrollmentRepository interface {
	FindByID(ctx context.Context, id int) (model.Enrollment, error)
	// Create menambah baris KRS. Menerima DBTX agar berjalan pada
	// transaksi yang sama dengan pemeriksaan kuota dan batas SKS.
	Create(ctx context.Context, db DBTX, e *model.Enrollment) error
	// Delete menghapus baris KRS (hard delete). Kuota kembali bertambah
	// karena jumlah terisi selalu dihitung dari tabel ini.
	Delete(ctx context.Context, db DBTX, id int) error
	ListByStudent(ctx context.Context, studentID int) ([]model.EnrollmentCourse, error)
	// Exists memeriksa duplikasi KRS di dalam transaksi.
	Exists(ctx context.Context, db DBTX, studentID, courseID int, tahunAkademik string) (bool, error)
	// CountByCourse menghitung jumlah pengambil sebuah mata kuliah
	// di dalam transaksi, dipakai untuk pemeriksaan kuota.
	CountByCourse(ctx context.Context, db DBTX, courseID int) (int, error)
	// TotalSKS menjumlahkan seluruh SKS yang diambil seorang mahasiswa.
	TotalSKS(ctx context.Context, db DBTX, studentID int) (int, error)
}

type enrollmentPostgresRepository struct {
	pool *pgxpool.Pool
}

func NewEnrollmentRepository(pool *pgxpool.Pool) EnrollmentRepository {
	return &enrollmentPostgresRepository{pool: pool}
}

func (r *enrollmentPostgresRepository) FindByID(ctx context.Context, id int) (model.Enrollment, error) {
	var e model.Enrollment
	err := r.pool.QueryRow(ctx,
		`SELECT id, student_id, course_id, tahun_akademik, created_at
		 FROM enrollments WHERE id = $1`, id,
	).Scan(&e.ID, &e.StudentID, &e.CourseID, &e.TahunAkademik, &e.CreatedAt)
	if err != nil {
		return model.Enrollment{}, translateError(err, "mencari enrollment")
	}
	return e, nil
}

func (r *enrollmentPostgresRepository) Create(ctx context.Context, db DBTX, e *model.Enrollment) error {
	err := db.QueryRow(ctx,
		`INSERT INTO enrollments (student_id, course_id, tahun_akademik)
		 VALUES ($1, $2, $3)
		 RETURNING id, created_at`,
		e.StudentID, e.CourseID, e.TahunAkademik,
	).Scan(&e.ID, &e.CreatedAt)
	if err != nil {
		return translateError(err, "membuat enrollment")
	}
	return nil
}

func (r *enrollmentPostgresRepository) Delete(ctx context.Context, db DBTX, id int) error {
	tag, err := db.Exec(ctx, `DELETE FROM enrollments WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("menghapus enrollment: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *enrollmentPostgresRepository) ListByStudent(ctx context.Context, studentID int) ([]model.EnrollmentCourse, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT e.id, c.id, c.kode_mk, c.nama_mk, c.sks, c.semester, e.tahun_akademik
		 FROM enrollments e
		 JOIN courses c ON c.id = e.course_id
		 WHERE e.student_id = $1
		 ORDER BY e.tahun_akademik, c.kode_mk`, studentID,
	)
	if err != nil {
		return nil, fmt.Errorf("mengambil KRS mahasiswa: %w", err)
	}
	defer rows.Close()

	hasil := []model.EnrollmentCourse{}
	for rows.Next() {
		var ec model.EnrollmentCourse
		if err := rows.Scan(&ec.EnrollmentID, &ec.CourseID, &ec.KodeMK, &ec.NamaMK, &ec.SKS, &ec.Semester, &ec.TahunAkademik); err != nil {
			return nil, fmt.Errorf("membaca baris KRS: %w", err)
		}
		hasil = append(hasil, ec)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("membaca hasil query: %w", err)
	}

	return hasil, nil
}

func (r *enrollmentPostgresRepository) TotalSKS(ctx context.Context, db DBTX, studentID int) (int, error) {
	var total int
	err := db.QueryRow(ctx,
		`SELECT COALESCE(SUM(c.sks), 0)::int
		 FROM enrollments e
		 JOIN courses c ON c.id = e.course_id
		 WHERE e.student_id = $1`, studentID,
	).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("menghitung total SKS: %w", err)
	}
	return total, nil
}

func (r *enrollmentPostgresRepository) Exists(ctx context.Context, db DBTX, studentID, courseID int, tahunAkademik string) (bool, error) {
	var ada bool
	err := db.QueryRow(ctx,
		`SELECT EXISTS (
			 SELECT 1 FROM enrollments
			 WHERE student_id = $1 AND course_id = $2 AND tahun_akademik = $3
		 )`, studentID, courseID, tahunAkademik,
	).Scan(&ada)
	if err != nil {
		return false, fmt.Errorf("memeriksa duplikasi KRS: %w", err)
	}
	return ada, nil
}

func (r *enrollmentPostgresRepository) CountByCourse(ctx context.Context, db DBTX, courseID int) (int, error) {
	var jumlah int
	err := db.QueryRow(ctx,
		`SELECT COUNT(*) FROM enrollments WHERE course_id = $1`, courseID,
	).Scan(&jumlah)
	if err != nil {
		return 0, fmt.Errorf("menghitung pengambil mata kuliah: %w", err)
	}
	return jumlah, nil
}
