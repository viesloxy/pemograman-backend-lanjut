package route

import (
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Dependencies mengumpulkan seluruh objek yang dibutuhkan route.
// Daftarnya bertambah seiring bertambahnya fitur.
type Dependencies struct {
	Pool *pgxpool.Pool
}

// Register memetakan seluruh URL aplikasi ke handler-nya.
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
}
