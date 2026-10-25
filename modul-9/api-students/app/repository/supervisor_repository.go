package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"api-students/app/model"
)

var (
	ErrCourseFull       = errors.New("kuota sudah penuh")
	ErrSupervisorFull   = errors.New("kuota pembimbing sudah penuh")
	ErrNotEnrolled      = errors.New("tidak terdaftar")
	ErrVersionConflict  = errors.New("data sudah diubah pihak lain")
	ErrQuotaBelowFilled = errors.New("kuota lebih kecil dari jumlah mahasiswa")
)

type SupervisorRepository interface {
	WithTx(tx DBTX) SupervisorRepository

	FindAll(ctx context.Context) ([]model.Supervisor, error)
	FindByID(ctx context.Context, id int) (model.Supervisor, error)
	UpdateQuota(ctx context.Context, id, quota, version int) (model.Supervisor, error)
	LockStudent(ctx context.Context, studentID int) error
	LockSupervisors(ctx context.Context, ids ...int) error
	IncrementSupervised(ctx context.Context, supervisorID int) (model.Supervisor, error)
	DecrementSupervised(ctx context.Context, supervisorID int) (model.Supervisor, error)
	FindSupervision(ctx context.Context, supervisorID, studentID int) error
	AssignSupervisor(ctx context.Context, supervisorID, studentID int) error
	DeleteSupervision(ctx context.Context, supervisorID, studentID int) error
	IsEnrolled(ctx context.Context, studentID int) (bool, error)
}

type supervisorRepository struct{ db DBTX }

func NewSupervisorRepository(db DBTX) SupervisorRepository {
	return &supervisorRepository{db: db}
}

func (r *supervisorRepository) WithTx(tx DBTX) SupervisorRepository {
	return &supervisorRepository{db: tx}
}

func scanSupervisor(row pgx.Row) (model.Supervisor, error) {
	var s model.Supervisor
	err := row.Scan(&s.ID, &s.Name, &s.Quota, &s.SupervisedCount,
		&s.Version, &s.CreatedAt)
	return s, err
}

func (r *supervisorRepository) FindAll(ctx context.Context) ([]model.Supervisor, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, name, quota, supervised_count, version, created_at
		 FROM supervisors ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("mengambil daftar pembimbing: %w", err)
	}
	defer rows.Close()

	result := []model.Supervisor{}
	for rows.Next() {
		s, err := scanSupervisor(rows)
		if err != nil {
			return nil, fmt.Errorf("membaca row pembimbing: %w", err)
		}
		result = append(result, s)
	}
	return result, rows.Err()
}

func (r *supervisorRepository) FindByID(ctx context.Context, id int) (model.Supervisor, error) {
	s, err := scanSupervisor(r.db.QueryRow(ctx,
		`SELECT id, name, quota, supervised_count, version, created_at
		 FROM supervisors WHERE id = $1`, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Supervisor{}, ErrNotFound
		}
		return model.Supervisor{}, fmt.Errorf("mengambil pembimbing: %w", err)
	}
	return s, nil
}

func (r *supervisorRepository) UpdateQuota(ctx context.Context, id, quota, version int) (model.Supervisor, error) {
	s, err := scanSupervisor(r.db.QueryRow(ctx,
		`UPDATE supervisors SET quota = $1, version = version + 1
		 WHERE id = $2 AND version = $3 AND $1 >= supervised_count
		 RETURNING id, name, quota, supervised_count, version, created_at`,
		quota, id, version))
	if err == nil {
		return s, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return model.Supervisor{}, fmt.Errorf("mengubah kuota: %w", err)
	}

	current, findErr := r.FindByID(ctx, id)
	if findErr != nil {
		return model.Supervisor{}, findErr
	}
	if current.Version != version {
		return model.Supervisor{}, ErrVersionConflict
	}
	return model.Supervisor{}, ErrQuotaBelowFilled
}

func (r *supervisorRepository) LockStudent(ctx context.Context, studentID int) error {
	var id int
	err := r.db.QueryRow(ctx,
		"SELECT id FROM students WHERE id = $1", studentID).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("mengunci data mahasiswa: %w", err)
	}
	return nil
}

func (r *supervisorRepository) LockSupervisors(ctx context.Context, ids ...int) error {
	rows, err := r.db.Query(ctx,
		"SELECT id FROM supervisors WHERE id = ANY($1) ORDER BY id FOR UPDATE", ids)
	if err != nil {
		return fmt.Errorf("mengunci pembimbing: %w", err)
	}
	defer rows.Close()

	found := 0
	for rows.Next() {
		found++
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("mengunci pembimbing: %w", err)
	}
	if found != len(ids) {
		return ErrNotFound
	}
	return nil
}

// IncrementSupervised mengambil satu kursi bimbingan secara ATOMIK.
//
// Pemeriksaan kuota dan penambahan jumlah bimbingan terjadi dalam satu
// pernyataan UPDATE (teknik UPDATE bersyarat, Bagian A.4 cara 1).
// PostgreSQL mengunci row tersebut selama UPDATE berjalan, sehingga
// dua penetapan yang datang bersamaan tidak mungkin sama-sama melihat
// "masih ada kursi".
func (r *supervisorRepository) IncrementSupervised(ctx context.Context, supervisorID int) (model.Supervisor, error) {
	s, err := scanSupervisor(r.db.QueryRow(ctx,
		`UPDATE supervisors SET supervised_count = supervised_count + 1
		 WHERE id = $1 AND supervised_count < quota
		 RETURNING id, name, quota, supervised_count, version, created_at`, supervisorID))
	if err == nil {
		return s, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return model.Supervisor{}, fmt.Errorf("menambah mahasiswa bimbingan: %w", err)
	}

	// Tidak ada row yang berubah. Dua kemungkinan: pembimbingnya tidak ada,
	// atau pembimbingnya ada tetapi sudah penuh. Bedakan keduanya.
	if _, findErr := r.FindByID(ctx, supervisorID); findErr != nil {
		return model.Supervisor{}, findErr
	}
	return model.Supervisor{}, ErrSupervisorFull
}

// DecrementSupervised melepas satu kursi bimbingan, selalu sebagai
// pasangan DeleteSupervision di dalam transaction yang sama. Bila
// penugasan baru ternyata gagal, ROLLBACK mengembalikan nilai ini.
func (r *supervisorRepository) DecrementSupervised(ctx context.Context, supervisorID int) (model.Supervisor, error) {
	s, err := scanSupervisor(r.db.QueryRow(ctx,
		`UPDATE supervisors SET supervised_count = supervised_count - 1
		 WHERE id = $1
		 RETURNING id, name, quota, supervised_count, version, created_at`, supervisorID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Supervisor{}, ErrNotFound
		}
		return model.Supervisor{}, fmt.Errorf("mengurangi mahasiswa bimbingan: %w", err)
	}
	return s, nil
}

func (r *supervisorRepository) FindSupervision(ctx context.Context, supervisorID, studentID int) error {
	var sid int
	err := r.db.QueryRow(ctx,
		"SELECT supervisor_id FROM supervisions WHERE supervisor_id = $1 AND student_id = $2",
		supervisorID, studentID).Scan(&sid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("mencari pendaftaran: %w", err)
	}
	return nil
}

func (r *supervisorRepository) AssignSupervisor(ctx context.Context, supervisorID, studentID int) error {
	_, err := r.db.Exec(ctx,
		"INSERT INTO supervisions (supervisor_id, student_id) VALUES ($1, $2)",
		supervisorID, studentID)
	if err != nil {
		if pgErrorCode(err) == codeUniqueViolation {
			return ErrDuplicate
		}
		return fmt.Errorf("menetapkan pembimbing: %w", err)
	}
	return nil
}

func (r *supervisorRepository) DeleteSupervision(ctx context.Context, supervisorID, studentID int) error {
	tag, err := r.db.Exec(ctx,
		"DELETE FROM supervisions WHERE supervisor_id = $1 AND student_id = $2",
		supervisorID, studentID)
	if err != nil {
		return fmt.Errorf("menghapus pembimbing: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotEnrolled
	}
	return nil
}

func (r *supervisorRepository) IsEnrolled(ctx context.Context, studentID int) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx,
		"SELECT EXISTS (SELECT 1 FROM supervisions WHERE student_id = $1)", studentID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("memeriksa pendaftaran: %w", err)
	}
	return exists, nil
}
