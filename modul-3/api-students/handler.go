package main

import (
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"

	"api-students/app/model"
)

var (
	mu       sync.RWMutex
	students []model.Student
	nextID   = 1
)

func findStudentIndex(id int) int {
	for i := range students {
		if students[i].ID == id {
			return i
		}
	}
	return -1
}

func nimDipakai(nim string, kecualiID int) bool {
	for _, s := range students {
		if s.ID != kecualiID && s.NIM == nim {
			return true
		}
	}
	return false
}

func cocokPencarian(s model.Student, kata string) bool {
	return strings.Contains(strings.ToLower(s.Name), strings.ToLower(kata))
}

func paramID(c *fiber.Ctx) (int, bool) {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id < 1 {
		return 0, false
	}
	return id, true
}

func listStudents(c *fiber.Ctx) error {
	q := parseListQuery(c)

	mu.RLock()
	defer mu.RUnlock()

	hasil := []model.Student{}
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

	total := len(hasil)
	totalPages := 0
	if total > 0 {
		totalPages = (total + q.Limit - 1) / q.Limit
	}
	mulai := q.Offset()
	if mulai > total {
		mulai = total
	}
	akhir := mulai + q.Limit
	if akhir > total {
		akhir = total
	}

	return okList(c, "daftar mahasiswa berhasil diambil", hasil[mulai:akhir], &model.Meta{
		Page:       q.Page,
		Limit:      q.Limit,
		Total:      total,
		TotalPages: totalPages,
	})
}

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

func createStudent(c *fiber.Ctx) error {
	var req model.CreateStudentRequest
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

	if nimDipakai(req.NIM, 0) {
		return fail(c, fiber.StatusConflict, "NIM sudah terdaftar")
	}

	baru := model.Student{
		ID:        nextID,
		NIM:       req.NIM,
		Name:      req.Name,
		Grade:     *req.Grade,
		CreatedAt: time.Now(),
	}
	baru.Activate()
	students = append(students, baru)
	nextID++

	return created(c, "mahasiswa berhasil ditambahkan", baru,
		"/api/v1/students/"+strconv.Itoa(baru.ID))
}

func replaceStudent(c *fiber.Ctx) error {
	id, valid := paramID(c)
	if !valid {
		return fail(c, fiber.StatusBadRequest, "id harus berupa angka positif")
	}

	var req model.ReplaceStudentRequest
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

func patchStudent(c *fiber.Ctx) error {
	id, valid := paramID(c)
	if !valid {
		return fail(c, fiber.StatusBadRequest, "id harus berupa angka positif")
	}

	var req model.PatchStudentRequest
	if err := c.BodyParser(&req); err != nil {
		return fail(c, fiber.StatusBadRequest, "body harus berupa JSON yang valid")
	}

	if req.NIM == nil && req.Name == nil && req.Grade == nil && req.IsActive == nil {
		return fail(c, fiber.StatusBadRequest, "tidak ada field yang diubah")
	}

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
		students[i].UpdateGrade(*req.Grade)
	}
	if req.IsActive != nil {
		if *req.IsActive {
			students[i].Activate()
		} else {
			students[i].Deactivate()
		}
	}

	return ok(c, "data mahasiswa berhasil diperbarui sebagian", students[i])
}

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

	return noContent(c)
}
