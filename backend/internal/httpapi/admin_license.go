package httpapi

import (
	"net/http"

	"github.com/akbarryyan/gopay-notifications/backend/internal/licensecheck"
)

// requireLicense menolak request kalau lisensi instalasi ini tidak aktif.
// Ditaruh paling luar (sebelum requireDevice/requireAdmin/requireAPIKey)
// karena pengecekannya cuma baca status di memori — murah dibanding
// membuka sesi/DB dulu untuk kemudian tetap ditolak. Lihat
// docs/superpowers/specs/2026-09-13-license-system-design.md §5.
func (a *API) requireLicense(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if a.license.Status != licensecheck.StatusActive {
			a.writeError(w, http.StatusPaymentRequired,
				"license_"+string(a.license.Status), licenseErrorMessage(a.license.Status))
			return
		}
		next.ServeHTTP(w, r)
	})
}

func licenseErrorMessage(status licensecheck.Status) string {
	switch status {
	case licensecheck.StatusMissing:
		return "lisensi belum terpasang di instalasi ini"
	case licensecheck.StatusExpired:
		return "lisensi instalasi ini sudah kedaluwarsa"
	case licensecheck.StatusInvalid:
		return "lisensi instalasi ini tidak valid"
	default:
		return "lisensi instalasi ini tidak aktif"
	}
}

type licenseJSON struct {
	Customer      string `json:"customer,omitempty"`
	Domain        string `json:"domain,omitempty"`
	Plan          string `json:"plan,omitempty"`
	IssuedAt      string `json:"issued_at,omitempty"`
	ExpiresAt     string `json:"expires_at,omitempty"`
	DaysRemaining *int   `json:"days_remaining,omitempty"`
	Status        string `json:"status"`
	Reason        string `json:"reason,omitempty"`
}

// handleAdminLicense mengembalikan status lisensi instalasi ini. SENGAJA
// tidak dibungkus requireLicense — admin harus selalu bisa melihat kenapa
// lisensinya tidak aktif, bukan cuma saat aktif.
func (a *API) handleAdminLicense(w http.ResponseWriter, r *http.Request) {
	lic := a.license
	out := licenseJSON{Status: string(lic.Status), Reason: lic.Reason}

	// Detail lisensi (customer/domain/plan/tanggal) cuma ditampilkan kalau
	// ada sesuatu yang benar-benar valid untuk ditunjukkan — active atau
	// expired (signature dan domain sama-sama sudah lolos verifikasi,
	// tinggal tanggalnya yang lewat). missing dan invalid tidak punya data
	// yang bisa dipercaya untuk ditampilkan.
	if lic.Status == licensecheck.StatusActive || lic.Status == licensecheck.StatusExpired {
		out.Customer = lic.Customer
		out.Domain = lic.Domain
		out.Plan = lic.Plan
		out.IssuedAt = lic.IssuedAt.Format(dateOnlyLayout)
		out.ExpiresAt = lic.ExpiresAt.Format(dateOnlyLayout)
		days := lic.DaysRemaining(a.now())
		out.DaysRemaining = &days
	}

	writeJSON(w, http.StatusOK, map[string]any{"success": true, "license": out})
}

const dateOnlyLayout = "2006-01-02"
