package helper

import (
	"errors"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
)

var validate = newValidator()

var (
	nimRegex           = regexp.MustCompile(`^\d{12}$`)
	tahunAkademikRegex = regexp.MustCompile(`^\d{4}/\d{4}-(Ganjil|Genap)$`)
)

func newValidator() *validator.Validate {
	v := validator.New()

	// Nama field pada kesalahan validasi mengikuti tag json, sehingga
	// respons 422 menunjuk field persis seperti yang dikirim client.
	v.RegisterTagNameFunc(func(field reflect.StructField) string {
		name := strings.SplitN(field.Tag.Get("json"), ",", 2)[0]
		if name == "" || name == "-" {
			return field.Name
		}
		return name
	})

	// NIM harus tepat 12 digit angka, sama seperti CHECK constraint
	// di database.
	_ = v.RegisterValidation("nim", func(fl validator.FieldLevel) bool {
		return nimRegex.MatchString(fl.Field().String())
	})

	// tahunakademik memeriksa bentuk 2026/2027-Ganjil atau -Genap.
	_ = v.RegisterValidation("tahunakademik", func(fl validator.FieldLevel) bool {
		return tahunAkademikRegex.MatchString(fl.Field().String())
	})

	// angkatan harus 4 digit dan tidak melebihi tahun berjalan.
	_ = v.RegisterValidation("angkatan", func(fl validator.FieldLevel) bool {
		n := fl.Field().Int()
		return n >= 2000 && n <= int64(time.Now().Year())
	})

	return v
}

// ValidateStruct memeriksa struct berdasarkan tag validate dan
// mengembalikan daftar kesalahan per field dalam bentuk yang siap
// dipakai pada respons 422 (nama field -> daftar pesan).
func ValidateStruct(s any) map[string][]string {
	err := validate.Struct(s)
	if err == nil {
		return nil
	}

	var invalid *validator.InvalidValidationError
	if errors.As(err, &invalid) {
		return map[string][]string{"_": {"objek yang divalidasi tidak sah"}}
	}

	var fieldErrors validator.ValidationErrors
	if !errors.As(err, &fieldErrors) {
		return map[string][]string{"_": {"validasi gagal"}}
	}

	result := make(map[string][]string, len(fieldErrors))
	for _, fe := range fieldErrors {
		result[fe.Field()] = append(result[fe.Field()], messageFor(fe))
	}
	return result
}

func messageFor(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "wajib diisi"
	case "email":
		return "format email tidak valid"
	case "min":
		if fe.Kind() == reflect.String {
			return "minimal " + fe.Param() + " karakter"
		}
		return "nilai minimal " + fe.Param()
	case "max":
		if fe.Kind() == reflect.String {
			return "maksimal " + fe.Param() + " karakter"
		}
		return "nilai maksimal " + fe.Param()
	case "gte":
		return "nilai minimal " + fe.Param()
	case "lte":
		return "nilai maksimal " + fe.Param()
	case "gt":
		return "harus lebih besar dari " + fe.Param()
	case "nim":
		return "NIM harus tepat 12 digit angka"
	case "tahunakademik":
		return "tahun_akademik harus berformat 2026/2027-Ganjil"
	case "angkatan":
		return "angkatan harus 4 digit dan tidak melebihi tahun berjalan"
	default:
		return "tidak memenuhi aturan " + fe.Tag()
	}
}
