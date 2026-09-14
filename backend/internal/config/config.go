// Package config membaca konfigurasi layanan dari environment variable.
package config

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/akbarryyan/gopay-notifications/backend/internal/secretbox"
)

type Config struct {
	DatabaseURL     string
	ListenAddr      string
	DeviceSecretKey []byte
	// Kunci penanda tangan sesi dashboard admin. Terpisah dari
	// DeviceSecretKey — keduanya melindungi hal yang berbeda, dan kompromi
	// pada satu tidak boleh ikut membongkar yang lain.
	AdminSessionKey []byte
	// Kunci enkripsi secret webhook (secretbox). Terpisah lagi dari dua
	// kunci di atas — tiga kunci, tiga tujuan, jangan dipakai ulang.
	WebhookSecretKey []byte
	// VendorSessionKey menandatangani cookie sesi Vendor Dashboard
	// (vendor_session) -- terpisah total dari AdminSessionKey (sesi
	// customer, admin_session), supaya dua jenis sesi ini tidak mungkin
	// tertukar walau tersimpan di browser yang sama.
	VendorSessionKey []byte
	// SettingsSecretKey mengenkripsi kredensial yang diatur vendor lewat
	// Vendor Dashboard dan disimpan di database (password SMTP, token bot
	// Telegram -- tabel notification_settings). Kunci kelima, tujuan
	// kelima: jangan dipakai ulang dengan WebhookSecretKey walau sama-sama
	// "mengenkripsi kredensial keluar" -- merotasi kunci webhook tidak boleh
	// ikut membuat password SMTP tidak bisa didekripsi, dan sebaliknya.
	SettingsSecretKey []byte
	// DashboardURL adalah alamat publik Customer Dashboard (tanpa "/" di
	// akhir), dipakai menyusun link di email reset password. Opsional:
	// kosong berarti fitur lupa password menjawab "belum tersedia".
	//
	// SENGAJA dari konfigurasi, tidak pernah dari header Host/Origin
	// request: kalau diambil dari request, penyerang bisa meminta reset
	// untuk email korban dengan Host palsu, dan korban menerima link asli
	// berisi token yang mengarah ke domain penyerang.
	DashboardURL string
}

// Load membaca dan memvalidasi seluruh konfigurasi. Konfigurasi yang salah
// harus menghentikan proses saat start, bukan saat request pertama masuk.
func Load() (Config, error) {
	c := Config{
		DatabaseURL: os.Getenv("DATABASE_URL"),
		ListenAddr:  os.Getenv("LISTEN_ADDR"),
	}
	if c.DatabaseURL == "" {
		return Config{}, errors.New("config: DATABASE_URL wajib diisi")
	}
	if c.ListenAddr == "" {
		c.ListenAddr = ":8080"
	}

	raw := os.Getenv("DEVICE_SECRET_KEY")
	if raw == "" {
		return Config{}, errors.New("config: DEVICE_SECRET_KEY wajib diisi")
	}
	key, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return Config{}, fmt.Errorf("config: DEVICE_SECRET_KEY bukan base64 yang sah: %w", err)
	}
	if len(key) != secretbox.KeySize {
		return Config{}, fmt.Errorf("config: DEVICE_SECRET_KEY harus %d byte setelah decode, dapat %d",
			secretbox.KeySize, len(key))
	}
	c.DeviceSecretKey = key

	rawSession := os.Getenv("ADMIN_SESSION_KEY")
	if rawSession == "" {
		return Config{}, errors.New("config: ADMIN_SESSION_KEY wajib diisi")
	}
	sessionKey, err := base64.StdEncoding.DecodeString(rawSession)
	if err != nil {
		return Config{}, fmt.Errorf("config: ADMIN_SESSION_KEY bukan base64 yang sah: %w", err)
	}
	if len(sessionKey) != secretbox.KeySize {
		return Config{}, fmt.Errorf("config: ADMIN_SESSION_KEY harus %d byte setelah decode, dapat %d",
			secretbox.KeySize, len(sessionKey))
	}
	c.AdminSessionKey = sessionKey

	rawWebhook := os.Getenv("WEBHOOK_SECRET_KEY")
	if rawWebhook == "" {
		return Config{}, errors.New("config: WEBHOOK_SECRET_KEY wajib diisi")
	}
	webhookKey, err := base64.StdEncoding.DecodeString(rawWebhook)
	if err != nil {
		return Config{}, fmt.Errorf("config: WEBHOOK_SECRET_KEY bukan base64 yang sah: %w", err)
	}
	if len(webhookKey) != secretbox.KeySize {
		return Config{}, fmt.Errorf("config: WEBHOOK_SECRET_KEY harus %d byte setelah decode, dapat %d",
			secretbox.KeySize, len(webhookKey))
	}
	c.WebhookSecretKey = webhookKey

	rawVendor := os.Getenv("VENDOR_SESSION_KEY")
	if rawVendor == "" {
		return Config{}, errors.New("config: VENDOR_SESSION_KEY wajib diisi")
	}
	vendorKey, err := base64.StdEncoding.DecodeString(rawVendor)
	if err != nil {
		return Config{}, fmt.Errorf("config: VENDOR_SESSION_KEY bukan base64 yang sah: %w", err)
	}
	if len(vendorKey) != secretbox.KeySize {
		return Config{}, fmt.Errorf("config: VENDOR_SESSION_KEY harus %d byte setelah decode, dapat %d",
			secretbox.KeySize, len(vendorKey))
	}
	c.VendorSessionKey = vendorKey

	rawSettings := os.Getenv("SETTINGS_SECRET_KEY")
	if rawSettings == "" {
		return Config{}, errors.New("config: SETTINGS_SECRET_KEY wajib diisi")
	}
	settingsKey, err := base64.StdEncoding.DecodeString(rawSettings)
	if err != nil {
		return Config{}, fmt.Errorf("config: SETTINGS_SECRET_KEY bukan base64 yang sah: %w", err)
	}
	if len(settingsKey) != secretbox.KeySize {
		return Config{}, fmt.Errorf("config: SETTINGS_SECRET_KEY harus %d byte setelah decode, dapat %d",
			secretbox.KeySize, len(settingsKey))
	}
	c.SettingsSecretKey = settingsKey

	if raw := strings.TrimRight(strings.TrimSpace(os.Getenv("DASHBOARD_URL")), "/"); raw != "" {
		u, err := url.Parse(raw)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			return Config{}, fmt.Errorf("config: DASHBOARD_URL harus URL http(s) lengkap, mis. https://whuzpay.com, dapat %q", raw)
		}
		c.DashboardURL = raw
	}

	return c, nil
}
