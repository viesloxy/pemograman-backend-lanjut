package service

import (
	"siakad-mini/app/model"
)

// BatasSKS menerjemahkan IPK menjadi batas pengambilan SKS per
// semester: 3,00 ke atas 24, 2,50 sampai 2,99 sebanyak 21, sisanya 18.
// IPK kosong dianggap bracket terendah.
func BatasSKS(ipk *float64) int {
	if ipk == nil {
		return 18
	}
	switch {
	case *ipk >= 3.00:
		return 24
	case *ipk >= 2.50:
		return 21
	default:
		return 18
	}
}

// BolehLihatStudent memutuskan siapa boleh melihat detail seorang
// mahasiswa: admin boleh melihat semua, mahasiswa hanya dirinya.
func BolehLihatStudent(current model.AuthUser, studentUserID int) bool {
	if current.Role == "admin" {
		return true
	}
	return current.ID == studentUserID
}
