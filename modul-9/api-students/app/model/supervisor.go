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

type Supervisor struct {
	ID              int       `json:"id"`
	Name            string    `json:"name"`
	Quota           int       `json:"quota"`
	SupervisedCount int       `json:"supervised_count"`
	Version         int       `json:"version"`
	CreatedAt       time.Time `json:"created_at"`
}

type SwapRequest struct {
	FromSupervisorID int `json:"from_supervisor_id" validate:"required,min=1"`
	ToSupervisorID   int `json:"to_supervisor_id" validate:"required,min=1"`
}

type UpdateQuotaRequest struct {
	Quota   int `json:"quota"   validate:"required,min=1,max=500"`
	Version int `json:"version" validate:"required,min=1"`
}
