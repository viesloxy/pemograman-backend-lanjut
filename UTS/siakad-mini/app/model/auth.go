package model

// AuthUser adalah identitas pemakai yang dibaca dari payload JWT.
// Disimpan di Locals oleh middleware RequireAuth.
type AuthUser struct {
	ID        int    `json:"id"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	StudentID int    `json:"student_id"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type UserProfile struct {
	ID    int    `json:"id"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

// LoginResponse adalah isi endpoint login sesuai spesifikasi soal:
// access token, tipe token, umur token dalam detik, dan data user.
type LoginResponse struct {
	AccessToken string      `json:"access_token"`
	TokenType   string      `json:"token_type"`
	ExpiresIn   int         `json:"expires_in"`
	User        UserProfile `json:"user"`
}

// StudentProfile adalah data students yang disertakan pada /auth/me
// bila yang login berperan mahasiswa.
type StudentProfile struct {
	NIM      string `json:"nim"`
	Nama     string `json:"nama"`
	Prodi    string `json:"prodi"`
	Angkatan int    `json:"angkatan"`
}

type MeResponse struct {
	ID        int             `json:"id"`
	Email     string          `json:"email"`
	Role      string          `json:"role"`
	Mahasiswa *StudentProfile `json:"mahasiswa,omitempty"`
}
