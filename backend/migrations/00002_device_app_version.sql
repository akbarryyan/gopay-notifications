-- +goose Up
-- Versi aplikasi Android per device.
--
-- Tanpa ini backend tidak tahu versi aplikasi yang mengiriminya, sehingga
-- backend baru dapat diam-diam merusak aplikasi lama dan kita baru tahu dari
-- keluhan pelanggan.
ALTER TABLE devices ADD COLUMN app_version TEXT;

-- +goose Down
ALTER TABLE devices DROP COLUMN app_version;
