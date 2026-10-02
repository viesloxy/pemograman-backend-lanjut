package helper

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
)

const requestTimeout = 5 * time.Second

// RequestContext memberi batas waktu pada setiap operasi database
// yang diturunkan dari sebuah request, agar query yang menggantung
// tidak menahan koneksi pool selamanya.
func RequestContext(c *fiber.Ctx) (context.Context, context.CancelFunc) {
	return context.WithTimeout(c.UserContext(), requestTimeout)
}
