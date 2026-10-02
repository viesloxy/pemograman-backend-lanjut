package service

import (
	"regexp"

	"siakad-mini/app/model"
)

var tahunAkademikFormat = regexp.MustCompile(`^\d{4}/\d{4}-(Ganjil|Genap)$`)

// validateCreateEnrollment memeriksa body pengambilan mata kuliah.
// Fungsi ini murni dan dapat diuji tanpa server maupun database.
func validateCreateEnrollment(req model.CreateEnrollmentRequest) map[string][]string {
	errs := map[string][]string{}

	if req.CourseID == 0 {
		errs["course_id"] = append(errs["course_id"], "course_id wajib diisi")
	} else if req.CourseID < 1 {
		errs["course_id"] = append(errs["course_id"], "course_id tidak valid")
	}

	switch {
	case req.TahunAkademik == "":
		errs["tahun_akademik"] = append(errs["tahun_akademik"], "tahun_akademik wajib diisi")
	case !tahunAkademikFormat.MatchString(req.TahunAkademik):
		errs["tahun_akademik"] = append(errs["tahun_akademik"],
			"tahun_akademik harus berformat 2026/2027-Ganjil")
	}

	if len(errs) == 0 {
		return nil
	}
	return errs
}

// SisaSKS menghitung sisa jatah SKS seorang mahasiswa.
func SisaSKS(batas, terpakai int) int {
	if terpakai >= batas {
		return 0
	}
	return batas - terpakai
}
