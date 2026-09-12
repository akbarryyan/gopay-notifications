package store

import (
	"context"
	"fmt"
	"time"
)

// Heartbeat adalah laporan kondisi yang dikirim perangkat secara berkala.
type Heartbeat struct {
	AndroidVersion    string
	ListenerConnected bool
	PendingCount      int
	FailedCount       int
}

// DeviceStatus adalah status yang ditampilkan dashboard.
type DeviceStatus string

const (
	// DeviceOnline: heartbeat terakhir masih dalam batas toleransi.
	DeviceOnline DeviceStatus = "ONLINE"
	// DeviceOffline: tidak ada heartbeat dalam batas toleransi.
	DeviceOffline DeviceStatus = "OFFLINE"
	// DeviceDisabled: dinonaktifkan dari backend.
	DeviceDisabled DeviceStatus = "DISABLED"
	// DevicePending: terdaftar tetapi belum pernah mengirim heartbeat.
	DevicePending DeviceStatus = "PENDING"
)

// HeartbeatInterval adalah jarak pengiriman heartbeat dari perangkat.
//
// 15 menit karena itu interval minimum WorkManager untuk periodic work.
const HeartbeatInterval = 15 * time.Minute

// HeartbeatTolerance adalah batas sebelum perangkat dianggap OFFLINE.
//
// Tiga kali interval, bukan satu. Android menunda periodic work saat Doze,
// dan menandai perangkat OFFLINE karena satu heartbeat tertunda akan
// menghasilkan alarm palsu yang membuat status ini diabaikan orang.
const HeartbeatTolerance = 3 * HeartbeatInterval

// StatusOf menurunkan status perangkat. Fungsi murni supaya dapat diuji tanpa
// database maupun waktu nyata.
func StatusOf(d Device, now time.Time) DeviceStatus {
	if !d.Enabled {
		return DeviceDisabled
	}
	if d.HeartbeatAt == nil {
		return DevicePending
	}
	if now.Sub(*d.HeartbeatAt) <= HeartbeatTolerance {
		return DeviceOnline
	}
	return DeviceOffline
}

// RecordHeartbeat menyimpan laporan kondisi perangkat.
func (s *Store) RecordHeartbeat(ctx context.Context, deviceID string, hb Heartbeat) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE devices
		 SET heartbeat_at       = now(),
		     last_seen_at       = now(),
		     android_version    = COALESCE(NULLIF($2, ''), android_version),
		     listener_connected = $3,
		     pending_count      = $4,
		     failed_count       = $5
		 WHERE device_id = $1`,
		deviceID, hb.AndroidVersion, hb.ListenerConnected, hb.PendingCount, hb.FailedCount)
	if err != nil {
		return fmt.Errorf("store: rekam heartbeat: %w", err)
	}
	return nil
}
