#!/usr/bin/env bash

BASE="http://localhost:3000/api/v1"
J="Content-Type: application/json"

judul() {
  echo ""
  echo "=================================================="
  echo "$1"
  echo "=================================================="
}

judul "1. REGISTER tiga akun mahasiswa, harapan 201 + Location"
curl -s -i -X POST $BASE/auth/register -H "$J" \
  -d '{"nim":"434241084","name":"Vito Aditya","grade":88,"password":"rahasia123"}'
echo ""
curl -s -i -X POST $BASE/auth/register -H "$J" \
  -d '{"nim":"434241085","name":"Bagas Pratama","grade":64,"password":"rahasia456"}'
echo ""
curl -s -i -X POST $BASE/auth/register -H "$J" \
  -d '{"nim":"434241086","name":"Citra Ayu","grade":91,"password":"rahasia789"}'

judul "2. REGISTER password lemah, harapan 422"
curl -s -i -X POST $BASE/auth/register -H "$J" \
  -d '{"nim":"434241087","name":"Dedi Kurnia","password":"password1"}'

judul "3. REGISTER diselipkan role admin, harapan 201 dengan role tetap user"
curl -s -i -X POST $BASE/auth/register -H "$J" \
  -d '{"nim":"434241088","name":"Eka Putri","grade":90,"password":"rahasia888","role":"admin"}'

judul "4. REGISTER NIM duplikat, harapan 409"
curl -s -i -X POST $BASE/auth/register -H "$J" \
  -d '{"nim":"434241084","name":"Peniru Identitas","grade":50,"password":"rahasia000"}'

judul "5. GET students tanpa token, harapan 401 + WWW-Authenticate"
curl -s -i $BASE/students

judul "6. LOGIN password salah, harapan 401"
curl -s -i -X POST $BASE/auth/login -H "$J" \
  -d '{"nim":"434241084","password":"salahsekali9"}'

judul "7. LOGIN NIM tidak terdaftar, harapan 401 dengan pesan sama persis"
curl -s -i -X POST $BASE/auth/login -H "$J" \
  -d '{"nim":"999999999","password":"salahsekali9"}'

judul "8. LOGIN benar, harapan 200 + access_token + refresh_token"
RESP=$(curl -s -X POST $BASE/auth/login -H "$J" \
  -d '{"nim":"434241084","password":"rahasia123"}')
echo "$RESP"
ACCESS=$(echo "$RESP" | grep -o '"access_token":"[^"]*"' | cut -d'"' -f4)
REFRESH=$(echo "$RESP" | grep -o '"refresh_token":"[^"]*"' | cut -d'"' -f4)

judul "9. GET students dengan token, harapan 200"
curl -s -i $BASE/students -H "Authorization: Bearer $ACCESS"

judul "10. GET /auth/me dengan token, harapan 200"
curl -s -i $BASE/auth/me -H "Authorization: Bearer $ACCESS"

judul "11. Token diubah satu huruf terakhir, harapan 401"
curl -s -i $BASE/students -H "Authorization: Bearer ${ACCESS%?}X"

judul "12. Token ber-alg none, harapan 401"
PAYLOAD=$(echo "$ACCESS" | cut -d. -f2)
curl -s -i $BASE/students -H "Authorization: Bearer eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0.$PAYLOAD."

judul "13. REFRESH, harapan 200 dengan pasangan token baru"
RESP2=$(curl -s -X POST $BASE/auth/refresh -H "$J" -d "{\"refresh_token\":\"$REFRESH\"}")
echo "$RESP2"
ACCESS2=$(echo "$RESP2" | grep -o '"access_token":"[^"]*"' | cut -d'"' -f4)
REFRESH2=$(echo "$RESP2" | grep -o '"refresh_token":"[^"]*"' | cut -d'"' -f4)

judul "14. Refresh token lama dipakai ulang, harapan 401 karena rotasi"
curl -s -i -X POST $BASE/auth/refresh -H "$J" -d "{\"refresh_token\":\"$REFRESH\"}"

judul "15. CRUD mahasiswa dengan token tetap berfungsi"
RESP3=$(curl -s -X POST $BASE/students -H "$J" -H "Authorization: Bearer $ACCESS2" \
  -d '{"nim":"434241099","name":"Mahasiswa Data","grade":70}')
echo "$RESP3"
ID=$(echo "$RESP3" | grep -o '"id":[0-9]*' | head -1 | cut -d: -f2)
curl -s -o /dev/null -w "PUT    : %{http_code}\n" -X PUT $BASE/students/$ID -H "$J" \
  -H "Authorization: Bearer $ACCESS2" \
  -d '{"nim":"434241099","name":"Mahasiswa Data Revisi","grade":80,"is_active":true}'
curl -s -o /dev/null -w "DELETE : %{http_code}\n" -X DELETE $BASE/students/$ID \
  -H "Authorization: Bearer $ACCESS2"

judul "16. /auth/me tanpa token, harapan 401"
curl -s -i $BASE/auth/me

judul "17. BRUTE FORCE enam kali login gagal"
echo "catatan: tiga login di skenario 6 sampai 8 ikut terhitung limiter,"
echo "maka 429 dapat muncul sebelum percobaan keenam."
for i in 1 2 3 4 5 6; do
  curl -s -o /dev/null -w "percobaan $i: %{http_code}\n" -X POST $BASE/auth/login -H "$J" \
    -d '{"nim":"434241084","password":"salahsekali9"}'
done

judul "18. Health tetap publik, harapan 200"
curl -s -i $BASE/health

echo ""
echo "Selesai. Untuk menjalankan ulang, tunggu satu menit (rate limiter)"
echo "atau jalankan TRUNCATE students RESTART IDENTITY via psql."
