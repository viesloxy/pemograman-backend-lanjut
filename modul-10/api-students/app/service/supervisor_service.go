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

var errCreditLimit = errors.New("batas SKS terlampaui")

type SupervisorService struct {
	supervisors repository.SupervisorRepository
	students    repository.StudentRepository
	tx          *repository.TxManager
}

func NewSupervisorService(
	supervisors repository.SupervisorRepository,
	students repository.StudentRepository,
	tx *repository.TxManager,
) *SupervisorService {
	return &SupervisorService{supervisors: supervisors, students: students, tx: tx}
}

func (s *SupervisorService) List(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	list, err := s.supervisors.FindAll(ctx)
	if err != nil {
		return helper.Internal(err)
	}
	return helper.Success(c, fiber.StatusOK, "daftar pembimbing berhasil diambil", list)
}

func (s *SupervisorService) Assign(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	if _, ok := helper.CurrentUser(c); !ok {
		return helper.Unauthorized("belum terautentikasi")
	}

	supervisorID, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id harus berupa angka positif")
	}

	var req model.AssignRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}
	if errs := helper.ValidateStruct(req); errs != nil {
		return helper.Validation(errs)
	}

	var assigned model.Supervisor

	err := s.tx.WithTx(ctx, func(tx repository.DBTX) error {
		repo := s.supervisors.WithTx(tx)

		// 1. Kunci baris mahasiswa (gembok KRS). Request lain untuk
		//    mahasiswa YANG SAMA harus menunggu sampai transaction ini
		//    selesai — satu mahasiswa tidak mungkin ditetapkan ke dua
		//    pembimbing secara serentak.
		if err := repo.LockStudent(ctx, req.StudentID); err != nil {
			return err
		}

		// 2. Kunci baris pembimbing dalam urutan id menaik sebelum
		//    menyentuh kuotanya, agar tidak ada pembacaan yang basi.
		if err := repo.LockSupervisors(ctx, supervisorID); err != nil {
			return err
		}

		// 3. Ambil satu kursi bimbingan secara atomik: pemeriksaan
		//    "supervised_count < quota" dan penambahannya terjadi dalam
		//    satu pernyataan UPDATE. Tidak ada celah check-then-act.
		sup, err := repo.IncrementSupervised(ctx, supervisorID)
		if err != nil {
			return err
		}
		assigned = sup

		// 4. Catat penetapannya. Bila mahasiswa ternyata sudah punya
		//    pembimbing (UNIQUE student_id), error di sini membatalkan
		//    juga penambahan kursi pada langkah 3 — itulah atomicity.
		return repo.AssignSupervisor(ctx, supervisorID, req.StudentID)
	})
	if err != nil {
		return translateSupervisorError(err)
	}

	return helper.Created(c, "pembimbing berhasil ditetapkan", assigned,
		"/api/v1/supervisors/"+strconv.Itoa(supervisorID))
}

func (s *SupervisorService) Switch(c *fiber.Ctx) error {
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
	if req.FromSupervisorID == req.ToSupervisorID {
		return helper.BadRequest("tidak boleh sama dengan from_supervisor_id")
	}

	err := s.tx.WithTx(ctx, func(tx repository.DBTX) error {
		repo := s.supervisors.WithTx(tx)

		if err := repo.LockStudent(ctx, current.StudentID); err != nil {
			return err
		}

		// Kunci KEDUA pembimbing lebih dulu, dalam urutan id menaik.
		// Tanpa urutan yang seragam, pindah A -> B dan B -> A pada saat
		// yang sama saling menunggu kunci: deadlock (Bagian A.6).
		if err := repo.LockSupervisors(ctx, req.FromSupervisorID, req.ToSupervisorID); err != nil {
			return err
		}

		// Pastikan mahasiswa memang ada di pembimbing asal.
		if err := repo.FindSupervision(ctx, req.FromSupervisorID, current.StudentID); err != nil {
			return err
		}

		// Keluar dari pembimbing lama.
		if err := repo.DeleteSupervision(ctx, req.FromSupervisorID, current.StudentID); err != nil {
			return err
		}
		if _, err := repo.DecrementSupervised(ctx, req.FromSupervisorID); err != nil {
			return err
		}

		// Masuk ke pembimbing baru secara atomik. Bila kuotanya penuh,
		// error di sini me-ROLLBACK seluruh transaction — mahasiswa
		// tetap berada di pembimbing lamanya.
		if _, err := repo.IncrementSupervised(ctx, req.ToSupervisorID); err != nil {
			return err
		}
		return repo.AssignSupervisor(ctx, req.ToSupervisorID, current.StudentID)
	})
	if err != nil {
		return translateSupervisorError(err)
	}

	return helper.Success(c, fiber.StatusOK, "pembimbing berhasil diganti", nil)
}

func (s *SupervisorService) UpdateQuota(c *fiber.Ctx) error {
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

	updated, err := s.supervisors.UpdateQuota(ctx, id, req.Quota, req.Version)
	if err != nil {
		return translateSupervisorError(err)
	}
	return helper.Success(c, fiber.StatusOK, "kuota berhasil diubah", updated)
}

func translateSupervisorError(err error) error {
	switch {
	case errors.Is(err, repository.ErrNotFound):
		return helper.NotFound("pembimbing atau mahasiswa tidak ditemukan")
	case errors.Is(err, repository.ErrSupervisorFull):
		return helper.NewError(fiber.StatusConflict, "SUPERVISOR_FULL", "kuota pembimbing sudah penuh")
	case errors.Is(err, repository.ErrDuplicate):
		return helper.NewError(fiber.StatusConflict, "ALREADY_SUPERVISED", "mahasiswa sudah memiliki pembimbing")
	case errors.Is(err, repository.ErrNotEnrolled):
		return helper.NewError(fiber.StatusConflict, "NOT_SUPERVISED", "tidak terdaftar pada pembimbing tersebut")
	case errors.Is(err, repository.ErrVersionConflict):
		return helper.NewError(fiber.StatusConflict, "VERSION_CONFLICT", "data sudah diubah pihak lain, muat ulang lalu coba lagi")
	case errors.Is(err, repository.ErrQuotaBelowFilled):
		return helper.NewError(fiber.StatusConflict, "QUOTA_BELOW_SUPERVISED", "kuota tidak boleh lebih kecil dari jumlah mahasiswa bimbingan saat ini")
	default:
		return helper.Internal(err)
	}
}

func cleanNIM(s string) string {
	return strings.TrimSpace(s)
}
