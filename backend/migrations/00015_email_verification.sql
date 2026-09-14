-- +goose Up
-- Verifikasi email saat signup -- menutup celah salah ketik email yang
-- membuat account tidak bisa dipulihkan (email dipakai untuk reset
-- password dan seluruh notifikasi, lihat migrasi 00013/00014).

-- NULL berarti belum diverifikasi. Tidak ditolak dari memakai produk
-- (trial tetap jalan) -- ini pengingat, bukan gerbang, supaya signup tidak
-- mendadak butuh langkah tambahan yang menambah gesekan.
ALTER TABLE accounts ADD COLUMN email_verified_at TIMESTAMPTZ NULL;

-- Pola sama persis password_reset_tokens: hash, bukan token mentah.
CREATE TABLE email_verification_tokens (
    id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    account_id TEXT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    token_hash BYTEA NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at    TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX email_verification_tokens_account_idx ON email_verification_tokens (account_id, created_at DESC);

ALTER TABLE notification_log DROP CONSTRAINT notification_log_kind_check;
ALTER TABLE notification_log ADD CONSTRAINT notification_log_kind_check
    CHECK (kind IN ('expiry_reminder', 'device_offline', 'device_online', 'test',
                    'password_reset', 'password_changed', 'email_verification'));

-- +goose Down
DELETE FROM notification_log WHERE kind = 'email_verification';
ALTER TABLE notification_log DROP CONSTRAINT notification_log_kind_check;
ALTER TABLE notification_log ADD CONSTRAINT notification_log_kind_check
    CHECK (kind IN ('expiry_reminder', 'device_offline', 'device_online', 'test',
                    'password_reset', 'password_changed'));
DROP TABLE email_verification_tokens;
ALTER TABLE accounts DROP COLUMN email_verified_at;
