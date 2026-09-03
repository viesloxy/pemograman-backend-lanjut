package main

import (
	"api-students/app/model"
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

const (
	defaultPage  = 1
	defaultLimit = 10
	maxLimit     = 50
	maxPage      = 100000
)

var allowedSort = map[string]bool{
	"id":         true,
	"nim":        true,
	"name":       true,
	"grade":      true,
	"created_at": true,
}

func ok(c *fiber.Ctx, message string, data any) error {
	return c.Status(fiber.StatusOK).JSON(model.WebResponse{
		Success: true, Message: message, Data: data,
	})
}

func okList(c *fiber.Ctx, message string, data any, meta *model.Meta) error {
	return c.Status(fiber.StatusOK).JSON(model.WebResponse{
		Success: true, Message: message, Data: data, Meta: meta,
	})
}

func created(c *fiber.Ctx, message string, data any, location string) error {
	c.Set("Location", location)
	return c.Status(fiber.StatusCreated).JSON(model.WebResponse{
		Success: true, Message: message, Data: data,
	})
}

func noContent(c *fiber.Ctx) error {
	return c.SendStatus(fiber.StatusNoContent)
}

func fail(c *fiber.Ctx, status int, message string) error {
	return c.Status(status).JSON(model.WebResponse{Success: false, Message: message})
}

func failValidation(c *fiber.Ctx, errs map[string]string) error {
	return c.Status(fiber.StatusUnprocessableEntity).JSON(model.WebResponse{
		Success: false, Message: "validasi gagal", Errors: errs,
	})
}

func reqCtx(c *fiber.Ctx) (context.Context, context.CancelFunc) {
	return context.WithTimeout(c.UserContext(), 5*time.Second)
}

func parseListQuery(c *fiber.Ctx) model.ListQuery {
	q := model.ListQuery{
		Page:   c.QueryInt("page", defaultPage),
		Limit:  c.QueryInt("limit", defaultLimit),
		Search: strings.TrimSpace(c.Query("search")),
		Sort:   strings.ToLower(strings.TrimSpace(c.Query("sort", "id"))),
		Order:  strings.ToLower(strings.TrimSpace(c.Query("order", "asc"))),
	}

	if q.Page < 1 {
		q.Page = defaultPage
	}
	if q.Page > maxPage {
		q.Page = maxPage
	}
	if q.Limit < 1 {
		q.Limit = defaultLimit
	}
	if q.Limit > maxLimit {
		q.Limit = maxLimit
	}
	if !allowedSort[q.Sort] {
		q.Sort = "id"
	}
	if q.Order != "desc" {
		q.Order = "asc"
	}

	if raw := c.Query("is_active"); raw != "" {
		if v, err := strconv.ParseBool(raw); err == nil {
			q.IsActive = &v
		}
	}
	if raw := c.Query("min_grade"); raw != "" {
		if v, err := strconv.ParseFloat(raw, 64); err == nil {
			q.MinGrade = &v
		}
	}
	if raw := c.Query("max_grade"); raw != "" {
		if v, err := strconv.ParseFloat(raw, 64); err == nil {
			q.MaxGrade = &v
		}
	}

	return q
}

func nimValid(nim string) bool {
	if len(nim) < 9 || len(nim) > 15 {
		return false
	}
	for _, r := range nim {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func validasiIsi(nim, name string, grade *float64, wajibLengkap bool) map[string]string {
	errs := map[string]string{}

	if nim == "" {
		if wajibLengkap {
			errs["nim"] = "wajib diisi"
		}
	} else if !nimValid(nim) {
		errs["nim"] = "harus berupa angka sepanjang 9 sampai 15 digit"
	}

	if name == "" {
		if wajibLengkap {
			errs["name"] = "wajib diisi"
		}
	} else if len(name) < 3 {
		errs["name"] = "minimal 3 karakter"
	} else if len(name) > 100 {
		errs["name"] = "maksimal 100 karakter"
	}

	if grade == nil {
		if wajibLengkap {
			errs["grade"] = "wajib diisi"
		}
	} else if *grade < 0 || *grade > 100 {
		errs["grade"] = "harus berada di rentang 0 sampai 100"
	}

	return errs
}
