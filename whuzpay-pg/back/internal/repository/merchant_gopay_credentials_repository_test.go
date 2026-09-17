package repository

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
)

var gopayCredsCols = []string{"api_key", "webhook_secret", "gopay_username", "qris_configured_at"}

func TestMerchantGopayCredentialsRepository_Get_Found(t *testing.T) {
	db, mock := newMockDB(t)
	repo := NewMerchantGopayCredentialsRepository(db)
	merchantID := uuid.New()
	apiKey, username := "sk_abc", "budi"
	qrisAt := time.Now()

	mock.ExpectQuery(`SELECT api_key, webhook_secret, gopay_username, qris_configured_at FROM merchant_gopay_credentials WHERE merchant_id = \$1`).
		WithArgs(merchantID).
		WillReturnRows(sqlmock.NewRows(gopayCredsCols).AddRow(apiKey, nil, username, qrisAt))

	got, err := repo.Get(context.Background(), merchantID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.APIKey == nil || *got.APIKey != apiKey {
		t.Errorf("APIKey = %v, want %q", got.APIKey, apiKey)
	}
	if got.WebhookSecret != nil {
		t.Errorf("WebhookSecret = %v, want nil", got.WebhookSecret)
	}
	if got.Username == nil || *got.Username != username {
		t.Errorf("Username = %v, want %q", got.Username, username)
	}
	if got.QRISConfiguredAt == nil {
		t.Error("QRISConfiguredAt = nil, want set")
	}
}

func TestMerchantGopayCredentialsRepository_Get_NotFound(t *testing.T) {
	db, mock := newMockDB(t)
	repo := NewMerchantGopayCredentialsRepository(db)
	merchantID := uuid.New()

	mock.ExpectQuery(`SELECT api_key, webhook_secret, gopay_username, qris_configured_at FROM merchant_gopay_credentials WHERE merchant_id = \$1`).
		WithArgs(merchantID).
		WillReturnError(sql.ErrNoRows)

	_, err := repo.Get(context.Background(), merchantID)
	if !errors.Is(err, ErrGopayCredentialsNotFound) {
		t.Fatalf("expected ErrGopayCredentialsNotFound, got %v", err)
	}
}

// TestMerchantGopayCredentialsRepository_Upsert_UsernameNilBerartiBiarkan
// mengunci pola tri-state: passing nil untuk username (persis yang
// dilakukan PaymentService.UpdateGopayCredentials, pemanggil lama sebelum
// kolom ini ada) tidak boleh menghapus username yang sudah tersimpan.
func TestMerchantGopayCredentialsRepository_Upsert_UsernameNilBerartiBiarkan(t *testing.T) {
	db, mock := newMockDB(t)
	repo := NewMerchantGopayCredentialsRepository(db)
	merchantID := uuid.New()
	apiKey := "sk_new"

	mock.ExpectExec(`INSERT INTO merchant_gopay_credentials`).
		WithArgs(merchantID, &apiKey, (*string)(nil), (*string)(nil), false, true, true).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := repo.Upsert(context.Background(), merchantID, &apiKey, nil, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestMerchantGopayCredentialsRepository_Upsert_SemuaFieldTerisi(t *testing.T) {
	db, mock := newMockDB(t)
	repo := NewMerchantGopayCredentialsRepository(db)
	merchantID := uuid.New()
	apiKey, webhookSecret, username := "sk_abc", "whsec_abc", "budi"

	mock.ExpectExec(`INSERT INTO merchant_gopay_credentials`).
		WithArgs(merchantID, &apiKey, &webhookSecret, &username, false, false, false).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := repo.Upsert(context.Background(), merchantID, &apiKey, &webhookSecret, &username); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMerchantGopayCredentialsRepository_MarkQRISConfigured(t *testing.T) {
	db, mock := newMockDB(t)
	repo := NewMerchantGopayCredentialsRepository(db)
	merchantID := uuid.New()

	mock.ExpectExec(`INSERT INTO merchant_gopay_credentials`).
		WithArgs(merchantID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := repo.MarkQRISConfigured(context.Background(), merchantID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}
