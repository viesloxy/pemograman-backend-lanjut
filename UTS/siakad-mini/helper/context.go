package helper

import (
	"github.com/gofiber/fiber/v2"

	"siakad-mini/app/model"
)

const LocalsAuthUser = "authUser"

// CurrentUser membaca identitas pemakai yang disimpan oleh middleware
// RequireAuth, sehingga service tidak perlu memeriksa ulang token.
func CurrentUser(c *fiber.Ctx) (model.AuthUser, bool) {
	user, ok := c.Locals(LocalsAuthUser).(model.AuthUser)
	return user, ok
}

// RequestID membaca id unik permintaan yang dipasang middleware
// requestid, dipakai untuk menelusuri log.
func RequestID(c *fiber.Ctx) string {
	id, _ := c.Locals("requestid").(string)
	return id
}
