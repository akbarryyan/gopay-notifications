package httpapi

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/akbarryyan/gopay-notifications/backend/internal/store"
)

var validActivityActions = map[string]bool{
	store.ActivityLoginSuccess:    true,
	store.ActivityLoginFailed:     true,
	store.ActivityPasswordChanged: true,
	store.ActivityPasswordReset:   true,
	store.ActivityAPIKeyCreated:   true,
	store.ActivityAPIKeyRevoked:   true,
	store.ActivityDeviceAdded:     true,
	store.ActivityDeviceDeleted:   true,
}

type activityLogJSON struct {
	ID        int64           `json:"id"`
	Action    string          `json:"action"`
	IPAddress *string         `json:"ip_address"`
	UserAgent *string         `json:"user_agent"`
	Metadata  json.RawMessage `json:"metadata"`
	CreatedAt string          `json:"created_at"`
}

// handleAdminActivityLog melayani halaman Logs Customer Dashboard --
// SELALU di-scope account dari sesi (accountID dari AccountFromContext),
// beda dari GET /vendor/notification-log yang sengaja lintas account.
func (a *API) handleAdminActivityLog(w http.ResponseWriter, r *http.Request) {
	accountID, _ := AccountFromContext(r.Context())

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

	var filter store.ActivityLogFilter
	var msg string
	if filter.Actions, msg = csvListParam(r, "action", validActivityActions); msg != "" {
		a.writeError(w, http.StatusBadRequest, "invalid_payload", msg)
		return
	}
	if raw := r.URL.Query().Get("from"); raw != "" {
		from, err := time.Parse("2006-01-02", raw)
		if err != nil {
			a.writeError(w, http.StatusBadRequest, "invalid_payload", "from harus YYYY-MM-DD")
			return
		}
		filter.From = &from
	}
	if raw := r.URL.Query().Get("to"); raw != "" {
		to, err := time.Parse("2006-01-02", raw)
		if err != nil {
			a.writeError(w, http.StatusBadRequest, "invalid_payload", "to harus YYYY-MM-DD")
			return
		}
		to = to.Add(24*time.Hour - time.Nanosecond)
		filter.To = &to
	}

	rows, err := a.store.ListActivityLog(r.Context(), accountID, limit, offset, filter)
	if err != nil {
		slog.Error("ambil activity log gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}

	out := make([]activityLogJSON, 0, len(rows))
	for _, row := range rows {
		out = append(out, activityLogJSON{
			ID: row.ID, Action: row.Action, IPAddress: row.IPAddress, UserAgent: row.UserAgent,
			Metadata: row.Metadata, CreatedAt: row.CreatedAt.Format(time.RFC3339),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "activity": out})
}
