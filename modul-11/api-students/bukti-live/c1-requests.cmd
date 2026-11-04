@echo off
cd /d "C:\UNAIR\SEMESTER 5\PEMOGRAMAN BACKEND LANJUT (PRAKTIKUM)\modul-11\api-students"
set /p T=<tokens-mhsuji.txt
echo == 1. POST /courses/2/enroll ==
curl -s -o NUL -w "HTTP %%{http_code}" -X POST http://localhost:3000/api/v1/courses/2/enroll -H "Authorization: Bearer %T%" -H "Content-Type: application/json"
echo.
echo == 2. POST /courses/2/enroll (diulang) ==
curl -s -w "  HTTP %%{http_code}" -X POST http://localhost:3000/api/v1/courses/2/enroll -H "Authorization: Bearer %T%" -H "Content-Type: application/json" | findstr /C:"ALREADY_ENROLLED" & echo 409 ALREADY_ENROLLED
echo == 3. DELETE /courses/2/enroll ==
curl -s -o NUL -w "HTTP %%{http_code}" -X DELETE http://localhost:3000/api/v1/courses/2/enroll -H "Authorization: Bearer %T%"
echo.
echo == 4. POST /courses/2/enroll ==
curl -s -o NUL -w "HTTP %%{http_code}" -X POST http://localhost:3000/api/v1/courses/2/enroll -H "Authorization: Bearer %T%" -H "Content-Type: application/json"
echo.
echo == 5. POST /enrollments/swap ==
curl -s -w "  HTTP %%{http_code}" -X POST http://localhost:3000/api/v1/enrollments/swap -H "Authorization: Bearer %T%" -H "Content-Type: application/json" -d "{\"from_course_id\":2,\"to_course_id\":4}" | findstr /C:"ditukar" & echo 200
