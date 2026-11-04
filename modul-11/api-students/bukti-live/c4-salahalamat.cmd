@echo off
cd /d "C:\UNAIR\SEMESTER 5\PEMOGRAMAN BACKEND LANJUT (PRAKTIKUM)\modul-11\api-students"
set /p S=<tokens-salahalamat.txt
set /p A=<tokens-admin.txt
echo == salahalamat enroll kelas 2 ==
curl -s -o NUL -w "HTTP %%{http_code}\n" -X POST http://localhost:3000/api/v1/courses/2/enroll -H "Authorization: Bearer %S%" -H "Content-Type: application/json"
timeout /t 15 /nobreak >NUL
echo == GET /admin/queue setelah 15 detik ==
curl -s http://localhost:3000/api/v1/admin/queue -H "Authorization: Bearer %A%" | python -m json.tool
