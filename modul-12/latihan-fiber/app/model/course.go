package model

import "time"

type Course struct {
	ID            int       `json:"id"`
	Code          string    `json:"code"`
	Name          string    `json:"name"`
	Credits       int       `json:"credits"`
	Quota         int       `json:"quota"`
	EnrolledCount int       `json:"enrolled_count"`
	Version       int       `json:"version"`
	CreatedAt     time.Time `json:"created_at"`
}

// SwapRequest dipakai endpoint pindah kelas.
type SwapRequest struct {
	FromCourseID int `json:"from_course_id" validate:"required,min=1"`
	ToCourseID   int `json:"to_course_id"   validate:"required,min=1"`
}

// UpdateQuotaRequest membawa version yang terakhir DILIHAT client.
type UpdateQuotaRequest struct {
	Quota   int `json:"quota"   validate:"required,min=1,max=500"`
	Version int `json:"version" validate:"required,min=1"`
}

// MyCourses adalah isi field data pada GET /courses/mine.
//
// Sebelumnya response ini disusun dengan fiber.Map. Hasil JSON-nya sama
// persis, tetapi map tidak memiliki nama dan bentuk yang dapat dibaca swag,
// sehingga dokumentasi API tidak dapat menjelaskan isinya.
type MyCourses struct {
	Courses      []Course `json:"courses"`
	TotalCredits int      `json:"total_credits"`
	MaxCredits   int      `json:"max_credits"`
}