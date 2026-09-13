// Package config membaca konfigurasi layanan dari environment variable.
package config

import (
	"encoding/base64"
	"errors"
	"fmt"
	"os"

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

	return c, nil
}
