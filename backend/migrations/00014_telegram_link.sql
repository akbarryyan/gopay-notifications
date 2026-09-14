-- +goose Up
-- Menghubungkan Telegram customer lewat tombol "Hubungkan Telegram"
-- (deep link t.me/<bot>?start=<kode>), menggantikan isian chat id manual.
--
-- Chat id manual tidak pernah benar-benar berfungsi: bot Telegram DILARANG
-- memulai obrolan dengan orang yang belum pernah menekan Start di bot itu,
-- jadi setiap kiriman ke chat id yang diketik tangan gagal "chat not found".
-- Lewat deep link, customer pasti sudah menekan Start -- dan chat id-nya
-- diambil dari pesan itu sendiri, bukan diketik.

-- Kode sekali pakai, disimpan sebagai SHA-256 (pola sama dengan
-- password_reset_tokens): siapa pun yang memegang kode yang masih berlaku
-- bisa mengarahkan notifikasi account itu ke Telegram miliknya.
CREATE TABLE telegram_link_codes (
    id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    account_id TEXT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    code_hash  BYTEA NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at    TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX telegram_link_codes_account_idx ON telegram_link_codes (account_id);

-- Posisi terakhir getUpdates yang sudah diproses. Disimpan di database,
-- bukan di memori: tanpa ini, setiap restart backend memproses ulang pesan
-- /start 24 jam terakhir dan membalas "link tidak berlaku" ke orang yang
-- sebenarnya sudah berhasil terhubung. Direset ke 0 saat token bot diganti
-- (update_id berurutan per bot, bukan global).
ALTER TABLE notification_settings ADD COLUMN telegram_update_offset BIGINT NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE notification_settings DROP COLUMN telegram_update_offset;
DROP TABLE telegram_link_codes;
