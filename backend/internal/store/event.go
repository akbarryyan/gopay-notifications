package store

import (
	"context"
	"fmt"
	"time"
)

type Event struct {
	EventID     string
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
		   (event_id, device_id, source, package_name,
		    title, body_text, big_text, amount_hint,
		    posted_at, received_at, raw_payload)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		 ON CONFLICT (event_id) DO NOTHING`,
		e.EventID, e.DeviceID, e.Source, e.PackageName,
		e.Title, e.BodyText, e.BigText, e.AmountHint,
		e.PostedAt, e.ReceivedAt, e.RawPayload)
	if err != nil {
		return false, fmt.Errorf("store: insert event: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}

// ListEvents mengembalikan event terbaru lebih dulu.
func (s *Store) ListEvents(ctx context.Context, limit, offset int) ([]Event, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT event_id, device_id, source, package_name,
		        title, body_text, big_text, amount_hint,
		        posted_at, received_at, raw_payload
		 FROM notification_events
		 ORDER BY ingested_at DESC, id DESC
		 LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("store: list events: %w", err)
	}
	defer rows.Close()

	var out []Event
	for rows.Next() {
		var e Event
		if err := rows.Scan(&e.EventID, &e.DeviceID, &e.Source, &e.PackageName,
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
