package model

// WebResponse adalah satu amplop untuk seluruh respons: sukses maupun gagal.
// Pada respons sukses terisi Success, Message, dan (bila ada) Data dan Meta.
// Pada respons kegagalan validasi terisi Errors per field.
type WebResponse struct {
	Success bool                `json:"success"`
	Message string              `json:"message"`
	Data    any                 `json:"data,omitempty"`
	Meta    *Meta               `json:"meta,omitempty"`
	Errors  map[string][]string `json:"errors,omitempty"`
}

// Meta adalah informasi pagination untuk endpoint daftar.
type Meta struct {
	CurrentPage int `json:"current_page"`
	PerPage     int `json:"per_page"`
	Total       int `json:"total"`
	LastPage    int `json:"last_page"`
}
