CREATE TABLE IF NOT EXISTS roles (
    name VARCHAR(20) PRIMARY KEY,
    description VARCHAR(150) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO roles (name, description) VALUES
    ('admin', 'Akses penuh terhadap seluruh data mahasiswa'),
    ('staff', 'Boleh melihat dan menambah data mahasiswa'),
    ('user', 'Hanya boleh mengelola datanya sendiri')
ON CONFLICT (name) DO NOTHING;

CREATE TABLE IF NOT EXISTS permissions (
    name VARCHAR(50) PRIMARY KEY,
    description VARCHAR(150) NOT NULL
);

CREATE TABLE IF NOT EXISTS role_permissions (
    role_name VARCHAR(20) NOT NULL
        REFERENCES roles(name) ON DELETE CASCADE,
    permission_name VARCHAR(50) NOT NULL
        REFERENCES permissions(name) ON DELETE CASCADE,
    PRIMARY KEY (role_name, permission_name)
);

UPDATE students SET role = 'user' WHERE role NOT IN (SELECT name FROM roles);

ALTER TABLE students DROP CONSTRAINT IF EXISTS students_role_fkey;
ALTER TABLE students
    ADD CONSTRAINT students_role_fkey
    FOREIGN KEY (role) REFERENCES roles(name) ON UPDATE CASCADE;

CREATE INDEX IF NOT EXISTS students_role_idx ON students (role);
