# api-students

REST API data mahasiswa memakai Go dan Fiber v2. Tugas Mandiri Modul 2 mata kuliah Praktikum Pemrograman Backend Lanjut, Program Studi D4 Teknik Informatika, Fakultas Vokasi, Universitas Airlangga.

Entitas `Student` beserta method `UpdateGrade`, `Activate`, dan `Deactivate` dibawa dari tugas Modul 1, lalu ditambah field `NIM` sebagai penanda unik. Method tersebut kini dipanggil langsung oleh handler POST, PUT, dan PATCH.

Data disimpan di memori, jadi seluruh isinya hilang setiap kali server dimatikan. Basis data baru dipakai pada modul 3.

## Cara menjalankan

```bash
git clone <url-repo-ini>
cd modul-2/api-students

go mod tidy
go run .
```

Server berjalan di `http://localhost:3000`. Karena kode tersebar di beberapa berkas, jalankan dengan `go run .` dan bukan `go run main.go`.

Pengujian berurutan bisa memakai skrip yang disediakan: jalankan server di satu terminal, lalu `bash uji.sh` di terminal lain.

## Struktur berkas

```text
api-students/
├── go.mod
├── main.go      konfigurasi aplikasi, middleware, dan route
├── model.go     struct entitas, request, dan respons
├── helper.go    amplop respons, parser query string, dan validasi
├── handler.go   fungsi penangan tiap endpoint
├── uji.sh       skrip pengujian seluruh skenario status
└── README.md    kontrak API
```

## Bentuk respons

Seluruh endpoint memakai amplop yang sama, termasuk ketika gagal.

```json
{ "success": true,  "message": "...", "data": { } }
{ "success": true,  "message": "...", "data": [ ], "meta": { } }
{ "success": false, "message": "mahasiswa tidak ditemukan" }
{ "success": false, "message": "validasi gagal", "errors": { "nim": "wajib diisi" } }
```

Satu-satunya pengecualian adalah 204 pada DELETE, yang memang tidak boleh membawa body.

## Entitas Student

| Field | Tipe | Keterangan |
|---|---|---|
| id | int | Dibuat server, tidak bisa dikirim klien |
| nim | string | 9 sampai 15 digit angka, unik |
| name | string | 3 sampai 100 karakter |
| grade | float | Rentang 0 sampai 100 |
| is_active | bool | Bernilai true saat data pertama dibuat |
| created_at | time | Dibuat server |

## Kontrak API

Basis URL: `http://localhost:3000/api/v1`

| Metode | Endpoint | Parameter | Contoh body permintaan | Status yang mungkin | Contoh respons |
|---|---|---|---|---|---|
| GET | /students | Query: `page`, `limit`, `search`, `sort`, `order`, `is_active`, `min_grade`, `max_grade` | tidak ada | 200 | `{"success":true,"message":"daftar mahasiswa berhasil diambil","data":[{"id":1,"nim":"082111633001","name":"Zagan Jade","grade":88,"is_active":true,"created_at":"2026-08-21T09:00:00Z"}],"meta":{"page":1,"limit":10,"total":1,"total_pages":1}}` |
| GET | /students/:id | Path: `id` (angka positif) | tidak ada | 200, 400, 404 | `{"success":true,"message":"mahasiswa ditemukan","data":{"id":1,"nim":"082111633001","name":"Zagan Jade","grade":88,"is_active":true,"created_at":"2026-08-21T09:00:00Z"}}` |
| POST | /students | Header: `Content-Type: application/json` | `{"nim":"082111633001","name":"Zagan Jade","grade":88}` | 201, 400, 409, 415, 422 | `{"success":true,"message":"mahasiswa berhasil ditambahkan","data":{"id":1,"nim":"082111633001","name":"Zagan Jade","grade":88,"is_active":true,"created_at":"2026-08-21T09:00:00Z"}}` disertai header `Location: /api/v1/students/1` |
| PUT | /students/:id | Path: `id`. Header: `Content-Type: application/json`. Seluruh field wajib dikirim | `{"nim":"082111633009","name":"Zagan Baru","grade":75,"is_active":false}` | 200, 400, 404, 409, 415, 422 | `{"success":true,"message":"data mahasiswa berhasil diganti seluruhnya","data":{"id":1,"nim":"082111633009","name":"Zagan Baru","grade":75,"is_active":false,"created_at":"2026-08-21T09:00:00Z"}}` |
| PATCH | /students/:id | Path: `id`. Header: `Content-Type: application/json`. Hanya field yang ingin diubah | `{"grade":91}` | 200, 400, 404, 409, 415, 422 | `{"success":true,"message":"data mahasiswa berhasil diperbarui sebagian","data":{"id":1,"nim":"082111633009","name":"Zagan Baru","grade":91,"is_active":false,"created_at":"2026-08-21T09:00:00Z"}}` |
| DELETE | /students/:id | Path: `id` | tidak ada | 204, 400, 404 | tanpa body |
| GET | /health | tidak ada | tidak ada | 200 | `{"success":true,"message":"server berjalan","data":{"timestamp":"2026-08-21T09:00:00Z"}}` |

## Parameter query pada endpoint daftar

| Parameter | Kegunaan | Bawaan | Catatan |
|---|---|---|---|
| page | Halaman keberapa | 1 | Nilai di bawah 1 dikembalikan ke 1 |
| limit | Jumlah baris per halaman | 10 | Dibatasi maksimal 50 |
| search | Kata kunci pada nama | kosong | Tidak membedakan huruf besar dan kecil |
| sort | Field pengurutan | id | Hanya menerima `id`, `nim`, `name`, `grade`, `created_at` |
| order | Arah pengurutan | asc | Selain `desc` dianggap `asc` |
| is_active | Menyaring status aktif | tidak menyaring | Menerima true dan false |
| min_grade | Batas bawah nilai | tidak menyaring | Angka |
| max_grade | Batas atas nilai | tidak menyaring | Angka |

Alasan batas atas limit dipilih 50: satu halaman daftar mahasiswa yang masih nyaman dibaca dan digambar antarmuka klien ada di kisaran puluhan baris. Tanpa batas ini, siapa pun bisa mengirim `?limit=99999999` dan memaksa server menyusun respons raksasa dalam sekali permintaan.

## Daftar status yang dipakai

| Status | Situasi |
|---|---|
| 200 | Pengambilan, penggantian, atau perubahan berhasil |
| 201 | Penambahan berhasil, disertai header `Location` |
| 204 | Penghapusan berhasil, tanpa body |
| 400 | JSON rusak, id bukan angka positif, atau PATCH tanpa satu pun field |
| 404 | Data atau endpoint tidak ditemukan |
| 409 | NIM sudah dipakai data lain |
| 415 | Content-Type bukan application/json |
| 422 | Bentuk permintaan benar tetapi isinya gagal validasi |

## Contoh pengujian

```bash
# 201 dan header Location
curl -i -X POST localhost:3000/api/v1/students \
  -H "Content-Type: application/json" \
  -d '{"nim":"082111633001","name":"Zagan Jade","grade":88}'

# 200 dengan paginasi, pencarian, dan penyaringan
curl -i "localhost:3000/api/v1/students?page=1&limit=2&sort=grade&order=desc&is_active=true"

# 409 karena NIM diulang
curl -i -X POST localhost:3000/api/v1/students \
  -H "Content-Type: application/json" \
  -d '{"nim":"082111633001","name":"Nama Lain","grade":70}'

# 415 karena Content-Type tidak disertakan
curl -i -X POST localhost:3000/api/v1/students -d '{"nim":"082111633009"}'

# 422 karena PUT tidak mengirim seluruh field
curl -i -X PUT localhost:3000/api/v1/students/1 \
  -H "Content-Type: application/json" \
  -d '{"name":"Kurang Lengkap"}'

# 200, PATCH hanya menyentuh field yang dikirim
curl -i -X PATCH localhost:3000/api/v1/students/1 \
  -H "Content-Type: application/json" \
  -d '{"grade":91}'

# 204 tanpa body
curl -i -X DELETE localhost:3000/api/v1/students/1
```

## Catatan

Endpoint daftar mengembalikan `data` berupa array kosong bila tidak ada yang cocok, bukan `null`, supaya klien tidak perlu memeriksa dua kemungkinan.
