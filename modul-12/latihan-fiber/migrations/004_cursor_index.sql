-- Urutan column pada index HARUS sama persis dengan ORDER BY pada
-- query, termasuk arah DESC-nya. Bila tidak, PostgreSQL tetap dapat
-- memakai index untuk sebagian kasus, tetapi tidak dapat memakainya
-- untuk melompat langsung ke posisi cursor.
CREATE INDEX IF NOT EXISTS users_created_at_id_desc_idx
    ON users (created_at DESC, id DESC);