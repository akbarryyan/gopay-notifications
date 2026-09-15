package gopay

// invoiceResponse -- bentuk respons POST/GET /api/v1/invoices gopay-notifications.
// Field yang tidak dipakai adapter ini (matched_event_id, paid_at) tetap
// didekode supaya unmarshal tidak gagal, meski tidak dipetakan ke ProviderPaymentResponse.
type invoiceResponse struct {
	ID              string  `json:"id"`
	ExternalRef     string  `json:"external_ref"`
	RequestedAmount int64   `json:"requested_amount"`
	UniqueAmount    int64   `json:"unique_amount"`
	Status          string  `json:"status"`
	MatchedEventID  *string `json:"matched_event_id"`
	CreatedAt       string  `json:"created_at"`
	ExpiresAt       string  `json:"expires_at"`
	PaidAt          *string `json:"paid_at"`
	QRISImage       *string `json:"qris_image"`
}

type errorResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
	Message string `json:"message"`
}

// webhookPayload -- bentuk body POST {APP_BASE_URL}/api/v1/provider-webhooks/gopay
// dari gopay-notifications (lihat API Docs gopay-notifications §Webhook).
type webhookPayload struct {
	Event   string `json:"event"`
	Invoice struct {
		ID          string  `json:"id"`
		ExternalRef string  `json:"external_ref"`
		Status      string  `json:"status"`
		PaidAt      *string `json:"paid_at"`
	} `json:"invoice"`
	SentAt string `json:"sent_at"`
}
