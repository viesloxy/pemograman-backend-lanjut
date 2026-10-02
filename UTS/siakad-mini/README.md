# SIAKAD Mini

RESTful API untuk layanan akademik sederhana: mengelola data mahasiswa, mata kuliah, dan Kartu Rencana Studi (KRS). Dibangun dengan Go, Fiber v2, dan PostgreSQL, menerapkan repository pattern, clean architecture, autentikasi JWT, dan pemeriksaan hak akses berbasis peran.

## Menjalankan Proyek

### 1. Siapkan database

Buat database `siakad_mini` pada PostgreSQL, lalu jalankan migration dan seeder berurutan:

```bash
psql -U postgres -d siakad_mini -f migrations/001_init.sql
psql -U postgres -d siakad_mini -f migrations/002_seed.sql
```

Seeder mengisi 1 admin, 20 mahasiswa, dan 11 mata kuliah. Hash password dihasilkan oleh program kecil berikut (dipakai saat menyusun seeder):

```bash
go run ./cmd/hashgen "password"
```

### 2. Konfigurasi

Salin `.env.example` menjadi `.env`, lalu isi minimal `DB_PASSWORD` dan `JWT_SECRET` (minimal 32 karakter acak; aplikasi menolak menyala bila kurang).

### 3. Jalankan

```bash
go run .
# atau
go build -o siakad-mini.exe . && ./siakad-mini.exe
```

Server berjalan di `http://localhost:3000`. Cek kesehatan: `GET /api/v1/health`.

## Akun Seed

| Peran | Email | Password |
|---|---|---|
| admin | admin@siakad.test | admin1234 |
| mahasiswa | `<nim>@siakad.test` (mis. 000434241084@siakad.test) | NIM-nya (mis. 000434241084) |

## Endpoint

| No | Method | Endpoint | Akses | Fungsi |
|---|---|---|---|---|
| 1 | POST | /api/v1/auth/login | publik (rate limit 5/menit) | Login, mengembalikan access token |
| 2 | GET | /api/v1/auth/me | semua role | Profil pengguna yang sedang login |
| 3 | GET | /api/v1/students | admin | Daftar mahasiswa dengan pagination, filter, search |
| 4 | POST | /api/v1/students | admin | Menambah mahasiswa sekaligus akun usernya |
| 5 | GET | /api/v1/students/{id} | admin, mahasiswa (data sendiri) | Detail mahasiswa + total SKS + batas SKS |
| 6 | PUT | /api/v1/students/{id} | admin | Memperbarui data mahasiswa (NIM tidak dapat diubah) |
| 7 | DELETE | /api/v1/students/{id} | admin | Soft delete mahasiswa |
| 8 | GET | /api/v1/courses | semua role | Daftar mata kuliah + terisi + sisa kuota |
| 9 | POST | /api/v1/enrollments | mahasiswa | Mengambil mata kuliah (menambah KRS) |
| 10 | DELETE | /api/v1/enrollments/{id} | mahasiswa (milik sendiri) | Membatalkan mata kuliah dari KRS |

Semua endpoint kecuali login memerlukan header `Authorization: Bearer <access_token>`.

## Aturan Bisnis

1. Batas SKS per tahun akademik menurut IPK terakhir: ≥ 3,00 → 24 SKS; 2,50–2,99 → 21 SKS; < 2,50 → 18 SKS.
2. Satu mata kuliah hanya dapat diambil sekali pada tahun akademik yang sama (dijaga unique constraint).
3. Mata kuliah yang kuotanya penuh tidak dapat diambil (pemeriksaan berjalan dalam satu transaksi dengan `SELECT ... FOR UPDATE`).
4. Mahasiswa hanya dapat melihat dan mengubah KRS miliknya sendiri.
5. Mahasiswa yang di-soft delete hilang dari daftar dan tidak dapat login.

## Contoh Pemakaian

### Login

```bash
curl -X POST http://localhost:3000/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"000434241084@siakad.test","password":"000434241084"}'
```

```json
{
  "success": true,
  "message": "login berhasil",
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIs...",
    "token_type": "Bearer",
    "expires_in": 900,
    "user": { "id": 6, "email": "000434241084@siakad.test", "role": "mahasiswa" }
  }
}
```

### Daftar mata kuliah

```bash
curl "http://localhost:3000/api/v1/courses?available=true" \
  -H "Authorization: Bearer <token>"
```

```json
{
  "success": true,
  "message": "Data mata kuliah berhasil diambil",
  "data": [
    {
      "id": 1, "kode_mk": "AGI401", "nama_mk": "Agama Islam II",
      "sks": 2, "semester": 3, "kuota": 40, "terisi": 0, "sisa_kuota": 40
    }
  ]
}
```

### Mengambil mata kuliah

```bash
curl -X POST http://localhost:3000/api/v1/enrollments \
  -H "Authorization: Bearer <token>" -H "Content-Type: application/json" \
  -d '{"course_id": 1, "tahun_akademik": "2026/2027-Ganjil"}'
```

### Contoh respons gagal validasi (422)

```json
{
  "success": false,
  "message": "Validasi gagal",
  "errors": {
    "nim": ["NIM harus tepat 12 digit angka"],
    "email": ["format email tidak valid"]
  }
}
```

### Contoh respons batas SKS (422)

```json
{
  "success": false,
  "message": "total SKS melebihi batas: mengambil 2 SKS menyisakan 1 SKS dari batas 18 SKS"
}
```

## Struktur Proyek

```
siakad-mini/
├── app/
│   ├── model/        struct entitas, request, respons
│   ├── repository/   interface + SQL + sentinel error
│   └── service/      business rules (murni) + handler fiber.Ctx
├── cmd/hashgen/      generator hash bcrypt untuk seeder
├── config/           env, logger, perakitan aplikasi + error handler
├── database/         connection pool pgx
├── helper/           jwt, bcrypt, validator, AppError, amplop respons
├── middleware/       auth (RequireAuth), authz (RequirePermission), log akses
├── migrations/       skema database + seeder
├── route/            peta URL, middleware, dan hak akses per endpoint
└── main.go           perakitan + graceful shutdown
```

Aturan utama arsitektur: repository tidak mengimpor Fiber, service tidak menulis SQL, dan model tidak mengimpor apa pun.

## Status HTTP

200 (sukses), 201 (dibuat), 204 (dihapus), 401 (belum login/token bermasalah), 403 (bukan haknya), 404 (tidak ditemukan), 409 (KRS duplikat), 422 (validasi gagal/kuota penuh/batas SKS), 429 (melebihi batas login), 500 (kesalahan server; detail teknis hanya di log).
