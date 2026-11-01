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
// terdaftar" tetap harus ditanyakan ke PostgreSQL.
type CourseRepository interface {
	FindByID(ctx context.Context, id int) (model.Course, error)
	IsEnrolled(ctx context.Context, courseID, studentID int) (bool, error)
}

type coursePostgresRepository struct{ db DBTX }

func NewCourseRepository(db DBTX) CourseRepository {
	return &coursePostgresRepository{db: db}
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
