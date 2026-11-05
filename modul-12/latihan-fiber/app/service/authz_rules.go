package service

import (
	"strings"

	"latihan-fiber/app/model"
	"latihan-fiber/helper"
)

// CanAccessUser memutuskan apakah seseorang boleh menyentuh data user lain.
//
// Dua jalur yang diizinkan:
// 1. Kepemilikan (ownership) --- data itu miliknya sendiri.
// 2. Permission --- role-nya memang berhak atas data siapa pun.
func CanAccessUser(
	current model.AuthUser,
	targetID int,
	perms *helper.PermissionSet,
	anyPermission string,
) bool {
	if current.UserID == targetID {
		return true
	}
	return perms.Can(current.Role, anyPermission)
}

// ValidateAssignRole memeriksa permintaan pergantian role.
func ValidateAssignRole(
	current model.AuthUser,
	targetID int,
	req model.AssignRoleRequest,
	perms *helper.PermissionSet,
) map[string]string {
	errs := map[string]string{}
	role := strings.TrimSpace(req.Role)

	if role == "" {
		errs["role"] = "wajib diisi"
		return errs
	}
	if !perms.IsKnownRole(role) {
		errs["role"] = "role tidak dikenal, pilih salah satu dari: " +
			strings.Join(perms.KnownRoles(), ", ")
	}
	if current.UserID == targetID {
		errs["role"] = "tidak boleh mengubah role diri sendiri"
	}

	return errs
}