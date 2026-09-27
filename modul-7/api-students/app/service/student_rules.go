package service

import (
	"strings"

	"api-students/app/model"
)

func ApplyPatch(current model.Student, req model.PatchStudentRequest) model.Student {
	if req.NIM != nil {
		current.NIM = strings.TrimSpace(*req.NIM)
	}
	if req.Name != nil {
		current.Name = strings.TrimSpace(*req.Name)
	}
	if req.Grade != nil {
		current.UpdateGrade(*req.Grade)
	}
	if req.IsActive != nil {
		if *req.IsActive {
			current.Activate()
		} else {
			current.Deactivate()
		}
	}
	return current
}

func IsEmptyPatch(req model.PatchStudentRequest) bool {
	return req.NIM == nil && req.Name == nil && req.Grade == nil && req.IsActive == nil
}

func CountTotalPages(total, limit int) int {
	if limit <= 0 {
		return 0
	}
	return (total + limit - 1) / limit
}
