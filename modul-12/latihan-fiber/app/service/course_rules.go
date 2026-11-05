package service

// MaxCreditsPerSemester adalah batas SKS yang boleh diambil dalam satu KRS.
const MaxCreditsPerSemester = 24

// WithinCreditLimit adalah aturan murni: tidak menyentuh database dan
// tidak menyentuh Fiber.
func WithinCreditLimit(taken, adding int) bool {
	return taken+adding <= MaxCreditsPerSemester
}

// CreditsAfterSwap menghitung SKS setelah pindah kelas.
func CreditsAfterSwap(taken, leaving, joining int) int {
	return taken - leaving + joining
}