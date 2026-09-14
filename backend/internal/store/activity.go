package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// Jenis aktivitas di account_activity_log.action.
const (
	ActivityLoginSuccess    = "login_success"
	ActivityLoginFailed     = "login_failed"
	ActivityPasswordChanged = "password_changed"
	ActivityPasswordReset   = "password_reset"
	ActivityAPIKeyCreated   = "api_key_created"
	ActivityAPIKeyRevoked   = "api_key_revoked"
	ActivityDeviceAdded     = "device_added"
	ActivityDeviceDeleted   = "device_deleted"
)

// ActivityLogEntry adalah satu kejadian untuk dicatat.
type ActivityLogEntry struct {
	AccountID string
	Action    string
	// IPAddress/UserAgent nil untuk kejadian yang tidak berasal dari satu
	// request HTTP tunggal (tidak ada saat ini, tapi dibiarkan opsional).
	IPAddress *string
	UserAgent *string
	Metadata  map[string]any
}

// LogActivity mencatat satu baris riwayat. Gagal mencatat TIDAK menggagalkan
// aksi yang memicunya -- riwayat adalah alat bantu, bukan syarat aksi itu
// sendiri berhasil (pola sama dengan LogAudit).
func (s *Store) LogActivity(ctx context.Context, e ActivityLogEntry) error {
	var raw []byte
	if e.Metadata != nil {
		var err error
		raw, err = json.Marshal(e.Metadata)
		if err != nil {
			return fmt.Errorf("store: marshal metadata aktivitas: %w", err)
		}
	}
	_, err := s.pool.Exec(ctx,
		`INSERT INTO account_activity_log (account_id, action, ip_address, user_agent, metadata)
		 VALUES ($1, $2, $3, $4, $5)`,
		e.AccountID, e.Action, e.IPAddress, e.UserAgent, raw)
	if err != nil {
		return fmt.Errorf("store: catat aktivitas: %w", err)
	}
	return nil
}

// ActivityLogRow adalah satu baris riwayat untuk Customer Dashboard.
type ActivityLogRow struct {
	ID        int64
	Action    string
	IPAddress *string
	UserAgent *string
	Metadata  json.RawMessage
	CreatedAt time.Time
}

// ActivityLogFilter menyaring ListActivityLog. Field kosong/nil berarti
// tidak difilter -- pola yang sama dengan NotificationLogFilter.
type ActivityLogFilter struct {
	Actions []string
	From    *time.Time
	To      *time.Time
}

// ListActivityLog SELALU di-scope satu account -- beda dari
// ListNotificationLog/ListAllWebhookDeliveries (Vendor Dashboard, lintas
// account), ini dilihat customer sendiri lewat requireAdmin.
func (s *Store) ListActivityLog(ctx context.Context, accountID string, limit, offset int, filter ActivityLogFilter) ([]ActivityLogRow, error) {
	query := `SELECT id, action, ip_address, user_agent, metadata, created_at
	          FROM account_activity_log WHERE account_id = $1`
	args := []any{accountID}
	arg := func(v any) string {
		args = append(args, v)
		return fmt.Sprintf("$%d", len(args))
	}

	if len(filter.Actions) > 0 {
		query += " AND action = ANY(" + arg(filter.Actions) + ")"
	}
	if filter.From != nil {
		query += " AND created_at >= " + arg(*filter.From)
	}
	if filter.To != nil {
		query += " AND created_at <= " + arg(*filter.To)
	}
	query += " ORDER BY created_at DESC, id DESC LIMIT " + arg(limit) + " OFFSET " + arg(offset)

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("store: list activity log: %w", err)
	}
	defer rows.Close()

	out := make([]ActivityLogRow, 0)
	for rows.Next() {
		var r ActivityLogRow
		if err := rows.Scan(&r.ID, &r.Action, &r.IPAddress, &r.UserAgent, &r.Metadata, &r.CreatedAt); err != nil {
			return nil, fmt.Errorf("store: scan activity log: %w", err)
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterasi activity log: %w", err)
	}
	return out, nil
}
