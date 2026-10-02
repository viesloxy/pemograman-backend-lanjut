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
	Pool        *pgxpool.Pool
	JWT         *helper.JWTManager
	Permissions *helper.PermissionSet
	Auth        *service.AuthService
	Students    *service.StudentService
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

	// Kelola mahasiswa: khusus admin, dijaga permission.
	students := api.Group("/students", middleware.RequireAuth(deps.JWT))
	students.Get("/", middleware.RequirePermission(deps.Permissions, "student:list"), deps.Students.List)
	students.Post("/", middleware.RequirePermission(deps.Permissions, "student:create"), deps.Students.Create)
}
