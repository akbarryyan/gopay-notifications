-- +goose Up

-- Akun vendor untuk Vendor Dashboard. Satu akun cukup untuk MVP, sama
-- seperti admin_users di database gopay milik tiap instalasi customer.
CREATE TABLE admin_users (
    id            BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    username      TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE customers (
    id         TEXT PRIMARY KEY,
    name       TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE licenses (
    id                        TEXT PRIMARY KEY,
    customer_id               TEXT NOT NULL REFERENCES customers(id),
    key_hash                  BYTEA NOT NULL UNIQUE,
    plan                      TEXT NOT NULL,
    status                    TEXT NOT NULL CHECK (status IN ('active', 'suspended', 'revoked')),
    max_devices               INT NOT NULL,
    production_installations  INT NOT NULL DEFAULT 1,
    uat_installations         INT NOT NULL DEFAULT 1,
    issued_at                 DATE NOT NULL,
    expires_at                DATE NOT NULL,
    created_at                TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at                TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX licenses_customer_id_idx ON licenses (customer_id);

-- Satu license bisa punya beberapa installation (production + uat), tapi
-- kuota tiap environment dibatasi kolom *_installations di atas — dihitung
-- dari baris yang released_at masih NULL, bukan dari total baris sepanjang
-- masa (installation yang di-reset harus membuka kuota lagi).
CREATE TABLE installations (
    id              TEXT PRIMARY KEY,
    license_id      TEXT NOT NULL REFERENCES licenses(id),
    environment     TEXT NOT NULL CHECK (environment IN ('production', 'uat')),
    product_version TEXT NOT NULL,
    activated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    released_at     TIMESTAMPTZ
);

CREATE INDEX installations_license_id_idx ON installations (license_id);
-- Menghitung kuota terpakai per license+environment adalah query yang
-- berjalan di setiap /activate — indeks partial ini menyaring baris yang
-- belum di-release, yang justru satu-satunya yang perlu dihitung.
CREATE INDEX installations_active_idx ON installations (license_id, environment)
    WHERE released_at IS NULL;

CREATE TABLE audit_log (
    id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    actor      TEXT NOT NULL,
    action     TEXT NOT NULL,
    resource   TEXT NOT NULL,
    metadata   JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX audit_log_created_at_idx ON audit_log (created_at DESC);

-- +goose Down
DROP TABLE audit_log;
DROP TABLE installations;
DROP TABLE licenses;
DROP TABLE customers;
DROP TABLE admin_users;
