# Praktikum Pemrograman Backend Lanjut

Repositori berisi seluruh pekerjaan praktikum mata kuliah **Pemrograman Backend Lanjut** — Universitas Airlangga, Semester 5.

## Tech Stack

- **Bahasa:** Go 1.26+
- **Framework:** [Fiber v2](https://docs.gofiber.io)
- **Database:** PostgreSQL, MongoDB, Redis (queue)
- **Tools:** Git, VS Code, Postman

## Struktur Repositori

```
.
├── modul-<N>/           ← kode praktikum + tugas mandiri per pertemuan
├── UTS/siakad-mini/     ← proyek Ujian Tengah Semester: RESTful API SIAKAD Mini
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

### Modul 4 — Clean Architecture

- **Topik:** empat layer Clean Architecture dan Dependency Rule, pemisahan business rules murni + unit test, logger terstruktur (`log/slog` + rotasi berkas), middleware & route terpisah, graceful shutdown.
- **Tugas:** [`modul-4/api-students/`](./modul-4/api-students/) — API mahasiswa Modul 3 direstrukturisasi ke struktur berlapis baku; perilaku HTTP tidak berubah.

### Modul 5 — Authentication & Security

- **Topik:** register dan login dengan bcrypt, JWT (access + refresh token dengan rotasi), middleware RequireAuth, rate limiter, user enumeration prevention.
- **Tugas:** [`modul-5/api-students/`](./modul-5/api-students/) — API mahasiswa Modul 4 dilengkapi autentikasi JWT; seluruh endpoint students terkunci tanpa token.

### Modul 6 — Authorization & RBAC

- **Topik:** Role Based Access Control (tabel roles, permissions, role_permissions), middleware RequirePermission, pemeriksaan kepemilikan di service, fail closed, 401 vs 403.
- **Tugas:** [`modul-6/api-students/`](./modul-6/api-students/) — API mahasiswa Modul 5 ditambah lapisan otorisasi RBAC; admin akses penuh, staff baca dan tambah, user hanya miliknya sendiri.

### Modul 7 — Advanced API Design

- **Topik:** validasi deklaratif dengan go-playground/validator, cursor pagination (keyset), content negotiation (JSON + CSV), error handling terpusat dengan AppError dan kode error stabil, log identitas.
- **Tugas:** [`modul-7/api-students/`](./modul-7/api-students/) — API mahasiswa Modul 6 dimodernisasi: validasi pindah ke tag, pagination berganti cursor, respons error seragam dengan kode dan request_id.

### UTS — RESTful API SIAKAD Mini

- **Studi Kasus:** implementasi materi pertemuan 1 sampai 7 dalam satu API utuh untuk layanan akademik sederhana: pengelolaan data mahasiswa, mata kuliah, dan Kartu Rencana Studi (KRS) dengan dua peran, admin dan mahasiswa, melalui 10 endpoint.
- **Topik yang diimplementasikan:** clean architecture (struktur baku pertemuan 4), PostgreSQL dengan repository pattern, autentikasi bcrypt + JWT dengan rate limiter login, kontrol akses peran dengan pemeriksaan kepemilikan data, transaksi PostgreSQL dengan row locking untuk aturan kuota dan batas SKS (IPK menentukan 18/21/24 SKS), soft delete, validasi deklaratif dengan go-playground/validator, dan error handling terpusat dengan format respons seragam.
- **Kode:** [`UTS/siakad-mini/`](./UTS/siakad-mini/) — lengkap dengan migration, seeder (1 admin, 20 mahasiswa, 11 mata kuliah), koleksi Postman di folder [`postman/`](./UTS/siakad-mini/postman/), dan README dokumentasi API.

### Modul 9 — Transactions & Concurrency

- **Topik:** transaksi database dengan TxManager, row locking (`FOR UPDATE`, urutan kunci anti-deadlock), UPDATE bersyarat atomik untuk kuota, optimistic locking (version), uji beban konkurensi.
- **Tugas:** [`modul-9/api-students/`](./modul-9/api-students/) — penetapan dan pergantian dosen pembimbing menjadi transaksi: satu mahasiswa satu pembimbing dijaga constraint, kuota tidak mungkin terlampaui walau diserang puluhan request serentak.

### Modul 10 — API dengan MongoDB

- **Topik:** document model, BSON/ObjectID, embedding vs referencing, update atomik lewat filter (`FindOneAndUpdate`), index dan aggregation pipeline (`$facet`, `$unwind`), pencegahan NoSQL injection, polyglot persistence.
- **Tugas:** [`modul-10/api-students/`](./modul-10/api-students/) — ulasan kelas dan portofolio mahasiswa tersimpan di MongoDB berdampingan dengan PostgreSQL; ringkasan ulasan lewat aggregation, portofolio dengan upsert atomik.

### Modul 11 — Background Jobs & Asynchronous Processing

- **Topik:** transactional outbox, relay dengan `FOR UPDATE SKIP LOCKED`, job queue Asynq (producer-broker-worker), at-least-once dan handler idempoten, retry/backoff/SkipRetry, satu kode dua proses dengan graceful shutdown.
- **Tugas:** [`modul-11/api-students/`](./modul-11/api-students/) — setiap perubahan KRS dikirim email konfirmasi lewat worker terpisah: API hanya menulis outbox (Redis boleh mati, notifikasi tertunda tidak hilang), worker mengirim email tiruan secara idempoten dengan klaim `(outbox_id, user_id)`, plus notifikasi perubahan kuota untuk seluruh peserta kelas.

### Modul 12 — Testing & Quality, API Documentation, dan Docker Deployment

- **Topik:** piramida test (unit, integration, end-to-end), table-driven test, test double (fake dengan embedding interface), component test dengan `app.Test`, integration test pada database terpisah dengan `TestMain`, pengukuran kualitas test (coverage, `-shuffle`, mutation testing), dokumentasi OpenAPI/Swagger dengan swaggo, multi-stage Docker build, dan docker compose.
- **Tugas:** [`modul-12/latihan-fiber/`](./modul-12/latihan-fiber/) — proyek KRS + ulasan + notifikasi Modul 11 disusun ulang sebagai `latihan-fiber` dan dilengkapi: test suite (unit test aturan murni, fake repository, component test endpoint, white-box test pemetaan error, integration test repository dan endpoint KRS termasuk swap kelas penuh dan keluar kelas), dokumentasi Swagger untuk 20 endpoint dengan gerbang `SWAGGER_ENABLED`, serta deployment lima service (api, worker, PostgreSQL, MongoDB, Redis) dengan satu perintah `docker compose up -d --build`.

<!-- Modul berikutnya ditambahkan di sini -->

## Cara Menjalankan

Tiap sub-proyek di dalam `modul-<N>/` adalah Go module independen:

```bash
cd modul-1/latihan-fiber
go mod tidy
go run main.go
```

Proyek yang terdiri dari beberapa berkas/paket (modul 2 sampai 7) dijalankan dengan `go run .` dari folder proyeknya. Proyek modul 3 sampai 7 membutuhkan PostgreSQL berjalan dan berkas `.env`. Proyek modul 5 sampai 7 juga membutuhkan `JWT_SECRET` yang valid. Unit test: `go test ./app/service/ ./helper/ -v` dari folder proyeknya.

Proyek UTS (`UTS/siakad-mini/`) dijalankan dengan `go run .` dari folder proyeknya: membutuhkan database `siakad_mini`, migration, seeder, `.env` dengan `JWT_SECRET` minimal 32 karakter, dan unit testnya berjalan dengan `go test ./...`. Rincian lengkap ada pada [`UTS/siakad-mini/README.md`](./UTS/siakad-mini/README.md).

Proyek modul 12 (`modul-12/latihan-fiber/`) membutuhkan PostgreSQL, MongoDB, dan Redis berjalan serta `.env` (lihat `.env.example`). Unit test: `go test ./...`. Integration test: `go test -tags=integration -count=1 ./tests/integration/` — membutuhkan database `praktikum_backend_test` dan berkas `.env.test`. Seluruh sistem juga dapat dijalankan sekaligus dengan `docker compose up -d --build`; Swagger UI tersedia di `http://localhost:8080/swagger/index.html` saat `SWAGGER_ENABLED=true`.

## Author

**Vito Aditya** — Universitas Airlangga, Fakultas Vokasi, Program Studi D4 Teknik Informatika.
