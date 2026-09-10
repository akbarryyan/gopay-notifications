package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/akbar/gopay-notifications/backend/internal/secretbox"
	"github.com/jackc/pgx/v5"
)

// ErrDeviceNotFound dikembalikan bila device_id tidak terdaftar.
var ErrDeviceNotFound = errors.New("store: device tidak ditemukan")

type Device struct {
	DeviceID   string
	Name       string
	Secret     []byte
	Enabled    bool
	LastSeenAt *time.Time
}

// CreateDevice menyimpan device baru dengan secret terenkripsi.
func (s *Store) CreateDevice(ctx context.Context, key []byte, deviceID, name string, secret []byte) error {
	enc, err := secretbox.Seal(key, secret)
	if err != nil {
		return fmt.Errorf("store: enkripsi secret: %w", err)
	}

	_, err = s.pool.Exec(ctx,
		`INSERT INTO devices (device_id, name, secret_enc) VALUES ($1, $2, $3)`,
		deviceID, name, enc)
	if err != nil {
		return fmt.Errorf("store: insert device: %w", err)
	}
	return nil
}

// GetDevice mengambil device beserta secret yang sudah didekripsi.
func (s *Store) GetDevice(ctx context.Context, key []byte, deviceID string) (Device, error) {
	var (
		d   Device
		enc []byte
	)
	err := s.pool.QueryRow(ctx,
		`SELECT device_id, name, secret_enc, enabled, last_seen_at
		 FROM devices WHERE device_id = $1`, deviceID).
		Scan(&d.DeviceID, &d.Name, &enc, &d.Enabled, &d.LastSeenAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return Device{}, ErrDeviceNotFound
	}
	if err != nil {
		return Device{}, fmt.Errorf("store: select device: %w", err)
	}

	secret, err := secretbox.Open(key, enc)
	if err != nil {
		return Device{}, fmt.Errorf("store: dekripsi secret device %s: %w", deviceID, err)
	}
	d.Secret = secret
	return d, nil
}

// TouchDevice memperbarui last_seen_at. Dipanggil setelah autentikasi berhasil,
// sehingga tidak dibutuhkan heartbeat berkala dari perangkat.
func (s *Store) TouchDevice(ctx context.Context, deviceID string) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE devices SET last_seen_at = now() WHERE device_id = $1`, deviceID)
	if err != nil {
		return fmt.Errorf("store: touch device: %w", err)
	}
	return nil
}
