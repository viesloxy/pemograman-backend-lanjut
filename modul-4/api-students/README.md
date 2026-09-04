# api-students

REST API data mahasiswa dengan Go, Fiber v2, dan PostgreSQL — disusun dengan struktur berlapis baku (Clean Architecture). Tugas Mandiri Modul 4 — Praktikum Pemrograman Backend Lanjut, D4 Teknik Informatika, Fakultas Vokasi, Universitas Airlangga.

Modul ini tidak menambah satu pun fitur: seluruh kode Modul 3 direstrukturisasi ke dalam lapisan, dan perilaku HTTP dipastikan tidak berubah (22 skenario `uji.sh` menghasilkan status identik). Struct `Student` beserta method `UpdateGrade`, `Activate`, `Deactivate` dari tugas Modul 1 tetap dipakai.

## Prasyarat

- Go 1.26+
- PostgreSQL 16 atau lebih baru
- `psql` (ikut terpasang bersama PostgreSQL)
- `curl` dan Git Bash (Windows) untuk pengujian

## Cara menjalankan dari nol

```bash
git clone <url-repo-ini>
cd modul-4/api-students

# 1. Buat basis data kosong
psql -U postgres -c "CREATE DATABASE praktikum_backend;"

# 2. Jalankan migrasi (membuat tabel students beserta indeksnya)
psql -U postgres -d praktikum_backend -f migrations/001_create_students.sql

# 3. Salin berkas contoh konfigurasi lalu isi kata sandi PostgreSQL Anda
cp .env.example .env
# edit .env: DB_PASSWORD=...

# 4. Ambil dependensi dan jalankan
go mod tidy
go run .
```

Server menyala di `http://localhost:3000` dan hanya akan menyala bila koneksi basis data berhasil. Cek:

```bash
curl -s -i http://localhost:3000/api/v1/health
```

Pengujian berurutan: jalankan server di satu terminal, lalu `bash uji.sh` di terminal lain. Unit test business rules berjalan tanpa server dan tanpa basis data: `go test ./app/service/ -v`.

## Variabel environment

| Variabel | Arti | Bawaan |
|---|---|---|
| APP_PORT | Port aplikasi | 3000 |
| APP_NAME | Nama aplikasi (header Server) | API Students |
| LOG_LEVEL | Level log: debug, info, warn, error | info |
| DB_HOST | Alamat PostgreSQL | localhost |
| DB_PORT | Port PostgreSQL | 5432 |
| DB_USER | Pengguna basis data | postgres |
| DB_PASSWORD | Kata sandi basis data | (wajib diisi) |
| DB_NAME | Nama basis data | praktikum_backend |
| DB_SSLMODE | Mode SSL koneksi | disable |
| DB_MAX_CONNS | Koneksi maksimum pada pool | 10 |

## Skema tabel

```sql
CREATE TABLE IF NOT EXISTS students (
  id SERIAL PRIMARY KEY,
  nim VARCHAR(15) NOT NULL,
  name VARCHAR(100) NOT NULL,
  grade NUMERIC(5,2) NOT NULL DEFAULT 0 CHECK (grade >= 0 AND grade <= 100),
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS students_nim_key ON students (nim);
CREATE INDEX IF NOT EXISTS students_name_lower_idx ON students (LOWER(name));
```

Keunikan NIM dijaga `UNIQUE INDEX` di basis data (kode 23505 → status 409). Indeks pada `LOWER(name)` melayani pencarian `?search=` yang memakai `ILIKE`.

## Struktur berkas dan pemetaan layer

```text
api-students/
├── app/
│   ├── model/                  Entities: entitas Student, struct request-respons
│   ├── repository/             Interface Adapters (gateway): SQL ke tabel students
│   └── service/
│       ├── student_rules.go    Use Cases: business rules murni (tanpa fiber)
│       ├── student_rules_test.go
│       └── student_service.go  Interface Adapters (controller): menerima fiber.Ctx
├── config/                     Frameworks: app.go, env.go, logger.go
├── database/                   Frameworks: koneksi PostgreSQL (pgxpool)
├── helper/                     Interface Adapters (presenter): respons + pembaca query
├── logs/                       output log (tidak di-commit)
├── middleware/                 Frameworks: requestid, recover, helmet, cors, logger, RequireJSON
├── route/                      Frameworks: pendaftaran alamat
├── .env                        konfigurasi lokal (tidak di-commit)
├── .env.example                contoh isian
└── main.go                     perakitan + graceful shutdown
```

Arah dependency satu arah ke dalam: `main.go → config/database → route/middleware → service → repository → model`. Tidak ada penyebutan fiber di `app/repository`, tidak ada SQL di `app/service`, dan `app/model` tidak mengimpor apa pun dari proyek.

## Bentuk respons

Seluruh endpoint memakai amplop yang sama, termasuk ketika gagal. Satu-satunya pengecualian adalah 204 pada DELETE, yang memang tidak boleh membawa body.

```json
{ "success": true,  "message": "...", "data": { } }
{ "success": true,  "message": "...", "data": [ ], "meta": { } }
{ "success": false, "message": "mahasiswa tidak ditemukan" }
{ "success": false, "message": "validasi gagal", "errors": { "nim": "wajib diisi" } }
```

## Kontrak API

Basis URL: `http://localhost:3000/api/v1`

| Metode | Endpoint | Parameter | Contoh body permintaan | Status yang mungkin |
|---|---|---|---|---|
| GET | /students | Query: `page`, `limit`, `search`, `sort`, `order`, `is_active`, `min_grade`, `max_grade` | tidak ada | 200, 500 |
| GET | /students/:id | Path: `id` (angka positif) | tidak ada | 200, 400, 404, 500 |
| POST | /students | Header: `Content-Type: application/json` | `{"nim":"434241084","name":"Vito Aditya","grade":88}` | 201, 400, 409, 415, 422, 500 |
| PUT | /students/:id | Path: `id`. Seluruh field wajib dikirim | `{"nim":"434241084","name":"Vito Aditya Revisi","grade":75,"is_active":false}` | 200, 400, 404, 409, 415, 422, 500 |
| PATCH | /students/:id | Path: `id`. Hanya field yang ingin diubah | `{"grade":91}` | 200, 400, 404, 409, 415, 422, 500 |
| DELETE | /students/:id | Path: `id` | tidak ada | 204, 400, 404, 500 |
| GET | /health | tidak ada | tidak ada | 200, 503 |

POST yang berhasil menyertakan header `Location: /api/v1/students/<id>`. DELETE yang berhasil menjawab 204 tanpa body.

## Parameter query pada endpoint daftar

| Parameter | Kegunaan | Bawaan | Catatan |
|---|---|---|---|
| page | Halaman keberapa | 1 | Diterjemahkan menjadi OFFSET |
| limit | Jumlah baris per halaman | 10 | Maksimal 50 |
| search | Kata kunci pada nama | kosong | `name ILIKE '%kata%'` |
| sort | Field pengurutan | id | Daftar putih: `id`, `nim`, `name`, `grade`, `created_at` |
| order | Arah pengurutan | asc | Selain `desc` dianggap `asc` |
| is_active | Menyaring status aktif | tidak menyaring | WHERE dengan parameter |
| min_grade | Batas bawah nilai | tidak menyaring | WHERE `grade >=` |
| max_grade | Batas atas nilai | tidak menyaring | WHERE `grade <=` |

Seluruh nilai dari klien dikirim sebagai parameter query (`$1`, `$2`, ...). Nama kolom ORDER BY tidak bisa memakai parameter, sehingga dipetakan lewat daftar putih.

## Log

Setiap request tercatat satu baris JSON oleh `middleware.RequestLogger` — memuat `request_id`, metode, jalur, status, durasi, dan IP — ke layar sekaligus ke `logs/app.log` (dirotasi otomatis oleh lumberjack: 10 MB per berkas, 5 cadangan, 14 hari). Folder `logs/` tidak ikut di-commit.

## Daftar status yang dipakai

| Status | Situasi |
|---|---|
| 200 | Pengambilan, penggantian, atau perubahan berhasil |
| 201 | Penambahan berhasil, disertai header `Location` |
| 204 | Penghapusan berhasil, tanpa body |
| 400 | JSON rusak, id bukan angka positif, atau PATCH tanpa satu pun field |
| 404 | Data atau endpoint tidak ditemukan (berasal dari `pgx.ErrNoRows`) |
| 409 | NIM sudah dipakai data lain (pelanggaran UNIQUE, kode 23505) |
| 415 | Content-Type bukan application/json |
| 422 | Bentuk permintaan benar tetapi isinya gagal validasi |
| 500 | Kesalahan tak terduga pada operasi basis data |
| 503 | `/health`: PostgreSQL tidak dapat dihubungi |

## Catatan

- Percobaan NIM duplikat membakar nomor id dari sequence: INSERT mengalokasikan nomor lebih dulu sebelum ditolak batasan UNIQUE, sehingga id bisa melompat. Untuk mengosongkan tabel sekaligus me-reset penomoran: `psql -U postgres -d praktikum_backend -c "TRUNCATE students RESTART IDENTITY;"`.
- Endpoint daftar mengembalikan `data` berupa array kosong bila tidak ada yang cocok, bukan `null`.
