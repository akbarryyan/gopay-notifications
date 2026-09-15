package gopay

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	domainProvider "github.com/akbarryyan/pg-aggregator-back/internal/domain/provider"
	providerPkg "github.com/akbarryyan/pg-aggregator-back/internal/provider"
	"github.com/akbarryyan/pg-aggregator-back/internal/repository"
	"github.com/google/uuid"
)

// credentialsRepository -- interface lokal ke package ini (bukan tipe
// konkret repository), supaya adapter bisa diuji dengan fake tanpa
// database sungguhan. *repository.MerchantGopayCredentialsRepository
// memenuhi ini secara struktural.
type credentialsRepository interface {
	Get(ctx context.Context, merchantID uuid.UUID) (*repository.GopayCredentials, error)
}

const ProviderName = "gopay"

type Adapter struct {
	baseURL    string
	credsRepo  credentialsRepository
	httpClient *http.Client
}

func NewAdapter(baseURL string, credsRepo credentialsRepository) *Adapter {
	return &Adapter{
		baseURL:    strings.TrimRight(baseURL, "/"),
		credsRepo:  credsRepo,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

func (a *Adapter) GetName() string { return ProviderName }

func (a *Adapter) CreatePayment(ctx context.Context, req *domainProvider.ProviderPaymentRequest) (*domainProvider.ProviderPaymentResponse, error) {
	creds, err := a.credsRepo.Get(ctx, req.MerchantID)
	if err != nil || creds.APIKey == nil || *creds.APIKey == "" {
		return nil, ErrCredentialsNotConfigured
	}

	body, _ := json.Marshal(map[string]interface{}{
		"external_ref": req.InternalReference,
		"amount":       req.Amount,
	})
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, a.baseURL+"/api/v1/invoices", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("gopay: build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+*creds.APIKey)

	resp, err := a.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("gopay: create invoice: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == http.StatusConflict {
		var errResp errorResponse
		if json.Unmarshal(respBody, &errResp) == nil && errResp.Error == "qris_not_configured" {
			return nil, ErrQRISNotConfigured
		}
		return nil, fmt.Errorf("gopay: create invoice ditolak: %s", string(respBody))
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("gopay: create invoice status %d: %s", resp.StatusCode, string(respBody))
	}

	var inv invoiceResponse
	if err := json.Unmarshal(respBody, &inv); err != nil {
		return nil, fmt.Errorf("gopay: decode invoice response: %w", err)
	}

	expiresAt, err := time.Parse(time.RFC3339, inv.ExpiresAt)
	if err != nil {
		return nil, fmt.Errorf("gopay: parse expires_at: %w", err)
	}

	return &domainProvider.ProviderPaymentResponse{
		ProviderReference: inv.ID,
		ProviderName:      ProviderName,
		Status:            "pending",
		Amount:            inv.UniqueAmount,
		QRISData:          inv.QRISImage,
		ExpiresAt:         expiresAt,
		RawResponse:       map[string]interface{}{"external_ref": inv.ExternalRef},
	}, nil
}

func (a *Adapter) GetPaymentStatus(ctx context.Context, providerReference string, merchantID uuid.UUID) (*domainProvider.NormalizedPaymentStatus, error) {
	creds, err := a.credsRepo.Get(ctx, merchantID)
	if err != nil || creds.APIKey == nil || *creds.APIKey == "" {
		return nil, ErrCredentialsNotConfigured
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, a.baseURL+"/api/v1/invoices/"+providerReference, nil)
	if err != nil {
		return nil, fmt.Errorf("gopay: build request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+*creds.APIKey)

	resp, err := a.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("gopay: get invoice: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gopay: get invoice status %d: %s", resp.StatusCode, string(respBody))
	}

	var inv invoiceResponse
	if err := json.Unmarshal(respBody, &inv); err != nil {
		return nil, fmt.Errorf("gopay: decode invoice response: %w", err)
	}

	return &domainProvider.NormalizedPaymentStatus{
		Status:            a.NormalizeStatus(inv.Status),
		ProviderReference: inv.ID,
	}, nil
}

func (a *Adapter) ValidateWebhook(rawPayload []byte, signature string, merchantID uuid.UUID) error {
	if signature == "" {
		return providerPkg.ErrInvalidWebhookSignature
	}
	creds, err := a.credsRepo.Get(context.Background(), merchantID)
	if err != nil || creds.WebhookSecret == nil || *creds.WebhookSecret == "" {
		return providerPkg.ErrInvalidWebhookSignature
	}

	expected := computeHMAC(rawPayload, *creds.WebhookSecret)
	if !hmac.Equal([]byte(expected), []byte(signature)) {
		return providerPkg.ErrInvalidWebhookSignature
	}
	return nil
}

func (a *Adapter) ParseWebhook(rawPayload []byte) (*domainProvider.ProviderWebhookPayload, error) {
	var wh webhookPayload
	if err := json.Unmarshal(rawPayload, &wh); err != nil {
		return nil, fmt.Errorf("gopay: decode webhook payload: %w", err)
	}

	var paidAt *time.Time
	if wh.Invoice.PaidAt != nil {
		t, err := time.Parse(time.RFC3339, *wh.Invoice.PaidAt)
		if err == nil {
			paidAt = &t
		}
	}

	return &domainProvider.ProviderWebhookPayload{
		ProviderName:      ProviderName,
		ProviderReference: wh.Invoice.ID,
		Status:            a.NormalizeStatus(wh.Invoice.Status),
		PaidAt:            paidAt,
		RawPayload:        map[string]interface{}{"event": wh.Event, "external_ref": wh.Invoice.ExternalRef},
	}, nil
}

func (a *Adapter) NormalizeStatus(providerStatus string) string {
	switch strings.ToUpper(providerStatus) {
	case "PAID":
		return "paid"
	case "EXPIRED":
		return "expired"
	default:
		return "pending"
	}
}

// computeHMAC dipakai test (dan boleh dipakai ulang siapa pun yang perlu
// menghasilkan tanda tangan uji secara manual).
func computeHMAC(payload []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}
