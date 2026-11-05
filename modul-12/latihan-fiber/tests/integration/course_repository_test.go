//go:build integration

package integration_test

import (
	"context"
	"errors"
	"testing"

	"latihan-fiber/app/model"
	"latihan-fiber/app/repository"
)

func TestIncrementEnrolled_StopsAtQuota(t *testing.T) {
	ctx := context.Background()
	course := seedCourse(t, 3, 2)
	repo := repository.NewCourseRepository(pool)

	for i := 1; i <= 2; i++ {
		if _, err := repo.IncrementEnrolled(ctx, course.ID); err != nil {
			t.Fatalf("kursi ke-%d: error tidak diharapkan: %v", i, err)
		}
	}

	_, err := repo.IncrementEnrolled(ctx, course.ID)
	if !errors.Is(err, repository.ErrCourseFull) {
		t.Fatalf("kursi ke-3: error = %v, seharusnya ErrCourseFull", err)
	}

	got, err := repo.FindByID(ctx, course.ID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if got.EnrolledCount != 2 {
		t.Errorf("enrolled_count = %d, seharusnya 2", got.EnrolledCount)
	}
}

func TestInsertEnrollment_Duplicate(t *testing.T) {
	ctx := context.Background()
	course := seedCourse(t, 3, 40)
	student := seedStudent(t)
	repo := repository.NewCourseRepository(pool)

	if err := repo.InsertEnrollment(ctx, course.ID, student.ID); err != nil {
		t.Fatalf("pendaftaran pertama: %v", err)
	}
	err := repo.InsertEnrollment(ctx, course.ID, student.ID)
	if !errors.Is(err, repository.ErrDuplicate) {
		t.Errorf("pendaftaran kedua: error = %v, seharusnya ErrDuplicate", err)
	}
}

// Transaction yang gagal harus membatalkan SEMUA langkahnya, termasuk
// catatan di outbox. Inilah jaminan yang menjadi dasar pola outbox Modul 11.
func TestWithTx_RollsBackEveryStep(t *testing.T) {
	ctx := context.Background()
	course := seedCourse(t, 3, 40)
	student := seedStudent(t)

	courses := repository.NewCourseRepository(pool)
	outbox := repository.NewOutboxRepository(pool)
	txm := repository.NewTxManager(pool)

	errStop := errors.New("gagal yang disengaja")
	err := txm.WithTx(ctx, func(tx repository.DBTX) error {
		if _, err := courses.WithTx(tx).IncrementEnrolled(ctx, course.ID); err != nil {
			return err
		}
		if err := outbox.WithTx(tx).Insert(ctx, model.EventEnrollmentCreated,
			model.EnrollmentEvent{UserID: student.ID, CourseID: course.ID}); err != nil {
			return err
		}
		return errStop
	})
	if !errors.Is(err, errStop) {
		t.Fatalf("WithTx: error = %v, seharusnya errStop", err)
	}

	got, _ := courses.FindByID(ctx, course.ID)
	if got.EnrolledCount != 0 {
		t.Errorf("enrolled_count = %d, seharusnya 0 setelah rollback", got.EnrolledCount)
	}
	var events int
	_ = pool.QueryRow(ctx, `SELECT COUNT(*) FROM outbox WHERE (payload->>'user_id')::INT = $1`,
		student.ID).Scan(&events)
	if events != 0 {
		t.Errorf("baris outbox = %d, seharusnya 0 setelah rollback", events)
	}
}

func TestClaim_OnlyOnce(t *testing.T) {
	ctx := context.Background()
	student := seedStudent(t)
	repo := repository.NewNotificationRepository(pool)

	var outboxID int64
	err := pool.QueryRow(ctx,
		`INSERT INTO outbox (event_type, payload) VALUES ($1, jsonb_build_object('user_id', $2::INT)) RETURNING id`,
		model.EventEnrollmentCreated, student.ID).Scan(&outboxID)
	if err != nil {
		t.Fatalf("seed outbox: %v", err)
	}

	n := model.Notification{OutboxID: outboxID, UserID: student.ID, Channel: "email", Subject: "s", Body: "b"}
	first, err := repo.Claim(ctx, n)
	if err != nil || !first {
		t.Fatalf("claim pertama = %v, %v; seharusnya true, nil", first, err)
	}
	second, err := repo.Claim(ctx, n)
	if err != nil || second {
		t.Errorf("claim kedua = %v, %v; seharusnya false, nil", second, err)
	}
}