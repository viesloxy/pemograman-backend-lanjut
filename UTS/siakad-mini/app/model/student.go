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
	NIM         string   `json:"nim"`
	Nama        string   `json:"nama"`
	Email       string   `json:"email"`
	Prodi       string   `json:"prodi"`
	Angkatan    int      `json:"angkatan"`
	IPKTerakhir *float64 `json:"ipk_terakhir"`
}

type UpdateStudentRequest struct {
	Nama        string   `json:"nama"`
	Prodi       string   `json:"prodi"`
	Angkatan    int      `json:"angkatan"`
	IPKTerakhir *float64 `json:"ipk_terakhir"`
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
