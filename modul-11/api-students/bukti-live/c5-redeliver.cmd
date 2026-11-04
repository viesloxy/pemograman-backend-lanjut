@echo off
cd /d "C:\UNAIR\SEMESTER 5\PEMOGRAMAN BACKEND LANJUT (PRAKTIKUM)\modul-11\api-students"
set /p T=<tokens-mhsuji.txt
set PGPASSWORD=vito123
echo == worker mati: enroll kelas 8 (201) ==
curl -s -o NUL -w "HTTP %%{http_code}\n" -X POST http://localhost:3000/api/v1/courses/8/enroll -H "Authorization: Bearer %T%" -H "Content-Type: application/json"
timeout /t 2 /nobreak >NUL
echo == id event tertahan ==
psql -U postgres -d praktikum_backend -c "SELECT id, event_type, published_at FROM outbox ORDER BY id DESC LIMIT 1"
