package httpapi

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/akbarryyan/gopay-notifications/backend/internal/licenseserver/store"
)

type installationJSON struct {
	ID             string  `json:"id"`
	Environment    string  `json:"environment"`
	ProductVersion string  `json:"product_version"`
	ActivatedAt    string  `json:"activated_at"`
	ReleasedAt     *string `json:"released_at"`
}

func toInstallationJSON(inst store.Installation) installationJSON {
	out := installationJSON{
		ID:             inst.ID,
		Environment:    inst.Environment,
		ProductVersion: inst.ProductVersion,
		ActivatedAt:    inst.ActivatedAt.Format(time.RFC3339),
	}
	if inst.ReleasedAt != nil {
		s := inst.ReleasedAt.Format(time.RFC3339)
		out.ReleasedAt = &s
	}
	return out
}

// handleResetInstallation ("Reset Installation", §22 license-spec.md) —
// dipakai saat customer migrasi VPS: installation lama dilepas, membuka
// kuota supaya installation baru bisa diaktivasi dengan key yang sama.
func (a *API) handleResetInstallation(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("installationID")
	if err := a.store.ResetInstallation(r.Context(), id); errors.Is(err, store.ErrInstallationNotFound) {
		a.writeError(w, http.StatusNotFound, "not_found", "installation tidak ditemukan atau sudah direset")
		return
	} else if err != nil {
		slog.Error("reset installation gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}
	if err := a.store.LogAudit(r.Context(), "vendor", "INSTALLATION_RESET", id, nil); err != nil {
		slog.Error("log audit gagal", "err", err)
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}
