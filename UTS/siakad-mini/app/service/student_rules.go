package service

import (
	"regexp"
	"strings"
	"time"

	"siakad-mini/app/model"
)

var (
	nimFormat   = regexp.MustCompile(`^\d{12}$`)
	angkatanMax = time.Now().Year()
	angkatanMin = 2000
	maxNamaLen  = 150
	maxProdiLen = 100
	maxEmailLen = 255
)

// validateCreateStudent memeriksa seluruh field pembuatan mahasiswa
// dan mengembalikan kesalahan per field. Fungsi ini murni: menerima
// struct, mengembalikan map, tanpa fiber.Ctx maupun database.
func validateCreateStudent(req model.CreateStudentRequest) map[string][]string {
	errs := map[string][]string{}

	req.NIM = strings.TrimSpace(req.NIM)
	switch {
	case req.NIM == "":
		errs["nim"] = append(errs["nim"], "NIM wajib diisi")
	case !nimFormat.MatchString(req.NIM):
		errs["nim"] = append(errs["nim"], "NIM harus tepat 12 digit angka")
	}

	req.Nama = strings.TrimSpace(req.Nama)
	switch {
	case req.Nama == "":
		errs["nama"] = append(errs["nama"], "nama wajib diisi")
	case len(req.Nama) > maxNamaLen:
		errs["nama"] = append(errs["nama"], "nama maksimal 150 karakter")
	}

	req.Email = strings.TrimSpace(req.Email)
	switch {
	case req.Email == "":
		errs["email"] = append(errs["email"], "email wajib diisi")
	case len(req.Email) > maxEmailLen:
		errs["email"] = append(errs["email"], "email maksimal 255 karakter")
		errs["email"] = append(errs["email"], "format email tidak valid")
	}

	req.Prodi = strings.TrimSpace(req.Prodi)
	switch {
	case req.Prodi == "":
		errs["prodi"] = append(errs["prodi"], "prodi wajib diisi")
	case len(req.Prodi) > maxProdiLen:
		errs["prodi"] = append(errs["prodi"], "prodi maksimal 100 karakter")
	}

	switch {
	case req.Angkatan == 0:
		errs["angkatan"] = append(errs["angkatan"], "angkatan wajib diisi")
	case req.Angkatan < angkatanMin || req.Angkatan > angkatanMax:
		errs["angkatan"] = append(errs["angkatan"],
			"angkatan harus 4 digit dan tidak melebihi tahun berjalan")
	}

	if req.IPKTerakhir != nil && (*req.IPKTerakhir < 0 || *req.IPKTerakhir > 4) {
		errs["ipk_terakhir"] = append(errs["ipk_terakhir"], "IPK harus di antara 0,00 dan 4,00")
	}

	if len(errs) == 0 {
		return nil
	}
	return errs
}
