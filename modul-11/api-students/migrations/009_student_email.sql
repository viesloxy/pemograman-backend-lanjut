-- Migration 009 (penyesuaian proyek): kolom email mahasiswa
--
-- Modul 11 mengirim notifikasi email ke mahasiswa, sedangkan skema
-- sebelumnya belum menyimpan alamat email. Kolom ini opsional:
-- bila kosong, alamat diturunkan dari NIM (<nim>@student.unair.ac.id).
--
--    psql -U postgres -d praktikum_backend -f migrations/009_student_email.sql

ALTER TABLE students
    ADD COLUMN IF NOT EXISTS email VARCHAR(200) NULL;
