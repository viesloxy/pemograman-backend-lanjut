package service

import (
	"testing"

	"latihan-fiber/app/model"
)

// Pengujian ini tidak menyalakan server, tidak menyentuh database,
// dan tidak membuat fiber.Ctx.
func TestCountTotalPages(t *testing.T) {
	cases := []struct{ total, limit, want int }{
		{0, 10, 0},
		{1, 10, 1},
		{10, 10, 1},
		{11, 10, 2},
		{137, 20, 7},
	}

	for _, tc := range cases {
		if got := CountTotalPages(tc.total, tc.limit); got != tc.want {
			t.Errorf("total=%d limit=%d: harap %d, dapat %d",
				tc.total, tc.limit, tc.want, got)
		}
	}
}

func TestApplyPatch(t *testing.T) {
	initial := model.User{ID: 1, Username: "sari", Email: "sari@mail.com", IsActive: true}
	inactive := false

	result := ApplyPatch(initial, model.PatchUserRequest{IsActive: &inactive})
	if result.IsActive {
		t.Error("is_active seharusnya berubah menjadi false")
	}
	if result.Username != "sari" {
		t.Error("field yang tidak dikirim seharusnya tidak berubah")
	}
}

func TestIsEmptyPatch(t *testing.T) {
	if !IsEmptyPatch(model.PatchUserRequest{}) {
		t.Error("body kosong seharusnya dianggap empty patch")
	}
	email := "x@y.ac.id"
	if IsEmptyPatch(model.PatchUserRequest{Email: &email}) {
		t.Error("body yang berisi email tidak boleh dianggap empty patch")
	}
}