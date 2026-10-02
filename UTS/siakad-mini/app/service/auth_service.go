package service

import (
	"errors"
	"regexp"
	"strings"

	"github.com/gofiber/fiber/v2"

	"siakad-mini/app/model"
	"siakad-mini/app/repository"
	"siakad-mini/helper"
)

var emailFormat = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

type AuthService struct {
	users    repository.UserRepository
	students repository.StudentRepository
	jwt      *helper.JWTManager
}

func NewAuthService(
	users repository.UserRepository,
	students repository.StudentRepository,
	jwtManager *helper.JWTManager,
) *AuthService {
	return &AuthService{users: users, students: students, jwt: jwtManager}
}

// validateLogin adalah business rule murni: menerima struct dan
// mengembalikan daftar kesalahan per field, tanpa menyentuh fiber.Ctx.
func validateLogin(req model.LoginRequest) map[string][]string {
	errs := map[string][]string{}

	req.Email = strings.TrimSpace(req.Email)
	if req.Email == "" {
		errs["email"] = append(errs["email"], "email wajib diisi")
	} else if !emailFormat.MatchString(req.Email) {
		errs["email"] = append(errs["email"], "format email tidak valid")
	}

	if req.Password == "" {
		errs["password"] = append(errs["password"], "password wajib diisi")
	} else if len(req.Password) < 8 {
		errs["password"] = append(errs["password"], "password minimal 8 karakter")
	}

	if len(errs) == 0 {
		return nil
	}
	return errs
}

func (s *AuthService) Login(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	var req model.LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}

	if errs := validateLogin(req); errs != nil {
		return helper.Validation(errs)
	}

	user, err := s.users.FindByEmail(ctx, strings.TrimSpace(req.Email))
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			// Pesan yang sama persis dengan password salah, dan tetap
			// menjalankan pembanding bcrypt agar waktunya mirip.
			helper.VerifyDummyPassword(req.Password)
			return helper.Unauthorized("email atau password salah")
		}
		return helper.Internal(err)
	}

	if !helper.VerifyPassword(user.Password, req.Password) {
		return helper.Unauthorized("email atau password salah")
	}

	studentID := 0
	if user.Role == "mahasiswa" {
		student, err := s.students.FindByUserID(ctx, user.ID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return helper.Unauthorized("email atau password salah")
			}
			return helper.Internal(err)
		}
		studentID = student.ID
	}

	token, err := s.jwt.Generate(user, studentID)
	if err != nil {
		return helper.Internal(err)
	}

	return helper.Success(c, fiber.StatusOK, "login berhasil", model.LoginResponse{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresIn:   int(s.jwt.AccessTTL().Seconds()),
		User: model.UserProfile{
			ID:    user.ID,
			Email: user.Email,
			Role:  user.Role,
		},
	})
}

// Me mengembalikan profil pemilik token. Bila perannya mahasiswa,
// data students (nim, nama, prodi, angkatan) ikut disertakan.
func (s *AuthService) Me(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	authUser, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized("belum terautentikasi")
	}

	resp := model.MeResponse{
		ID:    authUser.ID,
		Email: authUser.Email,
		Role:  authUser.Role,
	}

	if authUser.Role == "mahasiswa" {
		student, err := s.students.FindByUserID(ctx, authUser.ID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return helper.NotFound("data mahasiswa tidak ditemukan")
			}
			return helper.Internal(err)
		}
		resp.Mahasiswa = &model.StudentProfile{
			NIM:      student.NIM,
			Nama:     student.Nama,
			Prodi:    student.Prodi,
			Angkatan: student.Angkatan,
		}
	}

	return helper.Success(c, fiber.StatusOK, "profil berhasil diambil", resp)
}
