package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"siakad-mini/app/model"
)

// kolomUrut adalah daftar putih pengurutan. Nilai dari query param
// tidak pernah masuk langsung ke ORDER BY.
var kolomUrutStudent = map[string]string{
	"nama":          "nama ASC",
	"ipk_terakhir":  "ipk_terakhir ASC",
	"-ipk_terakhir": "ipk_terakhir DESC NULLS LAST",
}

type StudentRepository interface {
	FindAll(ctx context.Context, q model.ListStudentQuery) ([]model.Student, int, error)
	FindByID(ctx context.Context, id int) (model.Student, error)
	// FindByNIM dipakai untuk memeriksa duplikat. Mahasiswa yang sudah
	// di-soft delete tetap terhitung, sama seperti unique index di database.
	FindByNIM(ctx context.Context, nim string) (model.Student, error)
	Create(ctx context.Context, db DBTX, s *model.Student) error
	Update(ctx context.Context, s *model.Student) error
	SoftDelete(ctx context.Context, id int) error
}

type studentPostgresRepository struct {
	pool *pgxpool.Pool
}

func NewStudentRepository(pool *pgxpool.Pool) StudentRepository {
	return &studentPostgresRepository{pool: pool}
}

func buildStudentFilter(q model.ListStudentQuery) (string, []any) {
	where := " WHERE deleted_at IS NULL"
	args := []any{}

	if q.Prodi != "" {
		where += fmt.Sprintf(" AND prodi = $%d", len(args)+1)
		args = append(args, q.Prodi)
	}
	if q.Angkatan > 0 {
		where += fmt.Sprintf(" AND angkatan = $%d", len(args)+1)
		args = append(args, q.Angkatan)
	}
	if q.Search != "" {
		where += fmt.Sprintf(" AND (nim ILIKE $%d OR nama ILIKE $%d)", len(args)+1, len(args)+1)
		args = append(args, "%"+q.Search+"%")
	}

	return where, args
}

func (r *studentPostgresRepository) FindAll(ctx context.Context, q model.ListStudentQuery) ([]model.Student, int, error) {
	where, args := buildStudentFilter(q)

	var total int
	err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM students"+where, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("menghitung mahasiswa: %w", err)
	}

	urutan, ok := kolomUrutStudent[q.Sort]
	if !ok {
		urutan = kolomUrutStudent["nama"]
	}

	sqlText := fmt.Sprintf(
		`SELECT id, user_id, nim, nama, prodi, angkatan, ipk_terakhir, deleted_at, created_at
		 FROM students%s
		 ORDER BY %s
		 LIMIT $%d OFFSET $%d`,
		where, urutan, len(args)+1, len(args)+2,
	)
	args = append(args, q.PerPage, q.Offset())

	rows, err := r.pool.Query(ctx, sqlText, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("mengambil daftar mahasiswa: %w", err)
	}
	defer rows.Close()

	hasil := []model.Student{}
	for rows.Next() {
		var s model.Student
		if err := rows.Scan(&s.ID, &s.UserID, &s.NIM, &s.Nama, &s.Prodi, &s.Angkatan, &s.IPKTerakhir, &s.DeletedAt, &s.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("membaca baris mahasiswa: %w", err)
		}
		hasil = append(hasil, s)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("membaca hasil query: %w", err)
	}

	return hasil, total, nil
}

func (r *studentPostgresRepository) FindByID(ctx context.Context, id int) (model.Student, error) {
	var s model.Student
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, nim, nama, prodi, angkatan, ipk_terakhir, deleted_at, created_at
		 FROM students
		 WHERE id = $1 AND deleted_at IS NULL`, id,
	).Scan(&s.ID, &s.UserID, &s.NIM, &s.Nama, &s.Prodi, &s.Angkatan, &s.IPKTerakhir, &s.DeletedAt, &s.CreatedAt)
	if err != nil {
		return model.Student{}, translateError(err, "mencari mahasiswa")
	}
	return s, nil
}

func (r *studentPostgresRepository) FindByNIM(ctx context.Context, nim string) (model.Student, error) {
	var s model.Student
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, nim, nama, prodi, angkatan, ipk_terakhir, deleted_at, created_at
		 FROM students
		 WHERE nim = $1`, nim,
	).Scan(&s.ID, &s.UserID, &s.NIM, &s.Nama, &s.Prodi, &s.Angkatan, &s.IPKTerakhir, &s.DeletedAt, &s.CreatedAt)
	if err != nil {
		return model.Student{}, translateError(err, "mencari mahasiswa berdasarkan NIM")
	}
	return s, nil
}

func (r *studentPostgresRepository) Create(ctx context.Context, db DBTX, s *model.Student) error {
	err := db.QueryRow(ctx,
		`INSERT INTO students (user_id, nim, nama, prodi, angkatan, ipk_terakhir)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, created_at`,
		s.UserID, s.NIM, s.Nama, s.Prodi, s.Angkatan, s.IPKTerakhir,
	).Scan(&s.ID, &s.CreatedAt)
	if err != nil {
		return translateError(err, "membuat mahasiswa")
	}
	return nil
}

func (r *studentPostgresRepository) Update(ctx context.Context, s *model.Student) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE students
		 SET nama = $1, prodi = $2, angkatan = $3, ipk_terakhir = $4
		 WHERE id = $5 AND deleted_at IS NULL`,
		s.Nama, s.Prodi, s.Angkatan, s.IPKTerakhir, s.ID,
	)
	if err != nil {
		return translateError(err, "memperbarui mahasiswa")
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *studentPostgresRepository) SoftDelete(ctx context.Context, id int) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE students SET deleted_at = NOW()
		 WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return fmt.Errorf("soft delete mahasiswa: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
