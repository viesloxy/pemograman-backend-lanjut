package service

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"api-students/app/model"
	"api-students/app/repository"
	"api-students/helper"
)

// PortfolioService mengelola portofolio mahasiswa: identitas mahasiswanya
// ditanyakan ke PostgreSQL, dokumen portofolionya disimpan di MongoDB.
type PortfolioService struct {
	portfolios repository.PortfolioRepository
	students   repository.StudentRepository
}

func NewPortfolioService(
	portfolios repository.PortfolioRepository,
	students repository.StudentRepository,
) *PortfolioService {
	return &PortfolioService{portfolios: portfolios, students: students}
}

// ---------- GET /students/:id/portfolio ----------

func (s *PortfolioService) Get(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	studentID, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id mahasiswa harus berupa angka positif")
	}
	if _, err := s.students.FindByID(ctx, studentID); err != nil {
		return translatePortfolioError(err)
	}

	portfolio, err := s.portfolios.GetByStudent(ctx, studentID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return helper.NewError(fiber.StatusNotFound, "PORTFOLIO_NOT_FOUND",
				"portofolio belum dibuat")
		}
		return helper.Internal(err)
	}
	return helper.Success(c, fiber.StatusOK, "portofolio ditemukan", portfolio)
}

// ---------- PUT /students/:id/portfolio ----------

func (s *PortfolioService) Replace(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	current, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized("belum terautentikasi")
	}
	studentID, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id mahasiswa harus berupa angka positif")
	}
	if current.StudentID != studentID {
		return helper.Forbidden("hanya pemilik yang boleh mengubah portofolio ini")
	}

	var req model.PutPortfolioRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}
	if errs := helper.ValidateStruct(req); errs != nil {
		return helper.Validation(errs)
	}
	if _, err := s.students.FindByID(ctx, studentID); err != nil {
		return translatePortfolioError(err)
	}

	portfolio := model.Portfolio{
		StudentID: studentID,
		Headline:  req.Headline,
		Bio:       req.Bio,
	}
	for _, p := range req.Projects {
		portfolio.Projects = append(portfolio.Projects, model.Project{
			Title: p.Title, Technologies: p.Technologies, Link: p.Link, Year: p.Year,
		})
	}
	for _, ct := range req.Certificates {
		portfolio.Certificates = append(portfolio.Certificates, model.Certificate{
			Name: ct.Name, Issuer: ct.Issuer, IssuedAt: ct.IssuedAt,
		})
	}

	saved, err := s.portfolios.Replace(ctx, portfolio)
	if err != nil {
		return translatePortfolioError(err)
	}
	return helper.Success(c, fiber.StatusOK, "portofolio berhasil disimpan", saved)
}

// ---------- POST /students/:id/portfolio/projects ----------

func (s *PortfolioService) AddProject(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	current, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized("belum terautentikasi")
	}
	studentID, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id mahasiswa harus berupa angka positif")
	}
	if current.StudentID != studentID {
		return helper.Forbidden("hanya pemilik yang boleh menambah proyek")
	}

	var req model.ProjectRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}
	if errs := helper.ValidateStruct(req); errs != nil {
		return helper.Validation(errs)
	}

	updated, err := s.portfolios.AddProject(ctx, studentID, model.Project{
		Title: req.Title, Technologies: req.Technologies, Link: req.Link, Year: req.Year,
	})
	if err != nil {
		return translatePortfolioError(err)
	}
	return helper.Success(c, fiber.StatusCreated, "proyek berhasil ditambahkan", updated)
}

// ---------- DELETE /students/:id/portfolio/projects/:projectId ----------

func (s *PortfolioService) RemoveProject(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	current, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized("belum terautentikasi")
	}
	studentID, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id mahasiswa harus berupa angka positif")
	}
	if current.StudentID != studentID {
		return helper.Forbidden("hanya pemilik yang boleh menghapus proyek")
	}

	updated, err := s.portfolios.RemoveProject(ctx, studentID, c.Params("projectId"))
	if err != nil {
		return translatePortfolioError(err)
	}
	return helper.Success(c, fiber.StatusOK, "proyek berhasil dihapus", updated)
}

// ---------- GET /portfolios/stats/technologies ----------

func (s *PortfolioService) StatsTechnologies(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	stats, err := s.portfolios.StatsTechnologies(ctx)
	if err != nil {
		return helper.Internal(err)
	}
	return helper.Success(c, fiber.StatusOK, "statistik teknologi berhasil diambil", stats)
}

func translatePortfolioError(err error) error {
	switch {
	case errors.Is(err, repository.ErrInvalidID):
		return helper.BadRequest("id proyek tidak valid")
	case errors.Is(err, repository.ErrNotFound):
		return helper.NotFound("mahasiswa tidak ditemukan")
	case errors.Is(err, repository.ErrNotOwner):
		return helper.Forbidden("hanya pemilik yang boleh melakukan ini")
	case errors.Is(err, repository.ErrProjectLimit):
		return helper.NewError(fiber.StatusConflict, "PROJECT_LIMIT",
			"jumlah proyek sudah mencapai batas 20")
	case errors.Is(err, repository.ErrProjectNotFound):
		return helper.NewError(fiber.StatusNotFound, "PROJECT_NOT_FOUND",
			"proyek tidak ditemukan")
	default:
		return helper.Internal(err)
	}
}
