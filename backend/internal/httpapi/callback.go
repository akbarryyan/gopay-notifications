package httpapi

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"regexp"
	"time"

	"github.com/akbarryyan/gopay-notifications/backend/internal/connector"
	"github.com/akbarryyan/gopay-notifications/backend/internal/store"
)

// eventIDPattern mengikat bentuk event_id ke formula di api-contract.md §4.2.
var eventIDPattern = regexp.MustCompile(`^evt_[0-9a-f]{32}$`)

type callbackRequest struct {
	EventID      string `json:"event_id"`
	DeviceID     string `json:"device_id"`
	Source       string `json:"source"`
	Notification struct {
		PackageName string  `json:"package_name"`
		Title       *string `json:"title"`
		Text        *string `json:"text"`
		BigText     *string `json:"big_text"`
		PostedAt    int64   `json:"posted_at"`
	} `json:"notification"`
	AmountHint *int64 `json:"amount_hint"`
	ReceivedAt string `json:"received_at"`
}

type callbackResponse struct {
	Success bool   `json:"success"`
	EventID string `json:"event_id"`
	Status  string `json:"status"`
}

func (a *API) handleCallback(w http.ResponseWriter, r *http.Request) {
	device, ok := DeviceFromContext(r.Context())
	if !ok {
		a.writeError(w, http.StatusInternalServerError, "internal", "device tidak ada di context")
		return
	}
	raw, ok := RawBodyFromContext(r.Context())
	if !ok {
		a.writeError(w, http.StatusInternalServerError, "internal", "body tidak ada di context")
		return
	}

	var req callbackRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		a.writeError(w, http.StatusBadRequest, "invalid_payload", "JSON tidak dapat dibaca")
		return
	}

	if !eventIDPattern.MatchString(req.EventID) {
		a.writeError(w, http.StatusBadRequest, "invalid_payload",
			"event_id harus berbentuk evt_ diikuti 32 karakter heksadesimal")
		return
	}
	if req.DeviceID != device.DeviceID {
		a.writeError(w, http.StatusBadRequest, "invalid_payload",
			"device_id di body tidak sama dengan device yang terautentikasi")
		return
	}
	// Divalidasi terhadap registry connector, bukan literal. Sumber yang
	// belum diimplementasikan harus ditolak, bukan diterima lalu diam-diam
	// tidak pernah dicocokkan.
	if !connector.IsKnown(req.Source) {
		a.writeError(w, http.StatusBadRequest, "invalid_payload",
			"source tidak dikenal: "+req.Source)
		return
	}
	if req.Notification.PackageName == "" {
		a.writeError(w, http.StatusBadRequest, "invalid_payload", "package_name wajib diisi")
		return
	}
	if req.Notification.PostedAt <= 0 {
		a.writeError(w, http.StatusBadRequest, "invalid_payload",
			"posted_at wajib Unix epoch milidetik yang positif")
		return
	}

	receivedAt, err := time.Parse(time.RFC3339, req.ReceivedAt)
	if err != nil {
		a.writeError(w, http.StatusBadRequest, "invalid_payload",
			"received_at wajib RFC3339 dengan offset zona waktu")
		return
	}

	// raw disimpan apa adanya, bukan hasil re-encode: bentuk asli notifikasi
	// akan dibutuhkan saat sub-project 3 menyusun aturan matching.
	inserted, err := a.store.InsertEvent(r.Context(), store.Event{
		EventID:     req.EventID,
		AccountID:   device.AccountID,
		DeviceID:    device.DeviceID,
		Source:      req.Source,
		PackageName: req.Notification.PackageName,
		Title:       req.Notification.Title,
		BodyText:    req.Notification.Text,
		BigText:     req.Notification.BigText,
		AmountHint:  req.AmountHint,
		PostedAt:    time.UnixMilli(req.Notification.PostedAt),
		ReceivedAt:  receivedAt,
		RawPayload:  raw,
	})
	if err != nil {
		slog.Error("simpan event gagal", "event_id", req.EventID, "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}

	status := "duplicate"
	if inserted {
		status = "accepted"

		// Matching hanya untuk event yang benar-benar baru — event duplikat
		// sudah dicocokkan (atau memang tidak cocok) saat pertama kali
		// masuk, mengulanginya cuma kerja sia-sia. MatchEvent sendiri yang
		// melewati amount_hint nil, jadi tidak perlu dicek di sini.
		matchedInvoiceID, err := a.store.MatchEvent(r.Context(), a.now(), device.AccountID, req.EventID, req.AmountHint)
		if err != nil {
			// Event sudah tersimpan — kegagalan matching bukan alasan
			// membalas gagal ke device, yang tidak tahu-menahu soal invoice.
			slog.Error("matching invoice gagal", "event_id", req.EventID, "err", err)
		} else if matchedInvoiceID != "" {
			// Webhook dikirim di goroutine terpisah dengan context sendiri
			// (bukan r.Context()) — request ini boleh selesai dan koneksinya
			// ditutup tanpa ikut membatalkan pengiriman webhook yang mungkin
			// masih berlangsung ke server merchant.
			go a.triggerInvoiceWebhook(device.AccountID, store.WebhookEventInvoicePaid, matchedInvoiceID)
		}
	}
	slog.Info("event diterima", "event_id", req.EventID, "device_id", device.DeviceID, "status", status)

	writeJSON(w, http.StatusOK, callbackResponse{
		Success: true,
		EventID: req.EventID,
		Status:  status,
	})
}
