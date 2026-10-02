-- Migration 001: skema awal SIAKAD Mini
-- Tabel: users, students, courses, enrollments

CREATE TABLE IF NOT EXISTS users (
    id         SERIAL PRIMARY KEY,
    email      VARCHAR(255) NOT NULL,
    password   VARCHAR(255) NOT NULL,
    role       VARCHAR(20)  NOT NULL DEFAULT 'mahasiswa',
    deleted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- Role dibatasi dua nilai sesuai kebutuhan SIAKAD Mini.
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_role_check;
ALTER TABLE users
    ADD CONSTRAINT users_role_check
    CHECK (role IN ('admin', 'mahasiswa'));

-- Email dijaga unik tanpa membedakan huruf besar dan kecil.
CREATE UNIQUE INDEX IF NOT EXISTS users_email_lower_key
    ON users (LOWER(email));

CREATE TABLE IF NOT EXISTS students (
    id            SERIAL PRIMARY KEY,
    user_id       INTEGER      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    nim           VARCHAR(12)  NOT NULL,
    nama          VARCHAR(150) NOT NULL,
    prodi         VARCHAR(100) NOT NULL,
    angkatan      INTEGER      NOT NULL,
    ipk_terakhir  NUMERIC(3,2),
    deleted_at    TIMESTAMPTZ,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- Satu user tepat satu data student.
CREATE UNIQUE INDEX IF NOT EXISTS students_user_id_key
    ON students (user_id);

ALTER TABLE students DROP CONSTRAINT IF EXISTS students_nim_key;
ALTER TABLE students
    ADD CONSTRAINT students_nim_key UNIQUE (nim);

-- NIM wajib tepat 12 digit angka, sama seperti aturan validasi endpoint.
ALTER TABLE students DROP CONSTRAINT IF EXISTS students_nim_format_check;
ALTER TABLE students
    ADD CONSTRAINT students_nim_format_check CHECK (nim ~ '^\d{12}$');

ALTER TABLE students DROP CONSTRAINT IF EXISTS students_angkatan_check;
ALTER TABLE students
    ADD CONSTRAINT students_angkatan_check
    CHECK (angkatan BETWEEN 2000 AND 2999);

ALTER TABLE students DROP CONSTRAINT IF EXISTS students_ipk_check;
ALTER TABLE students
    ADD CONSTRAINT students_ipk_check
    CHECK (ipk_terakhir IS NULL OR (ipk_terakhir >= 0 AND ipk_terakhir <= 4));

CREATE TABLE IF NOT EXISTS courses (
    id       SERIAL PRIMARY KEY,
    kode_mk  VARCHAR(20)  NOT NULL,
    nama_mk  VARCHAR(150) NOT NULL,
    sks      INTEGER      NOT NULL,
    semester INTEGER      NOT NULL,
    kuota    INTEGER      NOT NULL
);

ALTER TABLE courses DROP CONSTRAINT IF EXISTS courses_kode_mk_key;
ALTER TABLE courses
    ADD CONSTRAINT courses_kode_mk_key UNIQUE (kode_mk);

ALTER TABLE courses DROP CONSTRAINT IF EXISTS courses_sks_check;
ALTER TABLE courses
    ADD CONSTRAINT courses_sks_check CHECK (sks BETWEEN 1 AND 6);

ALTER TABLE courses DROP CONSTRAINT IF EXISTS courses_semester_check;
ALTER TABLE courses
    ADD CONSTRAINT courses_semester_check CHECK (semester BETWEEN 1 AND 8);

ALTER TABLE courses DROP CONSTRAINT IF EXISTS courses_kuota_check;
ALTER TABLE courses
    ADD CONSTRAINT courses_kuota_check CHECK (kuota > 0);

CREATE TABLE IF NOT EXISTS enrollments (
    id             BIGSERIAL PRIMARY KEY,
    student_id     INTEGER     NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    course_id      INTEGER     NOT NULL REFERENCES courses(id),
    tahun_akademik VARCHAR(20) NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Satu mahasiswa tidak boleh mengambil mata kuliah yang sama
-- dua kali pada tahun akademik yang sama. Aturan dijaga database.
ALTER TABLE enrollments DROP CONSTRAINT IF EXISTS enrollments_krs_unik;
ALTER TABLE enrollments
    ADD CONSTRAINT enrollments_krs_unik
    UNIQUE (student_id, course_id, tahun_akademik);

-- Format tahun akademik: 2026/2027-Ganjil
ALTER TABLE enrollments DROP CONSTRAINT IF EXISTS enrollments_tahun_akademik_check;
ALTER TABLE enrollments
    ADD CONSTRAINT enrollments_tahun_akademik_check
    CHECK (tahun_akademik ~ '^\d{4}/\d{4}-(Ganjil|Genap)$');

CREATE INDEX IF NOT EXISTS enrollments_course_id_idx
    ON enrollments (course_id);

CREATE INDEX IF NOT EXISTS enrollments_student_id_idx
    ON enrollments (student_id);
