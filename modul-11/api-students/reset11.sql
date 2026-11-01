-- reset11.sql — mengembalikan keadaan akun uji Modul 11
--    psql -U postgres -d praktikum_backend -f reset11.sql
--
-- HANYA menyentuh dua akun uji: mhsuji (NIM 4342210901) dan
-- salahalamat (NIM 4342210902). Data mahasiswa
-- lain, kelas, dan seluruh tabel tetap utuh.

-- 1. Riwayat notifikasi dan event milik akun uji.
DELETE FROM notifications
WHERE user_id IN (SELECT id FROM students WHERE nim IN ('4342210901', '4342210902'));

DELETE FROM outbox
WHERE (payload->>'user_id')::INT IN (SELECT id FROM students WHERE nim IN ('4342210901', '4342210902'));

-- 2. KRS milik akun uji.
DELETE FROM enrollments
WHERE student_id IN (SELECT id FROM students WHERE nim IN ('4342210901', '4342210902'));

-- 3. Counter peserta dihitung ulang dari baris enrollments yang sebenarnya.
UPDATE courses c
SET enrolled_count = (SELECT COUNT(*) FROM enrollments e WHERE e.course_id = c.id);
