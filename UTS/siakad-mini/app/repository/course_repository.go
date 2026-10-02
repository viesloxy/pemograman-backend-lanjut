package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"siakad-mini/app/model"
)

type CourseRepository interface {
	// FindAll mengembalikan daftar mata kuliah beserta terisi dan
	// sisa kuota, yang dihitung dari tabel enrollments.
	FindAll(ctx context.Context, q model.ListCourseQuery) ([]model.CourseWithKuota, error)
	FindByID(ctx context.Context, id int) (model.Course, error)
	// LockByID mengunci baris mata kuliah (SELECT ... FOR UPDATE) sampai
	// transaksi selesai. Pemeriksaan kuota yang dilakukan setelah kunci
	// ini tidak dapat balapan dengan permintaan lain.
	LockByID(ctx context.Context, db DBTX, id int) (model.Course, error)
}

type coursePostgresRepository struct {
	pool *pgxpool.Pool
}

func NewCourseRepository(pool *pgxpool.Pool) CourseRepository {
	return &coursePostgresRepository{pool: pool}
}

func (r *coursePostgresRepository) FindAll(ctx context.Context, q model.ListCourseQuery) ([]model.CourseWithKuota, error) {
	where := ""
	args := []any{}

	if q.Semester > 0 {
		where += fmt.Sprintf(" AND c.semester = $%d", len(args)+1)
		args = append(args, q.Semester)
	}
	if q.Search != "" {
		where += fmt.Sprintf(" AND (c.kode_mk ILIKE $%d OR c.nama_mk ILIKE $%d)", len(args)+1, len(args)+1)
		args = append(args, "%"+q.Search+"%")
	}

	having := ""
	if q.Available {
		having = " HAVING COUNT(e.id) < c.kuota"
	}

	sqlText := fmt.Sprintf(
		`SELECT c.id, c.kode_mk, c.nama_mk, c.sks, c.semester, c.kuota,
		        COUNT(e.id)::int AS terisi,
		        c.kuota - COUNT(e.id)::int AS sisa_kuota
		 FROM courses c
		 LEFT JOIN enrollments e ON e.course_id = c.id
		 WHERE TRUE%s
		 GROUP BY c.id%s
		 ORDER BY c.kode_mk`,
		where, having,
	)

	rows, err := r.pool.Query(ctx, sqlText, args...)
	if err != nil {
		return nil, fmt.Errorf("mengambil daftar mata kuliah: %w", err)
	}
	defer rows.Close()

	hasil := []model.CourseWithKuota{}
	for rows.Next() {
		var c model.CourseWithKuota
		if err := rows.Scan(&c.ID, &c.KodeMK, &c.NamaMK, &c.SKS, &c.Semester, &c.Kuota, &c.Terisi, &c.SisaKuota); err != nil {
			return nil, fmt.Errorf("membaca baris mata kuliah: %w", err)
		}
		hasil = append(hasil, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("membaca hasil query: %w", err)
	}

	return hasil, nil
}

func (r *coursePostgresRepository) FindByID(ctx context.Context, id int) (model.Course, error) {
	var c model.Course
	err := r.pool.QueryRow(ctx,
		`SELECT id, kode_mk, nama_mk, sks, semester, kuota
		 FROM courses WHERE id = $1`, id,
	).Scan(&c.ID, &c.KodeMK, &c.NamaMK, &c.SKS, &c.Semester, &c.Kuota)
	if err != nil {
		return model.Course{}, translateError(err, "mencari mata kuliah")
	}
	return c, nil
}

func (r *coursePostgresRepository) LockByID(ctx context.Context, db DBTX, id int) (model.Course, error) {
	var c model.Course
	err := db.QueryRow(ctx,
		`SELECT id, kode_mk, nama_mk, sks, semester, kuota
		 FROM courses WHERE id = $1 FOR UPDATE`, id,
	).Scan(&c.ID, &c.KodeMK, &c.NamaMK, &c.SKS, &c.Semester, &c.Kuota)
	if err != nil {
		return model.Course{}, translateError(err, "mengunci mata kuliah")
	}
	return c, nil
}
