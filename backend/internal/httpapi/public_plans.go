package httpapi

import (
	"log/slog"
	"net/http"
	"strconv"
)

// publicPlanJSON SENGAJA tidak menyertakan field internal (id, visible,
// sort_order sudah tercermin dari urutan array, updated_at) -- ini
// dikonsumsi landing page publik tanpa auth, cukup yang benar-benar
// ditampilkan.
type publicPlanJSON struct {
	Name        string `json:"name"`
	PriceLabel  string `json:"price_label"`
	PricePeriod string `json:"price_period"`
	Description string `json:"description"`
	// DeviceLabel dihitung server-side dari max_devices (-1 => tanpa
	// batas) -- satu sumber kebenaran, bukan teks bebas yang bisa
	// diam-diam tidak sinkron dengan kuota device sungguhan.
	DeviceLabel string   `json:"device_label"`
	Features    []string `json:"features"`
	Highlighted bool     `json:"highlighted"`
}

func deviceLabel(maxDevices int) string {
	switch {
	case maxDevices < 0:
		return "Device tanpa batas"
	case maxDevices == 1:
		return "1 device"
	default:
		return strconv.Itoa(maxDevices) + " device"
	}
}

// handlePublicPricingPlans melayani section Harga di landing page publik
// (dashboard/, tanpa sesi apa pun) -- hanya plan visible=true, urutan
// sudah sesuai tampilan (ListVisiblePlans).
func (a *API) handlePublicPricingPlans(w http.ResponseWriter, r *http.Request) {
	plans, err := a.store.ListVisiblePlans(r.Context())
	if err != nil {
		slog.Error("ambil pricing plans gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}
	out := make([]publicPlanJSON, 0, len(plans))
	for _, p := range plans {
		out = append(out, publicPlanJSON{
			Name: p.Name, PriceLabel: p.PriceLabel, PricePeriod: p.PricePeriod,
			Description: p.Description, DeviceLabel: deviceLabel(p.MaxDevices),
			Features: p.Features, Highlighted: p.Highlighted,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "plans": out})
}
