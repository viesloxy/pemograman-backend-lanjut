package service

import (
	"errors"
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"

	"siakad-mini/app/model"
	"siakad-mini/app/repository"
	"siakad-mini/helper"
)

type EnrollmentService struct {
	enrollments repository.EnrollmentRepository
	students    repository.StudentRepository
	courses     repository.CourseRepository
	// pool dipakai untuk membuka transaksi dan sebagai DBTX ketika
	// operasi tidak butuh transaksi.
	pool *pgxpool.Pool
}

func NewEnrollmentService(
	enrollments repository.EnrollmentRepository,
	students repository.StudentRepository,
	courses repository.CourseRepository,
	pool *pgxpool.Pool,
) *EnrollmentService {
	return &EnrollmentService{
		enrollments: enrollments,
		students:    students,
		courses:     courses,
		pool:        pool,
	}
}

// Create mengambil mata kuliah ke dalam KRS. Seluruh pemeriksaan —
// duplikasi, kuota, dan batas SKS — berjalan dalam satu transaksi,
// dan baris mata kuliah dikunci (SELECT ... FOR UPDATE) agar dua
// permintaan serentak tidak dapat melewati pemeriksaan kuota bersamaan.
func (s *EnrollmentService) Create(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	authUser, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized("belum terautentikasi")
	}

	var req model.CreateEnrollmentRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}

	if errs := validateCreateEnrollment(req); errs != nil {
		return helper.Validation(errs)
	}

	// Data mahasiswa (untuk IPK) dibaca sebelum transaksi; nilainya
	// tidak bisa berubah di tengah permintaan ini.
	student, err := s.students.FindByID(ctx, authUser.StudentID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return helper.NotFound("data mahasiswa tidak ditemukan")
		}
		return helper.Internal(err)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return helper.Internal(err)
	}
	defer tx.Rollback(ctx)

	// 1. Kunci baris mata kuliah.
	course, err := s.courses.LockByID(ctx, tx, req.CourseID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return helper.NotFound("mata kuliah tidak ditemukan")
		}
		return helper.Internal(err)
	}

	// 2. Cek duplikasi pada tahun akademik yang sama.
	ada, err := s.enrollments.Exists(ctx, tx, student.ID, course.ID, req.TahunAkademik)
	if err != nil {
		return helper.Internal(err)
	}
	if ada {
		return helper.Conflict(fmt.Sprintf(
			"mata kuliah %s sudah diambil pada tahun akademik %s",
			course.KodeMK, req.TahunAkademik))
	}

	// 3. Cek kuota.
	terisi, err := s.enrollments.CountByCourse(ctx, tx, course.ID)
	if err != nil {
		return helper.Internal(err)
	}
	if terisi >= course.Kuota {
		return helper.Unprocessable(fmt.Sprintf(
			"kuota mata kuliah %s sudah penuh (%d/%d)",
			course.KodeMK, terisi, course.Kuota))
	}

	// 4. Cek batas SKS sesuai IPK.
	totalSKS, err := s.enrollments.TotalSKS(ctx, tx, student.ID)
	if err != nil {
		return helper.Internal(err)
	}
	batas := BatasSKS(student.IPKTerakhir)
	sisa := SisaSKS(batas, totalSKS)
	if course.SKS > sisa {
		return helper.Unprocessable(fmt.Sprintf(
			"total SKS melebihi batas: mengambil %d SKS menyisakan %d SKS dari batas %d SKS",
			course.SKS, sisa, batas))
	}

	// 5. Semua pemeriksaan lolos: simpan KRS.
	enrollment := model.Enrollment{
		StudentID:     student.ID,
		CourseID:      course.ID,
		TahunAkademik: req.TahunAkademik,
	}
	if err := s.enrollments.Create(ctx, tx, &enrollment); err != nil {
		// Pelanggaran unique constraint bila dua permintaan identik
		// lolos dari pemeriksaan pada saat hampir bersamaan.
		if errors.Is(err, repository.ErrDuplicate) {
			return helper.Conflict("mata kuliah sudah diambil pada tahun akademik ini")
		}
		return helper.Internal(err)
	}

	if err := tx.Commit(ctx); err != nil {
		return helper.Internal(err)
	}

	return helper.Created(c, "mata kuliah berhasil diambil", enrollment,
		"/api/v1/enrollments/"+fmt.Sprint(enrollment.ID))
}

// Delete membatalkan mata kuliah dari KRS milik mahasiswa yang login.
// Kuota mata kuliah otomatis kembali karena terisi selalu dihitung
// dari tabel enrollments.
func (s *EnrollmentService) Delete(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	authUser, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized("belum terautentikasi")
	}

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.NotFound("enrollment tidak ditemukan")
	}

	enrollment, err := s.enrollments.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return helper.NotFound("enrollment tidak ditemukan")
		}
		return helper.Internal(err)
	}

	// Pemeriksaan kepemilikan: bergantung isi data, maka di service.
	if enrollment.StudentID != authUser.StudentID {
		return helper.Forbidden("Anda hanya boleh membatalkan KRS Anda sendiri")
	}

	if err := s.enrollments.Delete(ctx, s.pool, id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return helper.NotFound("enrollment tidak ditemukan")
		}
		return helper.Internal(err)
	}

	return helper.NoContent(c)
}
