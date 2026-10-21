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

		if err := repo.LockStudent(ctx, req.StudentID); err != nil {
			return err
		}

		supervisor, err := repo.FindByID(ctx, supervisorID)
		if err != nil {
			return err
		}
		count, err := repo.CountBySupervisor(ctx, supervisorID)
		if err != nil {
			return err
		}
		if !WithinQuotaLimit(count, supervisor.Quota) {
			return helper.Conflict("kuota pembimbing sudah penuh")
		}

		if err := repo.AssignSupervisor(ctx, supervisorID, req.StudentID); err != nil {
			return err
		}

		assigned = supervisor
		return nil
	})
	if err != nil {
		return err
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

		if err := repo.LockSupervisors(ctx, req.FromSupervisorID, req.ToSupervisorID); err != nil {
			return err
		}

		if err := repo.FindSupervision(ctx, req.FromSupervisorID, current.StudentID); err != nil {
			return err
		}

		target, err := repo.FindByID(ctx, req.ToSupervisorID)
		if err != nil {
			return err
		}
		count, err := repo.CountBySupervisor(ctx, req.ToSupervisorID)
		if err != nil {
			return err
		}
		if !WithinQuotaLimit(count, target.Quota) {
			return helper.Conflict("kuota pembimbing sudah penuh")
		}

		if err := repo.DeleteSupervision(ctx, req.FromSupervisorID, current.StudentID); err != nil {
			return err
		}
		return repo.AssignSupervisor(ctx, req.ToSupervisorID, current.StudentID)
	})
	if err != nil {
		return err
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
		return helper.NotFound("pembimbing tidak ditemukan")
	case errors.Is(err, repository.ErrVersionConflict):
		return helper.Conflict("data sudah diubah pihak lain, muat ulang lalu coba lagi")
	case errors.Is(err, repository.ErrQuotaBelowFilled):
		return helper.Conflict("kuota tidak boleh lebih kecil dari jumlah mahasiswa saat ini")
	default:
		return helper.Internal(err)
	}
}

func cleanNIM(s string) string {
	return strings.TrimSpace(s)
}
