@echo off
cd /d "C:\UNAIR\SEMESTER 5\PEMOGRAMAN BACKEND LANJUT (PRAKTIKUM)\modul-11\api-students"
set /p A=<tokens-admin.txt
curl -s http://localhost:3000/api/v1/admin/queue -H "Authorization: Bearer %A%" | python -m json.tool
