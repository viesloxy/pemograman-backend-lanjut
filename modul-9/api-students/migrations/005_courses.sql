CREATE TABLE IF NOT EXISTS courses (
    id            SERIAL      PRIMARY KEY,
    code          VARCHAR(20) NOT NULL UNIQUE,
    name          VARCHAR(120) NOT NULL,
    credits       SMALLINT    NOT NULL CHECK (credits BETWEEN 1 AND 6),
    quota         INTEGER     NOT NULL CHECK (quota > 0),
    enrolled_count INTEGER    NOT NULL DEFAULT 0,
    version       INTEGER     NOT NULL DEFAULT 1,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT courses_enrolled_within_quota
        CHECK (enrolled_count >= 0 AND enrolled_count <= quota)
);

CREATE TABLE IF NOT EXISTS enrollments (
    id          BIGSERIAL   PRIMARY KEY,
    course_id   INTEGER     NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
    student_id  INTEGER     NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT enrollments_course_student_unique UNIQUE (course_id, student_id)
);

CREATE INDEX IF NOT EXISTS enrollments_student_idx ON enrollments (student_id);

INSERT INTO courses (code, name, credits, quota) VALUES
    ('IF101', 'Pemrograman Backend Lanjut', 3, 40),
    ('IF102', 'Basis Data Lanjut',          3, 40),
    ('IF301', 'Kecerdasan Buatan',          3, 40),
    ('IF302', 'Keamanan Informasi',         3, 40),
    ('IF303', 'Komputasi Awan',             3, 40),
    ('IF304', 'Data Mining',                3, 40),
    ('IF305', 'Pemrograman Mobile',         3, 40),
    ('IF306', 'Interaksi Manusia Komputer', 3, 40),
    ('IF307', 'Sistem Terdistribusi',       3, 40),
    ('IF308', 'Visi Komputer',              3, 40),
    ('IF309', 'Pemrosesan Bahasa Alami',    3, 40),
    ('IF310', 'Manajemen Proyek TI',        3, 40),
    ('IF399', 'Kelas Rebutan',              3, 5)
ON CONFLICT (code) DO NOTHING;

INSERT INTO permissions (name, description) VALUES
    ('course:manage', 'Mengubah kuota dan data kelas')
ON CONFLICT (name) DO NOTHING;

INSERT INTO role_permissions (role_name, permission_name) VALUES
    ('admin', 'course:manage')
ON CONFLICT DO NOTHING;
