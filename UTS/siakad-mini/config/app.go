package config

import (
	"errors"
	"log/slog"

	"github.com/gofiber/fiber/v2"

	"siakad-mini/app/model"
	"siakad-mini/helper"
	"siakad-mini/middleware"
	"siakad-mini/route"
)

// NewApp merakit aplikasi Fiber: konfigurasi dasar, error handler
// terpusat, lalu pendaftaran route. Middleware ditambahkan di sini
// seiring berkembangnya kebutuhan.
func NewApp(logger *slog.Logger, deps route.Dependencies) *fiber.App {
	app := fiber.New(fiber.Config{
		AppName:      GetEnv("APP_NAME", "Siakad Mini API"),
		ErrorHandler: newErrorHandler(logger),
		BodyLimit:    1 * 1024 * 1024,
	})

	middleware.Register(app, logger, GetEnv("ALLOWED_ORIGINS", ""))
	route.Register(app, deps)

	return app
}

// newErrorHandler menyatukan seluruh kegagalan menjadi satu bentuk
// respons. Service dan repository cukup mengembalikan error; penulisan
// status dan body hanya terjadi di satu tempat ini.
func newErrorHandler(logger *slog.Logger) fiber.ErrorHandler {
	return func(c *fiber.Ctx, err error) error {
		var appErr *helper.AppError

		switch {
		case errors.As(err, &appErr):
		case errors.Is(err, fiber.ErrRequestEntityTooLarge):
			appErr = helper.NewAppError(
				fiber.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE",
				"ukuran body melebihi batas yang diizinkan")
		default:
			var fiberErr *fiber.Error
			if errors.As(err, &fiberErr) {
				message := fiberErr.Message
				// Fiber membuat error 404/405 secara dinamis (bukan sentinel),
				// sehingga pemeriksaan memakai kode statusnya.
				switch fiberErr.Code {
				case fiber.StatusNotFound:
					message = "endpoint tidak ditemukan"
				case fiber.StatusMethodNotAllowed:
					message = "metode tidak diizinkan untuk endpoint ini"
				}
				appErr = &helper.AppError{
					Status:  fiberErr.Code,
					Code:    "HTTP_ERROR",
					Message: message,
				}
			} else {
				appErr = helper.Internal(err)
			}
		}

		// Kegagalan server dicatat lengkap dengan penyebabnya;
		// kegagalan dari sisi client cukup dicatat ringkas.
		if appErr.Status >= fiber.StatusInternalServerError {
			logger.Error("request_failed",
				slog.String("path", c.Path()),
				slog.String("method", c.Method()),
				slog.String("code", appErr.Code),
				slog.Int("status", appErr.Status),
				slog.String("error", err.Error()))
		} else {
			logger.Warn("request_rejected",
				slog.String("path", c.Path()),
				slog.String("method", c.Method()),
				slog.String("code", appErr.Code),
				slog.Int("status", appErr.Status))
		}

		return c.Status(appErr.Status).JSON(model.WebResponse{
			Success: false,
			Message: appErr.Message,
			Errors:  appErr.Fields,
		})
	}
}
