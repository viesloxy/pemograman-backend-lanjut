package service

import (
	"testing"

	"api-students/app/model"
	"api-students/helper"
)

func TestValidateRegisterDeclarative(t *testing.T) {
	grade := 88.0
	valid := model.RegisterRequest{
		NIM: "434241084", Name: "Vito Aditya", Grade: &grade, Password: "rahasia123",
	}
	if errs := helper.ValidateStruct(valid); errs != nil {
		t.Errorf("harusnya valid, dapat %v", errs)
	}

	kosong := model.RegisterRequest{}
	errs := helper.ValidateStruct(kosong)
	if errs["nim"] == "" || errs["name"] == "" || errs["password"] == "" {
		t.Errorf("nim, name, dan password wajib ditandai: %v", errs)
	}

	lemah := model.RegisterRequest{
		NIM: "4342410ab", Name: "Budi", Password: "password1",
	}
	errs = helper.ValidateStruct(lemah)
	if errs["nim"] == "" {
		t.Error("nim format salah seharusnya ditandai")
	}
	if errs["password"] != "password terlalu umum" {
		t.Errorf("harusnya password terlalu umum: %v", errs["password"])
	}
}

func TestValidateLoginDeclarative(t *testing.T) {
	kosong := model.LoginRequest{}
	errs := helper.ValidateStruct(kosong)
	if errs["nim"] != "wajib diisi" || errs["password"] != "wajib diisi" {
		t.Errorf("kedua field wajib ditandai: %v", errs)
	}

	isi := model.LoginRequest{NIM: "434241084", Password: "apapun123"}
	if errs := helper.ValidateStruct(isi); errs != nil {
		t.Errorf("login lengkap seharusnya lolos validasi bentuk: %v", errs)
	}
}

func TestApplyPatchSimplified(t *testing.T) {
	awal := model.Student{ID: 1, NIM: "434241084", Name: "Vito Aditya", Grade: 88, IsActive: true}

	grade := 91.0
	hasil := ApplyPatch(awal, model.PatchStudentRequest{Grade: &grade})
	if hasil.Grade != 91 {
		t.Errorf("grade seharusnya 91, dapat %v", hasil.Grade)
	}
	if hasil.Name != "Vito Aditya" || !hasil.IsActive {
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
