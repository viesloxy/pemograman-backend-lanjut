package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Kode SQLSTATE dari PostgreSQL yang perlu dikenali aplikasi.
const (
	codeUniqueViolation      = "23505"
	codeSerializationFailure = "40001"
	codeDeadlockDetected     = "40P01"
)

// DBTX adalah kemampuan minimum untuk menjalankan query.
//
// *pgxpool.Pool dan pgx.Tx sama-sama memenuhi interface ini. Karena itu
// repository yang hanya bergantung pada DBTX dapat bekerja di luar maupun
// di dalam transaction tanpa mengubah satu baris pun query-nya.
type DBTX interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func pgErrorCode(err error) string {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code
	}
	return ""
}