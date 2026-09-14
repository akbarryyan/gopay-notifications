package store

import (
	"context"
	"fmt"
	"time"
)

// OfflineAlertWindow membatasi seberapa lama HP boleh sudah offline untuk
// masih diberi peringatan. HP yang heartbeat terakhirnya lebih tua dari
// ini dilewati.
//
// Tanpa batas ini, begitu fitur ini pertama kali aktif (atau SMTP baru
// diisi di Vendor Dashboard), setiap HP lama yang sudah berminggu-minggu
// tidak dipakai langsung memicu email "HP kamu offline" sekaligus --
// peringatan untuk kejadian basi yang tidak lagi bisa ditindaklanjuti.
const OfflineAlertWindow = 24 * time.Hour

// DeviceAlert adalah device yang perlu dikabarkan statusnya ke pemiliknya,
// beserta kontak account-nya.
type DeviceAlert struct {
	DeviceID       string
	DeviceName     string
	AccountID      string
	BusinessName   string
	Email          string
	TelegramChatID *string
	HeartbeatAt    time.Time
	// OfflineSince adalah heartbeat_at saat peringatan offline dikirim
	// (offline_alert_for) -- nol untuk device yang belum pernah diberi
	// peringatan.
	OfflineSince time.Time
}

// deviceAlertQuery dipakai dua query di bawah. Account harus masih
// operasional (aktif dan belum kedaluwarsa): account yang sudah berhenti
// memang tidak lagi dilayani, HP-nya offline pun wajar.
const deviceAlertQuery = `SELECT d.device_id, d.name, a.id, a.business_name, a.email, a.telegram_chat_id, d.heartbeat_at,
	       COALESCE(d.offline_alert_for, 'epoch'::timestamptz)
	FROM devices d
	JOIN accounts a ON a.id = d.account_id
	WHERE d.enabled
	  AND a.admin_status = 'active'
	  AND a.expires_at > $1
	  AND d.heartbeat_at IS NOT NULL`

// DevicesNeedingOfflineAlert: heartbeat terakhir sudah melewati toleransi
// (status OFFLINE persis seperti StatusOf), masih dalam OfflineAlertWindow,
// dan peringatan untuk NILAI heartbeat_at itu belum dikirim.
func (s *Store) DevicesNeedingOfflineAlert(ctx context.Context, now time.Time) ([]DeviceAlert, error) {
	return s.queryDeviceAlerts(ctx, deviceAlertQuery+`
	  AND d.heartbeat_at < $2
	  AND d.heartbeat_at >= $3
	  AND d.offline_alert_for IS DISTINCT FROM d.heartbeat_at
	ORDER BY d.heartbeat_at ASC`,
		now, now.Add(-HeartbeatTolerance), now.Add(-OfflineAlertWindow))
}

// DevicesRecoveredFromOffline: sudah pernah diberi peringatan offline, dan
// sejak itu heartbeat baru masuk sehingga kembali ONLINE.
func (s *Store) DevicesRecoveredFromOffline(ctx context.Context, now time.Time) ([]DeviceAlert, error) {
	return s.queryDeviceAlerts(ctx, deviceAlertQuery+`
	  AND d.offline_alert_for IS NOT NULL
	  AND d.heartbeat_at > d.offline_alert_for
	  AND d.heartbeat_at >= $2
	ORDER BY d.heartbeat_at ASC`,
		now, now.Add(-HeartbeatTolerance))
}

func (s *Store) queryDeviceAlerts(ctx context.Context, query string, args ...any) ([]DeviceAlert, error) {
	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("store: query device alert: %w", err)
	}
	defer rows.Close()

	out := make([]DeviceAlert, 0)
	for rows.Next() {
		var d DeviceAlert
		if err := rows.Scan(&d.DeviceID, &d.DeviceName, &d.AccountID, &d.BusinessName, &d.Email,
			&d.TelegramChatID, &d.HeartbeatAt, &d.OfflineSince); err != nil {
			return nil, fmt.Errorf("store: scan device alert: %w", err)
		}
		out = append(out, d)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterasi device alert: %w", err)
	}
	return out, nil
}

// MarkDeviceOfflineAlerted menandai peringatan offline sudah dikirim untuk
// heartbeat_at tertentu.
func (s *Store) MarkDeviceOfflineAlerted(ctx context.Context, deviceID string, heartbeatAt time.Time) error {
	if _, err := s.pool.Exec(ctx,
		`UPDATE devices SET offline_alert_for = $2 WHERE device_id = $1`, deviceID, heartbeatAt); err != nil {
		return fmt.Errorf("store: tandai peringatan offline: %w", err)
	}
	return nil
}

// ClearDeviceOfflineAlert dipanggil setelah pemberitahuan "kembali online"
// terkirim, membuka peringatan untuk offline berikutnya.
func (s *Store) ClearDeviceOfflineAlert(ctx context.Context, deviceID string) error {
	if _, err := s.pool.Exec(ctx,
		`UPDATE devices SET offline_alert_for = NULL WHERE device_id = $1`, deviceID); err != nil {
		return fmt.Errorf("store: hapus tanda peringatan offline: %w", err)
	}
	return nil
}
