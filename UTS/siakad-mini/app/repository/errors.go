package repository

import "errors"

// Sentinel error milik repository. Handler tidak pernah melihat error
// dari driver; yang diterima handler hanyalah error-error ini, lalu
// diterjemahkan menjadi status HTTP.
var (
	ErrNotFound  = errors.New("data tidak ditemukan")
	ErrDuplicate = errors.New("data sudah ada")
)
