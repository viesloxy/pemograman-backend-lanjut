package model

import "time"

type Student struct {
	ID          int        `json:"id"`
	UserID      int        `json:"-"`
	NIM         string     `json:"nim"`
	Nama        string     `json:"nama"`
	Prodi       string     `json:"prodi"`
	Angkatan    int        `json:"angkatan"`
	IPKTerakhir *float64   `json:"ipk_terakhir"`
	DeletedAt   *time.Time `json:"-"`
	CreatedAt   time.Time  `json:"created_at"`
}

type CreateStudentRequest struct {
	NIM         string   `json:"nim" validate:"required,nim"`
	Nama        string   `json:"nama" validate:"required,min=3,max=150"`
	Email       string   `json:"email" validate:"required,max=255,email"`
	Prodi       string   `json:"prodi" validate:"required,max=100"`
	Angkatan    int      `json:"angkatan" validate:"required,angkatan"`
	IPKTerakhir *float64 `json:"ipk_terakhir" validate:"omitnil,gte=0,lte=4"`
}

type UpdateStudentRequest struct {
	Nama        string   `json:"nama" validate:"required,min=3,max=150"`
	Prodi       string   `json:"prodi" validate:"required,max=100"`
	Angkatan    int      `json:"angkatan" validate:"required,angkatan"`
	IPKTerakhir *float64 `json:"ipk_terakhir" validate:"omitnil,gte=0,lte=4"`
}

type ListStudentQuery struct {
	Page     int
	PerPage  int
	Prodi    string
	Angkatan int // 0 berarti tidak difilter
	Search   string
	Sort     string // "nama" atau "-ipk_terakhir"
}

func (q ListStudentQuery) Offset() int {
	return (q.Page - 1) * q.PerPage
}

// StudentDetailResponse adalah isi endpoint detail mahasiswa:
// data mahasiswa, daftar mata kuliah yang diambil, total SKS,
// dan batas SKS menurut IPK-nya.
type StudentDetailResponse struct {
	Student
	MataKuliah []EnrollmentCourse `json:"mata_kuliah"`
	TotalSKS   int                `json:"total_sks"`
	BatasSKS   int                `json:"batas_sks"`
}
