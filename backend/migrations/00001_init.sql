-- +goose Up
CREATE TABLE devices (
    device_id    TEXT PRIMARY KEY,
    name         TEXT NOT NULL,
    secret_enc   BYTEA NOT NULL,
    enabled      BOOLEAN NOT NULL DEFAULT TRUE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_seen_at TIMESTAMPTZ
);

CREATE TABLE notification_events (
    id           BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    event_id     TEXT NOT NULL UNIQUE,
    device_id    TEXT NOT NULL REFERENCES devices(device_id),
    source       TEXT NOT NULL,
    package_name TEXT NOT NULL,
    title        TEXT,
    body_text    TEXT,
    big_text     TEXT,
    amount_hint  BIGINT,
    posted_at    TIMESTAMPTZ NOT NULL,
    received_at  TIMESTAMPTZ NOT NULL,
    ingested_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    raw_payload  JSONB NOT NULL
);

CREATE INDEX notification_events_ingested_at_idx
    ON notification_events (ingested_at DESC);

-- +goose Down
DROP TABLE notification_events;
DROP TABLE devices;
