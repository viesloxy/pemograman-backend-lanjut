package service

import (
	"errors"
	"fmt"
	"testing"

	"latihan-fiber/app/repository"
	"latihan-fiber/helper"
)

// Test ini memastikan setiap error dari repository berubah menjadi response
// yang tepat — termasuk error yang sudah dibungkus dengan fmt.Errorf("...: %w", err).
//
// translateReviewError adalah fungsi yang TIDAK di-export. Karena itu file ini
// memakai package service (white-box): ia boleh memanggil fungsi yang tidak
// di-export, tidak seperti test black-box pada file lain di folder ini.
func TestTranslateReviewError(t *testing.T) {
	tests := []struct {
		err        error
		wantStatus int
		wantCode   string
	}{
		{repository.ErrInvalidID, 400, helper.CodeBadRequest},
		{repository.ErrNotFound, 404, helper.CodeNotFound},
		{repository.ErrDuplicate, 409, "ALREADY_REVIEWED"},
		{repository.ErrNotOwner, 403, helper.CodeForbidden},
		{repository.ErrOwnReview, 409, "OWN_REVIEW"},
		{repository.ErrAlreadyVoted, 409, "ALREADY_VOTED"},
		{repository.ErrReplyLimit, 409, "REPLY_LIMIT"},
		{fmt.Errorf("lapisan luar: %w", repository.ErrDuplicate), 409, "ALREADY_REVIEWED"},
		{errors.New("koneksi MongoDB putus"), 500, helper.CodeInternal},
	}

	for _, tt := range tests {
		t.Run(tt.err.Error(), func(t *testing.T) {
			var appErr *helper.AppError
			if !errors.As(translateReviewError(tt.err), &appErr) {
				t.Fatalf("hasil bukan *helper.AppError")
			}
			if appErr.Status != tt.wantStatus || appErr.Code != tt.wantCode {
				t.Errorf("dapat %d %s, seharusnya %d %s", appErr.Status, appErr.Code, tt.wantStatus, tt.wantCode)
			}
		})
	}
}