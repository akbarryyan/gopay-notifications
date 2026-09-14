package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/akbarryyan/gopay-notifications/backend/internal/secretbox"
	"github.com/jackc/pgx/v5"
)

// ErrDeviceNotFound dikembalikan bila device_id tidak terdaftar.
var ErrDeviceNotFound = errors.New("store: device tidak ditemukan")

type Device struct {
	DeviceID  string
	AccountID string
	Name      string
	Secret    []byte
	Enabled   bool
	CreatedAt time.Time

	LastSeenAt *time.Time
	// Versi aplikasi Android yang terakhir menghubungi backend. Kosong
	// sampai perangkat mengirim header X-App-Version.
	AppVersion *string

	// Diisi dari heartbeat berkala. Nil berarti perangkat belum pernah
	// mengirim heartbeat sama sekali.
	HeartbeatAt       *time.Time
	AndroidVersion    *string
	ListenerConnected *bool
	PendingCount      *int
	FailedCount       *int
}

// CreateDevice menyimpan device baru dengan secret terenkripsi, milik satu account.
func (s *Store) CreateDevice(ctx context.Context, key []byte, accountID, deviceID, name string, secret []byte) error {
	enc, err := secretbox.Seal(key, secret)
	if err != nil {
		return fmt.Errorf("store: enkripsi secret: %w", err)
	}

	_, err = s.pool.Exec(ctx,
		`INSERT INTO devices (device_id, account_id, name, secret_enc) VALUES ($1, $2, $3, $4)`,
		deviceID, accountID, name, enc)
	if err != nil {
		return fmt.Errorf("store: insert device: %w", err)
	}
	return nil
}

// GetDevice mengambil device beserta secret yang sudah didekripsi. device_id
// tetap unik global (bukan di-scope per account) -- ID acak, pemanggil
// (requireDevice) yang membaca AccountID dari hasilnya buat tahu pemiliknya.
func (s *Store) GetDevice(ctx context.Context, key []byte, deviceID string) (Device, error) {
	var (
		d   Device
		enc []byte
	)
	err := s.pool.QueryRow(ctx,
		`SELECT device_id, account_id, name, secret_enc, enabled, last_seen_at, app_version,
		        heartbeat_at, android_version, listener_connected,
		        pending_count, failed_count
		 FROM devices WHERE device_id = $1`, deviceID).
		Scan(&d.DeviceID, &d.AccountID, &d.Name, &enc, &d.Enabled, &d.LastSeenAt, &d.AppVersion,
			&d.HeartbeatAt, &d.AndroidVersion, &d.ListenerConnected,
			&d.PendingCount, &d.FailedCount)

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

// ListDevices mengembalikan seluruh device untuk dashboard admin.
//
// Secret SENGAJA tidak diambil sama sekali — dashboard tidak pernah butuh
// melihatnya, dan tidak menyentuh secretbox di sini berarti tidak ada
// jalan bagi endpoint admin untuk kebocoran mendekripsi secret secara
// tidak sengaja.
func (s *Store) ListDevices(ctx context.Context, accountID string) ([]Device, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT device_id, account_id, name, enabled, created_at, last_seen_at, app_version,
		        heartbeat_at, android_version, listener_connected,
		        pending_count, failed_count
		 FROM devices
		 WHERE account_id = $1
		 ORDER BY created_at DESC`, accountID)
	if err != nil {
		return nil, fmt.Errorf("store: list devices: %w", err)
	}
	defer rows.Close()

	var out []Device
	for rows.Next() {
		var d Device
		if err := rows.Scan(&d.DeviceID, &d.AccountID, &d.Name, &d.Enabled, &d.CreatedAt, &d.LastSeenAt,
			&d.AppVersion, &d.HeartbeatAt, &d.AndroidVersion, &d.ListenerConnected,
			&d.PendingCount, &d.FailedCount); err != nil {
			return nil, fmt.Errorf("store: scan device: %w", err)
		}
		out = append(out, d)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterasi devices: %w", err)
	}
	return out, nil
}

// ErrDeviceHasEvents dikembalikan DeleteDevice bila device ini pernah
// mengirim minimal satu event -- riwayat notification_events adalah jejak
// pembayaran, tidak boleh diam-diam ikut hilang gara-gara device-nya
// dihapus (notification_events.device_id punya foreign key ke devices
// TANPA ON DELETE CASCADE, sengaja begitu sejak migrasi awal).
var ErrDeviceHasEvents = errors.New("store: device punya riwayat event, nonaktifkan saja")

// DeleteDevice menghapus device permanen -- cuma boleh untuk device yang
// belum pernah mengirim event sama sekali (baru dibuat lalu tidak jadi
// dipakai, atau salah bikin). Device yang sudah punya riwayat wajib
// dinonaktifkan (SetDeviceEnabled), bukan dihapus, supaya jejak
// pembayarannya tetap utuh.
func (s *Store) DeleteDevice(ctx context.Context, accountID, deviceID string) error {
	var hasEvents bool
	if err := s.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM notification_events WHERE device_id = $1)`, deviceID).
		Scan(&hasEvents); err != nil {
		return fmt.Errorf("store: cek riwayat event device: %w", err)
	}
	if hasEvents {
		return ErrDeviceHasEvents
	}

	tag, err := s.pool.Exec(ctx,
		`DELETE FROM devices WHERE device_id = $1 AND account_id = $2`, deviceID, accountID)
	if err != nil {
		return fmt.Errorf("store: delete device: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrDeviceNotFound
	}
	return nil
}

// SetDeviceEnabled mengaktifkan atau menonaktifkan device dari dashboard.
// Mengembalikan ErrDeviceNotFound juga kalau device ada tapi milik account
// lain -- pemanggil tidak boleh bisa membedakan "tidak ada" dari "bukan
// milikmu" lewat error yang berbeda.
func (s *Store) SetDeviceEnabled(ctx context.Context, accountID, deviceID string, enabled bool) error {
	tag, err := s.pool.Exec(ctx,
		`UPDATE devices SET enabled = $3 WHERE device_id = $1 AND account_id = $2`,
		deviceID, accountID, enabled)
	if err != nil {
		return fmt.Errorf("store: set device enabled: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrDeviceNotFound
	}
	return nil
}

// TouchDevice memperbarui last_seen_at dan versi aplikasi. Dipanggil setelah
// autentikasi berhasil.
//
// appVersion kosong tidak menimpa nilai yang sudah tersimpan: aplikasi versi
// lama tidak mengirim header itu, dan menghapus versi yang sudah diketahui
// justru membuang informasi.
func (s *Store) TouchDevice(ctx context.Context, deviceID, appVersion string) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE devices
		 SET last_seen_at = now(),
		     app_version  = COALESCE(NULLIF($2, ''), app_version)
		 WHERE device_id = $1`, deviceID, appVersion)
	if err != nil {
		return fmt.Errorf("store: touch device: %w", err)
	}
	return nil
}
