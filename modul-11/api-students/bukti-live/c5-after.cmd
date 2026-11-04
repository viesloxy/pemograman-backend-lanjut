@echo off
cd /d "C:\UNAIR\SEMESTER 5\PEMOGRAMAN BACKEND LANJUT (PRAKTIKUM)\modul-11\api-students"
set PGPASSWORD=vito123
echo == jumlah notifikasi outbox terakhir (harap 1) ==
psql -U postgres -d praktikum_backend -t -A -c "SELECT COUNT(*) FROM notifications WHERE outbox_id = (SELECT MAX(id) FROM outbox)"
echo == baris mail.log kelas 8 ==
find /c "IF306" logs\mail.log
