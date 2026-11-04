@echo off
set PGPASSWORD=vito123
psql -U postgres -d praktikum_backend -c "SELECT o.id, o.event_type, o.published_at IS NOT NULL AS di_publish, n.subject FROM outbox o LEFT JOIN notifications n ON n.outbox_id = o.id WHERE (o.payload->>'user_id')::INT = 123 ORDER BY o.id"
