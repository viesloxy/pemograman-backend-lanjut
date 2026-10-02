package middleware

import (
	"github.com/gofiber/fiber/v2"

	"siakad-mini/helper"
)

// RequirePermission memeriksa hak yang tidak bergantung pada isi data
// (misalnya "boleh melihat daftar?"). Pemeriksaan yang bergantung pada
// isi data seperti kepemilikan ditempatkan di layer service.
func RequirePermission(perms *helper.PermissionSet, permission string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		user, ok := helper.CurrentUser(c)
		if !ok {
			return helper.Unauthorized("belum terautentikasi")
		}
		if !perms.Can(user.Role, permission) {
			return helper.Forbidden("role " + user.Role + " tidak memiliki hak " + permission)
		}
		return c.Next()
	}
}
