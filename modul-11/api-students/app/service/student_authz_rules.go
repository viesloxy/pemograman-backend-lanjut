package service

import (
	"api-students/app/model"
	"api-students/helper"
)

func CanAccessStudent(
	current model.AuthUser,
	ownerID int,
	perms *helper.PermissionSet,
	anyPermission string,
) bool {
	if current.StudentID == ownerID {
		return true
	}
	return perms.Can(current.Role, anyPermission)
}

func ownerValue(owner *int) int {
	if owner == nil {
		return 0
	}
	return *owner
}
