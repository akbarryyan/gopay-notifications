-- +goose Up
-- Sub-project 3 fase 4: konsol pengecualian, lihat
-- docs/superpowers/specs/2026-09-13-exception-console-design.md.
CREATE TABLE event_reviews (
    event_id     TEXT PRIMARY KEY REFERENCES notification_events(event_id),
    dismissed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    note         TEXT NULL
);

-- Tanpa ini, jalur pencocokan MANUAL (beda dari matching otomatis fase 1
-- yang cuma menyentuh nominal yang memang unik) bisa mereferensikan event
-- yang sama ke dua invoice berbeda — satu pembayaran dobel terhitung.
CREATE UNIQUE INDEX invoices_matched_event_id_idx
    ON invoices (matched_event_id) WHERE matched_event_id IS NOT NULL;

-- +goose Down
DROP INDEX invoices_matched_event_id_idx;
DROP TABLE event_reviews;
