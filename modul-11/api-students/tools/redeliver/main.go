// Command redeliver adalah alat uji idempotency (Bagian C.5 dan D.2).
//
// Alat ini membaca satu baris outbox lalu membuat sejumlah SALINAN task di
// Redis untuk event yang sama — meniru kejadian "task yang sama datang
// berkali-kali" yang menurut jaminan at-least-once memang mungkin terjadi.
// Worker yang benar harus tetap mengirim TEPAT SATU notifikasi, berapa pun
// jumlah salinannya.
//
// Cara pakai:
//
//	go run ./tools/redeliver -outbox <id> -copies 5
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5"

	"api-students/app/model"
	"api-students/config"
)

func main() {
	outboxID := flag.Int64("outbox", 0, "id baris outbox yang akan dikirim ulang")
	copies := flag.Int("copies", 1, "jumlah salinan task yang dibuat")
	flag.Parse()

	config.LoadEnv()
	if *outboxID <= 0 || *copies <= 0 {
		log.Fatal("pakai: go run ./tools/redeliver -outbox <id> -copies <n>")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		config.GetEnv("DB_USER", "postgres"),
		config.GetEnv("DB_PASSWORD", ""),
		config.GetEnv("DB_HOST", "localhost"),
		config.GetEnv("DB_PORT", "5432"),
		config.GetEnv("DB_NAME", "praktikum_backend"),
		config.GetEnv("DB_SSLMODE", "disable"),
	)

	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		log.Fatalf("terhubung ke database: %v", err)
	}
	defer conn.Close(ctx)

	var (
		id        int64
		eventType string
		payload   []byte
		published *time.Time
	)
	err = conn.QueryRow(ctx,
		`SELECT id, event_type, payload, published_at FROM outbox WHERE id = $1`,
		*outboxID).Scan(&id, &eventType, &payload, &published)
	if err != nil {
		log.Fatalf("membaca outbox %d: %v", *outboxID, err)
	}
	if published == nil {
		log.Printf("peringatan: outbox %d belum di-publish relay; salinan tetap dibuat", id)
	}

	client := asynq.NewClient(config.RedisOpt())
	defer client.Close()

	job := model.NotificationJob{OutboxID: id, EventType: eventType, Payload: payload}
	body, err := json.Marshal(job)
	if err != nil {
		log.Fatalf("menyusun payload: %v", err)
	}

	for i := 1; i <= *copies; i++ {
		// TaskID sengaja BERBEDA dari milik relay (outbox-<id>) dan antar
		// salinan: Redis harus menerima semuanya. Yang menjaga agar email
		// tetap satu adalah claim di database, bukan TaskID.
		task := asynq.NewTask(model.TaskSendNotification, body,
			asynq.TaskID(fmt.Sprintf("redeliver-%d-%d", id, i)),
			asynq.Queue(model.QueueNotifications),
			asynq.MaxRetry(5),
			asynq.Timeout(30*time.Second),
			asynq.Retention(24*time.Hour),
		)
		if _, err := client.EnqueueContext(ctx, task); err != nil {
			log.Fatalf("enqueue salinan %d: %v", i, err)
		}
		log.Printf("salinan %d/%d untuk outbox %d diantrekan", i, *copies, id)
	}
}
