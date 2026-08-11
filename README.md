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

<!-- Modul berikutnya ditambahkan di sini -->

## Cara Menjalankan

Tiap sub-proyek di dalam `modul-<N>/` adalah Go module independen:

```bash
cd modul-1/latihan-fiber
go mod tidy
go run main.go
```

## Author

**Vito Aditya** — Universitas Airlangga, Fakultas Vokasi, Program Studi D4 Teknik Informatika.
