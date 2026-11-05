package helper

import (
	"errors"
	"reflect"
	"strings"
	"unicode"

	"github.com/go-playground/validator/v10"
)

// validate dibuat SEKALI untuk seluruh aplikasi.
var validate = newValidator()

func newValidator() *validator.Validate {
	v := validator.New()

	// Pesan error memakai nama JSON ("username"), bukan nama field Go.
	v.RegisterTagNameFunc(func(field reflect.StructField) string {
		name := strings.SplitN(field.Tag.Get("json"), ",", 2)[0]
		if name == "" || name == "-" {
			return field.Name
		}
		return name
	})

	_ = v.RegisterValidation("nospace", func(fl validator.FieldLevel) bool {
		return !strings.ContainsAny(fl.Field().String(), " \t\n\r")
	})

	_ = v.RegisterValidation("username", func(fl validator.FieldLevel) bool {
		for _, r := range fl.Field().String() {
			if !unicode.IsLetter(r) && !unicode.IsDigit(r) &&
				r != '.' && r != '_' {
				return false
			}
		}
		return true
	})

	_ = v.RegisterValidation("strongpassword", func(fl validator.FieldLevel) bool {
		return checkPasswordStrength(fl.Field().String())
	})

	return v
}

// ValidateStruct menjalankan seluruh aturan pada tag struct dan
// mengembalikan peta nama field ke pesan berbahasa Indonesia.
func ValidateStruct(s any) map[string]string {
	err := validate.Struct(s)
	if err == nil {
		return nil
	}

	var invalid *validator.InvalidValidationError
	if errors.As(err, &invalid) {
		return map[string]string{"_": "objek yang divalidasi tidak sah"}
	}

	var fieldErrors validator.ValidationErrors
	if !errors.As(err, &fieldErrors) {
		return map[string]string{"_": "validasi gagal"}
	}

	result := make(map[string]string, len(fieldErrors))
	for _, fe := range fieldErrors {
		if _, exists := result[fe.Field()]; !exists {
			result[fe.Field()] = messageFor(fe)
		}
	}
	return result
}

// messageFor menerjemahkan nama tag menjadi kalimat yang dapat dibaca pemakai.
func messageFor(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "wajib diisi"
	case "email":
		return "format email tidak valid"
	case "min":
		switch fe.Kind() {
		case reflect.String:
			return "minimal " + fe.Param() + " karakter"
		case reflect.Slice, reflect.Array, reflect.Map:
			return "minimal " + fe.Param() + " item"
		}
		return "nilai minimal " + fe.Param()
	case "max":
		switch fe.Kind() {
		case reflect.String:
			return "maksimal " + fe.Param() + " karakter"
		case reflect.Slice, reflect.Array, reflect.Map:
			return "maksimal " + fe.Param() + " item"
		}
		return "nilai maksimal " + fe.Param()
	case "alphanum":
		return "hanya boleh berisi huruf dan angka"
	case "nospace":
		return "tidak boleh mengandung spasi"
	case "username":
		return "hanya boleh huruf, angka, titik, dan garis bawah"
	case "strongpassword":
		if value, ok := fe.Value().(string); ok {
			return passwordStrength(value)
		}
		return "password tidak memenuhi syarat"
	case "oneof":
		return "harus salah satu dari: " +
			strings.ReplaceAll(fe.Param(), " ", ", ")
	default:
		return "tidak memenuhi aturan " + fe.Tag()
	}
}

// checkPasswordStrength mengembalikan true bila password memenuhi syarat.
func checkPasswordStrength(password string) bool {
	return passwordStrength(password) == ""
}

// passwordStrength mengembalikan pesan kesalahan bila password lemah,
// atau string kosong bila password valid.
func passwordStrength(password string) string {
	const minPasswordLength = 8

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