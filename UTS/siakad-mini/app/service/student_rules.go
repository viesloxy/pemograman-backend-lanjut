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
	case !emailFormat.MatchString(req.Email):
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

// validateUpdateStudent memeriksa field pembaruan mahasiswa. NIM dan
// email sengaja tidak ada di struct request, sehingga tidak dapat
// diubah melalui endpoint ini.
func validateUpdateStudent(req model.UpdateStudentRequest) map[string][]string {
	errs := map[string][]string{}

	req.Nama = strings.TrimSpace(req.Nama)
	switch {
	case req.Nama == "":
		errs["nama"] = append(errs["nama"], "nama wajib diisi")
	case len(req.Nama) > maxNamaLen:
		errs["nama"] = append(errs["nama"], "nama maksimal 150 karakter")
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

// BatasSKS menerjemahkan IPK menjadi batas pengambilan SKS per
// semester: 3,00 ke atas 24, 2,50 sampai 2,99 sebanyak 21, sisanya 18.
// IPK kosong dianggap bracket terendah.
func BatasSKS(ipk *float64) int {
	if ipk == nil {
		return 18
	}
	switch {
	case *ipk >= 3.00:
		return 24
	case *ipk >= 2.50:
		return 21
	default:
		return 18
	}
}

// BolehLihatStudent memutuskan siapa boleh melihat detail seorang
// mahasiswa: admin boleh melihat semua, mahasiswa hanya dirinya.
func BolehLihatStudent(current model.AuthUser, studentUserID int) bool {
	if current.Role == "admin" {
		return true
	}
	return current.ID == studentUserID
}
