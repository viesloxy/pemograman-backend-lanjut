@echo off
cd /d "C:\UNAIR\SEMESTER 5\PEMOGRAMAN BACKEND LANJUT (PRAKTIKUM)\modul-11\api-students"
set /p T=<tokens-mhsuji.txt
set /p A=<tokens-admin.txt
echo == docker compose start redis ==
docker compose start redis
timeout /t 10 /nobreak >NUL
echo == GET /admin/queue ==
curl -s http://localhost:3000/api/v1/admin/queue -H "Authorization: Bearer %A%" | python -m json.tool
echo == notifikasi mhsuji (harap 7) ==
curl -s http://localhost:3000/api/v1/notifications/mine -H "Authorization: Bearer %T%" | python -c "import sys,json;d=json.load(sys.stdin)['data'];print('jumlah:',len(d))"
echo == mail.log mhsuji@ ==
find /c "mhsuji@" logs\mail.log
