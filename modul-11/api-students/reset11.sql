-- reset11.sql — mengembalikan keadaan akun uji Modul 11
--    psql -U postgres -d praktikum_backend -f reset11.sql
--
-- HANYA menyentuh dua akun uji: mhsuji dan salahalamat (dicocokkan lewat
-- NIM karena proyek ini tidak menyimpan username terpisah). Data mahasiswa
-- lain, kelas, dan seluruh tabel tetap utuh.

-- 1. Riwayat notifikasi dan event milik akun uji.
DELETE FROM notifications
WHERE user_id IN (SELECT id FROM students WHERE nim IN ('mhsuji', 'salahalamat'));

DELETE FROM outbox
WHERE (payload->>'user_id')::INT IN (SELECT id FROM students WHERE nim IN ('mhsuji', 'salahalamat'));

-- 2. KRS milik akun uji.
DELETE FROM enrollments
WHERE student_id IN (SELECT id FROM students WHERE nim IN ('mhsuji', 'salahalamat'));

-- 3. Counter peserta dihitung ulang dari baris enrollments yang sebenarnya.
UPDATE courses c
SET enrolled_count = (SELECT COUNT(*) FROM enrollments e WHERE e.course_id = c.id);
