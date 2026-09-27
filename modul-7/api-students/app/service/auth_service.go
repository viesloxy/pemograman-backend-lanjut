package service

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"api-students/app/model"
	"api-students/app/repository"
	"api-students/helper"
)

const refreshTokenBytes = 32

func gradeValue(grade *float64) float64 {
	if grade == nil {
		return 0
	}
	return *grade
}

type AuthService struct {
	students   repository.StudentRepository
	tokens     repository.TokenRepository
	jwt        *helper.JWTManager
	refreshTTL time.Duration
}

func NewAuthService(
	students repository.StudentRepository,
	tokens repository.TokenRepository,
	jwtManager *helper.JWTManager,
	refreshTTL time.Duration,
) *AuthService {
	return &AuthService{
		students: students, tokens: tokens, jwt: jwtManager, refreshTTL: refreshTTL,
	}
}

func (s *AuthService) Register(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	var req model.RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}

	req.NIM = strings.TrimSpace(req.NIM)
	req.Name = strings.TrimSpace(req.Name)

	if errs := ValidateRegister(req); len(errs) > 0 {
		return helper.Validation(errs)
	}

	hashed, err := helper.HashPassword(req.Password)
	if err != nil {
		return helper.Internal(err)
	}

	created, err := s.students.Create(ctx, model.Student{
		NIM:      req.NIM,
		Name:     req.Name,
		Grade:    gradeValue(req.Grade),
		Password: hashed,
		Role:     "user",
		IsActive: true,
	})
	if err != nil {
		if errors.Is(err, repository.ErrDuplicate) {
			return helper.Conflict("NIM sudah terdaftar")
		}
		return helper.Internal(err)
	}

	return helper.Created(c, "pendaftaran berhasil", created,
		"/api/v1/students/"+strconv.Itoa(created.ID))
}

func (s *AuthService) Login(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	var req model.LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}

	if errs := ValidateLogin(req); len(errs) > 0 {
		return helper.Validation(errs)
	}

	siswa, err := s.students.FindByNIM(ctx, strings.TrimSpace(req.NIM))
	if err != nil {
		helper.VerifyDummyPassword(req.Password)
		return helper.Unauthorized("NIM atau password salah")
	}

	if !helper.VerifyPassword(siswa.Password, req.Password) {
		return helper.Unauthorized("NIM atau password salah")
	}

	if !siswa.IsActive {
		return helper.Forbidden("akun dinonaktifkan")
	}

	pasangan, err := s.issueTokenPair(ctx, siswa)
	if err != nil {
		return helper.Internal(err)
	}

	return helper.Success(c, fiber.StatusOK, "login berhasil", pasangan)
}

func (s *AuthService) Refresh(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	var req model.RefreshRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}
	if strings.TrimSpace(req.RefreshToken) == "" {
		return helper.BadRequest("refresh_token wajib diisi")
	}

	hash := helper.SHA256Hex(req.RefreshToken)
	tersimpan, err := s.tokens.FindActive(ctx, hash)
	if err != nil {
		return helper.Unauthorized(
			"refresh token tidak valid atau sudah kedaluwarsa")
	}

	siswa, err := s.students.FindByID(ctx, tersimpan.StudentID)
	if err != nil || !siswa.IsActive {
		return helper.Unauthorized("akun tidak dapat dipakai")
	}

	if err := s.tokens.Revoke(ctx, hash); err != nil {
		return helper.Internal(err)
	}

	pasangan, err := s.issueTokenPair(ctx, siswa)
	if err != nil {
		return helper.Internal(err)
	}

	return helper.Success(c, fiber.StatusOK, "token berhasil diperbarui", pasangan)
}

func (s *AuthService) Logout(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	var req model.RefreshRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}

	if strings.TrimSpace(req.RefreshToken) != "" {
		_ = s.tokens.Revoke(ctx, helper.SHA256Hex(req.RefreshToken))
	}

	return helper.Success(c, fiber.StatusOK, "logout berhasil", nil)
}

func (s *AuthService) Me(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	authUser, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized("belum terautentikasi")
	}

	siswa, err := s.students.FindByID(ctx, authUser.StudentID)
	if err != nil {
		return helper.Unauthorized("mahasiswa tidak ditemukan")
	}

	return helper.Success(c, fiber.StatusOK, "profil berhasil diambil", siswa)
}

func (s *AuthService) issueTokenPair(
	ctx context.Context, siswa model.Student,
) (model.TokenPair, error) {
	accessToken, err := s.jwt.GenerateAccess(siswa)
	if err != nil {
		return model.TokenPair{}, err
	}

	refreshToken, err := helper.RandomToken(refreshTokenBytes)
	if err != nil {
		return model.TokenPair{}, err
	}

	err = s.tokens.Save(ctx, model.RefreshToken{
		StudentID: siswa.ID,
		TokenHash: helper.SHA256Hex(refreshToken),
		ExpiresAt: time.Now().Add(s.refreshTTL),
	})
	if err != nil {
		return model.TokenPair{}, err
	}

	return model.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int(s.jwt.AccessTTL().Seconds()),
	}, nil
}
