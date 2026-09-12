-- +goose Up
-- Heartbeat device.
--
-- Sebelum ini, last_seen_at hanya bergerak saat ada pembayaran. Toko dengan
-- tiga transaksi sehari akan tampak mati hampir sepanjang waktu, dan HP yang
-- benar-benar dibunuh OEM baru diketahui saat pembayaran berikutnya terlewat.
-- Untuk produk berbayar, menunggu pembayaran untuk tahu HP-nya mati tidak
-- dapat diterima.
ALTER TABLE devices ADD COLUMN heartbeat_at       TIMESTAMPTZ;
ALTER TABLE devices ADD COLUMN android_version    TEXT;
-- Status ikatan NotificationListenerService, bukan status izin. Keduanya bisa
-- berbeda, dan justru selisihnya yang menandakan service dibunuh diam-diam.
ALTER TABLE devices ADD COLUMN listener_connected BOOLEAN;
ALTER TABLE devices ADD COLUMN pending_count      INTEGER;
ALTER TABLE devices ADD COLUMN failed_count       INTEGER;

-- +goose Down
ALTER TABLE devices DROP COLUMN failed_count;
ALTER TABLE devices DROP COLUMN pending_count;
ALTER TABLE devices DROP COLUMN listener_connected;
ALTER TABLE devices DROP COLUMN android_version;
ALTER TABLE devices DROP COLUMN heartbeat_at;
