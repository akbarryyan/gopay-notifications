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

// LogAudit mencatat satu aksi vendor terhadap accounts (ACCOUNT_CREATED,
// ACCOUNT_RENEWED, ACCOUNT_SUSPENDED, ACCOUNT_REVOKED, dst).
func (s *Store) LogAudit(ctx context.Context, actor, action, resource string, metadata any) error {
	var raw []byte
	if metadata != nil {
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

// ListAuditLog dipakai Vendor Dashboard, terbaru lebih dulu.
func (s *Store) ListAuditLog(ctx context.Context, limit, offset int) ([]AuditEntry, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, actor, action, resource, metadata, created_at
		 FROM audit_log ORDER BY created_at DESC LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("store: list audit log: %w", err)
	}
	defer rows.Close()

	out := make([]AuditEntry, 0)
	for rows.Next() {
		var e AuditEntry
		if err := rows.Scan(&e.ID, &e.Actor, &e.Action, &e.Resource, &e.Metadata, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("store: scan audit entry: %w", err)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
