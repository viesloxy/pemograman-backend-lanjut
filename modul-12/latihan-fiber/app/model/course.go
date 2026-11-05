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