package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func isForeignKeyViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23503"
}

var (
	// ErrInvoiceNotEligibleForMatch: invoice tidak ditemukan, atau sudah
	// PAID — tidak boleh ditimpa oleh pencocokan manual.
	ErrInvoiceNotEligibleForMatch = errors.New("store: invoice tidak ditemukan atau sudah PAID")
	// ErrEventAlreadyMatched: event ini sudah dipakai invoice lain
	// (invoices_matched_event_id_idx).
	ErrEventAlreadyMatched = errors.New("store: event sudah dipakai untuk invoice lain")
	// ErrEventAlreadyDismissed: event_id sudah pernah di-dismiss
	// sebelumnya (event_reviews.event_id primary key).
	ErrEventAlreadyDismissed = errors.New("store: event sudah pernah diabaikan")
	// ErrEventNotFound: event_id tidak ada di notification_events sama
	// sekali (FK event_reviews -> notification_events).
	ErrEventNotFound = errors.New("store: event tidak ditemukan")
)

// ExceptionFilter menyaring ListExceptions. Field kosong/nil berarti tidak
// difilter pada dimensi itu — pola yang sama dengan EventFilter.
type ExceptionFilter struct {
	Query string // cocok sebagian ke device_id atau title
	From  *time.Time
	To    *time.Time
}

// ListExceptions mengembalikan event yang amount_hint-nya terisi, tidak
// direferensikan invoice manapun sebagai matched_event_id, dan belum pernah
// di-dismiss — "exception" dihitung lewat query, bukan status yang
// disimpan, supaya tidak ada dua sumber kebenaran yang bisa tidak sinkron.
func (s *Store) ListExceptions(ctx context.Context, limit, offset int, filter ExceptionFilter) ([]Event, error) {
	query := `SELECT e.event_id, e.device_id, e.source, e.package_name,
	                 e.title, e.body_text, e.big_text, e.amount_hint,
	                 e.posted_at, e.received_at, e.raw_payload
	          FROM notification_events e
	          WHERE e.amount_hint IS NOT NULL
	            AND NOT EXISTS (SELECT 1 FROM invoices i WHERE i.matched_event_id = e.event_id)
	            AND NOT EXISTS (SELECT 1 FROM event_reviews r WHERE r.event_id = e.event_id)`
	var args []any
	arg := func(v any) string {
		args = append(args, v)
		return fmt.Sprintf("$%d", len(args))
	}

	if filter.Query != "" {
		p := arg("%" + filter.Query + "%")
		query += " AND (e.device_id ILIKE " + p + " OR e.title ILIKE " + p + ")"
	}
	if filter.From != nil {
		query += " AND e.received_at >= " + arg(*filter.From)
	}
	if filter.To != nil {
		query += " AND e.received_at <= " + arg(*filter.To)
	}
	query += " ORDER BY e.received_at DESC LIMIT " + arg(limit) + " OFFSET " + arg(offset)

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("store: list exceptions: %w", err)
	}
	defer rows.Close()

	out := make([]Event, 0)
	for rows.Next() {
		var e Event
		if err := rows.Scan(&e.EventID, &e.DeviceID, &e.Source, &e.PackageName,
			&e.Title, &e.BodyText, &e.BigText, &e.AmountHint,
			&e.PostedAt, &e.ReceivedAt, &e.RawPayload); err != nil {
			return nil, fmt.Errorf("store: scan exception: %w", err)
		}
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterasi exceptions: %w", err)
	}
	return out, nil
}

// ManualMatchEvent mencocokkan event ke invoice pilihan admin — atomik,
// pola yang sama dengan MatchEvent (matching otomatis), tapi dikunci oleh
// id invoice, bukan nominal. Invoice boleh PENDING atau EXPIRED (customer
// bayar telat adalah kasus paling umum yang justru butuh konsol ini) —
// tidak boleh PAID, supaya tidak menimpa yang sudah lunas.
func (s *Store) ManualMatchEvent(ctx context.Context, now time.Time, invoiceID, eventID string) error {
	var id string
	err := s.pool.QueryRow(ctx,
		`UPDATE invoices SET status = $1, matched_event_id = $2, paid_at = $3
		 WHERE id = $4 AND status IN ($5, $6)
		 RETURNING id`,
		InvoiceStatusPaid, eventID, now, invoiceID, InvoiceStatusPending, InvoiceStatusExpired).
		Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrInvoiceNotEligibleForMatch
	}
	if isUniqueViolation(err, "invoices_matched_event_id_idx") {
		return ErrEventAlreadyMatched
	}
	if err != nil {
		return fmt.Errorf("store: manual match event: %w", err)
	}
	return nil
}

// DismissEvent menandai event sebagai sengaja diabaikan — tidak akan
// muncul lagi di ListExceptions. Tidak ada "undo", konsisten dengan pola
// tidak-ada-un-revoke di seluruh sistem ini (API key, webhook).
func (s *Store) DismissEvent(ctx context.Context, eventID string, note *string) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO event_reviews (event_id, note) VALUES ($1, $2)`, eventID, note)
	if isUniqueViolation(err, "event_reviews_pkey") {
		return ErrEventAlreadyDismissed
	}
	if isForeignKeyViolation(err) {
		return ErrEventNotFound
	}
	if err != nil {
		return fmt.Errorf("store: dismiss event: %w", err)
	}
	return nil
}
