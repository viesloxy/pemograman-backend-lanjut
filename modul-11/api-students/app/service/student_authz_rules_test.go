package service

import (
	"testing"

	"api-students/app/model"
	"api-students/helper"
)

func TestCanAccessStudent(t *testing.T) {
	perms := helper.NewPermissionSet(map[string][]string{
		"admin": {"student:read:any", "student:update:any", "student:delete"},
		"staff": {"student:read:any"},
		"user":  {},
	})

	pemilik := model.AuthUser{StudentID: 3, NIM: "434241086", Role: "user"}
	orangLain := model.AuthUser{StudentID: 4, NIM: "434241087", Role: "user"}
	admin := model.AuthUser{StudentID: 1, Role: "admin"}
	staff := model.AuthUser{StudentID: 2, Role: "staff"}

	if !CanAccessStudent(pemilik, 3, perms, "student:read:any") {
		t.Error("pemilik seharusnya boleh membaca datanya sendiri")
	}
	if !CanAccessStudent(admin, 3, perms, "student:read:any") {
		t.Error("admin seharusnya boleh membaca data siapa pun")
	}
	if !CanAccessStudent(staff, 3, perms, "student:read:any") {
		t.Error("staff seharusnya boleh membaca data siapa pun")
	}
	if CanAccessStudent(orangLain, 3, perms, "student:read:any") {
		t.Error("user tanpa kepemilikan seharusnya ditolak membaca")
	}
	if CanAccessStudent(orangLain, 3, perms, "student:update:any") {
		t.Error("user tanpa kepemilikan seharusnya ditolak mengubah")
	}
	if !CanAccessStudent(admin, 3, perms, "student:update:any") {
		t.Error("admin seharusnya boleh mengubah data siapa pun")
	}

	var permsNil *helper.PermissionSet
	if CanAccessStudent(admin, 3, permsNil, "student:read:any") {
		t.Error("permission set nil harus ditolak (fail closed)")
	}
}
