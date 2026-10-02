package service

import (
	"errors"
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"siakad-mini/app/model"
	"siakad-mini/app/repository"
	"siakad-mini/helper"
)

const (
	defaultPage    = 1
	defaultPerPage = 10
	maxPerPage     = 50
)

type StudentService struct {
	students    repository.StudentRepository
	users       repository.UserRepository
	enrollments repository.EnrollmentRepository
	pool        *pgxpool.Pool
}

func NewStudentService(
	students repository.StudentRepository,
	users repository.UserRepository,
	enrollments repository.EnrollmentRepository,
	pool *pgxpool.Pool,
) *StudentService {
	return &StudentService{students: students, users: users, enrollments: enrollments, pool: pool}
}

// List mengembalikan daftar mahasiswa dengan pagination, filter,
// pencarian, dan pengurutan. Seluruh penyaringan terjadi di SQL.
func (s *StudentService) List(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	q := model.ListStudentQuery{
		Page:    c.QueryInt("page", defaultPage),
		PerPage: c.QueryInt("per_page", defaultPerPage),
		Prodi:   strings.TrimSpace(c.Query("prodi")),
		Search:  strings.TrimSpace(c.Query("search")),
		Sort:    strings.TrimSpace(c.Query("sort")),
	}
	if raw := c.Query("angkatan"); raw != "" {
		// nilai tidak sah diabaikan: filter angkatan tidak diterapkan
		var v int
		if _, err := fmt.Sscanf(raw, "%d", &v); err == nil && v > 0 {
			q.Angkatan = v
		}
	}

	if q.Page < defaultPage {
		q.Page = defaultPage
	}
	if q.PerPage < 1 {
		q.PerPage = defaultPerPage
	}
	if q.PerPage > maxPerPage {
		q.PerPage = maxPerPage
	}

	data, total, err := s.students.FindAll(ctx, q)
	if err != nil {
		return helper.Internal(err)
	}

	lastPage := (total + q.PerPage - 1) / q.PerPage
	return helper.SuccessList(c, "Data mahasiswa berhasil diambil", data, &model.Meta{
		CurrentPage: q.Page,
		PerPage:     q.PerPage,
		Total:       total,
		LastPage:    lastPage,
	})
}

// Create menambah mahasiswa sekaligus akun usernya dalam satu transaksi.
// Password awal adalah NIM yang di-hash; role selalu ditentukan server.
func (s *StudentService) Create(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	var req model.CreateStudentRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}

	if errs := validateCreateStudent(req); errs != nil {
		return helper.Validation(errs)
	}

	// Pemeriksaan duplikat memberi pesan per field yang jelas;
	// unique constraint di database tetap menjadi pengaman terakhir.
	errs := map[string][]string{}
	if _, err := s.students.FindByNIM(ctx, req.NIM); err == nil {
		errs["nim"] = append(errs["nim"], "NIM sudah terdaftar")
	} else if !errors.Is(err, repository.ErrNotFound) {
		return helper.Internal(err)
	}
	if _, err := s.users.FindByEmail(ctx, req.Email); err == nil {
		errs["email"] = append(errs["email"], "Email sudah terdaftar")
	} else if !errors.Is(err, repository.ErrNotFound) {
		return helper.Internal(err)
	}
	if len(errs) > 0 {
		return helper.Validation(errs)
	}

	hashed, err := helper.HashPassword(req.NIM)
	if err != nil {
		return helper.Internal(err)
	}

	// Satu transaksi: user dan students harus tercipta bersama
	// atau sama-sama batal.
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return helper.Internal(err)
	}
	defer tx.Rollback(ctx)

	user := model.User{
		Email:    strings.TrimSpace(req.Email),
		Password: hashed,
		Role:     "mahasiswa",
	}
	if err := s.users.Create(ctx, tx, &user); err != nil {
		if field, ok := duplicateField(err); ok {
			return helper.Validation(map[string][]string{
				field: {"data sudah terdaftar"},
			})
		}
		return helper.Internal(err)
	}

	student := model.Student{
		UserID:      user.ID,
		NIM:         strings.TrimSpace(req.NIM),
		Nama:        strings.TrimSpace(req.Nama),
		Prodi:       strings.TrimSpace(req.Prodi),
		Angkatan:    req.Angkatan,
		IPKTerakhir: req.IPKTerakhir,
	}
	if err := s.students.Create(ctx, tx, &student); err != nil {
		if field, ok := duplicateField(err); ok {
			return helper.Validation(map[string][]string{
				field: {"data sudah terdaftar"},
			})
		}
		return helper.Internal(err)
	}

	if err := tx.Commit(ctx); err != nil {
		return helper.Internal(err)
	}

	return helper.Created(c, "mahasiswa berhasil ditambahkan", student,
		"/api/v1/students/"+fmt.Sprint(student.ID))
}

// duplicateField menerjemahkan pelanggaran unique constraint menjadi
// nama field request yang bentrok, agar jawaban 422 menunjuk field
// yang tepat.
func duplicateField(err error) (string, bool) {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23505" {
		return "", false
	}
	switch pgErr.ConstraintName {
	case "students_nim_key":
		return "nim", true
	default:
		return "email", true
	}
}

// Detail mengembalikan data mahasiswa beserta daftar mata kuliah
// yang diambil, total SKS, dan batas SKS menurut IPK-nya.
// Admin boleh melihat semua; mahasiswa hanya dirinya sendiri.
func (s *StudentService) Detail(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	authUser, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized("belum terautentikasi")
	}

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.NotFound("mahasiswa tidak ditemukan")
	}

	student, err := s.students.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return helper.NotFound("mahasiswa tidak ditemukan")
		}
		return helper.Internal(err)
	}

	// Pemeriksaan kepemilikan: bergantung pada isi data, maka
	// tempatnya di service, bukan di middleware.
	if !BolehLihatStudent(authUser, student.UserID) {
		return helper.Forbidden("Anda hanya boleh mengakses data Anda sendiri")
	}

	mataKuliah, err := s.enrollments.ListByStudent(ctx, student.ID)
	if err != nil {
		return helper.Internal(err)
	}

	totalSKS, err := s.enrollments.TotalSKS(ctx, s.pool, student.ID)
	if err != nil {
		return helper.Internal(err)
	}

	return helper.Success(c, fiber.StatusOK, "Detail mahasiswa berhasil diambil", model.StudentDetailResponse{
		Student:    student,
		MataKuliah: mataKuliah,
		TotalSKS:   totalSKS,
		BatasSKS:   BatasSKS(student.IPKTerakhir),
	})
}

// Update memperbarui data mahasiswa. NIM tidak dapat diubah karena
// tidak pernah dibaca dari body request.
func (s *StudentService) Update(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.NotFound("mahasiswa tidak ditemukan")
	}

	var req model.UpdateStudentRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}

	if errs := validateUpdateStudent(req); errs != nil {
		return helper.Validation(errs)
	}

	student, err := s.students.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return helper.NotFound("mahasiswa tidak ditemukan")
		}
		return helper.Internal(err)
	}

	student.Nama = strings.TrimSpace(req.Nama)
	student.Prodi = strings.TrimSpace(req.Prodi)
	student.Angkatan = req.Angkatan
	student.IPKTerakhir = req.IPKTerakhir

	if err := s.students.Update(ctx, &student); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return helper.NotFound("mahasiswa tidak ditemukan")
		}
		return helper.Internal(err)
	}

	return helper.Success(c, fiber.StatusOK, "Data mahasiswa berhasil diperbarui", student)
}

// Delete melakukan soft delete: students dan users diberi deleted_at,
// sehingga tidak muncul di daftar dan tidak dapat login lagi.
func (s *StudentService) Delete(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.NotFound("mahasiswa tidak ditemukan")
	}

	student, err := s.students.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return helper.NotFound("mahasiswa tidak ditemukan")
		}
		return helper.Internal(err)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return helper.Internal(err)
	}
	defer tx.Rollback(ctx)

	if err := s.students.SoftDelete(ctx, student.ID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return helper.NotFound("mahasiswa tidak ditemukan")
		}
		return helper.Internal(err)
	}
	if err := s.users.SoftDelete(ctx, tx, student.UserID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return helper.NotFound("mahasiswa tidak ditemukan")
		}
		return helper.Internal(err)
	}

	if err := tx.Commit(ctx); err != nil {
		return helper.Internal(err)
	}

	return helper.NoContent(c)
}
