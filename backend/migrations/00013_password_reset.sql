-- +goose Up
-- Reset password lewat email dan ganti password swalayan untuk customer.

-- password_changed_at: sesi dashboard stateless (cookie bertanda tangan,
-- tanpa tabel sesi), jadi satu-satunya cara mencabut sesi lama setelah
-- password diganti adalah menolak token yang diterbitkan SEBELUM waktu
-- ini (lihat requireAdmin). NULL = belum pernah diganti sejak dibuat.
ALTER TABLE accounts ADD COLUMN password_changed_at TIMESTAMPTZ NULL;

-- Token disimpan sebagai SHA-256, bukan nilai aslinya: siapa pun yang bisa
-- membaca tabel ini (backup bocor, akses database) tetap tidak bisa
-- memakai token yang masih berlaku untuk mengambil alih akun.
CREATE TABLE password_reset_tokens (
    id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    account_id TEXT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    token_hash BYTEA NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at    TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX password_reset_tokens_account_idx ON password_reset_tokens (account_id, created_at DESC);

ALTER TABLE notification_log DROP CONSTRAINT notification_log_kind_check;
ALTER TABLE notification_log ADD CONSTRAINT notification_log_kind_check
    CHECK (kind IN ('expiry_reminder', 'device_offline', 'device_online', 'test',
                    'password_reset', 'password_changed'));

-- +goose Down
DELETE FROM notification_log WHERE kind IN ('password_reset', 'password_changed');
ALTER TABLE notification_log DROP CONSTRAINT notification_log_kind_check;
ALTER TABLE notification_log ADD CONSTRAINT notification_log_kind_check
    CHECK (kind IN ('expiry_reminder', 'device_offline', 'device_online', 'test'));
DROP TABLE password_reset_tokens;
ALTER TABLE accounts DROP COLUMN password_changed_at;
