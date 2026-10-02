package helper

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
)

// ParamID membaca parameter :id dari URL. Bila bukan angka positif,
// dianggap tidak sah (nanti dijawab 404 oleh service).
func ParamID(c *fiber.Ctx) (int, bool) {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id < 1 {
		return 0, false
	}
	return id, true
}
