package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"api-students/app/model"
)

type NotificationRepository interface {
	WithTx(tx DBTX) NotificationRepository

	// Claim mencatat bahwa notifikasi untuk sebuah event akan dikirim.
	// Mengembalikan false bila event itu sudah pernah tercatat.
	Claim(ctx context.Context, n model.Notification) (bool, error)

	FindByUser(ctx context.Context, userID, limit int) ([]model.Notification, error)
	Count(ctx context.Context) (int, error)
}

type notificationPostgresRepository struct{ db DBTX }

func NewNotificationRepository(db DBTX) NotificationRepository {
	return &notificationPostgresRepository{db: db}
}

func (r *notificationPostgresRepository) WithTx(tx DBTX) NotificationRepository {
	return &notificationPostgresRepository{db: tx}
}

// Claim memakai ON CONFLICT DO NOTHING, bukan "SELECT dulu, lalu INSERT
// bila belum ada". Versi SELECT-lalu-INSERT adalah pola check-then-act dari
// Modul 9: dua worker dapat sama-sama melihat "belum ada" pada detik yang
// sama. UNIQUE constraint memang tetap menolak INSERT yang kedua, tetapi
// penolakan itu datang sebagai error 23505 — task-nya dianggap GAGAL dan
// di-retry, padahal yang benar adalah "sudah dikirim, lewati saja".
//
// Dengan ON CONFLICT, bila worker lain sedang memegang baris dengan
// outbox_id yang sama tetapi belum commit, pernyataan ini MENUNGGU sampai
// worker itu selesai — lalu melewatinya bila worker itu commit, atau
// melanjutkan bila worker itu rollback.
func (r *notificationPostgresRepository) Claim(ctx context.Context, n model.Notification) (bool, error) {
	// Claim memakai ON CONFLICT (outbox_id, user_id) — sejak migration 011,
	// satu event boleh menghasilkan banyak notifikasi (satu per penerima),
	// sehingga idempotency key-nya adalah pasangan event dan penerima.
	var id int64
	err := r.db.QueryRow(ctx, `
		INSERT INTO notifications (outbox_id, user_id, channel, subject, body)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (outbox_id, user_id) DO NOTHING
		RETURNING id`,
		n.OutboxID, n.UserID, n.Channel, n.Subject, n.Body).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("mencatat notifikasi: %w", err)
	}
	return true, nil
}

// FindByUser mengurutkan berdasarkan outbox_id, bukan id notifikasi.
// Worker mengerjakan beberapa task sekaligus, sehingga urutan
// notifikasi DICATAT tidak sama dengan urutan event-nya. outbox_id
// mengikuti urutan event, karena seluruh perubahan KRS milik satu
// mahasiswa bergiliran (LockStudent, Modul 9).
func (r *notificationPostgresRepository) FindByUser(ctx context.Context, userID, limit int) ([]model.Notification, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, outbox_id, user_id, channel, subject, body, sent_at
		FROM notifications
		WHERE user_id = $1
		ORDER BY outbox_id DESC
		LIMIT $2`, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("mengambil notifikasi: %w", err)
	}
	defer rows.Close()

	result := []model.Notification{}
	for rows.Next() {
		var n model.Notification
		if err := rows.Scan(&n.ID, &n.OutboxID, &n.UserID, &n.Channel,
			&n.Subject, &n.Body, &n.SentAt); err != nil {
			return nil, fmt.Errorf("membaca row notifikasi: %w", err)
		}
		result = append(result, n)
	}
	return result, rows.Err()
}

func (r *notificationPostgresRepository) Count(ctx context.Context) (int, error) {
	var total int
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM notifications`).Scan(&total); err != nil {
		return 0, fmt.Errorf("menghitung notifikasi: %w", err)
	}
	return total, nil
}
