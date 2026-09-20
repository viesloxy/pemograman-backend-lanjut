# api-students

REST API data mahasiswa dengan Go, Fiber v2, dan PostgreSQL, dilengkapi autentikasi JWT dan otorisasi Role Based Access Control. Tugas Mandiri Modul 6 — Praktikum Pemrograman Backend Lanjut, D4 Teknik Informatika, Fakultas Vokasi, Universitas Airlangga.

Modul ini melanjutkan Modul 5. Autentikasi (login, JWT, refresh token) tetap berjalan; yang baru adalah lapisan otorisasi: hak akses disimpan di database dalam tiga tabel RBAC (roles, permissions, role_permissions), diperiksa middleware `RequirePermission` untuk keputusan yang tidak bergantung pada data, dan pemeriksaan kepemilikan di layer service untuk keputusan yang bergantung pada `owner_id`.

## Prasyarat

- Go 1.26+
- PostgreSQL 16 atau lebih baru
- `psql`, `openssl`
- Postman untuk pengujian (koleksi tersedia di `postman/`)

## Cara menjalankan dari nol

```bash
git clone <url-repo-ini>
cd modul-6/api-students

# 1. Buat basis data kosong
psql -U postgres -c "CREATE DATABASE praktikum_backend;"

# 2. Jalankan tiga migrasi berurutan
psql -U postgres -d praktikum_backend -f migrations/001_create_students.sql
psql -U postgres -d praktikum_backend -f migrations/002_auth.sql
psql -U postgres -d praktikum_backend -f migrations/003_rbac.sql

# 3. Salin contoh konfigurasi, isi DB_PASSWORD dan JWT_SECRET
cp .env.example .env
# JWT_SECRET dibuat acak minimal 32 karakter:
openssl rand -hex 32

# 4. Ambil dependensi dan jalankan
go mod tidy
go run .
```

Server menolak menyala bila `JWT_SECRET` kosong atau pendek, atau bila koneksi database gagal. Cek kesehatan:

```bash
curl -s -i http://localhost:3000/api/v1/health
```

Unit test business rules dan otorisasi: `go test ./app/service/ -v`.

## Variabel environment

| Variabel | Arti | Bawaan |
|---|---|---|
| APP_PORT | Port aplikasi | 3000 |
| APP_NAME | Nama aplikasi | API Students |
| LOG_LEVEL | Level log | info |
| DB_HOST / DB_PORT / DB_USER / DB_PASSWORD / DB_NAME / DB_SSLMODE | Koneksi PostgreSQL | localhost / 5432 / postgres / (wajib) / praktikum_backend / disable |
| DB_MAX_CONNS | Koneksi maksimum pool | 10 |
| JWT_SECRET | Kunci tanda tangan token, minimal 32 karakter | (wajib) |
| JWT_ISSUER | Penerbit token | api-students |
| JWT_ACCESS_TTL_MINUTES | Umur access token (menit) | 15 |
| JWT_REFRESH_TTL_DAYS | Umur refresh token (hari) | 7 |
| ALLOWED_ORIGINS | Origin yang diizinkan CORS | http://localhost:5173 |

## Skema tabel

Tiga tabel RBAC:

```sql
roles            (name PK, description, created_at)
permissions      (name PK, description)
role_permissions (role_name FK, permission_name FK, PRIMARY KEY gabungan)

students.role    → FOREIGN KEY ke roles(name) ON UPDATE CASCADE
students.owner_id → FOREIGN KEY ke students(id), menandai pembuat data
```

Lima permission: `student:list`, `student:read:any`, `student:create`, `student:update:any`, `student:delete`.

Pembagian per role:

| Permission | admin | staff | user |
|---|---|---|---|
| student:list | ya | ya | tidak |
| student:read:any | ya | ya | tidak |
| student:create | ya | ya | tidak |
| student:update:any | ya | tidak | tidak |
| student:delete | ya | tidak | tidak |

## Struktur berkas

```text
api-students/
├── app/
│   ├── model/                  entitas, request-respons, AuthUser
│   ├── repository/             student_repository, token_repository, role_repository
│   └── service/
│       ├── student_rules.go    aturan CRUD (murni)
│       ├── student_authz_rules.go  aturan kepemilikan (murni)
│       ├── auth_rules.go       aturan register dan login (murni)
│       ├── student_service.go  CRUD mahasiswa (controller)
│       └── auth_service.go     register, login, refresh, logout, me
├── config/                     app.go, env.go, logger.go
├── database/                   koneksi PostgreSQL
├── helper/                     response, request, security, jwt, context, authz
├── logs/                       output log (tidak di-commit)
├── middleware/                 middleware.go (global) dan authz.go (RequirePermission)
├── route/                      peta endpoint beserta permission-nya
├── migrations/                 001, 002, 003
├── postman/                    koleksi Postman
├── .env / .env.example
└── main.go
```

## Endpoint dan permission

Basis URL: `http://localhost:3000/api/v1`

| Metode | Endpoint | Auth | Permission | Keterangan |
|---|---|---|---|---|
| GET | /health | publik | | cek server dan database |
| POST | /auth/register | publik | | daftar akun, role=user |
| POST | /auth/login | publik (rate limit 5/menit/IP) | | hasil: access + refresh token |
| POST | /auth/refresh | bawa refresh token | | rotasi token |
| POST | /auth/logout | bawa refresh token | | cabut refresh token |
| GET | /auth/me | Bearer token | | profil + daftar permission |
| GET | /students | Bearer token | student:list | daftar dengan query lengkap |
| POST | /students | Bearer token | student:create | tambah mahasiswa |
| GET | /students/:id | Bearer token + kepemilikan | | admin/staff boleh semua, user hanya miliknya |
| PUT | /students/:id | Bearer token + kepemilikan | | admin/staff boleh semua, user hanya miliknya |
| PATCH | /students/:id | Bearer token + kepemilikan | | admin/staff boleh semua, user hanya miliknya |
| DELETE | /students/:id | Bearer token | student:delete | admin saja |

Parameter query endpoint daftar: `page`, `limit`, `search`, `sort`, `order`, `is_active`, `min_grade`, `max_grade` (sama seperti Modul 3 dan 4).

## Matriks hak akses

| Aksi | admin | staff | user pemilik | user bukan pemilik |
|---|---|---|---|---|
| GET /students | 200 | 200 | 403 | 403 |
| POST /students | 201 | 201 | 403 | 403 |
| GET /students/:id milik sendiri | 200 | 200 | 200 | 403 |
| GET /students/:id milik orang lain | 200 | 200 | 403 | 403 |
| PUT milik sendiri | 200 | 200 | 200 | 403 |
| PUT milik orang lain | 200 | 403 | 403 | 403 |
| DELETE milik orang lain | 204 | 403 | 403 | 403 |
| DELETE diri sendiri | 403 | 403 | 403 | 403 |

## Daftar status

| Status | Situasi |
|---|---|
| 200 | Berhasil |
| 201 | Register dan tambah mahasiswa berhasil |
| 204 | Hapus berhasil, tanpa body |
| 400 | JSON rusak, id salah, PATCH kosong |
| 401 | Token tidak ada, tidak valid, atau kedaluwarsa |
| 403 | Role tidak punya permission, bukan pemilik data, atau coba hapus diri sendiri |
| 409 | NIM sudah terdaftar |
| 415 | Content-Type bukan application/json |
| 422 | Isi permintaan gagal validasi |
| 429 | Melebihi lima percobaan login per menit |
| 500 | Kesalahan tak terduga |
| 503 | Database tidak dapat dihubungi |

## Log

Setiap request tercatat satu baris JSON di layar dan `logs/app.log` (rotasi otomatis), memuat `request_id`, method, path, status, durasi, IP, dan identitas pemanggil bila sudah melewati RequireAuth.

## Postman

Koleksi `postman/api-students-modul-6.postman_collection.json` tinggal di-import: semua request sudah terkonfigurasi dan skrip Tests menyimpan token otomatis setiap login. Urutan pakai: folder Setup Akun, lalu promosi role lewat psql, lalu Login, lalu folder Students per role, lalu Refresh dan Negatif.
