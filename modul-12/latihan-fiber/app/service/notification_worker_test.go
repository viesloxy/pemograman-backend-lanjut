package service

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/hibiken/asynq"

	"latihan-fiber/app/model"
	"latihan-fiber/app/repository"
)

// fakeUsers memenuhi repository.UserRepository dengan cara embedding
// interface yang di-embed dan bernilai nil, sehingga test akan PANIC bila
// kode yang diuji memanggilnya. Itu disengaja: panic tersebut memberi tahu
// bahwa kode memakai dependency di luar yang kita perkirakan.
type fakeUsers struct {
	repository.UserRepository
	users map[int]model.User
}

func (f *fakeUsers) FindByID(_ context.Context, id int) (model.User, error) {
	u, ok := f.users[id]
	if !ok {
		return model.User{}, repository.ErrNotFound
	}
	return u, nil
}

func newTask(t *testing.T, eventType string, event any) *asynq.Task {
	t.Helper()
	payload, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("menyusun payload event: %v", err)
	}
	job, err := json.Marshal(model.NotificationJob{OutboxID: 1, EventType: eventType, Payload: payload})
	if err != nil {
		t.Fatalf("menyusun job: %v", err)
	}
	return asynq.NewTask(model.TaskSendNotification, job)
}

// Seluruh skenario di bawah ini berhenti SEBELUM worker menyentuh
// transaction dan mailer. Karena itu tx dan mailer boleh nil: test ini
// murni unit test, tanpa database dan tanpa Redis.
func TestHandleSend_PermanentFailures(t *testing.T) {
	worker := NewNotificationWorker(
		&fakeUsers{users: map[int]model.User{7: {ID: 7, Username: "mhsuji", Email: "mhsuji@student.unair.ac.id"}}},
		nil, nil, nil,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)

	tests := []struct {
		name string
		task *asynq.Task
	}{
		{name: "payload task bukan JSON", task: asynq.NewTask(model.TaskSendNotification, []byte("bukan json"))},
		{name: "user sudah dihapus", task: newTask(t, model.EventEnrollmentCreated, model.EnrollmentEvent{UserID: 99})},
		{name: "jenis event tidak dikenal", task: newTask(t, "enrollment.unknown", model.EnrollmentEvent{UserID: 7})},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := worker.HandleSend(context.Background(), tt.task)
			// Asynq memeriksa SkipRetry dengan errors.Is — worker MEMBUNGKUS
			// SkipRetry dengan %w, tidak mengembalikannya apa adanya. Test
			// harus memeriksa dengan cara yang sama, bukan membandingkan
			// pointer dengan == atau !=.
			if !errors.Is(err, asynq.SkipRetry) {
				t.Errorf("error = %v, seharusnya membungkus asynq.SkipRetry", err)
			}
		})
	}
}