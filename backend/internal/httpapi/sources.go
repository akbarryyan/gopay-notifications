package httpapi

import (
	"net/http"

	"github.com/akbarryyan/gopay-notifications/backend/internal/connector"
)

type sourcesResponse struct {
	Sources []connector.Info `json:"sources"`
}

// handleSources mengembalikan sumber pembayaran yang dikenal build ini.
//
// Dipakai dashboard agar halaman Payment Sources tidak perlu menanam daftar
// connector di sisi klien — daftar itu milik backend, dan ia berubah tiap
// connector baru ditambahkan.
func (a *API) handleSources(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, sourcesResponse{Sources: connector.All()})
}
