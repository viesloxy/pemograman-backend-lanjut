#!/usr/bin/env bash

BASE="http://localhost:3000/api/v1"

judul() {
  echo ""
  echo "=================================================="
  echo "$1"
  echo "=================================================="
}

judul "1. POST tiga mahasiswa, harapan 201 + header Location"
curl -s -i -X POST $BASE/students -H "Content-Type: application/json" \
  -d '{"nim":"082111633001","name":"Zagan Jade","grade":88}'
echo ""
curl -s -i -X POST $BASE/students -H "Content-Type: application/json" \
  -d '{"nim":"082111633002","name":"Bagas Pratama","grade":64}'
echo ""
curl -s -i -X POST $BASE/students -H "Content-Type: application/json" \
  -d '{"nim":"082111633003","name":"Citra Ayu","grade":91}'

judul "2. GET daftar dengan paginasi, harapan 200 + meta"
curl -s -i "$BASE/students?page=1&limit=2"

judul "3. GET daftar dengan pencarian dan pengurutan, harapan 200"
curl -s -i "$BASE/students?search=a&sort=grade&order=desc"

judul "4. GET daftar dengan filter status dan rentang nilai, harapan 200"
curl -s -i "$BASE/students?is_active=true&min_grade=70"

judul "5. GET satu mahasiswa, harapan 200"
curl -s -i $BASE/students/1

judul "6. GET id yang tidak ada, harapan 404"
curl -s -i $BASE/students/999

judul "7. GET id bukan angka, harapan 400"
curl -s -i $BASE/students/abc

judul "8. POST dengan NIM yang sudah dipakai, harapan 409"
curl -s -i -X POST $BASE/students -H "Content-Type: application/json" \
  -d '{"nim":"082111633001","name":"Nama Lain","grade":70}'

judul "9. POST tanpa Content-Type, harapan 415"
curl -s -i -X POST $BASE/students -d '{"nim":"082111633009","name":"Tanpa Header","grade":70}'

judul "10. POST dengan isi yang gagal validasi, harapan 422"
curl -s -i -X POST $BASE/students -H "Content-Type: application/json" \
  -d '{"nim":"abc","name":"Ka","grade":150}'

judul "11. POST dengan JSON rusak, harapan 400"
curl -s -i -X POST $BASE/students -H "Content-Type: application/json" \
  -d '{"nim":"082111633010",'

judul "12. Data sebelum diubah, simpan tangkapan layar ini"
curl -s -i $BASE/students/1

judul "13. PUT mengganti seluruh isi, harapan 200"
curl -s -i -X PUT $BASE/students/1 -H "Content-Type: application/json" \
  -d '{"nim":"082111633001","name":"Zagan Jade Revisi","grade":75,"is_active":false}'

judul "14. PUT tanpa mengirim seluruh field, harapan 422"
curl -s -i -X PUT $BASE/students/1 -H "Content-Type: application/json" \
  -d '{"name":"Kurang Lengkap"}'

judul "15. PATCH hanya satu field, harapan 200 dan field lain tetap"
curl -s -i -X PATCH $BASE/students/1 -H "Content-Type: application/json" \
  -d '{"grade":91}'

judul "16. PATCH tanpa satu pun field, harapan 400"
curl -s -i -X PATCH $BASE/students/1 -H "Content-Type: application/json" -d '{}'

judul "17. DELETE, harapan 204 tanpa body"
curl -s -i -X DELETE $BASE/students/2

judul "18. DELETE ulang pada id yang sama, harapan 404"
curl -s -i -X DELETE $BASE/students/2

judul "19. limit melebihi batas dan sort di luar daftar putih, harapan 200 dengan nilai aman"
curl -s -i "$BASE/students?limit=99999&sort=password"

judul "20. Endpoint yang tidak terdaftar, harapan 404"
curl -s -i $BASE/dosen

echo ""
echo "Selesai."
