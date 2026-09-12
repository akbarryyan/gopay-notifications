-- +goose Up
-- Satu instalasi self-hosted melayani satu merchant, sehingga satu akun admin
-- sudah cukup untuk MVP. Tabel dipakai (bukan env var) supaya password dapat
-- diganti tanpa redeploy, lewat cmd/admintool.
CREATE TABLE admin_users (
    id            BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    username      TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE admin_users;
