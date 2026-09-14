package store

import (
	"context"
	"fmt"
	"time"
)

// Jenis notifikasi di notification_log.kind.
const (
	NotificationKindExpiryReminder = "expiry_reminder"
	NotificationKindDeviceOffline  = "device_offline"
	NotificationKindDeviceOnline   = "device_online"
	NotificationKindTest           = "test"
)

// Status di notification_log.status.
const (
	NotificationStatusSent   = "sent"
	NotificationStatusFailed = "failed"
)

// NotificationLogEntry adalah SATU percobaan kirim ke SATU channel --
// pengingat yang dikirim lewat email dan Telegram tercatat dua baris,
// supaya "email sampai, Telegram gagal" terlihat apa adanya.
type NotificationLogEntry struct {
	AccountID  *string
	DeviceID   *string
	DeviceName *string
	Kind       string
	Channel    string // "email" | "telegram"
	Recipient  string
	Subject    string
	Status     string
	Error      *string
}

// LogNotification mencatat satu percobaan kirim.
func (s *Store) LogNotification(ctx context.Context, e NotificationLogEntry) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO notification_log
		   (account_id, device_id, device_name, kind, channel, recipient, subject, status, error)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		e.AccountID, e.DeviceID, e.DeviceName, e.Kind, e.Channel, e.Recipient, e.Subject, e.Status, e.Error)
	if err != nil {
		return fmt.Errorf("store: catat notifikasi: %w", err)
	}
	return nil
}

// NotificationLogRow adalah satu baris riwayat untuk Vendor Dashboard.
// BusinessName nil untuk pesan uji (tidak milik account mana pun).
type NotificationLogRow struct {
	ID           int64
	AccountID    *string
	BusinessName *string
	DeviceID     *string
	DeviceName   *string
	Kind         string
	Channel      string
	Recipient    string
	Subject      string
	Status       string
	Error        *string
	CreatedAt    time.Time
}

// NotificationLogFilter menyaring ListNotificationLog. Field kosong/nil
// berarti tidak difilter -- pola yang sama dengan WebhookDeliveryFilter.
type NotificationLogFilter struct {
	Kinds    []string
	Statuses []string
	Channels []string
	// Query cocok sebagian ke nama bisnis, alamat tujuan, atau nama device.
	Query string
	From  *time.Time
	To    *time.Time
}

// ListNotificationLog lintas SEMUA account, terbaru lebih dulu.
func (s *Store) ListNotificationLog(ctx context.Context, limit, offset int, filter NotificationLogFilter) ([]NotificationLogRow, error) {
	query := `SELECT n.id, n.account_id, a.business_name, n.device_id, n.device_name, n.kind,
	                 n.channel, n.recipient, n.subject, n.status, n.error, n.created_at
	          FROM notification_log n
	          LEFT JOIN accounts a ON a.id = n.account_id
	          WHERE 1 = 1`
	var args []any
	arg := func(v any) string {
		args = append(args, v)
		return fmt.Sprintf("$%d", len(args))
	}

	if len(filter.Kinds) > 0 {
		query += " AND n.kind = ANY(" + arg(filter.Kinds) + ")"
	}
	if len(filter.Statuses) > 0 {
		query += " AND n.status = ANY(" + arg(filter.Statuses) + ")"
	}
	if len(filter.Channels) > 0 {
		query += " AND n.channel = ANY(" + arg(filter.Channels) + ")"
	}
	if filter.Query != "" {
		p := arg("%" + filter.Query + "%")
		query += " AND (a.business_name ILIKE " + p + " OR n.recipient ILIKE " + p + " OR n.device_name ILIKE " + p + ")"
	}
	if filter.From != nil {
		query += " AND n.created_at >= " + arg(*filter.From)
	}
	if filter.To != nil {
		query += " AND n.created_at <= " + arg(*filter.To)
	}
	query += " ORDER BY n.created_at DESC, n.id DESC LIMIT " + arg(limit) + " OFFSET " + arg(offset)

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("store: list notification log: %w", err)
	}
	defer rows.Close()

	out := make([]NotificationLogRow, 0)
	for rows.Next() {
		var r NotificationLogRow
		if err := rows.Scan(&r.ID, &r.AccountID, &r.BusinessName, &r.DeviceID, &r.DeviceName, &r.Kind,
			&r.Channel, &r.Recipient, &r.Subject, &r.Status, &r.Error, &r.CreatedAt); err != nil {
			return nil, fmt.Errorf("store: scan notification log: %w", err)
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterasi notification log: %w", err)
	}
	return out, nil
}
