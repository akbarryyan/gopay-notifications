package httpapi

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"
)

type eventJSON struct {
	EventID     string          `json:"event_id"`
	DeviceID    string          `json:"device_id"`
	Source      string          `json:"source"`
	PackageName string          `json:"package_name"`
	Title       *string         `json:"title"`
	Text        *string         `json:"text"`
	BigText     *string         `json:"big_text"`
	AmountHint  *int64          `json:"amount_hint"`
	PostedAt    string          `json:"posted_at"`
	ReceivedAt  string          `json:"received_at"`
	RawPayload  json.RawMessage `json:"raw_payload"`
}

type eventsListResponse struct {
	Events []eventJSON `json:"events"`
}

// handleEvents dipakai untuk verifikasi manual: dibuka di browser lewat Caddy
// yang melindunginya dengan basic auth. Bukan halaman admin.
func (a *API) handleEvents(w http.ResponseWriter, r *http.Request) {
	limit, err := intParam(r, "limit", 50)
	if err != nil || limit < 1 || limit > 1000 {
		a.writeError(w, http.StatusBadRequest, "invalid_payload", "limit harus bilangan bulat 1..1000")
		return
	}
	offset, err := intParam(r, "offset", 0)
	if err != nil || offset < 0 {
		a.writeError(w, http.StatusBadRequest, "invalid_payload", "offset harus bilangan bulat >= 0")
		return
	}

	events, err := a.store.ListEvents(r.Context(), limit, offset)
	if err != nil {
		slog.Error("ambil events gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}

	// Selalu array, tidak pernah null.
	out := make([]eventJSON, 0, len(events))
	for _, e := range events {
		out = append(out, eventJSON{
			EventID:     e.EventID,
			DeviceID:    e.DeviceID,
			Source:      e.Source,
			PackageName: e.PackageName,
			Title:       e.Title,
			Text:        e.BodyText,
			BigText:     e.BigText,
			AmountHint:  e.AmountHint,
			PostedAt:    e.PostedAt.Format(time.RFC3339),
			ReceivedAt:  e.ReceivedAt.Format(time.RFC3339),
			RawPayload:  json.RawMessage(e.RawPayload),
		})
	}

	writeJSON(w, http.StatusOK, eventsListResponse{Events: out})
}

func intParam(r *http.Request, name string, def int) (int, error) {
	raw := r.URL.Query().Get(name)
	if raw == "" {
		return def, nil
	}
	return strconv.Atoi(raw)
}
