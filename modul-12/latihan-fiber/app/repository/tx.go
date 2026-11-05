package repository

import (
	"context"
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// TxManager adalah satu-satunya tempat transaction dibuka dan ditutup.
type TxManager struct {
	pool        *pgxpool.Pool
	maxAttempts int
}

func NewTxManager(pool *pgxpool.Pool) *TxManager {
	return &TxManager{pool: pool, maxAttempts: 3}
}

// WithTx menjalankan fn di dalam satu transaction.
//
// fn menerima DBTX, bukan pgx.Tx, supaya layer service tidak perlu
// mengimpor pgx sama sekali.
//
// Kegagalan karena deadlock (40P01) atau serialization failure (40001)
// dicoba ulang otomatis. Konsekuensinya: fn BISA dijalankan lebih dari
// sekali. Jangan pernah melakukan sesuatu di luar database di dalam fn.
func (m *TxManager) WithTx(ctx context.Context, fn func(tx DBTX) error) error {
	var err error
	for attempt := 1; attempt <= m.maxAttempts; attempt++ {
		err = m.runOnce(ctx, fn)
		if !isRetryable(err) {
			return err
		}

		wait := time.Duration(attempt*20+rand.IntN(30)) * time.Millisecond
		select {
		case <-ctx.Done():
			return err
		case <-time.After(wait):
		}
	}
	return err
}

// WithTxOnce sama dengan WithTx, tetapi TIDAK PERNAH melakukan retry.
//
// Dipakai ketika fn memang harus melakukan sesuatu di luar database —
// mengirim email, melakukan enqueue task — sambil memegang lock
// di database. Melakukan retry fn berarti mengulang tindakan luar itu.
// Tanggung jawab melakukan retry dipindahkan ke pemanggil: worker queue
// sudah punya mekanisme retry sendiri, dan relay outbox akan melakukan retry
// pada putaran berikutnya.
func (m *TxManager) WithTxOnce(ctx context.Context, fn func(tx DBTX) error) error {
	return m.runOnce(ctx, fn)
}

func (m *TxManager) runOnce(ctx context.Context, fn func(tx DBTX) error) error {
	tx, err := m.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("memulai transaction: %w", err)
	}

	// Rollback SELALU dijadwalkan. Setelah Commit berhasil, Rollback tidak
	// melakukan apa-apa (pgx mengembalikan ErrTxClosed yang kita abaikan).
	defer tx.Rollback(ctx)

	if err := fn(tx); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}

func isRetryable(err error) bool {
	code := pgErrorCode(err)
	return code == codeSerializationFailure || code == codeDeadlockDetected
}