-- +goose Up
-- Sub-project 3 fase 2: webhook delivery, lihat
-- docs/superpowers/specs/2026-09-13-webhook-delivery-design.md.
CREATE TABLE webhook_endpoints (
    id               TEXT PRIMARY KEY,
    name             TEXT NOT NULL,
    url              TEXT NOT NULL,
    -- Dienkripsi (secretbox), BUKAN di-hash seperti api_keys — pengiriman
    -- webhook butuh secret mentahnya lagi tiap kali menandatangani payload
    -- keluar, hash satu-arah tidak bisa dipakai di sini.
    secret_encrypted BYTEA NOT NULL,
    events           TEXT[] NOT NULL,
    enabled          BOOLEAN NOT NULL DEFAULT true,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE webhook_deliveries (
    id              TEXT PRIMARY KEY,
    endpoint_id     TEXT NOT NULL REFERENCES webhook_endpoints(id) ON DELETE CASCADE,
    event           TEXT NOT NULL,
    invoice_id      TEXT NULL REFERENCES invoices(id),
    payload         JSONB NOT NULL,
    status          TEXT NOT NULL DEFAULT 'PENDING',
    attempt         INT NOT NULL DEFAULT 0,
    next_attempt_at TIMESTAMPTZ NULL,
    http_status     INT NULL,
    duration_ms     INT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    delivered_at    TIMESTAMPTZ NULL
);

-- Dipakai worker berkala mencari delivery yang jatuh tempo — hanya
-- PENDING/RETRYING yang pernah relevan, FAILED/DELIVERED final.
CREATE INDEX webhook_deliveries_due_idx
    ON webhook_deliveries (next_attempt_at)
    WHERE status IN ('PENDING', 'RETRYING');

CREATE INDEX webhook_deliveries_endpoint_created_idx
    ON webhook_deliveries (endpoint_id, created_at DESC);

-- +goose Down
DROP TABLE webhook_deliveries;
DROP TABLE webhook_endpoints;
