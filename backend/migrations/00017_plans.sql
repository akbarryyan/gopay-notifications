-- +goose Up
-- Paket (plan) jadi data, bukan lagi peta hardcode di kode Go
-- (planPresets di vendor_accounts.go) -- vendor bisa menambah, mengubah,
-- menyembunyikan, dan mengurutkan ulang paket dari Vendor Dashboard,
-- termasuk teks yang tampil di section Harga landing page publik.
--
-- accounts.plan TETAP kolom teks bebas (bukan foreign key ke sini) --
-- account yang sudah dibuat sengaja TIDAK ikut berubah kalau nanti paket
-- yang namanya sama diedit/dihapus. max_devices sudah disalin ke kolom
-- accounts.max_devices milik account itu sendiri saat dibuat/diganti, jadi
-- kuota device yang sudah berjalan tidak pernah bergantung pada baris di
-- tabel ini lagi setelah momen itu.
CREATE TABLE plans (
    id           TEXT PRIMARY KEY,
    name         TEXT NOT NULL UNIQUE,
    -- -1 berarti unlimited, konvensi yang sama dengan accounts.max_devices.
    max_devices  INTEGER NOT NULL,
    -- Kosong berarti belum diisi vendor -- section Harga TIDAK menampilkan
    -- baris harga sama sekali untuk paket itu (bukan "Rp 0" atau angka
    -- karangan), supaya tampilan publik tidak pernah menampilkan nominal
    -- yang belum pernah benar-benar ditentukan vendor.
    price_label  TEXT NOT NULL DEFAULT '',
    price_period TEXT NOT NULL DEFAULT '',
    description  TEXT NOT NULL DEFAULT '',
    features     JSONB NOT NULL DEFAULT '[]'::jsonb,
    highlighted  BOOLEAN NOT NULL DEFAULT FALSE,
    -- false = tetap ada untuk dipakai vendor membuat/mengubah account,
    -- tapi disembunyikan dari section Harga publik (mis. paket lama/custom
    -- yang tidak lagi dijual ke umum).
    visible      BOOLEAN NOT NULL DEFAULT TRUE,
    sort_order   INTEGER NOT NULL DEFAULT 0,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX plans_sort_idx ON plans (sort_order, name);

-- Seed persis dari planPresets + teks yang SUDAH tampil di
-- pricing-faq-footer.tsx apa adanya -- migrasi ini cuma memindahkan
-- sumbernya dari kode ke database, sengaja TIDAK mengubah tampilan publik
-- sedikit pun (price_label kosong, sama seperti sebelumnya section Harga
-- tidak pernah menampilkan nominal rupiah).
INSERT INTO plans (id, name, max_devices, description, features, highlighted, sort_order) VALUES
  ('plan_starter', 'Starter', 3, 'Untuk usaha kecil yang baru mulai.',
   '["Webhook & retry", "Dashboard realtime", "Konsol pengecualian"]'::jsonb, FALSE, 1),
  ('plan_business', 'Business', 10, 'Untuk operasional yang sedang berkembang.',
   '["Webhook & retry", "Dashboard realtime", "Konsol pengecualian"]'::jsonb, TRUE, 2),
  ('plan_enterprise', 'Enterprise', -1, 'Untuk kebutuhan volume tinggi.',
   '["Webhook & retry", "Dashboard realtime", "Konsol pengecualian"]'::jsonb, FALSE, 3);

-- +goose Down
DROP TABLE plans;
