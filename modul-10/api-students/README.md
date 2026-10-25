# api-students

REST API data mahasiswa dengan Go, Fiber v2, dan PostgreSQL, dilengkapi autentikasi JWT, otorisasi RBAC, validasi deklaratif, cursor pagination, dan content negotiation. Tugas Mandiri Modul 7 — Praktikum Pemrograman Backend Lanjut, D4 Teknik Informatika, Fakultas Vokasi, Universitas Airlangga.

Modul ini memperbaiki empat kelemahan dari Modul 6: validasi manual diganti tag deklaratif (go-playground/validator), response kegagalan dipusatkan pada satu ErrorHandler dengan kode error yang stabil, pagination offset diganti keyset pagination dengan cursor, dan API mampu melayani JSON dan CSV melalui content negotiation.

## Prasyarat

- Go 1.26+
- PostgreSQL 16+
- `psql`, `openssl`
- Postman untuk pengujian

## Cara menjalankan dari nol

```bash
git clone <url-repo-ini>
cd modul-7/api-students

# 1. Buat basis data kosong
psql -U postgres -c "CREATE DATABASE praktikum_backend;"

# 2. Jalankan lima migrasi berurutan
psql -U postgres -d praktikum_backend -f migrations/001_create_students.sql
psql -U postgres -d praktikum_backend -f migrations/002_auth.sql
psql -U postgres -d praktikum_backend -f migrations/003_rbac.sql
psql -U postgres -d praktikum_backend -f migrations/004_student_permissions.sql
psql -U postgres -d praktikum_backend -f migrations/005_cursor_index.sql

# 3. Salin contoh konfigurasi, isi DB_PASSWORD dan JWT_SECRET
cp .env.example .env
# JWT_SECRET dibuat acak minimal 32 karakter:
openssl rand -hex 32

# 4. Ambil dependensi dan jalankan
go mod tidy
go run .
```

Unit test: `go test ./app/service/ ./helper/ -v`

## Variabel environment

| Variabel | Arti | Bawaan |
|---|---|---|
| APP_PORT | Port aplikasi | 3000 |
| APP_NAME | Nama aplikasi | API Students |
| LOG_LEVEL | Level log | info |
| DB_HOST/PORT/USER/PASSWORD/NAME/SSLMODE/MAX_CONNS | Koneksi PostgreSQL | |
| JWT_SECRET | Kunci token, min 32 karakter | (wajib) |
| JWT_ISSUER | Penerbit token | api-students |
| JWT_ACCESS_TTL_MINUTES | Umur access token | 15 |
| JWT_REFRESH_TTL_DAYS | Umur refresh token | 7 |
| ALLOWED_ORIGINS | Origin CORS | http://localhost:5173 |

## Struktur berkas

```text
api-students/
├── app/
│   ├── model/                  entitas + request (tag validate) + ErrorResponse + CursorMeta
│   ├── repository/             student_repository, token_repository, role_repository
│   └── service/
│       ├── student_rules.go    ApplyPatch, IsEmptyPatch, CountTotalPages
│       ├── auth_rules_test.go  unit test validasi deklaratif
│       ├── student_authz_rules.go  CanAccessStudent + ownerValue
│       ├── student_service.go  CRUD (balik error, bukan response)
│       └── auth_service.go     register, login, refresh, logout, me
├── config/                     app.go, env.go, logger.go
├── database/                   koneksi PostgreSQL
├── helper/                     response, request, security, jwt, context, authz,
│                               errors (AppError), validator, cursor, negotiate
├── middleware/                 middleware.go (global) + auth.go + authz.go
├── route/                      pendaftaran route + Dependencies
├── migrations/                 001-005
├── postman/                    koleksi Postman
├── logs/                       (tidak di-commit)
├── .env / .env.example
└── main.go
```

## Endpoint

Basis URL: `http://localhost:3000/api/v1`

| Metode | Endpoint | Auth | Permission |
|---|---|---|---|
| GET | /health | publik | |
| POST | /auth/register | publik | |
| POST | /auth/login | publik (rate limit) | |
| POST | /auth/refresh | refresh token | |
| POST | /auth/logout | refresh token | |
| GET | /auth/me | Bearer token | |
| GET | /students | Bearer + student:list | cursor pagination |
| POST | /students | Bearer + student:create | validasi deklaratif |
| GET | /students/:id | Bearer + pemilik atau student:read:any | |
| PUT | /students/:id | Bearer + pemilik atau student:update:any | |
| PATCH | /students/:id | Bearer + pemilik atau student:update:any | |
| DELETE | /students/:id | Bearer + student:delete | |

### Content negotiation

Header `Accept: text/csv` pada GET /students menghasilkan berkas CSV. Tanpa header Accept atau dengan `*/*` menghasilkan JSON. Format lain menjawab 406.

### Cursor pagination

GET /students mendukung `?limit=N&cursor=XXX`. Respons memuat `next_cursor` dan `has_more` alih-alih `page` dan `total_pages`. Cursor menambatkan posisi pada `created_at` + `id` sehingga tidak ada duplikat atau baris yang terlewat bila data berubah di antara dua halaman.

## Matriks hak akses

Lihat Modul 6 — tidak berubah.

## Daftar status

| Status | Code | Situasi |
|---|---|---|
| 200 | | Berhasil |
| 201 | | Register dan tambah berhasil |
| 204 | | Hapus berhasil |
| 400 | BAD_REQUEST | JSON rusak, id salah, cursor tidak valid |
| 401 | UNAUTHORIZED | Token tidak ada, tidak valid, kedaluwarsa |
| 403 | FORBIDDEN | Role tidak punya permission, bukan pemilik data |
| 404 | NOT_FOUND | Data atau endpoint tidak ditemukan |
| 406 | NOT_ACCEPTABLE | Format Accept tidak tersedia |
| 409 | CONFLICT | NIM sudah terdaftar |
| 415 | UNSUPPORTED_MEDIA_TYPE | Content-Type bukan application/json |
| 422 | VALIDATION_ERROR | Tag validate dilanggar |
| 429 | TOO_MANY_REQUESTS | Login melebihi lima kali per menit |
| 500 | INTERNAL_ERROR | Kesalahan tak terduga (detail hanya di log) |
| 503 | | Database tidak dapat dihubungi |

## Log

Setiap request tercatat satu baris JSON. Request yang gagal menghasilkan dua baris: `http_request` (dengan identitas bila ada) dan `request_rejected` atau `request_failed` (dengan kode error). Identitas `user_id`, `nim`, dan `role` dicatat untuk request yang membawa token.
