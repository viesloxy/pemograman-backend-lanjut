package helper

import (
	"encoding/csv"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"

	"latihan-fiber/app/model"
)

const (
	FormatJSON = fiber.MIMEApplicationJSON
	FormatCSV  = "text/csv"
)

// Negotiate memilih format response berdasarkan header Accept.
func Negotiate(c *fiber.Ctx, offered ...string) (string, error) {
	accept := strings.TrimSpace(c.Get(fiber.HeaderAccept))

	// Tidak menyebut Accept sama sekali berarti "terserah server".
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

// WriteUsersCSV menuliskan daftar user sebagai CSV.
func WriteUsersCSV(c *fiber.Ctx, users []model.User) error {
	c.Set(fiber.HeaderContentType, FormatCSV+"; charset=utf-8")
	c.Set(fiber.HeaderContentDisposition, `attachment; filename="users.csv"`)

	var buffer strings.Builder
	writer := csv.NewWriter(&buffer)

	header := []string{"id", "username", "email", "role", "is_active", "created_at"}
	if err := writer.Write(header); err != nil {
		return Internal(err)
	}

	for _, u := range users {
		row := []string{
			strconv.Itoa(u.ID), u.Username, u.Email, u.Role,
			strconv.FormatBool(u.IsActive),
			u.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		}
		if err := writer.Write(row); err != nil {
			return Internal(err)
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return Internal(err)
	}

	return c.SendString(buffer.String())
}