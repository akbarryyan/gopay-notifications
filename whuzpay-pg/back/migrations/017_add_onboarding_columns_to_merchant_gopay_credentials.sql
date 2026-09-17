-- Migration: Add onboarding columns to merchant_gopay_credentials
-- Dua kolom baru dari onboarding terpadu (lihat spec
-- 2026-09-17-whuzpay-pg-unified-onboarding-design.md):
-- gopay_username dicatat supaya merchant bisa login langsung ke
-- gopay-notifications kapan saja kalau mau, lepas dari whuzpay-pg.
-- qris_configured_at dicatat HANYA saat upload QRIS lewat wizard
-- onboarding ini berhasil -- kalau merchant upload manual langsung di
-- gopay-notifications belakangan, kolom ini TIDAK ikut ter-update (tidak
-- ada sesi gopay-notifications yang hidup untuk mendeteksinya). Ini
-- keterbatasan yang diterima, bukan bug -- pengecekan qris_not_configured
-- yang sungguhan tetap terjadi live di gopay-notifications saat invoice
-- dibuat, kolom ini cuma untuk tampilan status di Settings whuzpay-pg.

ALTER TABLE merchant_gopay_credentials
    ADD COLUMN gopay_username TEXT,
    ADD COLUMN qris_configured_at TIMESTAMPTZ;

COMMENT ON COLUMN merchant_gopay_credentials.gopay_username IS
    'Username akun gopay-notifications (dibuat manual atau otomatis lewat onboarding terpadu) -- ditampilkan di Settings, akun itu asli dan bisa dipakai login langsung ke gopay-notifications kapan saja.';
COMMENT ON COLUMN merchant_gopay_credentials.qris_configured_at IS
    'Diisi hanya saat QRIS diupload lewat wizard onboarding whuzpay-pg. NULL tidak selalu berarti QRIS belum ada di gopay-notifications -- bisa saja diupload manual langsung di sana, di luar sepengetahuan whuzpay-pg.';
