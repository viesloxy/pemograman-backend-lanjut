package helper

import (
	"github.com/gofiber/fiber/v2"

	"siakad-mini/app/model"
)

// Success mengirim respons sukses dengan amplop seragam.
func Success(c *fiber.Ctx, status int, message string, data any) error {
	return c.Status(status).JSON(model.WebResponse{
		Success: true,
		Message: message,
		Data:    data,
	})
}

// SuccessList mengirim respons daftar beserta meta pagination-nya.
func SuccessList(c *fiber.Ctx, message string, data any, meta *model.Meta) error {
	return c.Status(fiber.StatusOK).JSON(model.WebResponse{
		Success: true,
		Message: message,
		Data:    data,
		Meta:    meta,
	})
}

// Created mengirim respons 201. Bila location tidak kosong, header
// Location ikut disertakan sesuai kebiasaan REST.
func Created(c *fiber.Ctx, message string, data any, location string) error {
	if location != "" {
		c.Set("Location", location)
	}
	return c.Status(fiber.StatusCreated).JSON(model.WebResponse{
		Success: true,
		Message: message,
		Data:    data,
	})
}

// NoContent mengirim respons 204 tanpa body.
func NoContent(c *fiber.Ctx) error {
	return c.SendStatus(fiber.StatusNoContent)
}
