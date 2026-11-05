package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"latihan-fiber/app/model"
)

// Sentinel error: error milik lapisan repository, bukan error milik pgx.
var (
	ErrNotFound  = errors.New("data tidak ditemukan")
	ErrDuplicate = errors.New("data sudah ada")
)

// UserRepository adalah KONTRAK penyimpanan data user.
type UserRepository interface {
	FindAll(ctx context.Context, q model.ListQuery) ([]model.User, int, error)
	FindAfterCursor(ctx context.Context, q model.CursorQuery) ([]model.User, error)
	FindByID(ctx context.Context, id int) (model.User, error)
	FindByUsername(ctx context.Context, username string) (model.User, error)
	Create(ctx context.Context, u model.User) (model.User, error)
	Update(ctx context.Context, u model.User) (model.User, error)
	UpdateRole(ctx context.Context, id int, role string) (model.User, error)
	Delete(ctx context.Context, id int) error
}

// kolomUrut adalah daftar putih pemetaan nilai sort ke nama kolom.
var kolomUrut = map[string]string{
	"id":         "id",
	"username":   "username",
	"email":      "email",
	"created_at": "created_at",
}

type userPostgresRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) UserRepository {
	return &userPostgresRepository{pool: pool}
}

func buildFilter(q model.ListQuery) (string, []any) {
	where := " WHERE 1 = 1"
	args := []any{}

	if q.Search != "" {
		where += fmt.Sprintf(" AND (username ILIKE $%d OR email ILIKE $%d)",
			len(args)+1, len(args)+1)
		args = append(args, "%"+q.Search+"%")
	}
	if q.IsActive != nil {
		where += fmt.Sprintf(" AND is_active = $%d", len(args)+1)
		args = append(args, *q.IsActive)
	}

	return where, args
}

func (r *userPostgresRepository) FindAll(
	ctx context.Context, q model.ListQuery,
) ([]model.User, int, error) {
	where, args := buildFilter(q)

	var total int
	err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM users"+where, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("menghitung user: %w", err)
	}

	arah := "ASC"
	if q.Order == "desc" {
		arah = "DESC"
	}

	sqlText := fmt.Sprintf(
		`SELECT id, username, email, password, role, is_active, created_at
		 FROM users%s
		 ORDER BY %s %s
		 LIMIT $%d OFFSET $%d`,
		where, kolomUrut[q.Sort], arah, len(args)+1, len(args)+2,
	)
	args = append(args, q.Limit, q.Offset())

	rows, err := r.pool.Query(ctx, sqlText, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("mengambil daftar user: %w", err)
	}
	defer rows.Close()

	hasil := []model.User{}
	for rows.Next() {
		var u model.User
		if err := rows.Scan(&u.ID, &u.Username, &u.Email, &u.Password,
			&u.Role, &u.IsActive, &u.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("membaca baris user: %w", err)
		}
		hasil = append(hasil, u)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("membaca hasil query: %w", err)
	}

	return hasil, total, nil
}

func (r *userPostgresRepository) FindByID(
	ctx context.Context, id int,
) (model.User, error) {
	var u model.User
	err := r.pool.QueryRow(ctx,
		`SELECT id, username, email, password, role, is_active, created_at
		 FROM users WHERE id = $1`, id,
	).Scan(&u.ID, &u.Username, &u.Email, &u.Password, &u.Role, &u.IsActive, &u.CreatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.User{}, ErrNotFound
		}
		return model.User{}, fmt.Errorf("mengambil user: %w", err)
	}
	return u, nil
}

// FindByUsername dipakai saat login. Pencocokan tidak membedakan
// huruf besar dan kecil, sama seperti unique index-nya.
func (r *userPostgresRepository) FindByUsername(
	ctx context.Context, username string,
) (model.User, error) {
	var u model.User
	err := r.pool.QueryRow(ctx,
		`SELECT id, username, email, password, role, is_active, created_at
		 FROM users WHERE LOWER(username) = LOWER($1)`, username,
	).Scan(&u.ID, &u.Username, &u.Email, &u.Password, &u.Role,
		&u.IsActive, &u.CreatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.User{}, ErrNotFound
		}
		return model.User{}, fmt.Errorf("mengambil user: %w", err)
	}
	return u, nil
}

func (r *userPostgresRepository) Create(
	ctx context.Context, u model.User,
) (model.User, error) {
	// Role tidak disisipkan di sini: column role memakai DEFAULT 'user'.
	// Pengisian role selain 'user' dilakukan lewat jalur khusus (pertemuan 6).
	err := r.pool.QueryRow(ctx,
		`INSERT INTO users (username, email, password, is_active)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, role, created_at`,
		u.Username, u.Email, u.Password, u.IsActive,
	).Scan(&u.ID, &u.Role, &u.CreatedAt)

	if err != nil {
		if isUniqueViolation(err) {
			return model.User{}, ErrDuplicate
		}
		return model.User{}, fmt.Errorf("menyimpan user: %w", err)
	}
	return u, nil
}

func (r *userPostgresRepository) Update(
	ctx context.Context, u model.User,
) (model.User, error) {
	err := r.pool.QueryRow(ctx,
		`UPDATE users SET username = $1, email = $2, is_active = $3
		 WHERE id = $4
		 RETURNING id, username, email, password, role, is_active, created_at`,
		u.Username, u.Email, u.IsActive, u.ID,
	).Scan(&u.ID, &u.Username, &u.Email, &u.Password, &u.Role, &u.IsActive, &u.CreatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.User{}, ErrNotFound
		}
		if isUniqueViolation(err) {
			return model.User{}, ErrDuplicate
		}
		return model.User{}, fmt.Errorf("memperbarui user: %w", err)
	}
	return u, nil
}

func (r *userPostgresRepository) Delete(ctx context.Context, id int) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("menghapus user: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// FindAfterCursor mengambil satu halaman memakai keyset pagination.
//
// id ikut dibandingkan karena created_at TIDAK dijamin unik. Urutan column
// index HARUS sama persis dengan ORDER BY (keduanya DESC), supaya index
// dapat dipakai untuk melompat langsung ke posisi cursor.
//
// Jumlah yang diminta sengaja limit+1: baris tambahan itu hanya penanda
// "masih ada halaman berikutnya?" tanpa perlu COUNT(*) atas seluruh tabel.
func (r *userPostgresRepository) FindAfterCursor(
	ctx context.Context, q model.CursorQuery,
) ([]model.User, error) {
	args := []any{}
	where := " WHERE 1 = 1"

	if q.Search != "" {
		args = append(args, "%"+q.Search+"%")
		where += fmt.Sprintf(" AND (username ILIKE $%d OR email ILIKE $%d)",
			len(args), len(args))
	}
	if q.IsActive != nil {
		args = append(args, *q.IsActive)
		where += fmt.Sprintf(" AND is_active = $%d", len(args))
	}
	if q.After != nil {
		args = append(args, q.After.CreatedAt, q.After.ID)
		where += fmt.Sprintf(" AND (created_at, id) < ($%d, $%d)",
			len(args)-1, len(args))
	}

	args = append(args, q.Limit+1)
	query := fmt.Sprintf(
		"SELECT %s FROM users%s ORDER BY created_at DESC, id DESC LIMIT $%d",
		userColumns, where, len(args))

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("mengambil daftar user: %w", err)
	}
	defer rows.Close()

	result := []model.User{}
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, fmt.Errorf("membaca row user: %w", err)
		}
		result = append(result, u)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("membaca hasil query: %w", err)
	}

	return result, nil
}

// userColumns adalah daftar kolom user yang konsisten dipakai beberapa query.
const userColumns = "id, username, email, password, role, is_active, created_at"

// scanUser membaca satu baris hasil query menjadi model.User.
func scanUser(row pgx.Row) (model.User, error) {
	var u model.User
	err := row.Scan(&u.ID, &u.Username, &u.Email, &u.Password, &u.Role, &u.IsActive, &u.CreatedAt)
	return u, err
}

// UpdateRole sengaja dipisah dari Update. Mengubah role adalah tindakan
// istimewa yang dijaga permission tersendiri.
func (r *userPostgresRepository) UpdateRole(
	ctx context.Context, id int, role string,
) (model.User, error) {
	updated, err := scanUser(r.pool.QueryRow(ctx,
		"UPDATE users SET role = $1 WHERE id = $2 RETURNING "+userColumns,
		role, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.User{}, ErrNotFound
		}
		return model.User{}, fmt.Errorf("mengubah role user: %w", err)
	}
	return updated, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}