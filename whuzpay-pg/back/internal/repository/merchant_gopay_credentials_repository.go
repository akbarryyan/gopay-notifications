package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

var ErrGopayCredentialsNotFound = errors.New("gopay credentials not found")

// GopayCredentials -- semua field independen (bisa salah satu/beberapa
// nil kalau belum diisi/dicapai). Username dan QRISConfiguredAt ditambah
// untuk onboarding terpadu, lihat spec
// 2026-09-17-whuzpay-pg-unified-onboarding-design.md.
type GopayCredentials struct {
	MerchantID       uuid.UUID
	APIKey           *string
	WebhookSecret    *string
	Username         *string
	QRISConfiguredAt *time.Time
}

type MerchantGopayCredentialsRepository struct {
	db *sql.DB
}

func NewMerchantGopayCredentialsRepository(db *sql.DB) *MerchantGopayCredentialsRepository {
	return &MerchantGopayCredentialsRepository{db: db}
}

func (r *MerchantGopayCredentialsRepository) Get(ctx context.Context, merchantID uuid.UUID) (*GopayCredentials, error) {
	c := &GopayCredentials{MerchantID: merchantID}
	query := `SELECT api_key, webhook_secret, gopay_username, qris_configured_at FROM merchant_gopay_credentials WHERE merchant_id = $1`
	err := r.db.QueryRowContext(ctx, query, merchantID).Scan(&c.APIKey, &c.WebhookSecret, &c.Username, &c.QRISConfiguredAt)
	if err == sql.ErrNoRows {
		return nil, ErrGopayCredentialsNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get gopay credentials: %w", err)
	}
	return c, nil
}

// Upsert menerima *string per field -- pola tri-state: nil = biarkan
// nilai lama, non-nil (termasuk string kosong) = ganti. Kolom yang tidak
// disentuh dipertahankan lewat COALESCE terhadap baris yang sudah ada
// (atau NULL kalau baris belum ada sama sekali). username ditambah untuk
// onboarding terpadu -- pemanggil lama (Settings PUT, lihat
// PaymentService.UpdateGopayCredentials) selalu mengirim nil di situ,
// artinya "jangan ubah", persis perilaku sebelum kolom ini ada.
func (r *MerchantGopayCredentialsRepository) Upsert(ctx context.Context, merchantID uuid.UUID, apiKey, webhookSecret, username *string) error {
	query := `
		INSERT INTO merchant_gopay_credentials (merchant_id, api_key, webhook_secret, gopay_username, updated_at)
		VALUES ($1, $2, $3, $4, CURRENT_TIMESTAMP)
		ON CONFLICT (merchant_id) DO UPDATE SET
			api_key = CASE WHEN $5 THEN merchant_gopay_credentials.api_key ELSE EXCLUDED.api_key END,
			webhook_secret = CASE WHEN $6 THEN merchant_gopay_credentials.webhook_secret ELSE EXCLUDED.webhook_secret END,
			gopay_username = CASE WHEN $7 THEN merchant_gopay_credentials.gopay_username ELSE EXCLUDED.gopay_username END,
			updated_at = CURRENT_TIMESTAMP
	`
	_, err := r.db.ExecContext(ctx, query, merchantID, apiKey, webhookSecret, username,
		apiKey == nil, webhookSecret == nil, username == nil)
	if err != nil {
		return fmt.Errorf("failed to upsert gopay credentials: %w", err)
	}
	return nil
}

// MarkQRISConfigured mencatat bahwa QRIS berhasil diupload lewat wizard
// onboarding (lihat internal/gopayonboard.Client.UploadQRISImage) --
// dipanggil TEPAT SEKALI setelah upload sukses, tidak pernah dipanggil
// untuk "unmark".
func (r *MerchantGopayCredentialsRepository) MarkQRISConfigured(ctx context.Context, merchantID uuid.UUID) error {
	query := `
		INSERT INTO merchant_gopay_credentials (merchant_id, qris_configured_at, updated_at)
		VALUES ($1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		ON CONFLICT (merchant_id) DO UPDATE SET
			qris_configured_at = CURRENT_TIMESTAMP,
			updated_at = CURRENT_TIMESTAMP
	`
	_, err := r.db.ExecContext(ctx, query, merchantID)
	if err != nil {
		return fmt.Errorf("failed to mark qris configured: %w", err)
	}
	return nil
}
