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

	return c, nil
}
