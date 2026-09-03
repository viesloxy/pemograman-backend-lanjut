package main

import (
	"errors"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"

	"api-students/app/model"
	"api-students/app/repository"
)

type StudentHandler struct {
	repo repository.StudentRepository
}

func NewStudentHandler(repo repository.StudentRepository) *StudentHandler {
	return &StudentHandler{repo: repo}
}

func terjemahkanError(c *fiber.Ctx, err error, pesanUmum string) error {
	switch {
	case errors.Is(err, repository.ErrNotFound):
		return fail(c, fiber.StatusNotFound, "mahasiswa tidak ditemukan")
	case errors.Is(err, repository.ErrDuplicate):
		return fail(c, fiber.StatusConflict, "NIM sudah terdaftar")
	default:
		return fail(c, fiber.StatusInternalServerError, pesanUmum)
	}
}

func paramID(c *fiber.Ctx) (int, bool) {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id < 1 {
		return 0, false
	}
	return id, true
}

func (h *StudentHandler) List(c *fiber.Ctx) error {
	ctx, cancel := reqCtx(c)
	defer cancel()

	q := parseListQuery(c)

	daftar, total, err := h.repo.FindAll(ctx, q)
	if err != nil {
		return fail(c, fiber.StatusInternalServerError, "gagal mengambil data mahasiswa")
	}

	totalPages := 0
	if q.Limit > 0 {
		totalPages = (total + q.Limit - 1) / q.Limit
	}

	return okList(c, "daftar mahasiswa berhasil diambil", daftar, &model.Meta{
		Page: q.Page, Limit: q.Limit, Total: total, TotalPages: totalPages,
	})
}

func (h *StudentHandler) Get(c *fiber.Ctx) error {
	ctx, cancel := reqCtx(c)
	defer cancel()

	id, valid := paramID(c)
	if !valid {
		return fail(c, fiber.StatusBadRequest, "id harus berupa angka positif")
	}

	s, err := h.repo.FindByID(ctx, id)
	if err != nil {
		return terjemahkanError(c, err, "gagal mengambil data mahasiswa")
	}
	return ok(c, "mahasiswa ditemukan", s)
}

func (h *StudentHandler) Create(c *fiber.Ctx) error {
	ctx, cancel := reqCtx(c)
	defer cancel()

	var req model.CreateStudentRequest
	if err := c.BodyParser(&req); err != nil {
		return fail(c, fiber.StatusBadRequest, "body harus berupa JSON yang valid")
	}

	req.NIM = strings.TrimSpace(req.NIM)
	req.Name = strings.TrimSpace(req.Name)

	if errs := validasiIsi(req.NIM, req.Name, req.Grade, true); len(errs) > 0 {
		return failValidation(c, errs)
	}

	baru := model.Student{
		NIM:   req.NIM,
		Name:  req.Name,
		Grade: *req.Grade,
	}
	baru.Activate()

	hasil, err := h.repo.Create(ctx, baru)
	if err != nil {
		return terjemahkanError(c, err, "gagal menyimpan mahasiswa")
	}

	return created(c, "mahasiswa berhasil ditambahkan", hasil,
		"/api/v1/students/"+strconv.Itoa(hasil.ID))
}

func (h *StudentHandler) Replace(c *fiber.Ctx) error {
	ctx, cancel := reqCtx(c)
	defer cancel()

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

	s := model.Student{
		ID:    id,
		NIM:   req.NIM,
		Name:  req.Name,
		Grade: *req.Grade,
	}
	if *req.IsActive {
		s.Activate()
	} else {
		s.Deactivate()
	}

	hasil, err := h.repo.Update(ctx, s)
	if err != nil {
		return terjemahkanError(c, err, "gagal memperbarui mahasiswa")
	}
	return ok(c, "data mahasiswa berhasil diganti seluruhnya", hasil)
}

func (h *StudentHandler) Patch(c *fiber.Ctx) error {
	ctx, cancel := reqCtx(c)
	defer cancel()

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

	saatIni, err := h.repo.FindByID(ctx, id)
	if err != nil {
		return terjemahkanError(c, err, "gagal mengambil data mahasiswa")
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

	if req.NIM != nil {
		saatIni.NIM = nim
	}
	if req.Name != nil {
		saatIni.Name = name
	}
	if req.Grade != nil {
		saatIni.UpdateGrade(*req.Grade)
	}
	if req.IsActive != nil {
		if *req.IsActive {
			saatIni.Activate()
		} else {
			saatIni.Deactivate()
		}
	}

	hasil, err := h.repo.Update(ctx, saatIni)
	if err != nil {
		return terjemahkanError(c, err, "gagal memperbarui mahasiswa")
	}
	return ok(c, "data mahasiswa berhasil diperbarui sebagian", hasil)
}

func (h *StudentHandler) Delete(c *fiber.Ctx) error {
	ctx, cancel := reqCtx(c)
	defer cancel()

	id, valid := paramID(c)
	if !valid {
		return fail(c, fiber.StatusBadRequest, "id harus berupa angka positif")
	}

	if err := h.repo.Delete(ctx, id); err != nil {
		return terjemahkanError(c, err, "gagal menghapus mahasiswa")
	}
	return noContent(c)
}
