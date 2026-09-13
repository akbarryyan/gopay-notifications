package store

import (
	"context"
	"fmt"
	"time"
)

type Event struct {
	EventID     string
	AccountID   string
	DeviceID    string
	Source      string
	PackageName string
	Title       *string
	BodyText    *string
	BigText     *string
	AmountHint  *int64
	PostedAt    time.Time
	ReceivedAt  time.Time
	RawPayload  []byte
}

// InsertEvent menyimpan event dan melaporkan apakah ia benar-benar baru.
//
// Idempotency bersandar pada unique constraint, bukan pada SELECT lebih dulu.
// Pendekatan SELECT-lalu-INSERT salah bila dua request identik tiba bersamaan:
// keduanya akan melihat baris belum ada, lalu keduanya menyisipkan.
func (s *Store) InsertEvent(ctx context.Context, e Event) (bool, error) {
	tag, err := s.pool.Exec(ctx,
		`INSERT INTO notification_events
		   (event_id, account_id, device_id, source, package_name,
		    title, body_text, big_text, amount_hint,
		    posted_at, received_at, raw_payload)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		 ON CONFLICT (event_id) DO NOTHING`,
		e.EventID, e.AccountID, e.DeviceID, e.Source, e.PackageName,
		e.Title, e.BodyText, e.BigText, e.AmountHint,
		e.PostedAt, e.ReceivedAt, e.RawPayload)
	if err != nil {
		return false, fmt.Errorf("store: insert event: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}

// EventFilter menyaring ListEvents. Field kosong/nil berarti tidak difilter
// pada dimensi itu.
type EventFilter struct {
	Source string     // cocok persis dengan connector.Info.ID, mis. "gopay"
	Query  string     // cocok sebagian ke device_id ATAU title, tanpa peduli huruf besar/kecil
	From   *time.Time // received_at >= From
	To     *time.Time // received_at <= To
}

// ListEvents mengembalikan event terbaru lebih dulu.
func (s *Store) ListEvents(ctx context.Context, accountID string, limit, offset int, filter EventFilter) ([]Event, error) {
	query := `SELECT event_id, account_id, device_id, source, package_name,
	                 title, body_text, big_text, amount_hint,
	                 posted_at, received_at, raw_payload
	          FROM notification_events
	          WHERE account_id = $1`
	args := []any{accountID}
	arg := func(v any) string {
		args = append(args, v)
		return fmt.Sprintf("$%d", len(args))
	}

	if filter.Source != "" {
		query += " AND source = " + arg(filter.Source)
	}
	if filter.Query != "" {
		p := arg("%" + filter.Query + "%")
		query += " AND (device_id ILIKE " + p + " OR title ILIKE " + p + ")"
	}
	if filter.From != nil {
		query += " AND received_at >= " + arg(*filter.From)
	}
	if filter.To != nil {
		query += " AND received_at <= " + arg(*filter.To)
	}
	query += " ORDER BY ingested_at DESC, id DESC LIMIT " + arg(limit) + " OFFSET " + arg(offset)

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("store: list events: %w", err)
	}
	defer rows.Close()

	var out []Event
	for rows.Next() {
		var e Event
		if err := rows.Scan(&e.EventID, &e.AccountID, &e.DeviceID, &e.Source, &e.PackageName,
			&e.Title, &e.BodyText, &e.BigText, &e.AmountHint,
			&e.PostedAt, &e.ReceivedAt, &e.RawPayload); err != nil {
			return nil, fmt.Errorf("store: scan event: %w", err)
		}
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterasi events: %w", err)
	}
	return out, nil
}
