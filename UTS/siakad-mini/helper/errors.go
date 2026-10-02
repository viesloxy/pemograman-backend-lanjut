package helper

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
)

// Kode error yang stabil. Message boleh berubah kapan saja, tetapi
// Status dan bentuk responsnya adalah kontrak bagi client.
const (
	CodeValidation       = "VALIDATION_ERROR"
	CodeBadRequest       = "BAD_REQUEST"
	CodeUnauthorized     = "UNAUTHORIZED"
	CodeForbidden        = "FORBIDDEN"
	CodeNotFound         = "NOT_FOUND"
	CodeConflict         = "CONFLICT"
	CodeUnsupportedMedia = "UNSUPPORTED_MEDIA_TYPE"
	CodeTooManyRequests  = "TOO_MANY_REQUESTS"
	CodeInternal         = "INTERNAL_ERROR"
)

// AppError adalah satu-satunya bentuk kegagalan yang dikenal aplikasi.
// Ia tidak menyentuh fiber.Ctx: sebuah error hanya menggambarkan apa
// yang salah, urusan menuliskannya sebagai respons diserahkan kepada
// ErrorHandler terpusat di config/app.go.
//
// Fields dipakai untuk kegagalan validasi: nama field berisi daftar
// pesan kesalahan, sesuai format respons pada soal.
type AppError struct {
	Status  int
	Code    string
	Message string
	Fields  map[string][]string
	cause   error
}

func (e *AppError) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap membuat errors.Is dan errors.As tetap dapat menembus AppError
// untuk menemukan error asli di bawahnya.
func (e *AppError) Unwrap() error { return e.cause }

func NewAppError(status int, code, message string) *AppError {
	return &AppError{Status: status, Code: code, Message: message}
}

func BadRequest(message string) *AppError {
	return NewAppError(fiber.StatusBadRequest, CodeBadRequest, message)
}

func Unauthorized(message string) *AppError {
	return NewAppError(fiber.StatusUnauthorized, CodeUnauthorized, message)
}

func Forbidden(message string) *AppError {
	return NewAppError(fiber.StatusForbidden, CodeForbidden, message)
}

func NotFound(message string) *AppError {
	return NewAppError(fiber.StatusNotFound, CodeNotFound, message)
}

func Conflict(message string) *AppError {
	return NewAppError(fiber.StatusConflict, CodeConflict, message)
}

func Validation(fields map[string][]string) *AppError {
	return &AppError{
		Status:  fiber.StatusUnprocessableEntity,
		Code:    CodeValidation,
		Message: "Validasi gagal",
		Fields:  fields,
	}
}

func UnsupportedMediaType(message string) *AppError {
	return NewAppError(fiber.StatusUnsupportedMediaType, CodeUnsupportedMedia, message)
}

func TooManyRequests(message string) *AppError {
	return NewAppError(fiber.StatusTooManyRequests, CodeTooManyRequests, message)
}

// Internal sengaja memakai pesan yang seragam dan tidak informatif.
// Detail teknisnya disimpan pada cause dan hanya muncul di log, karena
// pesan error database sering membocorkan nama tabel dan struktur query.
func Internal(cause error) *AppError {
	return &AppError{
		Status:  fiber.StatusInternalServerError,
		Code:    CodeInternal,
		Message: "terjadi kesalahan pada server",
		cause:   cause,
	}
}
