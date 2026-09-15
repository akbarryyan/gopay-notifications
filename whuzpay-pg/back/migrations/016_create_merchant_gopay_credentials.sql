-- Migration: Create merchant_gopay_credentials table
-- Kredensial gopay-notifications milik tiap merchant -- API key untuk
-- memanggil POST /invoices, webhook secret untuk verifikasi
-- X-Webhook-Signature. Disimpan polos (plaintext), pola sama dengan
-- merchants.webhook_secret (migrasi 013): keamanan mengandalkan akses
-- database, bukan enkripsi aplikasi (whuzpay-pg belum punya sistem kripto).

CREATE TABLE IF NOT EXISTS merchant_gopay_credentials (
    merchant_id UUID PRIMARY KEY REFERENCES merchants(id) ON DELETE CASCADE,
    api_key TEXT,
    webhook_secret TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

COMMENT ON TABLE merchant_gopay_credentials IS 'Kredensial gopay-notifications per merchant. Kedua kolom independen -- merchant boleh isi api_key dulu sebelum sempat bikin webhook endpoint.';
COMMENT ON COLUMN merchant_gopay_credentials.api_key IS 'sk_... dari halaman API Keys gopay-notifications milik merchant ini.';
COMMENT ON COLUMN merchant_gopay_credentials.webhook_secret IS 'whsec_... dari webhook endpoint yang dibuat merchant di gopay-notifications, mengarah ke {APP_BASE_URL}/api/v1/provider-webhooks/gopay.';
