package model

type Course struct {
	ID       int    `json:"id"`
	KodeMK   string `json:"kode_mk"`
	NamaMK   string `json:"nama_mk"`
	SKS      int    `json:"sks"`
	Semester int    `json:"semester"`
	Kuota    int    `json:"kuota"`
}

// CourseWithKuota adalah mata kuliah beserta jumlah terisi dan sisa
// kuotanya. Keduanya dihitung dari tabel enrollments.
type CourseWithKuota struct {
	Course
	Terisi    int `json:"terisi"`
	SisaKuota int `json:"sisa_kuota"`
}

type ListCourseQuery struct {
	Semester  int // 0 berarti tidak difilter
	Search    string
	Available bool // true = hanya yang kuotanya belum penuh
}
