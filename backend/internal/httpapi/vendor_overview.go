package httpapi

import (
	"log/slog"
	"net/http"
)

// vendorOverviewTrendDays adalah panjang grafik tren di Dashboard Vendor --
// sama seperti overviewTrendDays di admin_overview.go (Customer Dashboard),
// 14 hari cukup terasa seperti tren tanpa titik yang terlalu rapat.
const vendorOverviewTrendDays = 14

type vendorOverviewResponse struct {
	Accounts struct {
		Total       int `json:"total"`
		Active      int `json:"active"`
		Expiring    int `json:"expiring"`
		Expired     int `json:"expired"`
		Suspended   int `json:"suspended"`
		Revoked     int `json:"revoked"`
		NewThisWeek int `json:"new_this_week"`
	} `json:"accounts"`
	Devices struct {
		Total int `json:"total"`
	} `json:"devices"`
	Revenue struct {
		// TotalPaidRp: total nominal invoice yang pernah lunas sepanjang
		// waktu, lintas seluruh account.
		TotalPaidRp int64 `json:"total_paid_rp"`
	} `json:"revenue"`
	Daily []vendorDailyStatJSON `json:"daily"`
	// RecentAccounts: account yang paling baru dibuat, terbanyak
	// vendorRecentAccountsLimit baris -- dipakai kartu "Customer Terbaru"
	// di Dashboard. Daftar lengkapnya ada di halaman Accounts.
	RecentAccounts []accountVendorJSON `json:"recent_accounts"`
}

// vendorRecentAccountsLimit membatasi kartu "Customer Terbaru" supaya
// tetap ringkas.
const vendorRecentAccountsLimit = 5

type vendorDailyStatJSON struct {
	Date         string `json:"date"`
	NewAccounts  int    `json:"new_accounts"`
	PaidAmountRp int64  `json:"paid_amount_rp"`
}

// handleVendorOverview meringkas seluruh platform (lintas account) untuk
// halaman Dashboard di Vendor Dashboard -- beda dari handleAdminOverview
// (Customer Dashboard) yang selalu di-scope satu account dari sesi login.
func (a *API) handleVendorOverview(w http.ResponseWriter, r *http.Request) {
	stats, err := a.store.VendorOverviewStats(r.Context(), a.now())
	if err != nil {
		slog.Error("ambil vendor overview stats gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}

	daily, err := a.store.VendorDailyStats(r.Context(), a.now(), vendorOverviewTrendDays)
	if err != nil {
		slog.Error("ambil vendor daily stats gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}

	// Diambil terpisah dari VendorOverviewStats (yang juga memanggil
	// ListAccounts secara internal untuk hitungan status) -- Dashboard
	// jarang dibuka (satu vendor, bukan trafik tinggi), jadi query ganda
	// yang murah ini lebih sederhana daripada mengubah bentuk
	// VendorOverviewStats supaya mengembalikan daftarnya juga.
	accounts, err := a.store.ListAccounts(r.Context())
	if err != nil {
		slog.Error("ambil recent accounts gagal", "err", err)
		a.writeError(w, http.StatusInternalServerError, "internal", "kesalahan internal")
		return
	}

	var resp vendorOverviewResponse
	resp.Accounts.Total = stats.TotalAccounts
	resp.Accounts.Active = stats.Active
	resp.Accounts.Expiring = stats.Expiring
	resp.Accounts.Expired = stats.Expired
	resp.Accounts.Suspended = stats.Suspended
	resp.Accounts.Revoked = stats.Revoked
	resp.Accounts.NewThisWeek = stats.NewThisWeek
	resp.Devices.Total = stats.TotalDevices
	resp.Revenue.TotalPaidRp = stats.TotalPaidAmountRp

	resp.Daily = make([]vendorDailyStatJSON, len(daily))
	for i, d := range daily {
		resp.Daily[i] = vendorDailyStatJSON{
			Date: d.Date.Format(dateOnlyLayout), NewAccounts: d.NewAccounts, PaidAmountRp: d.PaidAmountRp,
		}
	}

	// ListAccounts sudah terurut created_at DESC -- ambil sejumlah
	// vendorRecentAccountsLimit baris pertama.
	limit := min(vendorRecentAccountsLimit, len(accounts))
	resp.RecentAccounts = make([]accountVendorJSON, limit)
	for i := range limit {
		resp.RecentAccounts[i] = toAccountVendorJSON(accounts[i], a.now())
	}

	writeJSON(w, http.StatusOK, map[string]any{"success": true, "overview": resp})
}
