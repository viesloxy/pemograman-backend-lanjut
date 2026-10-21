package model

import "time"

type RegisterRequest struct {
	NIM      string   `json:"nim" validate:"required,nim"`
	Name     string   `json:"name" validate:"required,min=3,max=100"`
	Grade    *float64 `json:"grade" validate:"omitnil,min=0,max=100"`
	Password string   `json:"password" validate:"required,strongpassword"`
}

type LoginRequest struct {
	NIM      string `json:"nim" validate:"required"`
	Password string `json:"password" validate:"required"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
}

type RefreshToken struct {
	ID        int64
	StudentID int
	TokenHash string
	ExpiresAt time.Time
	RevokedAt *time.Time
	CreatedAt time.Time
}

type AuthUser struct {
	StudentID int    `json:"student_id"`
	NIM       string `json:"nim"`
	Role      string `json:"role"`
}
