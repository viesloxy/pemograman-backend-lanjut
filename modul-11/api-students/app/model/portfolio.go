package model

import "time"

// Portfolio adalah bentuk portofolio mahasiswa yang dikenal seluruh aplikasi.
// Data mahasiswanya sendiri tetap di PostgreSQL; yang disimpan di MongoDB
// hanyalah dokumen portofolionya. Tidak ada tipe driver di sini — aturan yang
// sama dengan Review: ID proyek berupa string, urusan BSON ada di repository.
type Portfolio struct {
	StudentID    int          `json:"student_id"`
	Headline     string       `json:"headline"`
	Bio          string       `json:"bio"`
	Projects     []Project    `json:"projects"`
	Certificates []Certificate `json:"certificates"`
	CreatedAt    time.Time    `json:"created_at"`
	UpdatedAt    time.Time    `json:"updated_at"`
}

// Project disimpan DI DALAM dokumen portofolio (embedding), dibatasi 20.
type Project struct {
	ID           string   `json:"id"`
	Title        string   `json:"title"`
	Technologies []string `json:"technologies"`
	Link         string   `json:"link"`
	Year         int      `json:"year"`
}

type Certificate struct {
	Name    string `json:"name"`
	Issuer  string `json:"issuer"`
	IssuedAt string `json:"issued_at"`
}

type PutPortfolioRequest struct {
	Headline     string              `json:"headline"     validate:"required,max=120"`
	Bio          string              `json:"bio"          validate:"required,max=2000"`
	Projects     []ProjectRequest    `json:"projects"     validate:"max=20,dive"`
	Certificates []CertificateRequest `json:"certificates" validate:"max=50,dive"`
}

type ProjectRequest struct {
	Title        string   `json:"title"        validate:"required,min=3,max=120"`
	Technologies []string `json:"technologies" validate:"required,min=1,max=15,dive,min=1,max=40"`
	Link         string   `json:"link"         validate:"omitempty,url,max=200"`
	Year         int      `json:"year"         validate:"required,min=2000,max=2100"`
}

type CertificateRequest struct {
	Name     string `json:"name"      validate:"required,max=120"`
	Issuer   string `json:"issuer"    validate:"required,max=120"`
	IssuedAt string `json:"issued_at" validate:"required,max=40"`
}

type TechnologyCount struct {
	Technology string `json:"technology"`
	Total      int    `json:"total"`
}
