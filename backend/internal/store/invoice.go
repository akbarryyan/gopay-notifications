package store

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const (
	InvoiceStatusPending = "PENDING"
	InvoiceStatusPaid    = "PAID"
	InvoiceStatusExpired = "EXPIRED"
)

const (
	invoiceExpiryDuration     = 15 * time.Minute
	invoiceOffsetMax          = 999
	invoiceAllocationAttempts = 20
)

var (
	// ErrInvoiceNotFound: id atau external_ref tidak ditemukan.
	ErrInvoiceNotFound = errors.New("store: invoice tidak ditemukan")
	// ErrInvoiceRefConflict: external_ref sudah dipakai invoice lain dengan
	// requested_amount yang BEDA — bukan retry aman, konflik sungguhan.
	ErrInvoiceRefConflict = errors.New("store: external_ref sudah dipakai dengan amount berbeda")
	// ErrInvoiceAllocationFull: seluruh percobaan offset nominal unik
	// bertabrakan — nominal dasar itu sedang sangat padat.
	ErrInvoiceAllocationFull = errors.New("store: ruang nominal unik penuh untuk amount ini")
)

type Invoice struct {
	ID              string
	ExternalRef     string
	RequestedAmount int64
	UniqueAmount    int64
	Status          string
	MatchedEventID  *string
	CreatedAt       time.Time
	ExpiresAt       time.Time
	PaidAt          *time.Time
}

const invoiceSelectCols = `SELECT id, external_ref, requested_amount, unique_amount,
	status, matched_event_id, created_at, expires_at, paid_at FROM invoices`

type invoiceScanner interface {
	Scan(dest ...any) error
}

func scanInvoice(row invoiceScanner) (Invoice, error) {
	var inv Invoice
	err := row.Scan(&inv.ID, &inv.ExternalRef, &inv.RequestedAmount, &inv.UniqueAmount,
		&inv.Status, &inv.MatchedEventID, &inv.CreatedAt, &inv.ExpiresAt, &inv.PaidAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Invoice{}, ErrInvoiceNotFound
	}
	if err != nil {
		return Invoice{}, fmt.Errorf("store: scan invoice: %w", err)
	}
	return inv, nil
}

// expireStaleInvoices menuliskan transisi PENDING -> EXPIRED untuk invoice
// yang sudah lewat expires_at.
//
// Ini transisi state sungguhan, bukan status yang dihitung saat baca seperti
// store.StatusOf untuk device — invoices_pending_unique_amount_idx adalah
// partial unique index, dan predicate-nya harus immutable, jadi tidak bisa
// memakai now() langsung di sana. Kedaluwarsa harus benar-benar dituliskan
// sebelum operasi yang bergantung pada slot nominal yang sudah bebas
// (alokasi baru maupun matching).
func (s *Store) expireStaleInvoices(ctx context.Context, now time.Time) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE invoices SET status = $1 WHERE status = $2 AND expires_at <= $3`,
		InvoiceStatusExpired, InvoiceStatusPending, now)
	if err != nil {
		return fmt.Errorf("store: expire invoices: %w", err)
	}
	return nil
}

func randomOffset(max int64) (int64, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(max))
	if err != nil {
		return 0, fmt.Errorf("store: random offset: %w", err)
	}
	return n.Int64() + 1, nil // geser ke [1, max], bukan [0, max)
}

func isUniqueViolation(err error, constraint string) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505" && pgErr.ConstraintName == constraint
	}
	return false
}

// CreateInvoice mengalokasikan nominal unik dan menyimpan invoice baru.
//
// Idempotent lewat external_ref: bila external_ref itu sudah pernah dipakai
// dengan requested_amount yang SAMA, invoice yang sudah ada dikembalikan
// (created=false) alih-alih membuat invoice kedua — retry jaringan dari sisi
// merchant jadi aman. Bila requested_amount BEDA, ErrInvoiceRefConflict.
func (s *Store) CreateInvoice(ctx context.Context, now time.Time, externalRef string, requestedAmount int64) (inv Invoice, created bool, err error) {
	if err := s.expireStaleInvoices(ctx, now); err != nil {
		return Invoice{}, false, err
	}

	existing, err := s.GetInvoiceByExternalRef(ctx, externalRef)
	switch {
	case err == nil:
		if existing.RequestedAmount == requestedAmount {
			return existing, false, nil
		}
		return Invoice{}, false, ErrInvoiceRefConflict
	case errors.Is(err, ErrInvoiceNotFound):
		// belum ada, lanjut alokasi di bawah
	default:
		return Invoice{}, false, err
	}

	id, err := randomPrefixedID("inv_")
	if err != nil {
		return Invoice{}, false, err
	}
	expiresAt := now.Add(invoiceExpiryDuration)

	for attempt := 0; attempt < invoiceAllocationAttempts; attempt++ {
		offset, err := randomOffset(invoiceOffsetMax)
		if err != nil {
			return Invoice{}, false, err
		}
		uniqueAmount := requestedAmount + offset

		_, err = s.pool.Exec(ctx,
			`INSERT INTO invoices (id, external_ref, requested_amount, unique_amount, status, created_at, expires_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			id, externalRef, requestedAmount, uniqueAmount, InvoiceStatusPending, now, expiresAt)
		if err == nil {
			return Invoice{
				ID:              id,
				ExternalRef:     externalRef,
				RequestedAmount: requestedAmount,
				UniqueAmount:    uniqueAmount,
				Status:          InvoiceStatusPending,
				CreatedAt:       now,
				ExpiresAt:       expiresAt,
			}, true, nil
		}

		if isUniqueViolation(err, "invoices_pending_unique_amount_idx") {
			continue // offset ini sedang dipakai invoice PENDING lain, coba lagi
		}
		if isUniqueViolation(err, "invoices_external_ref_idx") {
			// Race: request lain barusan membuat external_ref yang sama.
			existing, ferr := s.GetInvoiceByExternalRef(ctx, externalRef)
			if ferr != nil {
				return Invoice{}, false, ferr
			}
			if existing.RequestedAmount == requestedAmount {
				return existing, false, nil
			}
			return Invoice{}, false, ErrInvoiceRefConflict
		}
		return Invoice{}, false, fmt.Errorf("store: insert invoice: %w", err)
	}
	return Invoice{}, false, ErrInvoiceAllocationFull
}

// MatchEvent mencari invoice PENDING dengan unique_amount sama dengan amount
// dan menandainya PAID. Dilewati (matched=false, err=nil) bila amount nil
// atau tidak ada invoice PENDING yang cocok — bukan error, itu memang bentuk
// normal dari nominal yang salah ketik atau bayar setelah kedaluwarsa.
//
// Satu UPDATE...RETURNING atomik, pola yang sama dengan ON CONFLICT DO
// NOTHING di InsertEvent — dua event dengan amount sama yang tiba nyaris
// bersamaan tidak bisa keduanya "menang".
func (s *Store) MatchEvent(ctx context.Context, now time.Time, eventID string, amount *int64) (matched bool, err error) {
	if amount == nil {
		return false, nil
	}
	if err := s.expireStaleInvoices(ctx, now); err != nil {
		return false, err
	}

	tag, err := s.pool.Exec(ctx,
		`UPDATE invoices SET status = $1, matched_event_id = $2, paid_at = $3
		 WHERE status = $4 AND unique_amount = $5`,
		InvoiceStatusPaid, eventID, now, InvoiceStatusPending, *amount)
	if err != nil {
		return false, fmt.Errorf("store: match event: %w", err)
	}
	return tag.RowsAffected() > 0, nil
}

// GetInvoiceByID mengambil satu invoice untuk GET /invoices/{id}.
func (s *Store) GetInvoiceByID(ctx context.Context, id string) (Invoice, error) {
	return scanInvoice(s.pool.QueryRow(ctx, invoiceSelectCols+` WHERE id = $1`, id))
}

// GetInvoiceByExternalRef dipakai untuk pengecekan idempotency di CreateInvoice.
func (s *Store) GetInvoiceByExternalRef(ctx context.Context, externalRef string) (Invoice, error) {
	return scanInvoice(s.pool.QueryRow(ctx, invoiceSelectCols+` WHERE external_ref = $1`, externalRef))
}

// InvoiceFilter menyaring ListInvoices. Field kosong/nil berarti tidak
// difilter pada dimensi itu — pola yang sama dengan store.EventFilter.
type InvoiceFilter struct {
	Status string // PENDING | PAID | EXPIRED
	Query  string // cocok sebagian ke external_ref, tanpa peduli huruf besar/kecil
	From   *time.Time
	To     *time.Time
}

// ListInvoices mengembalikan invoice terbaru lebih dulu, untuk halaman
// Transactions di dashboard.
func (s *Store) ListInvoices(ctx context.Context, limit, offset int, filter InvoiceFilter) ([]Invoice, error) {
	query := invoiceSelectCols + ` WHERE 1 = 1`
	var args []any
	arg := func(v any) string {
		args = append(args, v)
		return fmt.Sprintf("$%d", len(args))
	}

	if filter.Status != "" {
		query += " AND status = " + arg(filter.Status)
	}
	if filter.Query != "" {
		query += " AND external_ref ILIKE " + arg("%"+filter.Query+"%")
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
		return nil, fmt.Errorf("store: list invoices: %w", err)
	}
	defer rows.Close()

	out := make([]Invoice, 0)
	for rows.Next() {
		inv, err := scanInvoice(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, inv)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterasi invoices: %w", err)
	}
	return out, nil
}
