package service

import (
	"testing"
	"time"

	"siakad-mini/app/model"
	"siakad-mini/helper"
)

func ipk(v float64) *float64 { return &v }

func TestBatasSKS(t *testing.T) {
	kasus := []struct {
		nama       string
		ipk        *float64
		diharapkan int
	}{
		{"IPK 4,00 bracket tertinggi", ipk(4.00), 24},
		{"IPK tepat 3,00 tetap 24", ipk(3.00), 24},
		{"IPK 2,99 bracket tengah", ipk(2.99), 21},
		{"IPK tepat 2,50 tetap 21", ipk(2.50), 21},
		{"IPK 2,49 bracket terendah", ipk(2.49), 18},
		{"IPK 0,00 bracket terendah", ipk(0.00), 18},
		{"IPK kosong dianggap terendah", nil, 18},
	}

	for _, k := range kasus {
		if hasil := BatasSKS(k.ipk); hasil != k.diharapkan {
			t.Errorf("%s: BatasSKS = %d, diharapkan %d", k.nama, hasil, k.diharapkan)
		}
	}
}

func TestValidateCreateStudent(t *testing.T) {
	// Seluruh field sah: tidak ada kesalahan yang dikembalikan.
	sah := model.CreateStudentRequest{
		NIM: "532147208001", Nama: "Budi Simanjuntak",
		Email: "budi@siakad.test", Prodi: "Sistem Informasi",
		Angkatan: 2025, IPKTerakhir: ipk(3.50),
	}
	if errs := helper.ValidateStruct(sah); errs != nil {
		t.Errorf("request sah justru ditolak: %v", errs)
	}

	// Seluruh field salah: setiap field menghasilkan pesannya.
	salah := model.CreateStudentRequest{
		NIM: "123", Nama: "", Email: "bukan-email",
		Prodi: "", Angkatan: time.Now().Year() + 1, IPKTerakhir: ipk(4.5),
	}
	errs := helper.ValidateStruct(salah)
	for _, field := range []string{"nim", "nama", "email", "prodi", "angkatan", "ipk_terakhir"} {
		if len(errs[field]) == 0 {
			t.Errorf("field %s seharusnya menghasilkan pesan kesalahan", field)
		}
	}

	// IPK kosong sah: field itu opsional.
	tanpaIPK := sah
	tanpaIPK.IPKTerakhir = nil
	if errs := helper.ValidateStruct(tanpaIPK); errs != nil {
		t.Errorf("tanpa IPK seharusnya sah: %v", errs)
	}
}

func TestBolehLihatStudent(t *testing.T) {
	admin := model.AuthUser{ID: 1, Role: "admin"}
	mahasiswa := model.AuthUser{ID: 7, Role: "mahasiswa", StudentID: 5}

	if !BolehLihatStudent(admin, 999) {
		t.Error("admin seharusnya boleh melihat data siapa pun")
	}
	if !BolehLihatStudent(mahasiswa, 7) {
		t.Error("mahasiswa seharusnya boleh melihat datanya sendiri")
	}
	if BolehLihatStudent(mahasiswa, 8) {
		t.Error("mahasiswa seharusnya ditolak melihat data orang lain")
	}
}
