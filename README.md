# Praktikum Pemrograman Backend Lanjut

Repositori berisi seluruh pekerjaan praktikum mata kuliah **Pemrograman Backend Lanjut** — Universitas Airlangga, Semester 5.

## Tech Stack

- **Bahasa:** Go 1.26+
- **Framework:** [Fiber v2](https://docs.gofiber.io)
- **Database:** PostgreSQL, MongoDB
- **Tools:** Git, VS Code, Postman

## Struktur Repositori

```
.
├── modul-<N>/           ← kode praktikum + tugas mandiri per pertemuan
└── README.md
```

## Daftar Modul

### Modul 1 — Persiapan Lingkungan & Sintaks Go dengan Fiber

- **Topik:** setup Go/Fiber, variabel & struktur data, pointer, struct + method.
- **Kode:** [`modul-1/`](./modul-1/)

### Modul 2 — REST API & HTTP Deep Dive

- **Topik:** metode HTTP (safe/idempotent), PUT vs PATCH, status code, header, query string (filter, sort, search, paginasi).
- **Tugas:** [`modul-2/api-students/`](./modul-2/api-students/) — REST API CRUD mahasiswa, lanjutan struct `Student` dari Modul 1.

### Modul 3 — Database & Repository Pattern

- **Topik:** PostgreSQL dengan pgx (connection pool, migrasi, query berparameter), pola repository dengan interface, penerjemahan error basis data menjadi status HTTP.
- **Tugas:** [`modul-3/api-students/`](./modul-3/api-students/) — API mahasiswa Modul 2 dipindahkan dari memori ke PostgreSQL melalui repository pattern; perilaku HTTP tidak berubah.

<!-- Modul berikutnya ditambahkan di sini -->

## Cara Menjalankan

Tiap sub-proyek di dalam `modul-<N>/` adalah Go module independen:

```bash
cd modul-1/latihan-fiber
go mod tidy
go run main.go
```

Proyek yang terdiri dari beberapa berkas/paket (modul 2 dan 3) dijalankan dengan `go run .` dari folder proyeknya. Proyek modul 3 membutuhkan PostgreSQL berjalan dan berkas `.env` — lihat README di dalam foldernya.

## Author

**Vito Aditya** — Universitas Airlangga, Fakultas Vokasi, Program Studi D4 Teknik Informatika.
