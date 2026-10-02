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
	Courses     *service.CourseService
	Enrollments *service.EnrollmentService
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
	// Detail tanpa permission: admin bebas, mahasiswa hanya dirinya —
	// keputusannya bergantung isi data, diperiksa di service.
	students.Get("/:id", deps.Students.Detail)
	students.Put("/:id", middleware.RequirePermission(deps.Permissions, "student:update:any"), deps.Students.Update)
	students.Delete("/:id", middleware.RequirePermission(deps.Permissions, "student:delete"), deps.Students.Delete)

	// Mata kuliah dapat dilihat semua role yang sudah login.
	courses := api.Group("/courses", middleware.RequireAuth(deps.JWT))
	courses.Get("/", deps.Courses.List)

	// KRS: khusus mahasiswa. Admin dijawab 403 oleh middleware
	// permission; kepemilikan per baris diperiksa di service.
	enrollments := api.Group("/enrollments", middleware.RequireAuth(deps.JWT))
	enrollments.Post("/", middleware.RequirePermission(deps.Permissions, "enrollment:create"), deps.Enrollments.Create)
	enrollments.Delete("/:id", middleware.RequirePermission(deps.Permissions, "enrollment:delete"), deps.Enrollments.Delete)
}
