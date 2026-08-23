package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/requestid"
)

// metodeBerbody menandai metode yang memang membawa body, jadi hanya
// metode inilah yang Content-Type nya perlu diperiksa.
var metodeBerbody = map[string]bool{
	fiber.MethodPost:  true,
	fiber.MethodPut:   true,
	fiber.MethodPatch: true,
}

// requireJSON menolak request berisi body yang Content-Type nya bukan JSON.
// Status yang tepat untuk kasus ini 415, bukan 400: bentuk kirimannya yang
// tidak didukung, bukan isinya yang salah.
func requireJSON(c *fiber.Ctx) error {
	if metodeBerbody[c.Method()] {
		ct := c.Get("Content-Type")
		// HasPrefix, bukan perbandingan sama dengan, karena Content-Type
		// sering membawa embel-embel seperti "; charset=utf-8".
		if !strings.HasPrefix(ct, fiber.MIMEApplicationJSON) {
			return fail(c, fiber.StatusUnsupportedMediaType,
				"Content-Type harus application/json")
		}
	}
	return c.Next()
}

func main() {
	app := fiber.New(fiber.Config{
		AppName: "API Students - Praktikum Backend Lanjut Modul 2",
		// ErrorHandler menangkap error yang lolos dari handler lalu
		// membungkusnya ke amplop respons yang sama seperti endpoint lain,
		// supaya klien tidak pernah menerima dua bentuk respons berbeda.
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			status := fiber.StatusInternalServerError
			pesan := "terjadi kesalahan pada server"
			if e, ok := err.(*fiber.Error); ok {
				status = e.Code
				pesan = e.Message
			}
			return fail(c, status, pesan)
		},
	})

	// Urutan pemasangan menentukan urutan eksekusi. requestid dipasang
	// paling awal supaya id-nya sudah tersedia ketika logger mencetak baris.
	app.Use(requestid.New())
	app.Use(logger.New(logger.Config{
		Format: "[${time}] ${locals:requestid} ${method} ${path} ${status} ${latency}\n",
	}))
	app.Use(cors.New())

	api := app.Group("/api/v1")

	api.Get("/health", func(c *fiber.Ctx) error {
		return ok(c, "server berjalan", fiber.Map{"timestamp": time.Now()})
	})

	// requireJSON dipasang pada grup students saja, bukan global, supaya
	// endpoint yang kelak menerima unggahan berkas tidak ikut tertolak.
	s := api.Group("/students", requireJSON)
	s.Get("/", listStudents)
	s.Get("/:id", getStudent)
	s.Post("/", createStudent)
	s.Put("/:id", replaceStudent)
	s.Patch("/:id", patchStudent)
	s.Delete("/:id", deleteStudent)

	// Jalur yang tidak dikenal. Ditaruh paling bawah karena middleware
	// tanpa path menangkap apa pun yang lolos sebelumnya.
	app.Use(func(c *fiber.Ctx) error {
		return fail(c, fiber.StatusNotFound, "endpoint tidak ditemukan")
	})

	fmt.Println("Server berjalan di http://localhost:3000")
	log.Fatal(app.Listen(":3000"))
}
