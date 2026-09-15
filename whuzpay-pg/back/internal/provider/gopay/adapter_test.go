package gopay

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	domainProvider "github.com/akbarryyan/pg-aggregator-back/internal/domain/provider"
	providerPkg "github.com/akbarryyan/pg-aggregator-back/internal/provider"
	"github.com/akbarryyan/pg-aggregator-back/internal/repository"
	"github.com/google/uuid"
)

type fakeCredsRepo struct {
	creds map[uuid.UUID]*repository.GopayCredentials
}

func (f *fakeCredsRepo) Get(ctx context.Context, merchantID uuid.UUID) (*repository.GopayCredentials, error) {
	c, ok := f.creds[merchantID]
	if !ok {
		return nil, repository.ErrGopayCredentialsNotFound
	}
	return c, nil
}

func TestCreatePayment_Success(t *testing.T) {
	merchantID := uuid.New()
	apiKey := "sk_test123"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+apiKey {
			t.Errorf("Authorization header = %q, want Bearer %s", r.Header.Get("Authorization"), apiKey)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":               "inv_abc123",
			"external_ref":     "ORDER-1",
			"requested_amount": 50000,
			"unique_amount":    50347,
			"status":           "PENDING",
			"created_at":       "2026-09-15T09:30:00Z",
			"expires_at":       "2026-09-15T09:45:00Z",
			"qris_image":       "data:image/png;base64,iVBORw0KGgo",
		})
	}))
	defer server.Close()

	repo := &fakeCredsRepo{creds: map[uuid.UUID]*repository.GopayCredentials{
		merchantID: {MerchantID: merchantID, APIKey: &apiKey},
	}}
	adapter := NewAdapter(server.URL, repo)

	resp, err := adapter.CreatePayment(context.Background(), &domainProvider.ProviderPaymentRequest{
		InternalReference: "ORDER-1",
		Amount:            50000,
		MerchantID:        merchantID,
	})
	if err != nil {
		t.Fatalf("CreatePayment: %v", err)
	}
	if resp.ProviderReference != "inv_abc123" {
		t.Errorf("ProviderReference = %q, want inv_abc123", resp.ProviderReference)
	}
	if resp.Amount != 50347 {
		t.Errorf("Amount = %d, want 50347 (unique_amount)", resp.Amount)
	}
	if resp.QRISData == nil || *resp.QRISData != "data:image/png;base64,iVBORw0KGgo" {
		t.Errorf("QRISData = %v, want the qris_image data URI", resp.QRISData)
	}
}

func TestCreatePayment_CredentialsNotConfigured(t *testing.T) {
	repo := &fakeCredsRepo{creds: map[uuid.UUID]*repository.GopayCredentials{}}
	adapter := NewAdapter("http://should-not-be-called.invalid", repo)

	_, err := adapter.CreatePayment(context.Background(), &domainProvider.ProviderPaymentRequest{
		InternalReference: "ORDER-1",
		Amount:            50000,
		MerchantID:        uuid.New(),
	})
	if err != ErrCredentialsNotConfigured {
		t.Errorf("err = %v, want ErrCredentialsNotConfigured", err)
	}
}

func TestCreatePayment_QRISNotConfigured(t *testing.T) {
	merchantID := uuid.New()
	apiKey := "sk_test123"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "qris_not_configured", "message": "QRIS belum diatur",
		})
	}))
	defer server.Close()

	repo := &fakeCredsRepo{creds: map[uuid.UUID]*repository.GopayCredentials{
		merchantID: {MerchantID: merchantID, APIKey: &apiKey},
	}}
	adapter := NewAdapter(server.URL, repo)

	_, err := adapter.CreatePayment(context.Background(), &domainProvider.ProviderPaymentRequest{
		InternalReference: "ORDER-1", Amount: 50000, MerchantID: merchantID,
	})
	if err != ErrQRISNotConfigured {
		t.Errorf("err = %v, want ErrQRISNotConfigured", err)
	}
}

func TestValidateWebhook_CorrectSignature(t *testing.T) {
	merchantID := uuid.New()
	secret := "whsec_abc123"
	repo := &fakeCredsRepo{creds: map[uuid.UUID]*repository.GopayCredentials{
		merchantID: {MerchantID: merchantID, WebhookSecret: &secret},
	}}
	adapter := NewAdapter("http://unused.invalid", repo)

	payload := []byte(`{"event":"invoice.paid","invoice":{"id":"inv_abc123"}}`)
	sig := computeHMAC(payload, secret)

	if err := adapter.ValidateWebhook(payload, sig, merchantID); err != nil {
		t.Errorf("ValidateWebhook: %v, want nil", err)
	}
}

func TestValidateWebhook_WrongSignature(t *testing.T) {
	merchantID := uuid.New()
	secret := "whsec_abc123"
	repo := &fakeCredsRepo{creds: map[uuid.UUID]*repository.GopayCredentials{
		merchantID: {MerchantID: merchantID, WebhookSecret: &secret},
	}}
	adapter := NewAdapter("http://unused.invalid", repo)

	payload := []byte(`{"event":"invoice.paid","invoice":{"id":"inv_abc123"}}`)
	err := adapter.ValidateWebhook(payload, "wrong-signature", merchantID)
	if err != providerPkg.ErrInvalidWebhookSignature {
		t.Errorf("err = %v, want ErrInvalidWebhookSignature", err)
	}
}

func TestValidateWebhook_SecretNotConfigured(t *testing.T) {
	merchantID := uuid.New()
	repo := &fakeCredsRepo{creds: map[uuid.UUID]*repository.GopayCredentials{}}
	adapter := NewAdapter("http://unused.invalid", repo)

	err := adapter.ValidateWebhook([]byte(`{}`), "any-signature", merchantID)
	if err != providerPkg.ErrInvalidWebhookSignature {
		t.Errorf("err = %v, want ErrInvalidWebhookSignature", err)
	}
}

func TestParseWebhook(t *testing.T) {
	adapter := NewAdapter("http://unused.invalid", &fakeCredsRepo{})
	payload := []byte(`{"event":"invoice.paid","invoice":{"id":"inv_abc123","external_ref":"ORDER-1","status":"PAID","paid_at":"2026-09-15T09:36:00Z"}}`)

	out, err := adapter.ParseWebhook(payload)
	if err != nil {
		t.Fatalf("ParseWebhook: %v", err)
	}
	if out.ProviderReference != "inv_abc123" {
		t.Errorf("ProviderReference = %q, want inv_abc123 (invoice.id, bukan external_ref)", out.ProviderReference)
	}
	if out.Status != "paid" {
		t.Errorf("Status = %q, want paid", out.Status)
	}
}

func TestNormalizeStatus(t *testing.T) {
	adapter := NewAdapter("http://unused.invalid", &fakeCredsRepo{})
	cases := map[string]string{"PAID": "paid", "EXPIRED": "expired", "PENDING": "pending", "unknown": "pending"}
	for in, want := range cases {
		if got := adapter.NormalizeStatus(in); got != want {
			t.Errorf("NormalizeStatus(%q) = %q, want %q", in, got, want)
		}
	}
}
