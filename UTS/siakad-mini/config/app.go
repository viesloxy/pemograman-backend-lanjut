package config

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"

	"siakad-mini/route"
)

// NewApp merakit aplikasi Fiber: konfigurasi dasar, lalu pendaftaran route.
// Middleware ditambahkan di sini seiring berkembangnya kebutuhan.
func NewApp(logger *slog.Logger, deps route.Dependencies) *fiber.App {
	app := fiber.New(fiber.Config{
		AppName:   GetEnv("APP_NAME", "Siakad Mini API"),
		BodyLimit: 1 * 1024 * 1024,
	})

	route.Register(app, deps)

	app.Use(func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"message": "endpoint tidak ditemukan",
		})
	})

	return app
}
