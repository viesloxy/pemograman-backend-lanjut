package service

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"

	"api-students/app/model"
	"api-students/app/repository"
	"api-students/helper"
)

// CourseService melayani KRS: mendaftar, keluar, dan pindah kelas. Sejak
// Modul 9 seluruh perubahan KRS adalah transaction: mengambil kursi dan
// mencatat pendaftaran harus terjadi bersama-sama atau batal bersama-sama.
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

		// 1. Kunci baris mahasiswa (gembok KRS). Request lain untuk
		//    mahasiswa YANG SAMA harus menunggu sampai transaction ini
		//    selesai, sehingga urutan event KRS seorang mahasiswa selalu
		//    jelas.
		if err := repo.LockStudent(ctx, current.StudentID); err != nil {
			return err
		}

		// 2. Kunci baris kelas sebelum menyentuh hitungannya.
		if err := repo.LockCourses(ctx, courseID); err != nil {
			return err
		}

		// 3. Ambil satu kursi secara atomik.
		var err error
		enrolled, err = repo.IncrementEnrolled(ctx, courseID)
		if err != nil {
			return err
		}

		// 4. Catat event-nya ke outbox, memakai transaction yang sama.
		//    PERNAYAAN: repo outbox WAJIB di-WithTx — bila tidak, catatan
		//    ditulis di LUAR transaction: pendaftaran yang batal (409) sudah
		//    meninggalkan event (notifikasi palsu), dan keduanya bisa tidak
		//    konsisten.
		if err := s.outbox.WithTx(tx).Insert(ctx, model.EventEnrollmentCreated, model.EnrollmentEvent{
			UserID: current.StudentID, CourseID: enrolled.ID,
			CourseCode: enrolled.Code, CourseName: enrolled.Name,
		}); err != nil {
			return err
		}

		// 5. Catat pendaftarannya. Bila ternyata sudah terdaftar, error
		//    di sini membatalkan seluruh langkah sebelumnya.
		return repo.InsertEnrollment(ctx, courseID, current.StudentID)
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

		if err := repo.LockStudent(ctx, current.StudentID); err != nil {
			return err
		}
		if err := repo.LockCourses(ctx, courseID); err != nil {
			return err
		}

		// Bila ternyata tidak terdaftar, error di sini membatalkan
		// transaction — tidak ada kursi yang ikut berkurang.
		if err := repo.DeleteEnrollment(ctx, courseID, current.StudentID); err != nil {
			return err
		}
		course, err := repo.DecrementEnrolled(ctx, courseID)
		if err != nil {
			return err
		}
		return s.outbox.WithTx(tx).Insert(ctx, model.EventEnrollmentDropped, model.EnrollmentEvent{
			UserID: current.StudentID, CourseID: course.ID,
			CourseCode: course.Code, CourseName: course.Name,
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

	var req model.SwapCourseRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}
	if errs := helper.ValidateStruct(req); errs != nil {
		return helper.Validation(errs)
	}
	if req.FromCourseID == req.ToCourseID {
		return helper.BadRequest("tidak boleh sama dengan from_course_id")
	}

	err := s.tx.WithTx(ctx, func(tx repository.DBTX) error {
		repo := s.courses.WithTx(tx)

		if err := repo.LockStudent(ctx, current.StudentID); err != nil {
			return err
		}

		// Kunci KEDUA kelas lebih dulu, dalam urutan id menaik.
		// Tanpa urutan yang seragam, pindah A -> B dan B -> A pada saat
		// yang sama saling menunggu kunci: deadlock (Modul 9).
		if err := repo.LockCourses(ctx, req.FromCourseID, req.ToCourseID); err != nil {
			return err
		}

		// Nama kelas asal dan kelas tujuan disalin ke event: notifikasi
		// memakai nama yang dilihat mahasiswa saat ia masih terdaftar.
		from, err := repo.FindByID(ctx, req.FromCourseID)
		if err != nil {
			return err
		}
		to, err := repo.FindByID(ctx, req.ToCourseID)
		if err != nil {
			return err
		}

		// Keluar dari kelas lama. Bila tidak terdaftar, seluruh
		// transaction dibatalkan.
		if err := repo.DeleteEnrollment(ctx, req.FromCourseID, current.StudentID); err != nil {
			return err
		}
		if _, err := repo.DecrementEnrolled(ctx, req.FromCourseID); err != nil {
			return err
		}

		// Masuk ke kelas baru secara atomik. Bila kuotanya penuh, error
		// di sini me-ROLLBACK seluruh transaction — mahasiswa tetap
		// terdaftar di kelas lamanya.
		if _, err := repo.IncrementEnrolled(ctx, req.ToCourseID); err != nil {
			return err
		}
		if err := repo.InsertEnrollment(ctx, req.ToCourseID, current.StudentID); err != nil {
			return err
		}

		// Satu event "pindah", bukan dua event "keluar" dan "masuk".
		// Mahasiswa melakukan satu tindakan, maka ia menerima satu notifikasi.
		return s.outbox.WithTx(tx).Insert(ctx, model.EventEnrollmentSwapped, model.EnrollmentEvent{
			UserID: current.StudentID, CourseID: to.ID,
			CourseCode: to.Code, CourseName: to.Name,
			FromCourseCode: from.Code, FromCourseName: from.Name,
		})
	})
	if err != nil {
		return translateCourseError(err)
	}

	return helper.Success(c, fiber.StatusOK, "kelas berhasil ditukar", nil)
}

// ---------- PATCH /courses/:id (admin) ----------

func (s *CourseService) UpdateQuota(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id harus berupa angka positif")
	}

	var req model.UpdateCourseQuotaRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}
	if errs := helper.ValidateStruct(req); errs != nil {
		return helper.Validation(errs)
	}

	var updated model.Course

	// Sejak D.2, mengubah kuota adalah DUA pernyataan: UPDATE kuota dan
	// INSERT event ke outbox. Tanpa transaction, kuota bisa berubah sementara
	// pencatatan eventnya gagal (atau sebaliknya) — peserta akan menerima
	// notifikasi untuk kuota yang tidak pernah terjadi.
	err := s.tx.WithTx(ctx, func(tx repository.DBTX) error {
		repo := s.courses.WithTx(tx)

		u, err := repo.UpdateQuota(ctx, id, req.Quota, req.Version)
		if err != nil {
			return err
		}
		updated = u

		return s.outbox.WithTx(tx).Insert(ctx, model.EventQuotaChanged, model.QuotaChangedEvent{
			CourseID: updated.ID, CourseCode: updated.Code,
			CourseName: updated.Name, Quota: updated.Quota,
		})
	})
	if err != nil {
		return translateCourseError(err)
	}
	return helper.Success(c, fiber.StatusOK, "kuota kelas berhasil diubah", updated)
}

func translateCourseError(err error) error {
	switch {
	case errors.Is(err, repository.ErrNotFound):
		return helper.NotFound("kelas atau mahasiswa tidak ditemukan")
	case errors.Is(err, repository.ErrCourseFull):
		return helper.NewError(fiber.StatusConflict, "COURSE_FULL", "kuota kelas sudah penuh")
	case errors.Is(err, repository.ErrDuplicate):
		return helper.NewError(fiber.StatusConflict, "ALREADY_ENROLLED", "sudah terdaftar di kelas tersebut")
	case errors.Is(err, repository.ErrNotEnrolled):
		return helper.NewError(fiber.StatusConflict, "NOT_ENROLLED", "tidak terdaftar di kelas tersebut")
	case errors.Is(err, repository.ErrVersionConflict):
		return helper.NewError(fiber.StatusConflict, "VERSION_CONFLICT", "data sudah diubah pihak lain, muat ulang lalu coba lagi")
	case errors.Is(err, repository.ErrQuotaBelowFilled):
		return helper.NewError(fiber.StatusConflict, "QUOTA_BELOW_ENROLLED", "kuota tidak boleh lebih kecil dari jumlah peserta saat ini")
	default:
		return helper.Internal(err)
	}
}
