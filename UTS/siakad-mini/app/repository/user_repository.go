package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"siakad-mini/app/model"
)

type UserRepository interface {
	// FindByEmail dipakai saat login. User yang sudah di-soft delete
	// tidak dianggap ada, sehingga tidak dapat login.
	FindByEmail(ctx context.Context, email string) (model.User, error)
	// Create membuat user baru. Menerima DBTX agar dapat dijalankan
	// di dalam transaksi yang sama dengan pembuatan student-nya.
	Create(ctx context.Context, db DBTX, u *model.User) error
	SoftDelete(ctx context.Context, db DBTX, id int) error
}

type userPostgresRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) UserRepository {
	return &userPostgresRepository{pool: pool}
}

func (r *userPostgresRepository) FindByEmail(ctx context.Context, email string) (model.User, error) {
	var u model.User
	err := r.pool.QueryRow(ctx,
		`SELECT id, email, password, role, deleted_at, created_at
		 FROM users
		 WHERE LOWER(email) = LOWER($1) AND deleted_at IS NULL`, email,
	).Scan(&u.ID, &u.Email, &u.Password, &u.Role, &u.DeletedAt, &u.CreatedAt)
	if err != nil {
		return model.User{}, translateError(err, "mencari user berdasarkan email")
	}
	return u, nil
}

func (r *userPostgresRepository) Create(ctx context.Context, db DBTX, u *model.User) error {
	err := db.QueryRow(ctx,
		`INSERT INTO users (email, password, role)
		 VALUES ($1, $2, $3)
		 RETURNING id, created_at`,
		u.Email, u.Password, u.Role,
	).Scan(&u.ID, &u.CreatedAt)
	if err != nil {
		return translateError(err, "membuat user")
	}
	return nil
}

func (r *userPostgresRepository) SoftDelete(ctx context.Context, db DBTX, id int) error {
	tag, err := db.Exec(ctx,
		`UPDATE users SET deleted_at = NOW()
		 WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return fmt.Errorf("soft delete user: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
