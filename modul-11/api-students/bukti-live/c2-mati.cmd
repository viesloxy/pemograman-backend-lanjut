@echo off
cd /d "C:\UNAIR\SEMESTER 5\PEMOGRAMAN BACKEND LANJUT (PRAKTIKUM)\modul-11\api-students"
set /p T=<tokens-mhsuji.txt
set /p A=<tokens-admin.txt
echo == docker compose stop redis ==
docker compose stop redis
timeout /t 2 /nobreak >NUL
echo == enroll kelas 5, 6, 7 ==
curl -s -o NUL -w "kelas 5: HTTP %%{http_code}\n" -X POST http://localhost:3000/api/v1/courses/5/enroll -H "Authorization: Bearer %T%" -H "Content-Type: application/json"
curl -s -o NUL -w "kelas 6: HTTP %%{http_code}\n" -X POST http://localhost:3000/api/v1/courses/6/enroll -H "Authorization: Bearer %T%" -H "Content-Type: application/json"
curl -s -o NUL -w "kelas 7: HTTP %%{http_code}\n" -X POST http://localhost:3000/api/v1/courses/7/enroll -H "Authorization: Bearer %T%" -H "Content-Type: application/json"
timeout /t 5 /nobreak >NUL
echo == GET /admin/queue saat Redis mati ==
curl -s http://localhost:3000/api/v1/admin/queue -H "Authorization: Bearer %A%" | python -m json.tool
