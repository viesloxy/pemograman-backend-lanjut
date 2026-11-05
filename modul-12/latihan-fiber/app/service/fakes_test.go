package service_test

import (
	"context"
	"strconv"
	"sync"
	"time"

	"latihan-fiber/app/model"
	"latihan-fiber/app/repository"
)

// Pastikan di waktu compile bahwa fake memenuhi interface yang ditiru.
// Bila suatu hari interface berubah, baris ini yang pertama kali gagal,
// bukan test yang memakainya.
var (
	_ repository.ReviewRepository = (*fakeReviewRepo)(nil)
	_ repository.CourseRepository = (*fakeCourseRepo)(nil)
)

// fakeReviewRepo menyimpan ulasan di memory, menggantikan MongoDB.
//
// Mutex diperlukan karena Fiber melayani request di goroutine yang berbeda.
type fakeReviewRepo struct {
	repository.ReviewRepository // method lain: panic bila terpanggil

	mu      sync.Mutex
	reviews []model.Review
}

func newFakeReviewRepo() *fakeReviewRepo {
	return &fakeReviewRepo{}
}

func (f *fakeReviewRepo) Create(_ context.Context, r model.Review) (model.Review, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Meniru unique index course_user_unique di MongoDB.
	for _, existing := range f.reviews {
		if existing.CourseID == r.CourseID && existing.UserID == r.UserID {
			return model.Review{}, repository.ErrDuplicate
		}
	}
	r.ID = strconv.Itoa(len(f.reviews) + 1)
	r.CreatedAt = time.Now()
	r.UpdatedAt = r.CreatedAt
	if r.Tags == nil {
		r.Tags = []string{}
	}
	r.Replies = []model.Reply{}
	f.reviews = append(f.reviews, r)
	return r, nil
}

// FindByCourse menerima ReviewQuery — persis seperti interface aslinya —
// bukan id kelas secara langsung. Bila tanda tangannya berbeda, method ini
// TIDAK menimpa method pada interface yang di-embed, dan ReviewService akan
// memanggil method nil dari embed (panic).
func (f *fakeReviewRepo) FindByCourse(_ context.Context, q model.ReviewQuery) ([]model.Review, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	result := []model.Review{}
	for i := len(f.reviews) - 1; i >= 0; i-- {
		if f.reviews[i].CourseID == q.CourseID {
			result = append(result, f.reviews[i])
		}
	}
	return result, nil
}

// fakeCourseRepo hanya mengisi dua method yang dipakai ReviewService.
type fakeCourseRepo struct {
	repository.CourseRepository

	courses  map[int]model.Course
	enrolled map[[2]int]bool // {courseID, userID}
}

func (f *fakeCourseRepo) FindByID(_ context.Context, id int) (model.Course, error) {
	c, ok := f.courses[id]
	if !ok {
		return model.Course{}, repository.ErrNotFound
	}
	return c, nil
}

func (f *fakeCourseRepo) IsEnrolled(_ context.Context, courseID, userID int) (bool, error) {
	return f.enrolled[[2]int{courseID, userID}], nil
}