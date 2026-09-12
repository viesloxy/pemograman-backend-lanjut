package service

import (
	"strings"

	"api-students/app/model"
)

func ValidateCreate(req model.CreateStudentRequest) map[string]string {
	errs := map[string]string{}

	if req.NIM == "" {
		errs["nim"] = "wajib diisi"
	} else if !nimValid(req.NIM) {
		errs["nim"] = "harus berupa angka sepanjang 9 sampai 15 digit"
	}

	if req.Name == "" {
		errs["name"] = "wajib diisi"
	} else if len(req.Name) < 3 {
		errs["name"] = "minimal 3 karakter"
	} else if len(req.Name) > 100 {
		errs["name"] = "maksimal 100 karakter"
	}

	if req.Grade == nil {
		errs["grade"] = "wajib diisi"
	} else if *req.Grade < 0 || *req.Grade > 100 {
		errs["grade"] = "harus berada di rentang 0 sampai 100"
	}

	return errs
}

func ValidateReplace(req model.ReplaceStudentRequest) map[string]string {
	errs := ValidateCreate(model.CreateStudentRequest{
		NIM:   req.NIM,
		Name:  req.Name,
		Grade: req.Grade,
	})
	if req.IsActive == nil {
		errs["is_active"] = "wajib dikirim pada PUT"
	}
	return errs
}

func ApplyPatch(
	current model.Student, req model.PatchStudentRequest,
) (model.Student, map[string]string) {
	errs := map[string]string{}

	if req.NIM != nil {
		nim := strings.TrimSpace(*req.NIM)
		if nim == "" {
			errs["nim"] = "tidak boleh dikosongkan"
		} else if !nimValid(nim) {
			errs["nim"] = "harus berupa angka sepanjang 9 sampai 15 digit"
		} else {
			current.NIM = nim
		}
	}
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			errs["name"] = "tidak boleh dikosongkan"
		} else if len(name) < 3 {
			errs["name"] = "minimal 3 karakter"
		} else if len(name) > 100 {
			errs["name"] = "maksimal 100 karakter"
		} else {
			current.Name = name
		}
	}
	if req.Grade != nil {
		if *req.Grade < 0 || *req.Grade > 100 {
			errs["grade"] = "harus berada di rentang 0 sampai 100"
		} else {
			current.UpdateGrade(*req.Grade)
		}
	}
	if req.IsActive != nil {
		if *req.IsActive {
			current.Activate()
		} else {
			current.Deactivate()
		}
	}

	return current, errs
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

func nimValid(nim string) bool {
	if len(nim) < 9 || len(nim) > 15 {
		return false
	}
	for _, r := range nim {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
