# api-students

REST API data mahasiswa dengan Go, Fiber v2, dan PostgreSQL. Tugas Mandiri Modul 3 — Praktikum Pemrograman Backend Lanjut, D4 Teknik Informatika, Fakultas Vokasi, Universitas Airlangga.

API ini melanjutkan tugas Modul 2: seluruh perilaku HTTP tidak berubah, tetapi penyimpanan dipindahkan dari memori ke PostgreSQL melalui pola repository. Struct `Student` beserta method `UpdateGrade`, `Activate`, dan `Deactivate` dari tugas Modul 1 tetap dipakai oleh handler. Data kini tersimpan permanen — bertahan walau server dimatikan.

## Prasyarat

- Go 1.26+
- PostgreSQL 16 atau lebih baru (dipakai 18.2 saat pengembangan)
- `psql` (ikut terpasang bersama PostgreSQL)
- `curl` dan Git Bash (Windows) untuk pengujian

## Cara menjalankan dari nol

```bash
git clone <url-repo-ini>
cd modul-3/api-students

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

Server menyala di `http://localhost:3000` — dan hanya akan menyala bila koneksi basis data berhasil (pool + ping diperiksa saat startup). Cek dengan:

```bash
curl -s -i http://localhost:3000/api/v1/health
```

Pengujian berurutan: jalankan server di satu terminal, lalu `bash uji.sh` di terminal lain. Karena kode tersebar di beberapa paket, perintahnya `go run .` dan bukan `go run main.go`.

## Variabel environment

Seluruh konfigurasi dibaca dari berkas `.env` (tidak pernah di-commit; lihat `.env.example`).

| Variabel | Arti | Bawaan |
|---|---|---|
| APP_PORT | Port aplikasi | 3000 |
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

Keunikan NIM dijaga `UNIQUE INDEX` di basis data (kode 23505 → status 409), bukan pemeriksaan manual di aplikasi, supaya dua permintaan bersamaan tidak bisa lolos bersama. Indeks pada `LOWER(name)` melayani pencarian `?search=` yang memakai `ILIKE`. Rentang nilai dijaga `CHECK` agar data tak sah tidak mungkin tersimpan walau lolos dari aplikasi.

## Struktur berkas

```text
api-students/
├── app/
│   ├── model/student.go                  struct entitas, request, dan respons
│   └── repository/student_repository.go  kontrak dan implementasi akses data
├── config/env.go                         memuat variabel environment
├── database/postgres.go                  koneksi dan connection pool
├── migrations/001_create_students.sql    skema tabel
├── .env                                  konfigurasi lokal (tidak di-commit)
├── .env.example                          contoh isian
├── main.go                               perakitan: pool → repository → handler
├── handler.go                            endpoint, memakai repository lewat interface
├── helper.go                             amplop respons, parser query, batas waktu
├── uji.sh                                skrip pengujian seluruh skenario
└── README.md
```

Arah ketergantungan satu arah: `handler → repository (interface) → model`. Tidak ada penyebutan fiber di paket repository.

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

| Metode | Endpoint | Parameter | Contoh body permintaan | Status yang mungkin | Contoh respons |
|---|---|---|---|---|---|
| GET | /students | Query: `page`, `limit`, `search`, `sort`, `order`, `is_active`, `min_grade`, `max_grade` | tidak ada | 200, 500 | `{"success":true,"message":"daftar mahasiswa berhasil diambil","data":[{"id":1,"nim":"434241084","name":"Vito Aditya","grade":88,"is_active":true,"created_at":"2026-08-30T18:42:02Z"}],"meta":{"page":1,"limit":10,"total":1,"total_pages":1}}` |
| GET | /students/:id | Path: `id` (angka positif) | tidak ada | 200, 400, 404, 500 | `{"success":true,"message":"mahasiswa ditemukan","data":{"id":1,"nim":"434241084","name":"Vito Aditya","grade":88,"is_active":true,"created_at":"2026-08-30T18:42:02Z"}}` |
| POST | /students | Header: `Content-Type: application/json` | `{"nim":"434241084","name":"Vito Aditya","grade":88}` | 201, 400, 409, 415, 422, 500 | `{"success":true,"message":"mahasiswa berhasil ditambahkan","data":{"id":1,"nim":"434241084","name":"Vito Aditya","grade":88,"is_active":true,"created_at":"2026-08-30T18:42:02Z"}}` disertai header `Location: /api/v1/students/1` |
| PUT | /students/:id | Path: `id`. Seluruh field wajib dikirim | `{"nim":"434241084","name":"Vito Aditya Revisi","grade":75,"is_active":false}` | 200, 400, 404, 409, 415, 422, 500 | `{"success":true,"message":"data mahasiswa berhasil diganti seluruhnya","data":{"id":1,...}}` |
| PATCH | /students/:id | Path: `id`. Hanya field yang ingin diubah | `{"grade":91}` | 200, 400, 404, 409, 415, 422, 500 | `{"success":true,"message":"data mahasiswa berhasil diperbarui sebagian","data":{"id":1,...}}` |
| DELETE | /students/:id | Path: `id` | tidak ada | 204, 400, 404, 500 | tanpa body |
| GET | /health | tidak ada | tidak ada | 200, 503 | `{"success":true,"message":"server dan database berjalan"}` |

## Parameter query pada endpoint daftar

| Parameter | Kegunaan | Bawaan | Catatan |
|---|---|---|---|
| page | Halaman keberapa | 1 | Diterjemahkan menjadi OFFSET |
| limit | Jumlah baris per halaman | 10 | Maksimal 50, diterjemahkan menjadi LIMIT |
| search | Kata kunci pada nama | kosong | Diterjemahkan menjadi `name ILIKE '%kata%'` |
| sort | Field pengurutan | id | Daftar putih: `id`, `nim`, `name`, `grade`, `created_at` |
| order | Arah pengurutan | asc | Selain `desc` dianggap `asc` |
| is_active | Menyaring status aktif | tidak menyaring | WHERE dengan parameter |
| min_grade | Batas bawah nilai | tidak menyaring | WHERE `grade >=` |
| max_grade | Batas atas nilai | tidak menyaring | WHERE `grade <=` |

Seluruh nilai dari klien dikirim sebagai parameter query (`$1`, `$2`, ...), tidak pernah disambung ke teks SQL. Nama kolom pada ORDER BY tidak bisa memakai parameter, sehingga dipetakan lewat daftar putih.

## Daftar status yang dipakai

| Status | Situasi |
|---|---|
| 200 | Pengambilan, penggantian, atau perubahan berhasil |
| 201 | Penambahan berhasil, disertai header `Location` |
| 204 | Penghapusan berhasil, tanpa body |
| 400 | JSON rusak, id bukan angka positif, atau PATCH tanpa satu pun field |
| 404 | Data atau endpoint tidak ditemukan (berasal dari `pgx.ErrNoRows`) |
| 409 | NIM sudah dipakai data lain (berasal dari pelanggaran UNIQUE, kode 23505) |
| 415 | Content-Type bukan application/json |
| 422 | Bentuk permintaan benar tetapi isinya gagal validasi |
| 500 | Kesalahan tak terduga pada operasi basis data |
| 503 | `/health`: PostgreSQL tidak dapat dihubungi |

## Catatan

- Endpoint daftar mengembalikan `data` berupa array kosong bila tidak ada yang cocok, bukan `null`.
- Percobaan NIM duplikat membakar nomor id dari sequence: INSERT mengalokasikan nomor lebih dulu sebelum ditolak batasan UNIQUE, sehingga id bisa melompat. Untuk mengosongkan tabel sekaligus me-reset penomoran: `psql -U postgres -d praktikum_backend -c "TRUNCATE students RESTART IDENTITY;"`.
