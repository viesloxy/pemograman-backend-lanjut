@echo off
cd /d "C:\UNAIR\SEMESTER 5\PEMOGRAMAN BACKEND LANJUT (PRAKTIKUM)\modul-11\api-students"
set /p T=<tokens-mhsuji.txt
echo == GET /notifications/mine ==
curl -s http://localhost:3000/api/v1/notifications/mine -H "Authorization: Bearer %T%" | python -m json.tool
echo == logs/mail.log ==
type logs\mail.log
echo.
echo == jumlah baris mhsuji@ ==
find /c "mhsuji@" logs\mail.log
