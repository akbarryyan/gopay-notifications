// Package httpapi berisi seluruh handler HTTP layanan ingestion.
package httpapi

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/akbar/gopay-notifications/backend/internal/store"
)

type API struct {
	store  *store.Store
	encKey []byte
	now    func() time.Time
}

// New membuat API. Parameter now disuntikkan agar test dapat memalsukan jam.
func New(s *store.Store, encKey []byte, now func() time.Time) *API {
	if now == nil {
		now = time.Now
	}
	return &API{store: s, encKey: encKey, now: now}
}

func (a *API) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/health", a.handleHealth)
	mux.Handle("GET /api/v1/device/me", a.requireDevice(http.HandlerFunc(a.handleDeviceMe)))
	return mux
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("tulis response gagal", "err", err)
	}
}

// errorResponse adalah bentuk baku response gagal, sesuai api-contract.md §4.5.
type errorResponse struct {
	Success    bool   `json:"success"`
	Error      string `json:"error"`
	Message    string `json:"message"`
	ServerTime int64  `json:"server_time,omitempty"`
}

func (a *API) writeError(w http.ResponseWriter, status int, code, msg string) {
	resp := errorResponse{Success: false, Error: code, Message: msg}
	if code == "clock_skew" {
		resp.ServerTime = a.now().Unix()
	}
	writeJSON(w, status, resp)
}
