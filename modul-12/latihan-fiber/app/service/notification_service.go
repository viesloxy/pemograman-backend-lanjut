package service

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/hibiken/asynq"

	"latihan-fiber/app/model"
	"latihan-fiber/app/repository"
	"latihan-fiber/helper"
)

// QueueInspector adalah kemampuan membaca keadaan queue.
// *asynq.Inspector memenuhi interface ini.
type QueueInspector interface {
	GetQueueInfo(queue string) (*asynq.QueueInfo, error)
}

// NotificationService melayani endpoint HTTP seputar notifikasi.
type NotificationService struct {
	notifications repository.NotificationRepository
	outbox        repository.OutboxRepository
	inspector     QueueInspector
}

func NewNotificationService(
	notifications repository.NotificationRepository,
	outbox repository.OutboxRepository,
	inspector QueueInspector,
) *NotificationService {
	return &NotificationService{notifications: notifications, outbox: outbox, inspector: inspector}
}

// ---------- GET /notifications/mine ----------

// Mine godoc
//
// @Summary  Notifikasi milik mahasiswa yang sedang login
// @Tags     notifications
// @Produce  json
// @Success  200  {object}  model.WebResponse{data=[]model.Notification}
// @Failure  401  {object}  model.ErrorResponse
// @Security BearerAuth
// @Router   /notifications/mine [get]
func (s *NotificationService) Mine(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	current, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized("belum terautentikasi")
	}

	list, err := s.notifications.FindByUser(ctx, current.UserID, 20)
	if err != nil {
		return helper.Internal(err)
	}
	return helper.Success(c, fiber.StatusOK, "notifikasi berhasil diambil", list)
}

// ---------- GET /admin/queue ----------
// Dijaga middleware dengan permission queue:monitor.

// Monitor godoc
//
// @Summary      Keadaan outbox dan queue (admin)
// @Description  Membutuhkan permission queue:monitor. Tetap menjawab 200 saat Redis mati; field queue_error menjelaskan sebabnya.
// @Tags         admin
// @Produce      json
// @Success      200  {object}  model.WebResponse{data=model.QueueStats}
// @Failure      403  {object}  model.ErrorResponse
// @Security     BearerAuth
// @Router       /admin/queue [get]
func (s *NotificationService) Monitor(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	pending, published, oldest, err := s.outbox.Stats(ctx)
	if err != nil {
		return helper.Internal(err)
	}
	sent, err := s.notifications.Count(ctx)
	if err != nil {
		return helper.Internal(err)
	}

	stats := model.QueueStats{
		OutboxPending:     pending,
		OutboxPublished:   published,
		OldestPendingAge:  oldest.Round(time.Millisecond).String(),
		NotificationsSent: sent,
	}

	// Redis yang mati tidak boleh membuat endpoint ini gagal seluruhnya.
	// Justru saat Redis mati, admin paling membutuhkan angka outbox_pending
	// untuk melihat berapa notifikasi yang sedang tertahan.
	info, err := s.inspector.GetQueueInfo(model.QueueNotifications)
	if err != nil {
		stats.QueueError = "queue tidak dapat dibaca: Redis tidak dapat dihubungi atau queue belum pernah dipakai"
	} else {
		stats.Queue = &model.QueueInfo{
			Name: info.Queue, Pending: info.Pending, Active: info.Active,
			Scheduled: info.Scheduled, Retry: info.Retry, Archived: info.Archived,
			Completed: info.Completed, Processed: info.Processed, Failed: info.Failed,
		}
	}
	return helper.Success(c, fiber.StatusOK, "keadaan queue berhasil diambil", stats)
}