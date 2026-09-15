package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

var ErrGopayCredentialsNotFound = errors.New("gopay credentials not found")

// GopayCredentials -- APIKey/WebhookSecret independen (bisa salah satu
// nil kalau merchant baru isi satu dari dua).
type GopayCredentials struct {
	MerchantID    uuid.UUID
	APIKey        *string
	WebhookSecret *string
}

type MerchantGopayCredentialsRepository struct {
	db *sql.DB
}

func NewMerchantGopayCredentialsRepository(db *sql.DB) *MerchantGopayCredentialsRepository {
	return &MerchantGopayCredentialsRepository{db: db}
}

func (r *MerchantGopayCredentialsRepository) Get(ctx context.Context, merchantID uuid.UUID) (*GopayCredentials, error) {
	c := &GopayCredentials{MerchantID: merchantID}
	query := `SELECT api_key, webhook_secret FROM merchant_gopay_credentials WHERE merchant_id = $1`
	err := r.db.QueryRowContext(ctx, query, merchantID).Scan(&c.APIKey, &c.WebhookSecret)
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
// (atau NULL kalau baris belum ada sama sekali).
func (r *MerchantGopayCredentialsRepository) Upsert(ctx context.Context, merchantID uuid.UUID, apiKey, webhookSecret *string) error {
	query := `
		INSERT INTO merchant_gopay_credentials (merchant_id, api_key, webhook_secret, updated_at)
		VALUES ($1, $2, $3, CURRENT_TIMESTAMP)
		ON CONFLICT (merchant_id) DO UPDATE SET
			api_key = CASE WHEN $4 THEN merchant_gopay_credentials.api_key ELSE EXCLUDED.api_key END,
			webhook_secret = CASE WHEN $5 THEN merchant_gopay_credentials.webhook_secret ELSE EXCLUDED.webhook_secret END,
			updated_at = CURRENT_TIMESTAMP
	`
	_, err := r.db.ExecContext(ctx, query, merchantID, apiKey, webhookSecret, apiKey == nil, webhookSecret == nil)
	if err != nil {
		return fmt.Errorf("failed to upsert gopay credentials: %w", err)
	}
	return nil
}
