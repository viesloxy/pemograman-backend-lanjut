package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// DBTX adalah kumpulan kemampuan query yang dimiliki baik oleh
// *pgxpool.Pool maupun pgx.Tx. Method repository yang perlu ikut
// sebuah transaksi menerima parameter bertipe ini, sehingga method
// yang sama dapat dipakai di dalam maupun di luar transaksi.
type DBTX interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, arguments ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, arguments ...any) pgx.Row
}
