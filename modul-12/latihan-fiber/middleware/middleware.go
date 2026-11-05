package middleware

import (
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/helmet"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"

	"latihan-fiber/helper"
)

// Register memasang seluruh middleware yang berlaku untuk semua route.
func Register(app *fiber.App, logger *slog.Logger, allowedOrigins string) {
	app.Use(requestid.New())
	app.Use(recover.New())
	app.Use(helmet.New())
	app.Use(corsPolicy(allowedOrigins))
	app.Use(RequestLogger(logger))
}

// corsPolicy membatasi origin yang boleh memanggil API.
func corsPolicy(allowedOrigins string) fiber.Handler {
	if strings.TrimSpace(allowedOrigins) == "" {
		allowedOrigins = "http://localhost:5173"
	}

	return cors.New(cors.Config{
		AllowOrigins: allowedOrigins,
		AllowMethods: "GET,POST,PUT,PATCH,DELETE,OPTIONS",
		AllowHeaders: "Origin,Content-Type,Accept,Authorization",
	})
}

// RequestLogger mencatat setiap request ke log terstruktur.
func RequestLogger(logger *slog.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		err := c.Next()

		requestID, _ := c.Locals("requestid").(string)

		// Sejak handler mengembalikan error alih-alih menulis response
		// sendiri, status pada c.Response() BELUM terisi ketika baris ini
		// dijalankan. Ekstrak status dari error bila ada.
		status := c.Response().StatusCode()
		if err != nil {
			var appErr *helper.AppError
			if errors.As(err, &appErr) {
				status = appErr.Status
			} else {
				status = fiber.StatusInternalServerError
			}
		}

		attrs := []any{
			slog.String("request_id", requestID),
			slog.String("method", c.Method()),
			slog.String("path", c.Path()),
			slog.Int("status", status),
			slog.Duration("duration", time.Since(start)),
			slog.String("ip", c.IP()),
		}

		// Identitas ikut dicatat bila request sudah melewati RequireAuth.
		if user, ok := helper.CurrentUser(c); ok {
			attrs = append(attrs,
				slog.Int("user_id", user.UserID),
				slog.String("role", user.Role))
		}

		logger.Info("http_request", attrs...)
		return err
	}
}

var methodsWithBody = map[string]bool{
	fiber.MethodPost:  true,
	fiber.MethodPut:   true,
	fiber.MethodPatch: true,
}

// RequireJSON menolak request berisi body yang Content-Type-nya bukan JSON.
func RequireJSON(c *fiber.Ctx) error {
	if methodsWithBody[c.Method()] {
		ct := c.Get("Content-Type")
		if !strings.HasPrefix(ct, fiber.MIMEApplicationJSON) {
			return helper.UnsupportedMediaType("Content-Type harus application/json")
		}
	}
	return c.Next()
}