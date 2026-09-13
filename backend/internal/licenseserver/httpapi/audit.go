package httpapi

import (
	"log/slog"
	"net/http"
	"time"
)

type auditEntryJSON struct {
	ID        int64  `json:"id"`
	Actor     string `json:"actor"`
	Action    string `json:"action"`
	Resource  string `json:"resource"`
	Metadata  any    `json:"metadata"`
	CreatedAt string `json:"created_at"`
}

func (a *API) handleListAuditLog(w http.ResponseWriter, r *http.Request) {
	entries, err := a.store.ListAuditLog(r.Context(), 100)
	if err != nil {
		slog.Error("list audit log gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}
	out := make([]auditEntryJSON, 0, len(entries))
	for _, e := range entries {
		out = append(out, auditEntryJSON{
			ID: e.ID, Actor: e.Actor, Action: e.Action, Resource: e.Resource,
			Metadata: e.Metadata, CreatedAt: e.CreatedAt.Format(time.RFC3339),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "entries": out})
}
