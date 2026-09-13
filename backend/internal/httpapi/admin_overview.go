package httpapi

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/akbarryyan/gopay-notifications/backend/internal/store"
)

type overviewResponse struct {
	Devices struct {
		Total    int `json:"total"`
		Online   int `json:"online"`
		Offline  int `json:"offline"`
		Disabled int `json:"disabled"`
		Pending  int `json:"pending"`
	} `json:"devices"`
	Events struct {
		Today     int              `json:"today"`
		Last7Days int              `json:"last_7_days"`
		LatestAt  *string          `json:"latest_at"`
		Daily     []dailyCountJSON `json:"daily"`
	} `json:"events"`
	// System selalu "operational" bila endpoint ini sempat menjawab — ia
	// menjawab dari proses backend dan database yang sama yang dipakai
	// seluruh sistem. Field ini tetap eksplisit di response supaya bentuk
	// JSON dashboard tidak perlu berubah saat komponen lain (mis. webhook
	// worker di sub-project 3) ditambahkan ke sini.
	System struct {
		Backend  string `json:"backend"`
		Database string `json:"database"`
	} `json:"system"`
}

type dailyCountJSON struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}

// overviewTrendDays adalah panjang grafik tren di Overview. 14 hari cukup
// untuk terasa seperti tren tanpa membuat titiknya terlalu rapat di layar
// sempit.
const overviewTrendDays = 14

// handleAdminOverview meringkas kondisi sistem untuk halaman Overview.
func (a *API) handleAdminOverview(w http.ResponseWriter, r *http.Request) {
	accountID, _ := AccountFromContext(r.Context())
	devices, err := a.store.ListDevices(r.Context(), accountID)
	if err != nil {
		slog.Error("ambil devices gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}

	stats, err := a.store.EventStats(r.Context(), accountID, a.now())
	if err != nil {
		slog.Error("ambil event stats gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}

	daily, err := a.store.DailyEventCounts(r.Context(), accountID, a.now(), overviewTrendDays)
	if err != nil {
		slog.Error("ambil daily event counts gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}

	var resp overviewResponse
	resp.Devices.Total = len(devices)
	for _, d := range devices {
		switch store.StatusOf(d, a.now()) {
		case store.DeviceOnline:
			resp.Devices.Online++
		case store.DeviceOffline:
			resp.Devices.Offline++
		case store.DeviceDisabled:
			resp.Devices.Disabled++
		case store.DevicePending:
			resp.Devices.Pending++
		}
	}

	resp.Events.Today = stats.Today
	resp.Events.Last7Days = stats.Last7Days
	if stats.LatestEventAt != nil {
		s := stats.LatestEventAt.Format(time.RFC3339)
		resp.Events.LatestAt = &s
	}
	resp.Events.Daily = make([]dailyCountJSON, len(daily))
	for i, d := range daily {
		resp.Events.Daily[i] = dailyCountJSON{Date: d.Date.Format("2006-01-02"), Count: d.Count}
	}

	resp.System.Backend = "operational"
	resp.System.Database = "operational"

	writeJSON(w, http.StatusOK, resp)
}
