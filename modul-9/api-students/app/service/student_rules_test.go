package service

import (
	"testing"

	"api-students/app/model"
	"api-students/helper"
)

func TestApplyPatch(t *testing.T) {
	awal := model.Student{ID: 1, NIM: "434241084", Name: "Vito Aditya", Grade: 88, IsActive: true}

	grade := 91.0
	hasil := ApplyPatch(awal, model.PatchStudentRequest{Grade: &grade})
	if hasil.Grade != 91 {
		t.Errorf("grade seharusnya 91, dapat %v", hasil.Grade)
	}
	if hasil.Name != "Vito Aditya" || hasil.NIM != "434241084" || !hasil.IsActive {
		t.Error("field yang tidak dikirim seharusnya tidak berubah")
	}

	tidakAktif := false
	hasil = ApplyPatch(awal, model.PatchStudentRequest{IsActive: &tidakAktif})
	if hasil.IsActive {
		t.Error("is_active seharusnya false")
	}
	if hasil.Grade != 88 {
		t.Errorf("grade seharusnya tetap 88, dapat %v", hasil.Grade)
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

func TestValidateCreateDeclarative(t *testing.T) {
	grade := 88.0
	valid := model.CreateStudentRequest{NIM: "434241084", Name: "Vito Aditya", Grade: &grade}
	if errs := helper.ValidateStruct(valid); errs != nil {
		t.Errorf("harusnya valid, dapat %v", errs)
	}

	kosong := model.CreateStudentRequest{}
	errs := helper.ValidateStruct(kosong)
	if errs["nim"] == "" || errs["name"] == "" || errs["grade"] == "" {
		t.Errorf("nim, name, dan grade wajib ditandai: %v", errs)
	}

	gradeSalah := 150.0
	salah := model.CreateStudentRequest{NIM: "abc", Name: "Ka", Grade: &gradeSalah}
	errs = helper.ValidateStruct(salah)
	if len(errs) < 3 {
		t.Errorf("harusnya minimal 3 error, dapat %d: %v", len(errs), errs)
	}
}
