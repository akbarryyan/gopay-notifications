-- +goose Up
-- Peringatan "HP bridge offline" ke customer, dan riwayat seluruh
-- notifikasi yang dikirim backend (pengingat kedaluwarsa, peringatan
-- offline/online, pesan uji) supaya vendor bisa melihat mana yang gagal.

-- offline_alert_for menyimpan NILAI heartbeat_at saat peringatan offline
-- dikirim -- pola yang sama dengan accounts.expiry_reminder_sent_for.
-- Selama heartbeat_at masih sama dengan nilai ini, HP belum kembali dan
-- peringatan tidak dikirim ulang. Begitu heartbeat baru masuk, heartbeat_at
-- bergerak melewati nilai ini -> pemberitahuan "kembali online" dikirim
-- lalu kolom dikosongkan, sehingga offline berikutnya memicu peringatan
-- baru.
ALTER TABLE devices ADD COLUMN offline_alert_for TIMESTAMPTZ NULL;

CREATE TABLE notification_log (
    id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    -- NULL untuk pesan uji dari Vendor Dashboard (tidak milik account mana pun).
    account_id  TEXT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    -- Tanpa foreign key, dan nama device disalin: device boleh dihapus
    -- permanen, riwayat notifikasinya tetap harus terbaca.
    device_id   TEXT NULL,
    device_name TEXT NULL,
    kind        TEXT NOT NULL CHECK (kind IN ('expiry_reminder', 'device_offline', 'device_online', 'test')),
    channel     TEXT NOT NULL CHECK (channel IN ('email', 'telegram')),
    recipient   TEXT NOT NULL,
    subject     TEXT NOT NULL,
    status      TEXT NOT NULL CHECK (status IN ('sent', 'failed')),
    error       TEXT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX notification_log_created_at_idx ON notification_log (created_at DESC, id DESC);
CREATE INDEX notification_log_account_idx ON notification_log (account_id, created_at DESC);

-- +goose Down
DROP TABLE notification_log;
ALTER TABLE devices DROP COLUMN offline_alert_for;
