-- +goose Up
-- Pivot self-hosted -> hosted multi-tenant. Lihat
-- docs/superpowers/specs/2026-09-13-multitenant-accounts-design.md.
--
-- accounts menggantikan admin_users (login) DAN customers+licenses License
-- Server (identitas + plan/kuota) -- satu baris = satu customer = satu
-- login, sesuai keputusan MVP (bukan multi-user per akun).
CREATE TABLE accounts (
    id            TEXT PRIMARY KEY,
    business_name TEXT NOT NULL,
    email         TEXT NOT NULL UNIQUE,
    username      TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    plan          TEXT NOT NULL,
    max_devices   INTEGER NOT NULL,
    admin_status  TEXT NOT NULL DEFAULT 'active',
    expires_at    TIMESTAMPTZ NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- vendor_admins: akun Akbar (superadmin) buat login ke Vendor Dashboard.
-- Skema identik admin_users lama -- satu instalasi backend, tapi dua jenis
-- sesi yang tidak boleh pernah tertukar (cookie vendor_session, BUKAN
-- admin_session).
CREATE TABLE vendor_admins (
    id            BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    username      TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Akun pertama: toko Akbar sendiri, migrasi dari admin_users yang sekarang
-- (kalau ada) supaya password yang sudah dipakai tetap jalan tanpa reset.
-- Di database test yang kosong, admin_users tidak punya baris -- fallback
-- ke akun default dengan hash yang sengaja tidak valid (tidak pernah
-- dipakai login di test manapun, cuma supaya FK backfill di bawah tidak
-- menabrak NOT NULL).
INSERT INTO accounts (id, business_name, email, username, password_hash, plan, max_devices, expires_at)
SELECT
    'acc_default',
    'Akun Utama',
    'akun-utama@whuzpay.local',
    admin_users.username,
    admin_users.password_hash,
    'Enterprise',
    -1,
    now() + interval '10 years'
FROM admin_users
LIMIT 1;

INSERT INTO accounts (id, business_name, email, username, password_hash, plan, max_devices, expires_at)
SELECT 'acc_default', 'Akun Utama', 'akun-utama@whuzpay.local', 'akun-utama-default',
       '$2a$10$00000000000000000000000000000000000000000000000000000',
       'Enterprise', -1, now() + interval '10 years'
WHERE NOT EXISTS (SELECT 1 FROM accounts WHERE id = 'acc_default');

ALTER TABLE devices             ADD COLUMN account_id TEXT REFERENCES accounts(id);
ALTER TABLE invoices            ADD COLUMN account_id TEXT REFERENCES accounts(id);
ALTER TABLE notification_events ADD COLUMN account_id TEXT REFERENCES accounts(id);
ALTER TABLE api_keys            ADD COLUMN account_id TEXT REFERENCES accounts(id);
ALTER TABLE webhook_endpoints   ADD COLUMN account_id TEXT REFERENCES accounts(id);
ALTER TABLE webhook_deliveries  ADD COLUMN account_id TEXT REFERENCES accounts(id);
ALTER TABLE event_reviews       ADD COLUMN account_id TEXT REFERENCES accounts(id);

UPDATE devices             SET account_id = 'acc_default' WHERE account_id IS NULL;
UPDATE invoices            SET account_id = 'acc_default' WHERE account_id IS NULL;
UPDATE notification_events SET account_id = 'acc_default' WHERE account_id IS NULL;
UPDATE api_keys            SET account_id = 'acc_default' WHERE account_id IS NULL;
UPDATE webhook_endpoints   SET account_id = 'acc_default' WHERE account_id IS NULL;
UPDATE webhook_deliveries  SET account_id = 'acc_default' WHERE account_id IS NULL;
UPDATE event_reviews       SET account_id = 'acc_default' WHERE account_id IS NULL;

ALTER TABLE devices             ALTER COLUMN account_id SET NOT NULL;
ALTER TABLE invoices            ALTER COLUMN account_id SET NOT NULL;
ALTER TABLE notification_events ALTER COLUMN account_id SET NOT NULL;
ALTER TABLE api_keys            ALTER COLUMN account_id SET NOT NULL;
ALTER TABLE webhook_endpoints   ALTER COLUMN account_id SET NOT NULL;
ALTER TABLE webhook_deliveries  ALTER COLUMN account_id SET NOT NULL;
ALTER TABLE event_reviews       ALTER COLUMN account_id SET NOT NULL;

CREATE INDEX devices_account_id_idx ON devices (account_id);
CREATE INDEX invoices_account_id_idx ON invoices (account_id);
CREATE INDEX notification_events_account_id_idx ON notification_events (account_id);
CREATE INDEX api_keys_account_id_idx ON api_keys (account_id);
CREATE INDEX webhook_endpoints_account_id_idx ON webhook_endpoints (account_id);

-- Constraint unik invoice di-scope ulang per akun (spec §2.3) -- dua
-- merchant beda BOLEH pakai external_ref/nominal yang sama, mereka toko
-- yang berbeda sama sekali.
DROP INDEX invoices_external_ref_idx;
CREATE UNIQUE INDEX invoices_account_external_ref_idx ON invoices (account_id, external_ref);

DROP INDEX invoices_pending_unique_amount_idx;
CREATE UNIQUE INDEX invoices_account_pending_unique_amount_idx
    ON invoices (account_id, unique_amount) WHERE status = 'PENDING';

DROP TABLE admin_users;

-- +goose Down
CREATE TABLE admin_users (
    id            BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    username      TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
INSERT INTO admin_users (username, password_hash)
SELECT username, password_hash FROM accounts WHERE id = 'acc_default';

DROP INDEX invoices_account_pending_unique_amount_idx;
CREATE UNIQUE INDEX invoices_pending_unique_amount_idx
    ON invoices (unique_amount) WHERE status = 'PENDING';
DROP INDEX invoices_account_external_ref_idx;
CREATE UNIQUE INDEX invoices_external_ref_idx ON invoices (external_ref);

ALTER TABLE event_reviews       DROP COLUMN account_id;
ALTER TABLE webhook_deliveries  DROP COLUMN account_id;
ALTER TABLE webhook_endpoints   DROP COLUMN account_id;
ALTER TABLE api_keys            DROP COLUMN account_id;
ALTER TABLE notification_events DROP COLUMN account_id;
ALTER TABLE invoices            DROP COLUMN account_id;
ALTER TABLE devices             DROP COLUMN account_id;

DROP TABLE vendor_admins;
DROP TABLE accounts;
