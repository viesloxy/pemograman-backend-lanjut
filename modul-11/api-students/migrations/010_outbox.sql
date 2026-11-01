-- Migration 010: transactional outbox dan catatan notifikasi
-- (migration 008 pada modul; nomor bergeser karena 008 sudah dipakai Modul 10)
--    psql -U postgres -d praktikum_backend -f migrations/010_outbox.sql

-- -----------------------------------------------------------------------
-- outbox — "kotak surat keluar" milik database
--
-- Setiap event penting (mendaftar, keluar, pindah kelas) dicatat di
-- sini DI DALAM transaction yang sama dengan perubahan datanya. Karena
-- satu transaction, hanya ada dua kemungkinan: perubahan data dan
-- catatan event-nya sama-sama tersimpan, atau sama-sama batal.
-- -----------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS outbox (
    id          BIGSERIAL   PRIMARY KEY,
    event_type  VARCHAR(50) NOT NULL,
    payload     JSONB       NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- NULL berarti belum di-publish ke queue.
    published_at TIMESTAMPTZ NULL
);

-- Relay hanya pernah mencari baris yang belum di-publish. Partial index
-- membuat index ini tetap kecil walaupun tabel outbox tumbuh jutaan baris,
-- karena baris yang sudah di-publish ikut diindeks.
CREATE INDEX IF NOT EXISTS outbox_unpublished_idx
    ON outbox (id) WHERE published_at IS NULL;


-- -----------------------------------------------------------------------
-- notifications — bukti bahwa sebuah event SUDAH dikirimkan
--
-- UNIQUE (outbox_id) adalah idempotency key: satu event hanya boleh
-- menghasilkan satu notifikasi, berapa kali pun task-nya dijalankan.
--
-- outbox_id sengaja TIDAK diberi FOREIGN KEY ke outbox. Baris outbox yang
-- sudah lama di-publish suatu saat perlu dibersihkan, sedangkan riwayat
-- notifikasi milik mahasiswa harus tetap ada.
-- -----------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS notifications (
    id          BIGSERIAL   PRIMARY KEY,
    outbox_id   BIGINT      NOT NULL,
    user_id     INTEGER     NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    channel     VARCHAR(20) NOT NULL,
    subject     VARCHAR(200) NOT NULL,
    body        TEXT        NOT NULL,
    sent_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT notifications_outbox_unique UNIQUE (outbox_id)
);
CREATE INDEX IF NOT EXISTS notifications_user_idx ON notifications (user_id, outbox_id DESC);

INSERT INTO permissions (name, description) VALUES
    ('queue:monitor', 'Melihat keadaan outbox dan queue task')
ON CONFLICT (name) DO NOTHING;

INSERT INTO role_permissions (role_name, permission_name) VALUES
    ('admin', 'queue:monitor')
ON CONFLICT DO NOTHING;
