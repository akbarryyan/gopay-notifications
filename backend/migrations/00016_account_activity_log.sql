-- +goose Up
-- Riwayat aktivitas akun untuk halaman Logs di Customer Dashboard --
-- login, ganti password, API key dibuat/dicabut, device ditambah/dihapus.
-- Berguna saat customer curiga ada akses yang bukan dari mereka.
--
-- Terpisah dari audit_log (aksi VENDOR terhadap account, dilihat vendor)
-- dan notification_log (pesan KELUAR ke customer) -- tabel ini mencatat
-- aksi CUSTOMER terhadap account_nya sendiri, dilihat customer.
CREATE TABLE account_activity_log (
    id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    account_id TEXT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    action     TEXT NOT NULL CHECK (action IN (
        'login_success', 'login_failed', 'password_changed', 'password_reset',
        'api_key_created', 'api_key_revoked', 'device_added', 'device_deleted'
    )),
    -- NULL untuk login_failed via username yang tidak terdaftar sama sekali
    -- -- ditolak sebelum account manapun diketahui, jadi tidak ada baris
    -- yang bisa dibuat (lihat komentar di handleAdminLogin).
    ip_address TEXT NULL,
    user_agent TEXT NULL,
    metadata   JSONB NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX account_activity_log_account_idx ON account_activity_log (account_id, created_at DESC, id DESC);

-- +goose Down
DROP TABLE account_activity_log;
