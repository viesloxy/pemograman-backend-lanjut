package service

import (
	"strings"
	"unicode"

	"api-students/app/model"
)

const minPasswordLength = 8

func ValidateRegister(req model.RegisterRequest) map[string]string {
	errs := map[string]string{}

	nim := strings.TrimSpace(req.NIM)
	switch {
	case nim == "":
		errs["nim"] = "wajib diisi"
	case !nimValid(nim):
		errs["nim"] = "harus berupa angka sepanjang 9 sampai 15 digit"
	}

	name := strings.TrimSpace(req.Name)
	switch {
	case name == "":
		errs["name"] = "wajib diisi"
	case len(name) < 3:
		errs["name"] = "minimal 3 karakter"
	case len(name) > 100:
		errs["name"] = "maksimal 100 karakter"
	}

	if req.Grade != nil && (*req.Grade < 0 || *req.Grade > 100) {
		errs["grade"] = "harus berada di rentang 0 sampai 100"
	}

	if msg := checkPasswordStrength(req.Password); msg != "" {
		errs["password"] = msg
	}

	return errs
}

func ValidateLogin(req model.LoginRequest) map[string]string {
	errs := map[string]string{}
	if strings.TrimSpace(req.NIM) == "" {
		errs["nim"] = "wajib diisi"
	}
	if req.Password == "" {
		errs["password"] = "wajib diisi"
	}
	return errs
}

func checkPasswordStrength(password string) string {
	if len(password) < minPasswordLength {
		return "minimal 8 karakter"
	}

	var hasLetter, hasDigit bool
	for _, r := range password {
		switch {
		case unicode.IsLetter(r):
			hasLetter = true
		case unicode.IsDigit(r):
			hasDigit = true
		}
	}
	if !hasLetter || !hasDigit {
		return "harus memuat huruf dan angka"
	}

	weak := map[string]bool{
		"password1": true, "12345678": true, "qwerty123": true,
		"admin123": true, "password123": true,
	}
	if weak[strings.ToLower(password)] {
		return "password terlalu umum"
	}

	return ""
}
