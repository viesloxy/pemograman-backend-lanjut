-- Skrip reset untuk uji beban Modul 9
-- Hapus pendaftaran/penetapan milik akun uji (nim LIKE 'mhs%'), lalu hitung ulang.
DELETE FROM enrollments
WHERE student_id IN (SELECT id FROM students WHERE nim LIKE 'mhs%');

DELETE FROM supervisions
WHERE student_id IN (SELECT id FROM students WHERE nim LIKE 'mhs%');

UPDATE courses c
SET enrolled_count = (SELECT COUNT(*) FROM enrollments e WHERE e.course_id = c.id);

UPDATE supervisors s
SET supervised_count = (SELECT COUNT(*) FROM supervisions v WHERE v.supervisor_id = s.id);
