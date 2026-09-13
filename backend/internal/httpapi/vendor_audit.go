package httpapi

import (
	"net/http"
	"time"
)

type auditEntryJSON struct {
	ID        int64  `json:"id"`
	Actor     string `json:"actor"`
	Action    string `json:"action"`
	Resource  string `json:"resource"`
	CreatedAt string `json:"created_at"`
}

func (a *API) handleVendorAuditLog(w http.ResponseWriter, r *http.Request) {
	entries, err := a.store.ListAuditLog(r.Context(), 100, 0)
	if err != nil {
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}
	out := make([]auditEntryJSON, 0, len(entries))
	for _, e := range entries {
		out = append(out, auditEntryJSON{
			ID: e.ID, Actor: e.Actor, Action: e.Action, Resource: e.Resource,
			CreatedAt: e.CreatedAt.Format(time.RFC3339),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "entries": out})
}
