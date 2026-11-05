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
	return helper.Success(c, fiber.StatusOK, "KRS berhasil diambil", fiber.Map{
		"courses":       courses,
		"total_credits": total,
		"max_credits":   MaxCreditsPerSemester,
	})
}

// ---------- GET /courses/:id ----------

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