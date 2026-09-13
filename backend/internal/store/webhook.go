package store

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/akbarryyan/gopay-notifications/backend/internal/secretbox"
	"github.com/jackc/pgx/v5"
)

const (
	WebhookEventInvoicePaid    = "invoice.paid"
	WebhookEventInvoiceExpired = "invoice.expired"
	WebhookEventTest           = "test"
)

const (
	WebhookDeliveryPending   = "PENDING"
	WebhookDeliveryRetrying  = "RETRYING"
	WebhookDeliveryDelivered = "DELIVERED"
	WebhookDeliveryFailed    = "FAILED"
	webhookMaxAttempts       = 5
)

// webhookBackoff[i] adalah jeda sebelum percobaan ke-(i+2), dalam menit —
// percobaan pertama selalu langsung (§3.2 spec), backoff baru berlaku
// setelah percobaan itu gagal.
var webhookBackoff = []time.Duration{
	1 * time.Minute,
	2 * time.Minute,
	4 * time.Minute,
	8 * time.Minute,
	16 * time.Minute,
}

var (
	ErrWebhookNotFound = errors.New("store: webhook endpoint tidak ditemukan")
)

type WebhookEndpoint struct {
	ID        string
	AccountID string
	Name      string
	URL       string
	Events    []string
	Enabled   bool
	CreatedAt time.Time
}

// WebhookEndpointSummary menambahkan ringkasan percobaan terakhir, untuk
// tabel Webhooks di dashboard — dihitung lewat LEFT JOIN LATERAL, bukan
// N+1 query per baris.
type WebhookEndpointSummary struct {
	WebhookEndpoint
	LastDeliveryAt     *time.Time
	LastDeliveryStatus *string
}

type WebhookDelivery struct {
	ID            string
	EndpointID    string
	Event         string
	InvoiceID     *string
	Payload       []byte
	Status        string
	Attempt       int
	NextAttemptAt *time.Time
	HTTPStatus    *int
	DurationMs    *int
	CreatedAt     time.Time
	DeliveredAt   *time.Time
}

// WebhookDeliveryDue menyertakan data endpoint yang dibutuhkan untuk
// mengirim (URL dan secret terdekripsi) — dipakai worker berkala supaya
// tidak perlu query endpoint terpisah per delivery.
type WebhookDeliveryDue struct {
	WebhookDelivery
	EndpointURL    string
	EndpointSecret []byte
}

// NewWebhookID menghasilkan ID endpoint baru, pola sama seperti device_id
// dan invoice/api-key ID: crypto/rand + hex.
func NewWebhookID() (string, error) {
	return randomPrefixedID("wh_")
}

func newWebhookDeliveryID() (string, error) {
	return randomPrefixedID("whd_")
}

// GenerateWebhookSecret membuat secret mentah baru (ditampilkan sekali ke
// pemanggil, lalu dienkripsi lewat CreateWebhookEndpoint) — pola prefix
// "whsec_" mengikuti kebiasaan umum (Stripe dkk) supaya gampang dikenali
// dari sekilas pandang sebagai webhook secret, bukan API key ("sk_...").
func GenerateWebhookSecret() (raw string, secretBytes []byte, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", nil, fmt.Errorf("store: random webhook secret: %w", err)
	}
	raw = "whsec_" + hex.EncodeToString(b)
	return raw, []byte(raw), nil
}

// CreateWebhookEndpoint menyimpan endpoint baru dengan secret terenkripsi.
//
// Dienkripsi (secretbox), BUKAN di-hash seperti api_keys — pengiriman
// webhook butuh secret mentahnya lagi tiap kali menandatangani payload
// keluar, jadi hash satu-arah tidak berlaku di sini (lihat invoice.go/
// apikey.go untuk pola hash yang dipakai di tempat yang memang cukup
// verifikasi satu arah).
func (s *Store) CreateWebhookEndpoint(ctx context.Context, key []byte, accountID, id, name, url string, events []string, secret []byte) error {
	enc, err := secretbox.Seal(key, secret)
	if err != nil {
		return fmt.Errorf("store: enkripsi secret webhook: %w", err)
	}
	_, err = s.pool.Exec(ctx,
		`INSERT INTO webhook_endpoints (id, account_id, name, url, secret_encrypted, events)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		id, accountID, name, url, enc, events)
	if err != nil {
		return fmt.Errorf("store: create webhook endpoint: %w", err)
	}
	return nil
}

// ListWebhookEndpoints tidak pernah membaca secret_encrypted — dashboard
// tidak butuh melihatnya, pola yang sama dengan ListDevices/ListAPIKeys.
func (s *Store) ListWebhookEndpoints(ctx context.Context, accountID string) ([]WebhookEndpointSummary, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT e.id, e.account_id, e.name, e.url, e.events, e.enabled, e.created_at,
		        d.created_at, d.status
		 FROM webhook_endpoints e
		 LEFT JOIN LATERAL (
		   SELECT created_at, status FROM webhook_deliveries
		   WHERE endpoint_id = e.id ORDER BY created_at DESC LIMIT 1
		 ) d ON true
		 WHERE e.account_id = $1
		 ORDER BY e.created_at DESC`, accountID)
	if err != nil {
		return nil, fmt.Errorf("store: list webhook endpoints: %w", err)
	}
	defer rows.Close()

	out := make([]WebhookEndpointSummary, 0)
	for rows.Next() {
		var w WebhookEndpointSummary
		if err := rows.Scan(&w.ID, &w.AccountID, &w.Name, &w.URL, &w.Events, &w.Enabled, &w.CreatedAt,
			&w.LastDeliveryAt, &w.LastDeliveryStatus); err != nil {
			return nil, fmt.Errorf("store: scan webhook endpoint: %w", err)
		}
		out = append(out, w)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterasi webhook endpoints: %w", err)
	}
	return out, nil
}

// GetWebhookEndpoint mengembalikan endpoint beserta secret yang sudah
// didekripsi — dipakai saat mengirim (attemptDelivery) dan saat test.
// Dibatasi account_id -- endpoint milik akun lain diperlakukan sama
// seperti tidak ada.
func (s *Store) GetWebhookEndpoint(ctx context.Context, key []byte, accountID, id string) (WebhookEndpoint, []byte, error) {
	var (
		w   WebhookEndpoint
		enc []byte
	)
	err := s.pool.QueryRow(ctx,
		`SELECT id, account_id, name, url, events, enabled, created_at, secret_encrypted
		 FROM webhook_endpoints WHERE id = $1 AND account_id = $2`, id, accountID).
		Scan(&w.ID, &w.AccountID, &w.Name, &w.URL, &w.Events, &w.Enabled, &w.CreatedAt, &enc)
	if errors.Is(err, pgx.ErrNoRows) {
		return WebhookEndpoint{}, nil, ErrWebhookNotFound
	}
	if err != nil {
		return WebhookEndpoint{}, nil, fmt.Errorf("store: select webhook endpoint: %w", err)
	}
	secret, err := secretbox.Open(key, enc)
	if err != nil {
		return WebhookEndpoint{}, nil, fmt.Errorf("store: dekripsi secret webhook %s: %w", id, err)
	}
	return w, secret, nil
}

// SetWebhookEndpointEnabled mengaktifkan/menonaktifkan endpoint milik akun ini.
func (s *Store) SetWebhookEndpointEnabled(ctx context.Context, accountID, id string, enabled bool) error {
	tag, err := s.pool.Exec(ctx,
		`UPDATE webhook_endpoints SET enabled = $3 WHERE id = $1 AND account_id = $2`, id, accountID, enabled)
	if err != nil {
		return fmt.Errorf("store: set webhook endpoint enabled: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrWebhookNotFound
	}
	return nil
}

// DeleteWebhookEndpoint menghapus endpoint milik akun ini. ON DELETE CASCADE
// di migrasi ikut menghapus seluruh webhook_deliveries miliknya.
func (s *Store) DeleteWebhookEndpoint(ctx context.Context, accountID, id string) error {
	tag, err := s.pool.Exec(ctx,
		`DELETE FROM webhook_endpoints WHERE id = $1 AND account_id = $2`, id, accountID)
	if err != nil {
		return fmt.Errorf("store: delete webhook endpoint: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrWebhookNotFound
	}
	return nil
}

// EnqueueWebhookDeliveries menyisipkan satu baris per endpoint yang enabled
// dan berlangganan event itu. next_attempt_at diisi eksplisit = now, bukan
// dibiarkan NULL, supaya baris itu langsung "jatuh tempo" tanpa menunggu
// putaran worker berikutnya (lihat webhook_deliveries_due_idx).
func (s *Store) EnqueueWebhookDeliveries(ctx context.Context, now time.Time, event string, invoiceID *string, payload []byte) ([]string, error) {
	// account_id ikut diambil dari endpoint-nya (denormalisasi ke
	// webhook_deliveries, spec §2.2) -- worker ini sengaja lintas akun,
	// jadi tidak menerima accountID sebagai parameter.
	rows, err := s.pool.Query(ctx,
		`SELECT id, account_id FROM webhook_endpoints WHERE enabled = true AND $1 = ANY(events)`, event)
	if err != nil {
		return nil, fmt.Errorf("store: cari webhook endpoints untuk %s: %w", event, err)
	}
	type endpointRef struct{ id, accountID string }
	var endpoints []endpointRef
	for rows.Next() {
		var e endpointRef
		if err := rows.Scan(&e.id, &e.accountID); err != nil {
			rows.Close()
			return nil, fmt.Errorf("store: scan endpoint id: %w", err)
		}
		endpoints = append(endpoints, e)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, fmt.Errorf("store: iterasi endpoint ids: %w", err)
	}
	rows.Close()

	deliveryIDs := make([]string, 0, len(endpoints))
	for _, ep := range endpoints {
		id, err := newWebhookDeliveryID()
		if err != nil {
			return nil, err
		}
		_, err = s.pool.Exec(ctx,
			`INSERT INTO webhook_deliveries
			   (id, account_id, endpoint_id, event, invoice_id, payload, status, next_attempt_at, created_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
			id, ep.accountID, ep.id, event, invoiceID, payload, WebhookDeliveryPending, now, now)
		if err != nil {
			return nil, fmt.Errorf("store: enqueue webhook delivery: %w", err)
		}
		deliveryIDs = append(deliveryIDs, id)
	}
	return deliveryIDs, nil
}

// DueWebhookDeliveries mengambil delivery yang siap dicoba (PENDING atau
// RETRYING dengan next_attempt_at sudah lewat), beserta data endpoint yang
// dibutuhkan untuk mengirim.
//
// Mengklaim baris secara atomik lewat UPDATE...RETURNING (pola yang sama
// dengan MatchEvent) — bukan SELECT polos. Tanpa ini, dua pemroses yang
// berjalan bersamaan (ticker 1 menit di cmd/server dan goroutine percobaan
// pertama yang dipicu invoice.paid) bisa mengambil delivery yang SAMA dan
// mengirimnya dua kali ke merchant. Klaim dilakukan dengan menuliskan
// next_attempt_at = NULL sebelum dikembalikan; pemanggil lain yang query di
// saat hampir bersamaan tidak lagi melihat baris itu sebagai "jatuh tempo".
func (s *Store) DueWebhookDeliveries(ctx context.Context, key []byte, now time.Time) ([]WebhookDeliveryDue, error) {
	rows, err := s.pool.Query(ctx,
		`WITH claimed AS (
		   UPDATE webhook_deliveries
		   SET next_attempt_at = NULL
		   WHERE status IN ($1, $2) AND next_attempt_at <= $3
		   RETURNING id, endpoint_id, event, invoice_id, payload, status, attempt, created_at
		 )
		 SELECT claimed.id, claimed.endpoint_id, claimed.event, claimed.invoice_id,
		        claimed.payload, claimed.status, claimed.attempt, claimed.created_at,
		        e.url, e.secret_encrypted
		 FROM claimed
		 JOIN webhook_endpoints e ON e.id = claimed.endpoint_id
		 ORDER BY claimed.created_at ASC`,
		WebhookDeliveryPending, WebhookDeliveryRetrying, now)
	if err != nil {
		return nil, fmt.Errorf("store: due webhook deliveries: %w", err)
	}
	defer rows.Close()

	out := make([]WebhookDeliveryDue, 0)
	for rows.Next() {
		var d WebhookDeliveryDue
		var enc []byte
		if err := rows.Scan(&d.ID, &d.EndpointID, &d.Event, &d.InvoiceID, &d.Payload, &d.Status,
			&d.Attempt, &d.CreatedAt, &d.EndpointURL, &enc); err != nil {
			return nil, fmt.Errorf("store: scan due webhook delivery: %w", err)
		}
		// Baru saja diklaim (next_attempt_at ditulis NULL oleh UPDATE di
		// atas) — bukan lagi bagian dari data yang relevan untuk dipakai
		// attemptDelivery, sengaja tidak di-scan.
		d.NextAttemptAt = nil
		secret, err := secretbox.Open(key, enc)
		if err != nil {
			return nil, fmt.Errorf("store: dekripsi secret webhook untuk delivery %s: %w", d.ID, err)
		}
		d.EndpointSecret = secret
		out = append(out, d)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterasi due webhook deliveries: %w", err)
	}
	return out, nil
}

// RecordDeliverySuccess menandai delivery sebagai DELIVERED.
func (s *Store) RecordDeliverySuccess(ctx context.Context, id string, now time.Time, httpStatus, durationMs int) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE webhook_deliveries
		 SET status = $2, http_status = $3, duration_ms = $4, delivered_at = $5, next_attempt_at = NULL
		 WHERE id = $1`,
		id, WebhookDeliveryDelivered, httpStatus, durationMs, now)
	if err != nil {
		return fmt.Errorf("store: record delivery success: %w", err)
	}
	return nil
}

// RecordDeliveryFailure menandai delivery gagal: RETRYING dengan jadwal
// percobaan berikutnya bila attempt belum habis, atau FAILED permanen kalau
// sudah mencapai webhookMaxAttempts.
func (s *Store) RecordDeliveryFailure(ctx context.Context, id string, now time.Time, attempt int, httpStatus *int, durationMs int) error {
	if attempt >= webhookMaxAttempts {
		_, err := s.pool.Exec(ctx,
			`UPDATE webhook_deliveries
			 SET status = $2, attempt = $3, http_status = $4, duration_ms = $5, next_attempt_at = NULL
			 WHERE id = $1`,
			id, WebhookDeliveryFailed, attempt, httpStatus, durationMs)
		if err != nil {
			return fmt.Errorf("store: record delivery failure (final): %w", err)
		}
		return nil
	}

	next := now.Add(webhookBackoff[attempt-1])
	_, err := s.pool.Exec(ctx,
		`UPDATE webhook_deliveries
		 SET status = $2, attempt = $3, http_status = $4, duration_ms = $5, next_attempt_at = $6
		 WHERE id = $1`,
		id, WebhookDeliveryRetrying, attempt, httpStatus, durationMs, next)
	if err != nil {
		return fmt.Errorf("store: record delivery failure (retry): %w", err)
	}
	return nil
}

// EnqueueTestDelivery menyisipkan satu baris delivery untuk SATU endpoint
// tertentu (bukan dicari lewat events seperti EnqueueWebhookDeliveries),
// dengan invoice_id NULL — dipakai tombol "Test" di dashboard.
// EnqueueTestDelivery hanya menyisipkan baris kalau endpointID itu memang
// milik accountID -- EXISTS di bawah mencegah account A memicu test
// delivery ke endpoint milik account B walau ID-nya diketahui/ditebak.
func (s *Store) EnqueueTestDelivery(ctx context.Context, now time.Time, accountID, endpointID string, payload []byte) (string, error) {
	id, err := newWebhookDeliveryID()
	if err != nil {
		return "", err
	}
	tag, err := s.pool.Exec(ctx,
		`INSERT INTO webhook_deliveries
		   (id, account_id, endpoint_id, event, invoice_id, payload, status, next_attempt_at, created_at)
		 SELECT $1, $7, $2, $3, NULL, $4, $5, NULL, $6
		 WHERE EXISTS (SELECT 1 FROM webhook_endpoints WHERE id = $2 AND account_id = $7)`,
		id, endpointID, WebhookEventTest, payload, WebhookDeliveryPending, now, accountID)
	if err != nil {
		return "", fmt.Errorf("store: enqueue test delivery: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return "", ErrWebhookNotFound
	}
	return id, nil
}

// RecordTestDeliveryResult menuliskan hasil percobaan test — selalu
// status akhir (DELIVERED atau FAILED), TIDAK PERNAH RETRYING. Test dipicu
// manual dari dashboard; admin sudah melihat hasilnya seketika lewat
// response HTTP, mengulang otomatis nanti hanya akan mengejutkan dan
// berpotensi membebani endpoint mereka tanpa diminta.
func (s *Store) RecordTestDeliveryResult(ctx context.Context, id string, now time.Time, success bool, httpStatus, durationMs int) error {
	status := WebhookDeliveryFailed
	var deliveredAt *time.Time
	if success {
		status = WebhookDeliveryDelivered
		deliveredAt = &now
	}
	var httpStatusPtr *int
	if httpStatus != 0 {
		httpStatusPtr = &httpStatus
	}
	_, err := s.pool.Exec(ctx,
		`UPDATE webhook_deliveries
		 SET status = $2, attempt = 1, http_status = $3, duration_ms = $4,
		     delivered_at = $5, next_attempt_at = NULL
		 WHERE id = $1`,
		id, status, httpStatusPtr, durationMs, deliveredAt)
	if err != nil {
		return fmt.Errorf("store: record test delivery result: %w", err)
	}
	return nil
}

// ListWebhookDeliveries mengembalikan riwayat pengiriman satu endpoint,
// terbaru lebih dulu — dipakai baris yang diperluas di dashboard.
func (s *Store) ListWebhookDeliveries(ctx context.Context, accountID, endpointID string, limit, offset int) ([]WebhookDelivery, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT d.id, d.endpoint_id, d.event, d.invoice_id, d.payload, d.status, d.attempt,
		        d.next_attempt_at, d.http_status, d.duration_ms, d.created_at, d.delivered_at
		 FROM webhook_deliveries d
		 JOIN webhook_endpoints e ON e.id = d.endpoint_id
		 WHERE d.endpoint_id = $1 AND e.account_id = $2
		 ORDER BY d.created_at DESC
		 LIMIT $3 OFFSET $4`, endpointID, accountID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("store: list webhook deliveries: %w", err)
	}
	defer rows.Close()

	out := make([]WebhookDelivery, 0)
	for rows.Next() {
		var d WebhookDelivery
		if err := rows.Scan(&d.ID, &d.EndpointID, &d.Event, &d.InvoiceID, &d.Payload, &d.Status,
			&d.Attempt, &d.NextAttemptAt, &d.HTTPStatus, &d.DurationMs, &d.CreatedAt, &d.DeliveredAt); err != nil {
			return nil, fmt.Errorf("store: scan webhook delivery: %w", err)
		}
		out = append(out, d)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterasi webhook deliveries: %w", err)
	}
	return out, nil
}
