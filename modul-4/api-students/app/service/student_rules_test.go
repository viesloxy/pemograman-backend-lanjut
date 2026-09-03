package service

import (
	"testing"

	"api-students/app/model"
)

func TestValidateCreate(t *testing.T) {
	grade := 88.0

	valid := model.CreateStudentRequest{NIM: "434241084", Name: "Vito Aditya", Grade: &grade}
	if errs := ValidateCreate(valid); len(errs) != 0 {
		t.Errorf("harusnya valid, dapat %v", errs)
	}

	kosong := model.CreateStudentRequest{}
	errs := ValidateCreate(kosong)
	if errs["nim"] != "wajib diisi" || errs["name"] != "wajib diisi" || errs["grade"] != "wajib diisi" {
		t.Errorf("pesan wajib diisi tidak lengkap: %v", errs)
	}

	gradeSalah := 150.0
	salah := model.CreateStudentRequest{NIM: "abc", Name: "Ka", Grade: &gradeSalah}
	errs = ValidateCreate(salah)
	if len(errs) != 3 {
		t.Errorf("harusnya 3 error, dapat %d: %v", len(errs), errs)
	}
}

func TestValidateReplace(t *testing.T) {
	grade := 75.0
	aktif := true

	lengkap := model.ReplaceStudentRequest{NIM: "434241084", Name: "Vito Aditya", Grade: &grade, IsActive: &aktif}
	if errs := ValidateReplace(lengkap); len(errs) != 0 {
		t.Errorf("harusnya valid, dapat %v", errs)
	}

	kurang := model.ReplaceStudentRequest{NIM: "434241084", Name: "Vito Aditya", Grade: &grade}
	errs := ValidateReplace(kurang)
	if errs["is_active"] != "wajib dikirim pada PUT" {
		t.Errorf("harusnya is_active ditandai: %v", errs)
	}
}

func TestApplyPatch(t *testing.T) {
	awal := model.Student{ID: 1, NIM: "434241084", Name: "Vito Aditya", Grade: 88, IsActive: true}

	grade := 91.0
	hasil, errs := ApplyPatch(awal, model.PatchStudentRequest{Grade: &grade})
	if len(errs) != 0 {
		t.Fatalf("tidak seharusnya ada error: %v", errs)
	}
	if hasil.Grade != 91 {
		t.Errorf("grade seharusnya 91, dapat %v", hasil.Grade)
	}
	if hasil.Name != "Vito Aditya" || hasil.NIM != "434241084" || !hasil.IsActive {
		t.Error("field yang tidak dikirim seharusnya tidak berubah")
	}

	nimKosong := ""
	_, errs = ApplyPatch(awal, model.PatchStudentRequest{NIM: &nimKosong})
	if errs["nim"] != "tidak boleh dikosongkan" {
		t.Errorf("harusnya nim ditolak: %v", errs)
	}

	nimFormat := "abc"
	_, errs = ApplyPatch(awal, model.PatchStudentRequest{NIM: &nimFormat})
	if errs["nim"] == "" {
		t.Error("harusnya format nim ditolak")
	}

	tidakAktif := false
	hasil, errs = ApplyPatch(awal, model.PatchStudentRequest{IsActive: &tidakAktif})
	if len(errs) != 0 || hasil.IsActive {
		t.Errorf("is_active seharusnya false, dapat %v %v", hasil.IsActive, errs)
	}
}

func TestIsEmptyPatch(t *testing.T) {
	if !IsEmptyPatch(model.PatchStudentRequest{}) {
		t.Error("patch kosong seharusnya true")
	}

	grade := 90.0
	if IsEmptyPatch(model.PatchStudentRequest{Grade: &grade}) {
		t.Error("patch dengan grade seharusnya false")
	}
}

func TestCountTotalPages(t *testing.T) {
	cases := []struct{ total, limit, want int }{
		{0, 10, 0},
		{1, 10, 1},
		{10, 10, 1},
		{11, 10, 2},
		{137, 20, 7},
	}

	for _, tc := range cases {
		if got := CountTotalPages(tc.total, tc.limit); got != tc.want {
			t.Errorf("total=%d limit=%d: harap %d, dapat %d", tc.total, tc.limit, tc.want, got)
		}
	}
}
