package service

import (
	"errors"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"

	"api-students/app/model"
	"api-students/app/repository"
	"api-students/helper"
)

type StudentService struct {
	repo  repository.StudentRepository
	perms *helper.PermissionSet
}

func NewStudentService(
	repo repository.StudentRepository,
	perms *helper.PermissionSet,
) *StudentService {
	return &StudentService{repo: repo, perms: perms}
}

func (s *StudentService) List(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	q := helper.ParseListQuery(c)

	daftar, total, err := s.repo.FindAll(ctx, q)
	if err != nil {
		return helper.Internal(err)
	}

	return helper.SuccessList(c, "daftar mahasiswa berhasil diambil", daftar, &model.Meta{
		Page:       q.Page,
		Limit:      q.Limit,
		Total:      total,
		TotalPages: CountTotalPages(total, q.Limit),
	})
}

func (s *StudentService) Get(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	current, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized("belum terautentikasi")
	}

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id harus berupa angka positif")
	}

	siswa, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return translateError(err)
	}

	if !CanAccessStudent(current, ownerValue(siswa.OwnerID), s.perms, "student:read:any") {
		return helper.Forbidden("tidak berhak mengakses data mahasiswa lain")
	}
	return helper.Success(c, fiber.StatusOK, "mahasiswa ditemukan", siswa)
}

func (s *StudentService) Create(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	current, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized("belum terautentikasi")
	}

	var req model.CreateStudentRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}

	req.NIM = strings.TrimSpace(req.NIM)
	req.Name = strings.TrimSpace(req.Name)

	if errs := helper.ValidateStruct(req); errs != nil {
		return helper.Validation(errs)
	}

	baru := model.Student{
		NIM:     req.NIM,
		Name:    req.Name,
		Grade:   *req.Grade,
		Role:    "user",
		OwnerID: &current.StudentID,
	}
	baru.Activate()

	hasil, err := s.repo.Create(ctx, baru)
	if err != nil {
		return translateError(err)
	}

	return helper.Created(c, "mahasiswa berhasil ditambahkan", hasil,
		"/api/v1/students/"+strconv.Itoa(hasil.ID))
}

func (s *StudentService) Replace(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	current, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized("belum terautentikasi")
	}

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id harus berupa angka positif")
	}

	var req model.ReplaceStudentRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}

	req.NIM = strings.TrimSpace(req.NIM)
	req.Name = strings.TrimSpace(req.Name)

	if errs := helper.ValidateStruct(req); errs != nil {
		return helper.Validation(errs)
	}

	siswa, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return translateError(err)
	}

	if !CanAccessStudent(current, ownerValue(siswa.OwnerID), s.perms, "student:update:any") {
		return helper.Forbidden("tidak berhak mengubah data mahasiswa lain")
	}

	siswa.NIM = req.NIM
	siswa.Name = req.Name
	siswa.UpdateGrade(*req.Grade)
	if *req.IsActive {
		siswa.Activate()
	} else {
		siswa.Deactivate()
	}

	hasil, err := s.repo.Update(ctx, siswa)
	if err != nil {
		return translateError(err)
	}
	return helper.Success(c, fiber.StatusOK, "data mahasiswa berhasil diganti seluruhnya", hasil)
}

func (s *StudentService) Patch(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	current, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized("belum terautentikasi")
	}

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id harus berupa angka positif")
	}

	var req model.PatchStudentRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}

	if IsEmptyPatch(req) {
		return helper.BadRequest("tidak ada field yang diubah")
	}

	saatIni, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return translateError(err)
	}

	if !CanAccessStudent(current, ownerValue(saatIni.OwnerID), s.perms, "student:update:any") {
		return helper.Forbidden("tidak berhak mengubah data mahasiswa lain")
	}

	diubah := ApplyPatch(saatIni, req)

	hasil, err := s.repo.Update(ctx, diubah)
	if err != nil {
		return translateError(err)
	}
	return helper.Success(c, fiber.StatusOK, "data mahasiswa berhasil diperbarui sebagian", hasil)
}

func (s *StudentService) Delete(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	current, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized("belum terautentikasi")
	}

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id harus berupa angka positif")
	}

	if current.StudentID == id {
		return helper.Forbidden("tidak boleh menghapus akun sendiri")
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return translateError(err)
	}
	return helper.NoContent(c)
}

func translateError(err error) error {
	switch {
	case errors.Is(err, repository.ErrNotFound):
		return helper.NotFound("mahasiswa tidak ditemukan")
	case errors.Is(err, repository.ErrDuplicate):
		return helper.Conflict("NIM sudah terdaftar")
	default:
		return helper.Internal(err)
	}
}
