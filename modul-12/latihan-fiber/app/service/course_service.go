package service

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"

	"latihan-fiber/app/model"
	"latihan-fiber/app/repository"
	"latihan-fiber/helper"
)

// errCreditLimit adalah kesalahan khusus layer service.
var errCreditLimit = errors.New("batas SKS terlampaui")

type CourseService struct {
	courses repository.CourseRepository
	outbox  repository.OutboxRepository
	tx      *repository.TxManager
}

func NewCourseService(
	courses repository.CourseRepository,
	outbox repository.OutboxRepository,
	tx *repository.TxManager,
) *CourseService {
	return &CourseService{courses: courses, outbox: outbox, tx: tx}
}

// ---------- GET /courses ----------

// List godoc
//
// @Summary  Daftar seluruh kelas
// @Tags     courses
// @Produce  json
// @Success  200  {object}  model.WebResponse{data=[]model.Course}
// @Failure  401  {object}  model.ErrorResponse
// @Security BearerAuth
// @Router   /courses [get]
func (s *CourseService) List(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	courses, err := s.courses.FindAll(ctx)
	if err != nil {
		return helper.Internal(err)
	}
	return helper.Success(c, fiber.StatusOK, "daftar kelas berhasil diambil", courses)
}

// ---------- GET /courses/mine ----------

// Mine godoc
//
// @Summary  KRS milik mahasiswa yang sedang login
// @Tags     courses
// @Produce  json
// @Success  200  {object}  model.WebResponse{data=model.MyCourses}
// @Failure  401  {object}  model.ErrorResponse
// @Security BearerAuth
// @Router   /courses/mine [get]
func (s *CourseService) Mine(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	current, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized("belum terautentikasi")
	}

	courses, err := s.courses.FindByStudent(ctx, current.UserID)
	if err != nil {
		return helper.Internal(err)
	}

	total := 0
	for _, course := range courses {
		total += course.Credits
	}
	return helper.Success(c, fiber.StatusOK, "KRS berhasil diambil", model.MyCourses{
		Courses:      courses,
		TotalCredits: total,
		MaxCredits:   MaxCreditsPerSemester,
	})
}

// ---------- GET /courses/:id ----------

// Get godoc
//
// @Summary  Detail satu kelas
// @Tags     courses
// @Produce  json
// @Param    id  path  int  true  "ID kelas"
// @Success  200  {object}  model.WebResponse{data=model.Course}
// @Failure  400  {object}  model.ErrorResponse  "ID bukan angka positif"
// @Failure  404  {object}  model.ErrorResponse
// @Security BearerAuth
// @Router   /courses/{id} [get]
func (s *CourseService) Get(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id harus berupa angka positif")
	}

	course, err := s.courses.FindByID(ctx, id)
	if err != nil {
		return translateCourseError(err)
	}
	return helper.Success(c, fiber.StatusOK, "kelas ditemukan", course)
}

// ---------- POST /courses/:id/enroll ----------

// Enroll godoc
//
// @Summary      Mendaftar ke sebuah kelas
// @Description  Mengambil satu kursi dan mencatat event enrollment.created ke outbox dalam SATU transaction.
// @Description  Kode error 409: COURSE_FULL, ALREADY_ENROLLED, CREDIT_LIMIT_EXCEEDED.
// @Tags         courses
// @Produce      json
// @Param        id   path  int  true  "ID kelas"
// @Success      201  {object}  model.WebResponse{data=model.Course}
// @Header       201  {string}  Location  "URL kelas yang baru didaftari"
// @Failure      401  {object}  model.ErrorResponse
// @Failure      404  {object}  model.ErrorResponse
// @Failure      409  {object}  model.ErrorResponse  "Kuota penuh, sudah terdaftar, atau batas SKS terlampaui"
// @Security     BearerAuth
// @Router       /courses/{id}/enroll [post]
func (s *CourseService) Enroll(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	current, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized("belum terautentikasi")
	}

	courseID, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id harus berupa angka positif")
	}

	var enrolled model.Course

	err := s.tx.WithTx(ctx, func(tx repository.DBTX) error {
		repo := s.courses.WithTx(tx)

		// 1. Kunci KRS milik mahasiswa ini.
		if err := repo.LockStudent(ctx, current.UserID); err != nil {
			return err
		}

		// 2. Periksa batas SKS.
		course, err := repo.FindByID(ctx, courseID)
		if err != nil {
			return err
		}
		taken, err := repo.SumCredits(ctx, current.UserID)
		if err != nil {
			return err
		}
		if !WithinCreditLimit(taken, course.Credits) {
			return errCreditLimit
		}

		// 3. Ambil satu kursi secara atomik.
		enrolled, err = repo.IncrementEnrolled(ctx, courseID)
		if err != nil {
			return err
		}

		// 4. Catat event-nya ke outbox, memakai transaction yang sama.
		//    Repo outbox WAJIB di-WithTx: bila tidak, catatan ditulis di
		//    LUAR transaction, sehingga pendaftaran yang batal (sudah
		//    terdaftar) sudah meninggalkan event (notifikasi palsu).
		if err := s.outbox.WithTx(tx).Insert(ctx, model.EventEnrollmentCreated, model.EnrollmentEvent{
			UserID:     current.UserID,
			CourseID:   enrolled.ID,
			CourseCode: enrolled.Code,
			CourseName: enrolled.Name,
		}); err != nil {
			return err
		}

		// 5. Catat pendaftarannya. Bila ternyata sudah terdaftar, error
		//    di sini membatalkan seluruh langkah sebelumnya.
		return repo.InsertEnrollment(ctx, courseID, current.UserID)
	})
	if err != nil {
		return translateCourseError(err)
	}

	return helper.Created(c, "berhasil mendaftar kelas", enrolled,
		"/api/v1/courses/"+strconv.Itoa(courseID))
}

// ---------- DELETE /courses/:id/enroll ----------

// Drop godoc
//
// @Summary  Keluar dari sebuah kelas
// @Tags     courses
// @Param    id  path  int  true  "ID kelas"
// @Success  204  "Berhasil keluar; tanpa body"
// @Failure  401  {object}  model.ErrorResponse
// @Failure  409  {object}  model.ErrorResponse  "NOT_ENROLLED"
// @Security BearerAuth
// @Router   /courses/{id}/enroll [delete]
func (s *CourseService) Drop(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	current, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized("belum terautentikasi")
	}

	courseID, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id harus berupa angka positif")
	}

	err := s.tx.WithTx(ctx, func(tx repository.DBTX) error {
		repo := s.courses.WithTx(tx)

		if err := repo.LockStudent(ctx, current.UserID); err != nil {
			return err
		}
		if err := repo.DeleteEnrollment(ctx, courseID, current.UserID); err != nil {
			return err
		}
		course, err := repo.DecrementEnrolled(ctx, courseID)
		if err != nil {
			return err
		}
		return s.outbox.WithTx(tx).Insert(ctx, model.EventEnrollmentDropped, model.EnrollmentEvent{
			UserID:     current.UserID,
			CourseID:   course.ID,
			CourseCode: course.Code,
			CourseName: course.Name,
		})
	})
	if err != nil {
		return translateCourseError(err)
	}

	return helper.NoContent(c)
}

// ---------- POST /enrollments/swap ----------

// Swap godoc
//
// @Summary      Pindah kelas
// @Description  Keluar dari kelas lama dan masuk ke kelas baru sebagai satu tindakan. Bila kelas baru penuh, mahasiswa tetap di kelas lama.
// @Tags         courses
// @Accept       json
// @Produce      json
// @Param        body  body  model.SwapRequest  true  "Kelas asal dan kelas tujuan"
// @Success      200   {object}  model.WebResponse{data=model.Course}
// @Failure      401   {object}  model.ErrorResponse
// @Failure      409   {object}  model.ErrorResponse  "COURSE_FULL, NOT_ENROLLED, CREDIT_LIMIT_EXCEEDED"
// @Failure      422   {object}  model.ErrorResponse
// @Security     BearerAuth
// @Router       /enrollments/swap [post]
func (s *CourseService) Swap(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	current, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized("belum terautentikasi")
	}

	var req model.SwapRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}
	if errs := helper.ValidateStruct(req); errs != nil {
		return helper.Validation(errs)
	}
	if req.FromCourseID == req.ToCourseID {
		return helper.Validation(map[string]string{
			"to_course_id": "tidak boleh sama dengan from_course_id",
		})
	}

	var joined model.Course

	err := s.tx.WithTx(ctx, func(tx repository.DBTX) error {
		repo := s.courses.WithTx(tx)

		if err := repo.LockStudent(ctx, current.UserID); err != nil {
			return err
		}

		// Kunci KEDUA kelas lebih dulu, urut id menaik.
		if err := repo.LockCourses(ctx, req.FromCourseID, req.ToCourseID); err != nil {
			return err
		}

		from, err := repo.FindByID(ctx, req.FromCourseID)
		if err != nil {
			return err
		}
		to, err := repo.FindByID(ctx, req.ToCourseID)
		if err != nil {
			return err
		}
		taken, err := repo.SumCredits(ctx, current.UserID)
		if err != nil {
			return err
		}
		if CreditsAfterSwap(taken, from.Credits, to.Credits) > MaxCreditsPerSemester {
			return errCreditLimit
		}

		if err := repo.DeleteEnrollment(ctx, req.FromCourseID, current.UserID); err != nil {
			return err
		}
		if _, err := repo.DecrementEnrolled(ctx, req.FromCourseID); err != nil {
			return err
		}

		joined, err = repo.IncrementEnrolled(ctx, req.ToCourseID)
		if err != nil {
			return err
		}
		if err := repo.InsertEnrollment(ctx, req.ToCourseID, current.UserID); err != nil {
			return err
		}

		// Satu event "pindah", bukan dua event "keluar" dan "masuk".
		// Mahasiswa melakukan satu tindakan, maka ia menerima satu notifikasi.
		return s.outbox.WithTx(tx).Insert(ctx, model.EventEnrollmentSwapped, model.EnrollmentEvent{
			UserID:         current.UserID,
			CourseID:       to.ID,
			CourseCode:     to.Code,
			CourseName:     to.Name,
			FromCourseCode: from.Code,
			FromCourseName: from.Name,
		})
	})
	if err != nil {
		return translateCourseError(err)
	}

	return helper.Success(c, fiber.StatusOK, "berhasil pindah kelas", joined)
}

// ---------- PATCH /courses/:id ----------
// Dijaga middleware dengan permission course:manage.

// UpdateQuota godoc
//
// @Summary      Mengubah kuota kelas (admin)
// @Description  Optimistic locking: kirim version yang terakhir dilihat. Membutuhkan permission course:manage.
// @Tags         courses
// @Accept       json
// @Produce      json
// @Param        id    path  int                      true  "ID kelas"
// @Param        body  body  model.UpdateQuotaRequest  true  "Kuota baru dan version"
// @Success      200   {object}  model.WebResponse{data=model.Course}
// @Failure      403   {object}  model.ErrorResponse
// @Failure      409   {object}  model.ErrorResponse  "VERSION_CONFLICT atau QUOTA_BELOW_ENROLLED"
// @Security     BearerAuth
// @Router       /courses/{id} [patch]
func (s *CourseService) UpdateQuota(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id harus berupa angka positif")
	}

	var req model.UpdateQuotaRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}
	if errs := helper.ValidateStruct(req); errs != nil {
		return helper.Validation(errs)
	}

	// Satu pernyataan UPDATE saja, tidak perlu WithTx.
	updated, err := s.courses.UpdateQuota(ctx, id, req.Quota, req.Version)
	if err != nil {
		return translateCourseError(err)
	}
	return helper.Success(c, fiber.StatusOK, "kuota berhasil diubah", updated)
}

// translateCourseError memetakan kegagalan ke response.
func translateCourseError(err error) error {
	switch {
	case errors.Is(err, repository.ErrNotFound):
		return helper.NotFound("kelas tidak ditemukan")
	case errors.Is(err, repository.ErrCourseFull):
		return helper.NewError(fiber.StatusConflict, "COURSE_FULL", "kuota kelas sudah penuh")
	case errors.Is(err, repository.ErrDuplicate):
		return helper.NewError(fiber.StatusConflict, "ALREADY_ENROLLED", "Anda sudah terdaftar di kelas ini")
	case errors.Is(err, repository.ErrNotEnrolled):
		return helper.NewError(fiber.StatusConflict, "NOT_ENROLLED", "Anda tidak terdaftar di kelas tersebut")
	case errors.Is(err, errCreditLimit):
		return helper.NewError(fiber.StatusConflict, "CREDIT_LIMIT_EXCEEDED",
			"batas "+strconv.Itoa(MaxCreditsPerSemester)+" SKS terlampaui")
	case errors.Is(err, repository.ErrVersionConflict):
		return helper.NewError(fiber.StatusConflict, "VERSION_CONFLICT",
			"data kelas sudah diubah pihak lain, muat ulang lalu coba lagi")
	case errors.Is(err, repository.ErrQuotaBelowFilled):
		return helper.NewError(fiber.StatusConflict, "QUOTA_BELOW_ENROLLED",
			"kuota tidak boleh lebih kecil dari jumlah peserta saat ini")
	default:
		return helper.Internal(err)
	}
}