@echo off
cd /d "C:\UNAIR\SEMESTER 5\PEMOGRAMAN BACKEND LANJUT (PRAKTIKUM)\modul-11\api-students"
set /p T=<tokens-mhsuji.txt
echo == enroll kelas 1 sampai 6 (MAILER_FAIL_EVERY=3) ==
for %%c in (1 2 3 4 5 6) do curl -s -o NUL -w "kelas %%c: HTTP %%{http_code}\n" -X POST http://localhost:3000/api/v1/courses/%%c/enroll -H "Authorization: Bearer %T%" -H "Content-Type: application/json"
timeout /t 20 /nobreak >NUL
set PGPASSWORD=vito123
echo == baris notifications milik mhsuji ==
psql -U postgres -d praktikum_backend -t -A -c "SELECT COUNT(*) FROM notifications n JOIN students s ON s.id=n.user_id WHERE s.nim='4342210901'"
echo == GET /notifications/mine ==
curl -s http://localhost:3000/api/v1/notifications/mine -H "Authorization: Bearer %T%" | python -c "import sys,json;print('jumlah:',len(json.load(sys.stdin)['data']))"
echo == baris mail.log mhsuji@ ==
find /c "mhsuji@" logs\mail.log
