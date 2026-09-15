-- +goose Up
-- QRIS statis per account -- lihat spec
-- docs/superpowers/specs/2026-09-15-account-qris-image-design.md. Relasi
-- 1:1 murni dengan accounts: account_id sebagai primary key (bukan id
-- sendiri) karena tidak perlu riwayat/versi -- upload baru menimpa yang
-- lama lewat UPSERT.
CREATE TABLE account_qris_images (
    account_id   TEXT PRIMARY KEY REFERENCES accounts(id) ON DELETE CASCADE,
    image_data   BYTEA NOT NULL,
    content_type TEXT NOT NULL,
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- account_activity_log (migrasi 00016) perlu dua aksi baru untuk upload/
-- hapus QRIS dari halaman Settings.
ALTER TABLE account_activity_log DROP CONSTRAINT account_activity_log_action_check;
ALTER TABLE account_activity_log ADD CONSTRAINT account_activity_log_action_check
    CHECK (action IN (
        'login_success', 'login_failed', 'password_changed', 'password_reset',
        'api_key_created', 'api_key_revoked', 'device_added', 'device_deleted',
        'qris_image_updated', 'qris_image_removed'
    ));

-- +goose Down
DELETE FROM account_activity_log WHERE action IN ('qris_image_updated', 'qris_image_removed');
ALTER TABLE account_activity_log DROP CONSTRAINT account_activity_log_action_check;
ALTER TABLE account_activity_log ADD CONSTRAINT account_activity_log_action_check
    CHECK (action IN (
        'login_success', 'login_failed', 'password_changed', 'password_reset',
        'api_key_created', 'api_key_revoked', 'device_added', 'device_deleted'
    ));
DROP TABLE account_qris_images;
