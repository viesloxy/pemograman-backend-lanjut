package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/hibiken/asynq"

	"api-students/app/model"
	"api-students/app/repository"
	"api-students/helper"
)

// NotificationWorker mengerjakan task "notification:send".
//
// Bagi worker, Asynq adalah framework-nya — sama seperti Fiber bagi API.
// Karena itu method di sini menerima *asynq.Task, seperti method service
// lain menerima *fiber.Ctx.
type NotificationWorker struct {
	users         repository.StudentRepository
	courses       repository.CourseRepository
	notifications repository.NotificationRepository
	tx            *repository.TxManager
	mailer        helper.Mailer
	logger        *slog.Logger
}

func NewNotificationWorker(
	users repository.StudentRepository,
	courses repository.CourseRepository,
	notifications repository.NotificationRepository,
	tx *repository.TxManager,
	mailer helper.Mailer,
	logger *slog.Logger,
) *NotificationWorker {
	return &NotificationWorker{users: users, courses: courses,
		notifications: notifications, tx: tx, mailer: mailer, logger: logger}
}

// HandleSend mengembalikan error untuk meminta Asynq melakukan retry nanti,
// atau error yang membungkus asynq.SkipRetry untuk menyatakan "jangan di-retry,
// task ini tidak mungkin berhasil".
//
// Asynq menjamin setiap task dijalankan PALING SEDIKIT satu kali, bukan
// TEPAT satu kali. Task yang sama bisa datang dua kali untuk, misalnya bila
// worker mati setelah mengirim email tetapi sebelum sempat melapor ke Redis.
// Karena itu handler ini harus idempoten: menjalankannya dua kali untuk
// task yang sama harus berakibat sama dengan menjalankannya sekali.
func (w *NotificationWorker) HandleSend(ctx context.Context, t *asynq.Task) error {
	var job model.NotificationJob
	if err := json.Unmarshal(t.Payload(), &job); err != nil {
		return fmt.Errorf("%w: payload task rusak: %v", asynq.SkipRetry, err)
	}
	var event model.EnrollmentEvent
	if err := json.Unmarshal(job.Payload, &event); err != nil {
		return fmt.Errorf("%w: payload event %d rusak: %v", asynq.SkipRetry, job.OutboxID, err)
	}

	user, err := w.users.FindByID(ctx, event.UserID)
	if errors.Is(err, repository.ErrNotFound) {
		// Akunnya sudah dihapus. Melakukan retry tidak akan memunculkannya.
		return fmt.Errorf("%w: user %d tidak ditemukan", asynq.SkipRetry, event.UserID)
	}
	if err != nil {
		return err // database sedang bermasalah: layak di-retry
	}

	msg, err := BuildMessage(job.EventType, event, user)
	if err != nil {
		return fmt.Errorf("%w: %v", asynq.SkipRetry, err)
	}

	sent, err := w.deliverOnce(ctx, job.OutboxID, user, msg)
	if err != nil {
		return err
	}

	retry, _ := asynq.GetRetryCount(ctx)
	w.logger.Info("notification_processed",
		slog.Int64("outbox_id", job.OutboxID),
		slog.String("event", job.EventType),
		slog.Int("retry", retry),
		slog.Bool("already_sent", !sent))
	return nil
}

// deliverOnce meng-claim lalu mengirim pesan untuk SATU penerima di dalam
// SATU transaction (aturan yang sama dengan HandleSend versi satu penerima).
// Mengembalikan true bila pengiriman benar-benar terjadi, false bila claim
// menunjukkan email ini sudah pernah terkirim.
func (w *NotificationWorker) deliverOnce(ctx context.Context, outboxID int64, user model.Student, msg model.Message) (bool, error) {
	sent := false
	err := w.tx.WithTxOnce(ctx, func(tx repository.DBTX) error {
		// Claim dan pengiriman berada di dalam SATU transaction:
		//   - pengiriman gagal  -> claim ikut dibatalkan; attempt
		//     berikutnya masih dapat mengirim;
		//   - pengiriman sukses -> claim di-commit -> attempt berikutnya
		//     melihat claim ini dan tidak mengirim lagi.
		// PERNAYAAN: repo notifikasi WAJIB di-WithTx(tx). Bila tidak, claim
		// langsung commit di koneksi lain: bila pengiriman gagal, claimnya
		// tetap ada dan attempt berikutnya mengira email sudah terkirim.
		claimed, err := w.notifications.WithTx(tx).Claim(ctx, model.Notification{
			OutboxID: outboxID, UserID: user.ID, Channel: "email",
			Subject: msg.Subject, Body: msg.Body,
		})
		if err != nil {
			return err
		}
		if !claimed {
			return nil
		}
		sent = true
		return w.mailer.Send(ctx, msg)
	})
	return sent, err
}

// HandleQuotaChanged mengerjakan event course.quota_changed (D.2): satu
// event untuk SELURUH peserta kelas.
//
// Keputusan perancangan (jawaban D.2.3): task TIDAK dipecah per mahasiswa
// oleh relay; worker yang memecahnya saat mengerjakan. Setiap penerima
// di-claim terpisah dengan key (outbox_id, user_id), masing-masing dalam
// transaction sendiri sehingga koneksi database tidak dipegang lama.
// Bila email ke-7 dari 40 gagal sementara, task di-retry dan attempt
// berikutnya hanya mengirim yang BELUM ter-claim — email ke-1 sampai ke-6
// tidak dikirim ulang. Kegagalan PERMANEN satu alamat tidak meng-archive
// task: penerima itu dilewati dan dicatat, sisanya tetap terkirim — lebih
// adil daripada menggagalkan 39 orang karena satu alamat salah.
func (w *NotificationWorker) HandleQuotaChanged(ctx context.Context, t *asynq.Task) error {
	var job model.NotificationJob
	if err := json.Unmarshal(t.Payload(), &job); err != nil {
		return fmt.Errorf("%w: payload task rusak: %v", asynq.SkipRetry, err)
	}
	var event model.QuotaChangedEvent
	if err := json.Unmarshal(job.Payload, &event); err != nil {
		return fmt.Errorf("%w: payload event %d rusak: %v", asynq.SkipRetry, job.OutboxID, err)
	}

	participants, err := w.courses.FindParticipantIDs(ctx, event.CourseID)
	if err != nil {
		return err // database sedang bermasalah: layak di-retry
	}

	skipped := 0
	for _, pid := range participants {
		user, err := w.users.FindByID(ctx, pid)
		if errors.Is(err, repository.ErrNotFound) {
			// Akun dihapus setelah event dicatat: tidak ada yang bisa
			// dikirim, dan meng-archive seluruh task karenanya salah.
			skipped++
			w.logger.Warn("quota_recipient_skipped",
				slog.Int64("outbox_id", job.OutboxID), slog.Int("user_id", pid))
			continue
		}
		if err != nil {
			return err
		}

		msg, err := BuildQuotaMessage(event, user)
		if err != nil {
			return fmt.Errorf("%w: %v", asynq.SkipRetry, err)
		}

		sent, err := w.deliverOnce(ctx, job.OutboxID, user, msg)
		if errors.Is(err, helper.ErrPermanentDelivery) {
			// Alamat tujuan tidak ada: lewati penerima ini, jangan
			// gambunkan penerima lain.
			skipped++
			w.logger.Warn("quota_recipient_skipped",
				slog.Int64("outbox_id", job.OutboxID), slog.Int("user_id", pid),
				slog.String("error", err.Error()))
			continue
		}
		if err != nil {
			// Kegagalan sementara: hentikan; Asynq me-retry task ini dan
			// penerima yang sudah ter-claim tidak akan dikirim ulang.
			return err
		}
		if !sent {
			skipped++
		}
	}

	retry, _ := asynq.GetRetryCount(ctx)
	w.logger.Info("quota_changed_processed",
		slog.Int64("outbox_id", job.OutboxID),
		slog.Int("recipients", len(participants)),
		slog.Int("skipped", skipped),
		slog.Int("retry", retry))
	return nil
}
