package service

import (
	"testing"

	"api-students/app/model"
)

func TestValidateRegister(t *testing.T) {
	grade := 88.0
	valid := model.RegisterRequest{
		NIM: "434241084", Name: "Vito Aditya", Grade: &grade, Password: "rahasia123",
	}
	if errs := ValidateRegister(valid); len(errs) != 0 {
		t.Errorf("harusnya valid, dapat %v", errs)
	}

	kosong := model.RegisterRequest{Password: "rahasia123"}
	errs := ValidateRegister(kosong)
	if errs["nim"] != "wajib diisi" || errs["name"] != "wajib diisi" {
		t.Errorf("nim dan name wajib ditandai: %v", errs)
	}

	lemah := model.RegisterRequest{
		NIM: "4342410ab", Name: "Budi", Password: "password1",
	}
	errs = ValidateRegister(lemah)
	if errs["password"] != "password terlalu umum" {
		t.Errorf("harusnya password ditolak sebagai umum: %v", errs)
	}
	if errs["nim"] == "" {
		t.Error("harusnya nim ditandai karena bukan digit")
	}
}

func TestValidateLogin(t *testing.T) {
	kosong := model.LoginRequest{}
	errs := ValidateLogin(kosong)
	if errs["nim"] != "wajib diisi" || errs["password"] != "wajib diisi" {
		t.Errorf("kedua field wajib ditandai: %v", errs)
	}

	isi := model.LoginRequest{NIM: "434241084", Password: "apapun123"}
	if errs := ValidateLogin(isi); len(errs) != 0 {
		t.Errorf("login lengkap seharusnya lolos validasi bentuk: %v", errs)
	}
}

func TestCheckPasswordStrength(t *testing.T) {
	cases := []struct {
		password string
		want     string
	}{
		{"rahasia123", ""},
		{"pendek1", "minimal 8 karakter"},
		{"tanpaangka", "harus memuat huruf dan angka"},
		{"12345678", "harus memuat huruf dan angka"},
		{"qwerty123", "password terlalu umum"},
		{"admin123", "password terlalu umum"},
	}

	for _, tc := range cases {
		if got := checkPasswordStrength(tc.password); got != tc.want {
			t.Errorf("password %q: harap %q, dapat %q", tc.password, tc.want, got)
		}
	}
}
