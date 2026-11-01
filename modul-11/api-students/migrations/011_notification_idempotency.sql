-- Migration 011 (migration 009 pada modul): idempotency key notifikasi
-- untuk event BERBANYAK penerima
--    psql -U postgres -d praktikum_backend -f migrations/011_notification_idempotency.sql
--
-- Latar (D.2): event course.quota_changed menghasilkan SATU event dengan
-- BANYAK notifikasi — satu untuk setiap peserta kelas. UNIQUE (outbox_id)
-- tidak lagi cocok: mahasiswa kedua dan seterusnya akan dianggap "sudah
-- dikirim" padahal belum. Idempotency key yang tepat adalah pasangan
-- (outbox_id, user_id): satu event dikirim paling sekali ke satu mahasiswa.
--
-- Unique constraint lama diganti, bukan ditambah: memakai dua constraint
-- berarti mahasiswa kedua tetap ditolak oleh constraint lama.

ALTER TABLE notifications
    DROP CONSTRAINT IF EXISTS notifications_outbox_unique;

ALTER TABLE notifications
    ADD CONSTRAINT notifications_outbox_user_unique UNIQUE (outbox_id, user_id);
