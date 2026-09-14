-- +goose Up
-- Konfigurasi SMTP + bot Telegram untuk pengingat kedaluwarsa, diatur vendor
-- lewat Vendor Dashboard (Settings) -- menggantikan env var SMTP_*/
-- TELEGRAM_BOT_TOKEN yang sempat dipakai sebelum pernah ter-deploy.
--
-- Satu baris saja: `id BOOLEAN PRIMARY KEY DEFAULT TRUE CHECK (id)` membuat
-- baris kedua mustahil disisipkan (primary key cuma bisa bernilai TRUE),
-- jadi tidak pernah ada pertanyaan "baris mana yang berlaku".
--
-- Password SMTP dan token bot DIENKRIPSI (secretbox, SETTINGS_SECRET_KEY),
-- bukan disimpan polos -- keduanya kredensial yang dipakai ulang tiap kali
-- mengirim, jadi tidak bisa di-hash satu arah (pola sama dengan
-- webhook_endpoints.secret_encrypted). NULL berarti belum/tidak diisi.
CREATE TABLE notification_settings (
    id                     BOOLEAN PRIMARY KEY DEFAULT TRUE CHECK (id),
    smtp_host              TEXT NOT NULL DEFAULT '',
    smtp_port              INTEGER NOT NULL DEFAULT 587,
    smtp_username          TEXT NOT NULL DEFAULT '',
    smtp_password_enc      BYTEA NULL,
    smtp_from              TEXT NOT NULL DEFAULT '',
    telegram_bot_token_enc BYTEA NULL,
    updated_at             TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_by             TEXT NULL
);

-- +goose Down
DROP TABLE notification_settings;
