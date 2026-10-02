package service

// SisaSKS menghitung sisa jatah SKS seorang mahasiswa.
func SisaSKS(batas, terpakai int) int {
	if terpakai >= batas {
		return 0
	}
	return batas - terpakai
}
