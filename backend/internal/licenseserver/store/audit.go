package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

type AuditEntry struct {
	ID        int64
	Actor     string
	Action    string
	Resource  string
	Metadata  json.RawMessage
	CreatedAt time.Time
}

// LogAudit mencatat satu aksi admin vendor. metadata boleh nil (disimpan
// sebagai objek kosong). Dipanggil dari setiap handler admin yang mengubah
// state (§26 license-spec.md) — create/renew/suspend/revoke license, reset
// installation.
func (s *Store) LogAudit(ctx context.Context, actor, action, resource string, metadata any) error {
	var raw []byte
	if metadata == nil {
		raw = []byte("{}")
	} else {
		var err error
		raw, err = json.Marshal(metadata)
		if err != nil {
			return fmt.Errorf("store: marshal audit metadata: %w", err)
		}
	}
	_, err := s.pool.Exec(ctx,
		`INSERT INTO audit_log (actor, action, resource, metadata) VALUES ($1, $2, $3, $4)`,
		actor, action, resource, raw)
	if err != nil {
		return fmt.Errorf("store: log audit: %w", err)
	}
	return nil
}

func (s *Store) ListAuditLog(ctx context.Context, limit int) ([]AuditEntry, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := s.pool.Query(ctx,
		`SELECT id, actor, action, resource, metadata, created_at
		 FROM audit_log ORDER BY created_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("store: list audit log: %w", err)
	}
	defer rows.Close()

	var out []AuditEntry
	for rows.Next() {
		var e AuditEntry
		if err := rows.Scan(&e.ID, &e.Actor, &e.Action, &e.Resource, &e.Metadata, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("store: scan audit entry: %w", err)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
