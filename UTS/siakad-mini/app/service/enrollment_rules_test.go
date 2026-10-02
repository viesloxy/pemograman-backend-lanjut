package service

import (
	"testing"

	"siakad-mini/app/model"
)

func TestValidateCreateEnrollment(t *testing.T) {
	sah := model.CreateEnrollmentRequest{CourseID: 1, TahunAkademik: "2026/2027-Ganjil"}
	if errs := validateCreateEnrollment(sah); errs != nil {
		t.Errorf("request sah justru ditolak: %v", errs)
	}

	salah := model.CreateEnrollmentRequest{CourseID: 0, TahunAkademik: "2026-Ganjil"}
	errs := validateCreateEnrollment(salah)
	if len(errs["course_id"]) == 0 {
		t.Error("course_id kosong seharusnya menghasilkan pesan")
	}
	if len(errs["tahun_akademik"]) == 0 {
		t.Error("tahun_akademik salah format seharusnya menghasilkan pesan")
	}

	genap := model.CreateEnrollmentRequest{CourseID: 1, TahunAkademik: "2026/2027-Genap"}
	if errs := validateCreateEnrollment(genap); errs != nil {
		t.Errorf("format Genap seharusnya sah: %v", errs)
	}
}

func TestSisaSKS(t *testing.T) {
	if s := SisaSKS(24, 10); s != 14 {
		t.Errorf("SisaSKS(24,10) = %d, diharapkan 14", s)
	}
	if s := SisaSKS(18, 20); s != 0 {
		t.Errorf("SisaSKS(18,20) = %d, diharapkan 0 (tidak boleh negatif)", s)
	}
}
