-- +goose Up
-- Sub-project 3 fase 1: invoice + nominal unik + matching, lihat
-- docs/superpowers/specs/2026-09-12-invoice-nominal-matching-design.md.
CREATE TABLE invoices (
    id               TEXT PRIMARY KEY,
    external_ref     TEXT NOT NULL,
    requested_amount BIGINT NOT NULL,
    unique_amount    BIGINT NOT NULL,
    status           TEXT NOT NULL DEFAULT 'PENDING',
    matched_event_id TEXT NULL REFERENCES notification_events(event_id),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at       TIMESTAMPTZ NOT NULL,
    paid_at          TIMESTAMPTZ NULL
);

CREATE UNIQUE INDEX invoices_external_ref_idx ON invoices (external_ref);

-- Partial unique index — inilah yang mencegah dua invoice PENDING berbagi
-- nominal yang sama di saat bersamaan. Predicate index harus immutable,
-- makanya status kedaluwarsa harus benar-benar dituliskan (bukan dihitung
-- saat baca seperti DeviceStatus) sebelum alokasi/matching berikutnya.
CREATE UNIQUE INDEX invoices_pending_unique_amount_idx
    ON invoices (unique_amount) WHERE status = 'PENDING';

CREATE INDEX invoices_created_at_idx ON invoices (created_at DESC);

-- API key merchant untuk memanggil POST/GET /invoices — tabel sungguhan
-- (bukan satu key statis di .env) supaya bisa banyak key bernama dan
-- dicabut satu per satu lewat dashboard.
CREATE TABLE api_keys (
    id         TEXT PRIMARY KEY,
    name       TEXT NOT NULL,
    key_hash   BYTEA NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    revoked_at TIMESTAMPTZ NULL
);

CREATE UNIQUE INDEX api_keys_key_hash_idx ON api_keys (key_hash);

-- +goose Down
DROP TABLE api_keys;
DROP TABLE invoices;
