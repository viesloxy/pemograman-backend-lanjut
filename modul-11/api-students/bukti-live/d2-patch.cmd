@echo off
cd /d "C:\UNAIR\SEMESTER 5\PEMOGRAMAN BACKEND LANJUT (PRAKTIKUM)\modul-11\api-students"
set /p A=<tokens-admin.txt
set PGPASSWORD=vito123
psql -U postgres -d praktikum_backend -c "SELECT id, code, quota, enrolled_count, version FROM courses WHERE id = 13"
echo == PATCH /courses/13 ==
curl -s -w "  HTTP %%{http_code}\n" -X PATCH http://localhost:3000/api/v1/courses/13 -H "Authorization: Bearer %A%" -H "Content-Type: application/json" -d "{\"quota\":42,\"version\":4}"
timeout /t 12 /nobreak >NUL
echo == event kuota terakhir ==
psql -U postgres -d praktikum_backend -c "SELECT id, event_type, published_at IS NOT NULL AS di_publish FROM outbox WHERE event_type='course.quota_changed' ORDER BY id DESC LIMIT 1"
echo == notifikasi kuota terakhir ==
psql -U postgres -d praktikum_backend -t -A -c "SELECT COUNT(*) FROM notifications WHERE outbox_id = (SELECT MAX(id) FROM outbox WHERE event_type='course.quota_changed')"
echo == email kuota di logs/mail.log ==
find /c "kuota IF399" logs\mail.log
