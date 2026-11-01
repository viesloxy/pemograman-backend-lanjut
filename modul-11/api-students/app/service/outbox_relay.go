package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/hibiken/asynq"

	"api-students/app/model"
	"api-students/app/repository"
)

// TaskEnqueuer adalah kemampuan melakukan enqueue task.
// *asynq.Client memenuhi interface ini.
type TaskEnqueuer interface {
	EnqueueContext(ctx context.Context, task *asynq.Task, opts ...asynq.Option) (*asynq.TaskInfo, error)
}

// OutboxRelay memindahkan event dari tabel outbox ke queue Redis.
//
// Relay adalah satu-satunya komponen yang membaca outbox. API tidak pernah
// berbicara dengan Redis; ia cukup menulis ke PostgreSQL. Karena itu
// mahasiswa tetap dapat mengisi KRS walaupun Redis sedang mati — notifikasinya
// hanya tertunda, tidak hilang.
type OutboxRelay struct {
	outbox   repository.OutboxRepository
	tx       *repository.TxManager
	queue    TaskEnqueuer
	logger   *slog.Logger
	interval time.Duration
	batch    int
}

func NewOutboxRelay(
	outbox repository.OutboxRepository,
	tx *repository.TxManager,
	queue TaskEnqueuer,
	logger *slog.Logger,
	interval time.Duration,
	batch int,
) *OutboxRelay {
	return &OutboxRelay{outbox: outbox, tx: tx, queue: queue, logger: logger,
		interval: interval, batch: batch}
}

// Run berjalan terus sampai ctx dibatalkan.
func (r *OutboxRelay) Run(ctx context.Context) {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		n, err := r.RelayOnce(ctx)
		if err != nil {
			r.logger.Warn("relay_delayed",
				slog.Int("published", n), slog.String("error", err.Error()))
		} else if n > 0 {
			r.logger.Info("relay_done", slog.Int("published", n))
		}

		// Satu batch penuh berarti kemungkinan besar masih ada sisa:
		// langsung lanjutkan tanpa menunggu. Selain itu, tunggu giliran
		// berikutnya.
		if err == nil && n == r.batch && ctx.Err() == nil {
			continue
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// RelayOnce mem-publish paling banyak satu batch. Mengembalikan jumlah
// event yang berhasil di-publish.
func (r *OutboxRelay) RelayOnce(ctx context.Context) (int, error) {
	var published int
	var enqueueErr error

	err := r.tx.WithTxOnce(ctx, func(tx repository.DBTX) error {
		repo := r.outbox.WithTx(tx)

		events, err := repo.FetchUnpublished(ctx, r.batch)
		if err != nil {
			return err
		}

		ids := make([]int64, 0, len(events))
		for _, e := range events {
			// Berhenti pada kegagalan pertama. Bila Redis mati, event
			// berikutnya pun pasti gagal; tidak ada gunanya dicoba satu per
			// satu. Yang belum di-publish tetap NULL dan diambil lagi pada
			// putaran berikutnya.
			if err := r.publish(ctx, e); err != nil {
				enqueueErr = err
				break
			}
			// PERNAYAAN: id dicatat HANYA setelah enqueue berhasil. Bila
			// dicatat sebelumnya, event yang gagal masuk Redis ikut
			// ditandai published — notifikasinya hilang tanpa jejak.
			ids = append(ids, e.ID)
		}

		// Hanya event yang SUDAH berhasil di-enqueue yang ditandai.
		if err := repo.MarkPublished(ctx, ids); err != nil {
			return err
		}
		published = len(ids)
		return nil
	})
	if err != nil {
		return 0, err
	}
	return published, enqueueErr
}

func (r *OutboxRelay) publish(ctx context.Context, e model.OutboxEvent) error {
	job := model.NotificationJob{
		OutboxID: e.ID, EventType: e.EventType, Payload: e.Payload,
	}

	// asynq.NewTask menerima payload []byte — task di Redis hanyalah
	// sekumpulan byte. Struct job harus disusun menjadi JSON lebih dulu.
	payload, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("menyusun payload task: %w", err)
	}
	// Jenis task menentukan handler yang mengerjakannya: event kuota
	// punya handler sendiri karena diteruskan ke banyak penerima.
	taskType := model.TaskSendNotification
	if e.EventType == model.EventQuotaChanged {
		taskType = model.TaskQuotaChanged
	}
	task := asynq.NewTask(taskType, payload,
		// TaskID yang diturunkan dari id outbox membuat enqueue ulang
		// event yang sama ditolak Redis. Ini menutup celah ketika relay
		// mati SETELAH melakukan enqueue tetapi SEBELUM sempat commit.
		asynq.TaskID(fmt.Sprintf("outbox-%d", e.ID)),
		asynq.Queue(model.QueueNotifications),
		asynq.MaxRetry(5),
		asynq.Timeout(30*time.Second),
		// Task yang selesai disimpan 24 jam. Selama itu, TaskID-nya
		// tetap tercatat sehingga perlindungan di atas tetap berlaku.
		asynq.Retention(24*time.Hour),
	)

	_, err = r.queue.EnqueueContext(ctx, task)
	if errors.Is(err, asynq.ErrTaskIDConflict) {
		// Sudah pernah di-enqueue pada putaran sebelumnya. Bagi relay,
		// itu sama dengan berhasil.
		return nil
	}
	if err != nil {
		return fmt.Errorf("enqueue outbox %d: %w", e.ID, err)
	}
	return nil
}
