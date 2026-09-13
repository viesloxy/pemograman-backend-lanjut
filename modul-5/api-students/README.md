# api-students

REST API data mahasiswa dengan Go, Fiber v2, dan PostgreSQL, dilengkapi autentikasi JWT: register, login, refresh token dengan rotasi, logout, dan proteksi seluruh endpoint mahasiswa. Tugas Mandiri Modul 5 — Praktikum Pemrograman Backend Lanjut, D4 Teknik Informatika, Fakultas Vokasi, Universitas Airlangga.

Modul ini melanjutkan Modul 4. Tidak ada perilaku CRUD yang berubah; yang baru: password disimpan sebagai hash bcrypt, endpoint `/auth/*` untuk pendaftaran akun dan login, serta middleware yang menuntut access token pada seluruh endpoint mahasiswa. Login memakai NIM dan password karena NIM sudah unik sejak Modul 3.

## Prasyarat

- Go 1.26+
- PostgreSQL 16 atau lebih baru
- `psql` dan `openssl` (ikut terpasang bersama PostgreSQL dan Git Bash)
- `curl` untuk pengujian

## Cara menjalankan dari nol

```bash
git clone <url-repo-ini>
cd modul-5/api-students

# 1. Buat basis data kosong
psql -U postgres -c "CREATE DATABASE praktikum_backend;"

# 2. Jalankan dua migrasi (struktur tabel + kolom dan tabel autentikasi)
psql -U postgres -d praktikum_backend -f migrations/001_create_students.sql
psql -U postgres -d praktikum_backend -f migrations/002_auth.sql

# 3. Salin contoh konfigurasi, isi kata sandi database dan JWT_SECRET
cp .env.example .env
# JWT_SECRET dibuat acak, minimal 32 karakter:
openssl rand -hex 32

# 4. Ambil dependensi dan jalankan
go mod tidy
go run .
```

Server menolak menyala bila `JWT_SECRET` kosong atau pendek dari 32 karakter. Cek kesehatan:

```bash
curl -s -i http://localhost:3000/api/v1/health
```

Unit test business rules berjalan tanpa server dan tanpa basis data: `go test ./app/service/ -v`.

## Variabel environment

| Variabel | Arti | Bawaan |
|---|---|---|
| APP_PORT | Port aplikasi | 3000 |
| APP_NAME | Nama aplikasi | API Students |
| LOG_LEVEL | Level log: debug, info, warn, error | info |
| DB_HOST / DB_PORT / DB_USER / DB_PASSWORD / DB_NAME / DB_SSLMODE | Koneksi PostgreSQL | localhost / 5432 / postgres / (wajib) / praktikum_backend / disable |
| DB_MAX_CONNS | Koneksi maksimum pool | 10 |
| JWT_SECRET | Kunci tanda tangan token, minimal 32 karakter | (wajib, tanpa bawaan) |
| JWT_ISSUER | Penerbit token | api-students |
| JWT_ACCESS_TTL_MINUTES | Umur access token (menit) | 15 |
| JWT_REFRESH_TTL_DAYS | Umur refresh token (hari) | 7 |
| ALLOWED_ORIGINS | Origin yang boleh memanggil API, dipisah koma | http://localhost:5173 |

## Skema tabel

```sql
ALTER TABLE students
  ADD COLUMN IF NOT EXISTS password VARCHAR(255) NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS role VARCHAR(20) NOT NULL DEFAULT 'user';

CREATE TABLE IF NOT EXISTS refresh_tokens (
    id BIGSERIAL PRIMARY KEY,
    student_id INTEGER NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

Password disimpan sebagai hash bcrypt cost 12 dengan salt bawaan bcrypt. Refresh token disimpan sebagai hash SHA-256; yang beredar di klien adalah nilai aslinya. Role disiapkan untuk pengaturan hak akses pada pertemuan berikutnya.

## Struktur berkas

```text
api-students/
├── app/
│   ├── model/                  entitas, request-respons CRUD, dan model autentikasi
│   ├── repository/             student_repository + token_repository
│   └── service/
│       ├── student_rules.go    aturan CRUD (murni, tanpa fiber)
│       ├── auth_rules.go       aturan register dan login (murni, tanpa fiber)
│       ├── auth_rules_test.go  unit test aturan
│       ├── student_service.go  CRUD mahasiswa (controller)
│       └── auth_service.go     register, login, refresh, logout, me
├── config/                     app.go, env.go, logger.go
├── database/                   koneksi PostgreSQL
├── helper/                     response.go, request.go, security.go, jwt.go, context.go
├── logs/                       output log (tidak di-commit)
├── middleware/                 middleware.go (global) dan auth.go (RequireAuth, limiter)
├── route/                      pendaftaran alamat beserta pembagian publik dan terlindungi
├── migrations/                 001_create_students.sql, 002_auth.sql
├── .env / .env.example
└── main.go                     perakitan, pemeriksaan secret, graceful shutdown
```

## Endpoint

Basis URL: `http://localhost:3000/api/v1`

| Metode | Endpoint | Auth | Keterangan |
|---|---|---|---|
| GET | /health | tidak perlu | memeriksa server dan database |
| POST | /auth/register | tidak perlu | daftar akun mahasiswa, 201 |
| POST | /auth/login | tidak perlu (rate limit 5 per menit per IP) | menghasilkan access token 15 menit dan refresh token 7 hari |
| POST | /auth/refresh | bawa refresh token | rotasi: token lama dicabut, pasangan baru diterbitkan |
| POST | /auth/logout | bawa refresh token | mencabut refresh token |
| GET | /auth/me | Bearer access token | profil mahasiswa yang sedang login |
| GET/POST | /students | Bearer access token | daftar (query lengkap) dan tambah |
| GET/PUT/PATCH/DELETE | /students/:id | Bearer access token | ambil satu, ganti, ubah sebagian, hapus |

Contoh alur:

```bash
B=http://localhost:3000/api/v1
J="Content-Type: application/json"

# Daftar akun
curl -s -i -X POST $B/auth/register -H "$J" \
  -d '{"nim":"434241084","name":"Vito Aditya","grade":88,"password":"rahasia123"}'

# Login, simpan access_token dan refresh_token dari respons
curl -s -X POST $B/auth/login -H "$J" \
  -d '{"nim":"434241084","password":"rahasia123"}'

# Akses endpoint terlindungi
curl -s $B/students -H "Authorization: Bearer <ACCESS_TOKEN>"
```

Parameter query pada endpoint daftar tetap sama seperti Modul 3 dan 4: `page`, `limit`, `search`, `sort`, `order`, `is_active`, `min_grade`, `max_grade`.

## Perlindungan keamanan yang diterapkan

| Perlindungan | Cara |
|---|---|
| Password | bcrypt cost 12 dengan salt bawaan; tidak pernah dikirim balik (`json:"-"`) |
| NIM ganda | UNIQUE INDEX di basis data, register menjawab 409 |
| Mass assignment | Role ditentukan server; struct request tidak memuat role |
| User enumeration | Pesan login identik, ditambah hash palsu agar waktu tanggap mirip |
| Algorithm confusion | keyfunc menolak algoritma selain HMAC |
| Secret lemah | Aplikasi menolak menyala bila JWT_SECRET kurang dari 32 karakter |
| Brute force | Rate limiter 5 percobaan login per menit per IP, jawab 429 + Retry-After |
| Token curian | Access token 15 menit; refresh token dapat dicabut dan dirotasi |
| Payload raksasa | BodyLimit 1 MB |
| CORS longgar | Daftar origin diatur lewat ALLOWED_ORIGINS |

## Daftar status

| Status | Situasi |
|---|---|
| 200 | Berhasil |
| 201 | Register dan tambah mahasiswa berhasil, disertai Location |
| 204 | Hapus berhasil, tanpa body |
| 400 | JSON rusak, id salah bentuk, PATCH kosong, refresh token kosong |
| 401 | Belum membawa token, token tidak valid atau kedaluwarsa, kredensial salah, refresh token tidak aktif |
| 403 | Akun dinonaktifkan |
| 409 | NIM sudah terdaftar |
| 415 | Content-Type bukan application/json |
| 422 | Isi permintaan gagal validasi |
| 429 | Melebihi lima percobaan login per menit |
| 500 | Kesalahan tak terduga |
| 503 | /health: database tidak dapat dihubungi |

## Catatan

- Percobaan register NIM duplikat membakar nomor id dari sequence (INSERT dialokasikan sebelum ditolak UNIQUE), sehingga id dapat melompat.
- `JWT_SECRET` yang berganti membuat seluruh token lama seketika tidak sah; seluruh pemakai harus login ulang. Itu memang perilaku yang diinginkan ketika secret dicurigai bocor.
- Endpoint yang belum tercakup: verifikasi email, lupa password, multi-factor, OAuth, dan penguncian akun.
