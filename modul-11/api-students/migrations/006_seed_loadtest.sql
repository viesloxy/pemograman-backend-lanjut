INSERT INTO students (nim, name, grade, password, role, is_active)
SELECT
    'mhs' || LPAD(g::TEXT, 3, '0'),
    'Mahasiswa Uji ' || g,
    75,
    '!',
    'user',
    true
FROM generate_series(1, 60) AS g
ON CONFLICT DO NOTHING;
