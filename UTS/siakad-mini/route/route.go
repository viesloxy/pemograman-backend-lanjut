package route

import (
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"

	"siakad-mini/app/service"
	"siakad-mini/helper"
	"siakad-mini/middleware"
)

// Dependencies mengumpulkan seluruh objek yang dibutuhkan route.
// Daftarnya bertambah seiring bertambahnya fitur.
type Dependencies struct {
	Pool *pgxpool.Pool
	JWT  *helper.JWTManager
	Auth *service.AuthService
}

// Register memetakan seluruh URL aplikasi ke handler-nya.
// Middleware terpasang di sini sehingga file ini terbaca sebagai
// peta perlindungan: route mana publik, route mana butuh token.
func Register(app *fiber.App, deps Dependencies) {
	api := app.Group("/api/v1")

	api.Get("/health", func(c *fiber.Ctx) error {
		if err := deps.Pool.Ping(c.UserContext()); err != nil {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"status":   "down",
				"database": "tidak terhubung",
			})
		}
		return c.JSON(fiber.Map{
			"status":   "ok",
			"database": "terhubung",
		})
	})

	// Login satu-satunya endpoint publik, dengan rate limiter sendiri.
	auth := api.Group("/auth")
	auth.Post("/login", middleware.LoginRateLimiter(), deps.Auth.Login)

	// Selebihnya wajib membawa access token.
	auth.Get("/me", middleware.RequireAuth(deps.JWT), deps.Auth.Me)
}
