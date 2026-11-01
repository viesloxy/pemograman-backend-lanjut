CREATE TABLE IF NOT EXISTS supervisors (
    id               SERIAL      PRIMARY KEY,
    name             VARCHAR(120) NOT NULL,
    quota            INTEGER     NOT NULL DEFAULT 3 CHECK (quota > 0),
    supervised_count INTEGER     NOT NULL DEFAULT 0,
    version          INTEGER     NOT NULL DEFAULT 1,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT supervisors_within_quota
        CHECK (supervised_count >= 0 AND supervised_count <= quota)
);

CREATE TABLE IF NOT EXISTS supervisions (
    student_id    INTEGER     NOT NULL UNIQUE REFERENCES students(id) ON DELETE CASCADE,
    supervisor_id INTEGER     NOT NULL REFERENCES supervisors(id) ON DELETE CASCADE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS supervisions_supervisor_idx
    ON supervisions (supervisor_id);

INSERT INTO supervisors (name, quota) VALUES
    ('Dr. Andi Wijaya, M.Kom',    3),
    ('Prof. Siti Rahayu, Ph.D',   3),
    ('Rina Kartika, M.T',         2)
ON CONFLICT DO NOTHING;

INSERT INTO permissions (name, description) VALUES
    ('supervision:manage', 'Menetapkan dan mengubah pembimbing tugas akhir')
ON CONFLICT (name) DO NOTHING;

INSERT INTO role_permissions (role_name, permission_name) VALUES
    ('admin', 'supervision:manage')
ON CONFLICT DO NOTHING;
