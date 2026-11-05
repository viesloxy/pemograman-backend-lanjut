package helper

import (
	"github.com/gofiber/fiber/v2"

	"latihan-fiber/app/model"
)

// LocalsAuthUser adalah kunci penyimpanan identitas pemakai di dalam
// context request.
const LocalsAuthUser = "authUser"

func CurrentUser(c *fiber.Ctx) (model.AuthUser, bool) {
	user, ok := c.Locals(LocalsAuthUser).(model.AuthUser)
	return user, ok
}