package helper

import (
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"

	"api-students/app/model"
)

const (
	FormatJSON = "application/json"
	FormatCSV  = "text/csv"
)

func Negotiate(c *fiber.Ctx, offered ...string) (string, error) {
	accept := strings.TrimSpace(c.Get("Accept"))

	if accept == "" {
		return offered[0], nil
	}

	chosen := c.Accepts(offered...)
	if chosen == "" {
		return "", NotAcceptable(
			"format yang diminta tidak tersedia, pilih salah satu dari: " +
				strings.Join(offered, ", "))
	}
	return chosen, nil
}

func WriteStudentsCSV(c *fiber.Ctx, students []model.Student) error {
	c.Set("Content-Type", FormatCSV+"; charset=utf-8")
	c.Set("Content-Disposition", `attachment; filename="students.csv"`)

	var lines strings.Builder
	lines.WriteString("id,nim,name,grade,role,is_active,created_at\n")

	for _, s := range students {
		grade := strconv.FormatFloat(s.Grade, 'f', 2, 64)
		created := s.CreatedAt.UTC().Format("2006-01-02T15:04:05Z")
		lines.WriteString(strconv.Itoa(s.ID) + "," + s.NIM + "," + s.Name + "," +
			grade + "," + s.Role + "," + strconv.FormatBool(s.IsActive) + "," + created + "\n")
	}

	return c.SendString(lines.String())
}

func ParseCursorQuery(c *fiber.Ctx) (model.CursorQuery, error) {
	q := model.CursorQuery{
		Limit:    c.QueryInt("limit", 10),
		Search:   strings.TrimSpace(c.Query("search")),
		IsActive: nil,
	}

	if q.Limit < 1 {
		q.Limit = 10
	}
	if q.Limit > 50 {
		q.Limit = 50
	}

	if raw := c.Query("is_active"); raw != "" {
		if v, err := strconv.ParseBool(raw); err == nil {
			q.IsActive = &v
		}
	}

	if encoded := c.Query("cursor"); encoded != "" {
		cursor, err := DecodeCursor(encoded)
		if err != nil {
			return q, BadRequest("cursor tidak valid")
		}
		q.Cursor = &cursor
	}

	return q, nil
}
