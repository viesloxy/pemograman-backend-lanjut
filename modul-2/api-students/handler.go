package main

import (
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
)

// Data masih disimpan di memori; seluruh isinya hilang setiap kali server
// dimatikan. Basis data baru masuk pada modul 3.
//
// Fiber melayani banyak request secara bersamaan, jadi slice di bawah
// dijaga mutex supaya dua request yang menulis bersamaan tidak merusak
// data: RLock untuk yang hanya membaca, Lock untuk yang menulis.
var (
	mu       sync.RWMutex
	students []Student
	nextID   = 1
)

// ---------- Fungsi bantuan ----------
// Semua fungsi di bawah dipanggil ketika mutex sudah dipegang pemanggilnya,
// jadi tidak ada satu pun yang mengunci ulang.

func findStudentIndex(id int) int {
	for i := range students {
		if students[i].ID == id {
			return i
		}
	}
	return -1
}

// nimDipakai memeriksa keunikan NIM. Parameter kecualiID dipakai saat
// mengubah data, supaya NIM milik sendiri tidak dianggap bentrok.
func nimDipakai(nim string, kecualiID int) bool {
	for _, s := range students {
		if s.ID != kecualiID && s.NIM == nim {
			return true
		}
	}
	return false
}

// cocokPencarian mencari kata kunci pada nama tanpa membedakan huruf
// besar dan kecil.
func cocokPencarian(s Student, kata string) bool {
	return strings.Contains(strings.ToLower(s.Name), strings.ToLower(kata))
}

// paramID membaca :id dari jalur URL. Gagal di sini berarti permintaannya
// salah bentuk, jadi statusnya 400, bukan 404.
func paramID(c *fiber.Ctx) (int, bool) {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id < 1 {
		return 0, false
	}
	return id, true
}

// ---------- GET /api/v1/students ----------

func listStudents(c *fiber.Ctx) error {
	q := parseListQuery(c)

	mu.RLock()
	defer mu.RUnlock()

	// 1) Saring
	hasil := []Student{}
	for _, s := range students {
		if q.IsActive != nil && s.IsActive != *q.IsActive {
			continue
		}
		if q.MinGrade != nil && s.Grade < *q.MinGrade {
			continue
		}
		if q.MaxGrade != nil && s.Grade > *q.MaxGrade {
			continue
		}
		if q.Search != "" && !cocokPencarian(s, q.Search) {
			continue
		}
		hasil = append(hasil, s)
	}

	// 2) Urutkan. Nilai q.Sort sudah lolos daftar putih di parseListQuery,
	// jadi switch di bawah tidak mungkin menerima nama field asing.
	sort.SliceStable(hasil, func(i, j int) bool {
		var lebihKecil bool
		switch q.Sort {
		case "nim":
			lebihKecil = hasil[i].NIM < hasil[j].NIM
		case "name":
			lebihKecil = strings.ToLower(hasil[i].Name) < strings.ToLower(hasil[j].Name)
		case "grade":
			lebihKecil = hasil[i].Grade < hasil[j].Grade
		case "created_at":
			lebihKecil = hasil[i].CreatedAt.Before(hasil[j].CreatedAt)
		default:
			lebihKecil = hasil[i].ID < hasil[j].ID
		}
		if q.Order == "desc" {
			return !lebihKecil
		}
		return lebihKecil
	})

	// 3) Potong sesuai halaman. Urutan saring-urutkan-potong tidak boleh
	// ditukar, kalau tidak meta.total jadi bohong.
	total := len(hasil)
	totalPages := 0
	if total > 0 {
		totalPages = (total + q.Limit - 1) / q.Limit
	}
	mulai := (q.Page - 1) * q.Limit
	if mulai > total {
		mulai = total
	}
	akhir := mulai + q.Limit
	if akhir > total {
		akhir = total
	}

	return okList(c, "daftar mahasiswa berhasil diambil", hasil[mulai:akhir], &Meta{
		Page:       q.Page,
		Limit:      q.Limit,
		Total:      total,
		TotalPages: totalPages,
	})
}

// ---------- GET /api/v1/students/:id ----------

func getStudent(c *fiber.Ctx) error {
	id, valid := paramID(c)
	if !valid {
		return fail(c, fiber.StatusBadRequest, "id harus berupa angka positif")
	}

	mu.RLock()
	defer mu.RUnlock()

	i := findStudentIndex(id)
	if i == -1 {
		return fail(c, fiber.StatusNotFound, "mahasiswa tidak ditemukan")
	}
	return ok(c, "mahasiswa ditemukan", students[i])
}

// ---------- POST /api/v1/students ----------

func createStudent(c *fiber.Ctx) error {
	var req CreateStudentRequest
	if err := c.BodyParser(&req); err != nil {
		return fail(c, fiber.StatusBadRequest, "body harus berupa JSON yang valid")
	}

	req.NIM = strings.TrimSpace(req.NIM)
	req.Name = strings.TrimSpace(req.Name)

	if errs := validasiIsi(req.NIM, req.Name, req.Grade, true); len(errs) > 0 {
		return failValidation(c, errs)
	}

	mu.Lock()
	defer mu.Unlock()

	// NIM ganda bukan salah bentuk dan bukan salah isi, melainkan bentrok
	// dengan data yang sudah ada, jadi statusnya 409.
	if nimDipakai(req.NIM, 0) {
		return fail(c, fiber.StatusConflict, "NIM sudah terdaftar")
	}

	baru := Student{
		ID:        nextID,
		NIM:       req.NIM,
		Name:      req.Name,
		Grade:     *req.Grade,
		CreatedAt: time.Now(),
	}
	baru.Activate() // method dari modul 1: mahasiswa baru selalu berstatus aktif
	students = append(students, baru)
	nextID++

	return created(c, "mahasiswa berhasil ditambahkan", baru,
		"/api/v1/students/"+strconv.Itoa(baru.ID))
}

// ---------- PUT /api/v1/students/:id ----------

// replaceStudent mengganti SELURUH isi. Field yang tidak dikirim dianggap
// dikosongkan, karena itu semuanya wajib ada dan ditolak 422 bila kurang.
func replaceStudent(c *fiber.Ctx) error {
	id, valid := paramID(c)
	if !valid {
		return fail(c, fiber.StatusBadRequest, "id harus berupa angka positif")
	}

	var req ReplaceStudentRequest
	if err := c.BodyParser(&req); err != nil {
		return fail(c, fiber.StatusBadRequest, "body harus berupa JSON yang valid")
	}

	req.NIM = strings.TrimSpace(req.NIM)
	req.Name = strings.TrimSpace(req.Name)

	errs := validasiIsi(req.NIM, req.Name, req.Grade, true)
	if req.IsActive == nil {
		errs["is_active"] = "wajib dikirim pada PUT"
	}
	if len(errs) > 0 {
		return failValidation(c, errs)
	}

	mu.Lock()
	defer mu.Unlock()

	i := findStudentIndex(id)
	if i == -1 {
		return fail(c, fiber.StatusNotFound, "mahasiswa tidak ditemukan")
	}
	if nimDipakai(req.NIM, id) {
		return fail(c, fiber.StatusConflict, "NIM sudah dipakai mahasiswa lain")
	}

	// ID dan CreatedAt sengaja tidak ikut diganti: keduanya milik server.
	// Perubahan nilai dan status memakai method dari modul 1.
	students[i].NIM = req.NIM
	students[i].Name = req.Name
	students[i].UpdateGrade(*req.Grade)
	if *req.IsActive {
		students[i].Activate()
	} else {
		students[i].Deactivate()
	}

	return ok(c, "data mahasiswa berhasil diganti seluruhnya", students[i])
}

// ---------- PATCH /api/v1/students/:id ----------

// patchStudent hanya menyentuh field yang benar-benar dikirim klien.
// Empat blok if di bawah adalah wujud nyata perbedaan PATCH dan PUT:
// field yang nil dilewati, isinya yang lama tidak tersentuh.
func patchStudent(c *fiber.Ctx) error {
	id, valid := paramID(c)
	if !valid {
		return fail(c, fiber.StatusBadRequest, "id harus berupa angka positif")
	}

	var req PatchStudentRequest
	if err := c.BodyParser(&req); err != nil {
		return fail(c, fiber.StatusBadRequest, "body harus berupa JSON yang valid")
	}

	if req.NIM == nil && req.Name == nil && req.Grade == nil && req.IsActive == nil {
		return fail(c, fiber.StatusBadRequest, "tidak ada field yang diubah")
	}

	// Hanya field yang dikirim yang divalidasi, jadi wajibLengkap false.
	nim, name := "", ""
	if req.NIM != nil {
		nim = strings.TrimSpace(*req.NIM)
	}
	if req.Name != nil {
		name = strings.TrimSpace(*req.Name)
	}
	errs := validasiIsi(nim, name, req.Grade, false)
	if req.NIM != nil && nim == "" {
		errs["nim"] = "tidak boleh dikosongkan"
	}
	if req.Name != nil && name == "" {
		errs["name"] = "tidak boleh dikosongkan"
	}
	if len(errs) > 0 {
		return failValidation(c, errs)
	}

	mu.Lock()
	defer mu.Unlock()

	i := findStudentIndex(id)
	if i == -1 {
		return fail(c, fiber.StatusNotFound, "mahasiswa tidak ditemukan")
	}
	if req.NIM != nil && nimDipakai(nim, id) {
		return fail(c, fiber.StatusConflict, "NIM sudah dipakai mahasiswa lain")
	}

	if req.NIM != nil {
		students[i].NIM = nim
	}
	if req.Name != nil {
		students[i].Name = name
	}
	if req.Grade != nil {
		students[i].UpdateGrade(*req.Grade) // method dari modul 1
	}
	if req.IsActive != nil {
		if *req.IsActive {
			students[i].Activate() // method dari modul 1
		} else {
			students[i].Deactivate() // method dari modul 1
		}
	}

	return ok(c, "data mahasiswa berhasil diperbarui sebagian", students[i])
}

// ---------- DELETE /api/v1/students/:id ----------

func deleteStudent(c *fiber.Ctx) error {
	id, valid := paramID(c)
	if !valid {
		return fail(c, fiber.StatusBadRequest, "id harus berupa angka positif")
	}

	mu.Lock()
	defer mu.Unlock()

	i := findStudentIndex(id)
	if i == -1 {
		return fail(c, fiber.StatusNotFound, "mahasiswa tidak ditemukan")
	}
	students = append(students[:i], students[i+1:]...)

	return noContent(c) // 204: berhasil, dan memang tidak ada yang perlu dikirim
}
