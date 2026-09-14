-- +goose Up
-- Pengingat "akun mendekati kedaluwarsa" yang dikirim ke customer.
--
-- telegram_chat_id OPSIONAL: email selalu ada (kolom wajib sejak
-- 00008_accounts.sql), sedangkan Telegram cuma terisi kalau customer
-- memang punya Telegram dan mau mengisinya sendiri di dashboard mereka.
-- Jadi email adalah jalur utama, Telegram tambahan.
ALTER TABLE accounts ADD COLUMN telegram_chat_id TEXT NULL;

-- expiry_reminder_sent_for menyimpan NILAI expires_at yang pengingatnya
-- sudah dikirim -- bukan sekadar timestamp "kapan terakhir kirim".
-- Alasannya: begitu akun diperpanjang, expires_at berubah, dan kolom ini
-- otomatis tidak lagi cocok sehingga pengingat periode berikutnya boleh
-- dikirim lagi. Kalau cuma menyimpan "kapan terakhir kirim", perpanjangan
-- tidak akan pernah mereset apa pun dan pengingat berikutnya tidak pernah
-- terkirim.
ALTER TABLE accounts ADD COLUMN expiry_reminder_sent_for TIMESTAMPTZ NULL;

-- +goose Down
ALTER TABLE accounts DROP COLUMN expiry_reminder_sent_for;
ALTER TABLE accounts DROP COLUMN telegram_chat_id;
