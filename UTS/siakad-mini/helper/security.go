package helper

import "golang.org/x/crypto/bcrypt"

const bcryptCost = 12

// dummyHash dipakai saat login dengan email yang tidak terdaftar.
// Percobaan tetap menjalankan pembanding bcrypt agar waktunya mirip
// dengan login yang benar-benar memeriksa password (mitigasi timing
// attack sekaligus penutupan user enumeration).
var dummyHash = []byte("$2a$12$abcdefghijklmnopqrstuuLKa3Bt1TCmU/6zvhZ8x4nq1yBiuGvS")

// HashPassword meng-hash password dengan bcrypt. Cost 12 dipilih agar
// pemeriksaan sengaja memakan waktu sekitar sepersekian detik.
func HashPassword(plain string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(plain), bcryptCost)
	if err != nil {
		return "", err
	}
	return string(hashed), nil
}

// VerifyPassword membandingkan password polos dengan hash di database.
func VerifyPassword(hash, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}

// VerifyDummyPassword sengaja membuang waktunya untuk membandingkan
// password dengan hash palsu.
func VerifyDummyPassword(plain string) {
	_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(plain))
}
