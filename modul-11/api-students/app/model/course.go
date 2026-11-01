package model

// SwapCourseRequest adalah body POST /enrollments/swap: pindah dari satu
// kelas ke kelas lain dalam satu langkah.
type SwapCourseRequest struct {
	FromCourseID int `json:"from_course_id" validate:"required,min=1"`
	ToCourseID   int `json:"to_course_id" validate:"required,min=1"`
}

// UpdateCourseQuotaRequest adalah body PATCH /courses/:id. Version adalah
// optimistic concurrency control: bila data sudah diubah pihak lain,
// permintaan ditolak agar perubahan tidak saling menimpa.
type UpdateCourseQuotaRequest struct {
	Quota   int `json:"quota" validate:"required,min=1,max=500"`
	Version int `json:"version" validate:"required,min=1"`
}
