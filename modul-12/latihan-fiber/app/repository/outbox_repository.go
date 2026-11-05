package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"latihan-fiber/app/model"
)

type OutboxRepository interface {
	WithTx(tx DBTX) OutboxRepository

	// Insert WAJIB dipanggil lewat repository hasil WithTx, di dalam
	// transaction yang sama dengan perubahan data yang dicatatnya.
	Insert(ctx context.Context, eventType string, event any) error

	// FetchUnpublished mengunci baris yang diambilnya sampai transaction
	// selesai. Karena itu ia hanya bermakna bila dipanggil di dalam
	// transaction.
	FetchUnpublished(ctx context.Context, limit int) ([]model.OutboxEvent, error)
	MarkPublished(ctx context.Context, ids []int64) error

	Stats(ctx context.Context) (pending, published int, oldest time.Duration, err error)
}

type outboxPostgresRepository struct{ db DBTX }

func NewOutboxRepository(db DBTX) OutboxRepository {
	return &outboxPostgresRepository{db: db}
}

func (r *outboxPostgresRepository) WithTx(tx DBTX) OutboxRepository {
	return &outboxPostgresRepository{db: tx}
}

func (r *outboxPostgresRepository) Insert(ctx context.Context, eventType string, event any) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("menyusun payload outbox: %w", err)
	}
	_, err = r.db.Exec(ctx,
		`INSERT INTO outbox (event_type, payload) VALUES ($1, $2)`,
		eventType, payload)
	if err != nil {
		return fmt.Errorf("mencatat outbox: %w", err)
	}
	return nil
}

// FetchUnpublished memakai FOR UPDATE SKIP LOCKED.
//
//   - FOR UPDATE   : baris yang diambil di-lock, sehingga relay lain tidak
//     dapat mengambil baris yang sama pada saat yang sama.
//   - SKIP LOCKED  : baris yang sedang di-lock relay lain DILEWATI, bukan
//     ditunggu. Dua relay yang berjalan bersamaan masing-masing mendapat
//     bagian yang berbeda, tidak saling menunggu.
func (r *outboxPostgresRepository) FetchUnpublished(ctx context.Context, limit int) ([]model.OutboxEvent, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, event_type, payload, created_at
		FROM outbox
		WHERE published_at IS NULL
		ORDER BY id
		LIMIT $1
		FOR UPDATE SKIP LOCKED`, limit)
	if err != nil {
		return nil, fmt.Errorf("mengambil outbox: %w", err)
	}
	defer rows.Close()

	events := []model.OutboxEvent{}
	for rows.Next() {
		var e model.OutboxEvent
		if err := rows.Scan(&e.ID, &e.EventType, &e.Payload, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("membaca row outbox: %w", err)
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

func (r *outboxPostgresRepository) MarkPublished(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	_, err := r.db.Exec(ctx,
		`UPDATE outbox SET published_at = NOW() WHERE id = ANY($1)`, ids)
	if err != nil {
		return fmt.Errorf("menandai outbox: %w", err)
	}
	return nil
}

func (r *outboxPostgresRepository) Stats(ctx context.Context) (int, int, time.Duration, error) {
	var pending, published int
	var oldestSeconds float64
	err := r.db.QueryRow(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE published_at IS NULL),
			COUNT(*) FILTER (WHERE published_at IS NOT NULL),
			COALESCE(EXTRACT(EPOCH FROM NOW() - MIN(created_at)) FILTER (WHERE published_at IS NULL), 0)
		FROM outbox`).Scan(&pending, &published, &oldestSeconds)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("menghitung outbox: %w", err)
	}
	return pending, published, time.Duration(oldestSeconds * float64(time.Second)), nil
}