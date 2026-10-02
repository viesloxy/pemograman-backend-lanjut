package service

import (
	"strings"

	"github.com/gofiber/fiber/v2"

	"siakad-mini/app/model"
	"siakad-mini/app/repository"
	"siakad-mini/helper"
)

type CourseService struct {
	courses repository.CourseRepository
}

func NewCourseService(courses repository.CourseRepository) *CourseService {
	return &CourseService{courses: courses}
}

// List mengembalikan daftar mata kuliah untuk semua role, lengkap
// dengan jumlah terisi dan sisa kuota yang dihitung dari enrollments.
func (s *CourseService) List(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	q := model.ListCourseQuery{
		Semester:  c.QueryInt("semester", 0),
		Search:    strings.TrimSpace(c.Query("search")),
		Available: c.Query("available") == "true",
	}

	data, err := s.courses.FindAll(ctx, q)
	if err != nil {
		return helper.Internal(err)
	}

	return helper.Success(c, fiber.StatusOK, "Data mata kuliah berhasil diambil", data)
}
