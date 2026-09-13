-- +goose Up
-- Audit log aksi vendor terhadap accounts -- pindah dari License Server
-- (dibongkar) ke database utama. Lihat
-- docs/superpowers/specs/2026-09-13-multitenant-accounts-design.md §4.
CREATE TABLE audit_log (
    id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    actor      TEXT NOT NULL,
    action     TEXT NOT NULL,
    resource   TEXT NOT NULL,
    metadata   JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX audit_log_created_at_idx ON audit_log (created_at DESC);

-- +goose Down
DROP TABLE audit_log;
