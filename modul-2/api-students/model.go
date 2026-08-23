package main

import "time"

// Student adalah entitas utama API ini. Struct ini dibawa dari tugas
// modul 1 (soal struct Student), lalu ditambah NIM sebagai penanda unik
// dan CreatedAt supaya data bisa diurutkan berdasarkan waktu pembuatan.
// Tag json menentukan nama field saat dikirim sebagai JSON.
type Student struct {
	ID        int       `json:"id"`
	NIM       string    `json:"nim"`
	Name      string    `json:"name"`
	Grade     float64   `json:"grade"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
}

// Tiga method di bawah dibawa langsung dari tugas modul 1. Kini mereka
// dipanggil oleh handler PUT dan PATCH untuk mengubah isi data.

// UpdateGrade mengganti nilai mahasiswa.
func (s *Student) UpdateGrade(grade float64) {
	s.Grade = grade
}

// Activate mengubah status mahasiswa menjadi aktif.
func (s *Student) Activate() {
	s.IsActive = true
}

// Deactivate mengubah status mahasiswa menjadi tidak aktif.
func (s *Student) Deactivate() {
	s.IsActive = false
}

// CreateStudentRequest dipakai pada POST. Seluruh field wajib diisi.
// Grade bertipe pointer karena nilai 0 adalah nilai yang sah, sehingga
// "tidak dikirim" harus bisa dibedakan dari "dikirim bernilai nol".
// NIM dan Name cukup bertipe string biasa karena string kosong memang
// bukan nilai yang sah, jadi "" sudah cukup menjadi penanda tidak dikirim.
type CreateStudentRequest struct {
	NIM   string   `json:"nim"`
	Name  string   `json:"name"`
	Grade *float64 `json:"grade"`
}

// ReplaceStudentRequest dipakai pada PUT. PUT mengganti seluruh isi, jadi
// klien wajib mengirim semua field. Grade dan IsActive tetap pointer
// dengan alasan yang sama: 0 dan false adalah nilai sah, bukan tanda kosong.
type ReplaceStudentRequest struct {
	NIM      string   `json:"nim"`
	Name     string   `json:"name"`
	Grade    *float64 `json:"grade"`
	IsActive *bool    `json:"is_active"`
}

// PatchStudentRequest dipakai pada PATCH. Semua field bertipe pointer
// supaya field yang tidak dikirim (nil) bisa dibedakan dari yang dikirim,
// dan tidak ikut menimpa data lama. Inilah kegunaan pointer dari modul 1
// pada kasus sesungguhnya.
type PatchStudentRequest struct {
	NIM      *string  `json:"nim,omitempty"`
	Name     *string  `json:"name,omitempty"`
	Grade    *float64 `json:"grade,omitempty"`
	IsActive *bool    `json:"is_active,omitempty"`
}

// WebResponse adalah amplop baku untuk seluruh respons, termasuk yang
// gagal. omitempty membuat field yang tidak dipakai hilang dari JSON,
// jadi respons sukses tidak membawa "errors": null yang membingungkan.
type WebResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
	Meta    *Meta  `json:"meta,omitempty"`
	Errors  any    `json:"errors,omitempty"`
}

// Meta menemani respons berbentuk daftar supaya klien bisa menggambar
// navigasi halaman tanpa harus menebak.
type Meta struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

// ListQuery menampung query string yang sudah dibersihkan dan diberi
// nilai bawaan. Field bertipe pointer berarti filter tersebut opsional:
// nil artinya jangan menyaring.
type ListQuery struct {
	Page     int
	Limit    int
	Search   string
	Sort     string
	Order    string
	IsActive *bool
	MinGrade *float64
	MaxGrade *float64
}
