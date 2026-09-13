package httpapi

import (
	"net/http"
	"time"

	"github.com/akbarryyan/gopay-notifications/backend/internal/licensecheck"
)

// requireLicense menolak request kalau lisensi instalasi ini tidak
// operasional (aktif/akan berakhir). Ditaruh paling luar (sebelum
// requireDevice/requireAdmin/requireAPIKey) karena pengecekannya cuma baca
// file lokal di memori — murah dibanding membuka sesi/DB dulu untuk
// kemudian tetap ditolak. Lihat
// docs/superpowers/specs/2026-09-13-online-license-platform-design.md §7.
func (a *API) requireLicense(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lic := a.loadLicense()
		if !lic.Status.Operational() {
			a.writeError(w, http.StatusPaymentRequired,
				"license_"+string(lic.Status), licenseErrorMessage(lic.Status))
			return
		}
		next.ServeHTTP(w, r)
	})
}

func licenseErrorMessage(status licensecheck.Status) string {
	switch status {
	case licensecheck.StatusMissing:
		return "lisensi belum diaktivasi di instalasi ini"
	case licensecheck.StatusExpired:
		return "lisensi instalasi ini sudah kedaluwarsa"
	case licensecheck.StatusSuspended:
		return "lisensi instalasi ini sedang disuspend"
	case licensecheck.StatusRevoked:
		return "lisensi instalasi ini sudah dicabut"
	case licensecheck.StatusUnreachable:
		return "instalasi ini sudah lama tidak berhasil menghubungi License Server"
	case licensecheck.StatusInvalid:
		return "lisensi instalasi ini tidak valid"
	default:
		return "lisensi instalasi ini tidak aktif"
	}
}

type licenseJSON struct {
	LicenseID      string `json:"license_id,omitempty"`
	InstallationID string `json:"installation_id,omitempty"`
	Customer       string `json:"customer,omitempty"`
	Plan           string `json:"plan,omitempty"`
	MaxDevices     int    `json:"max_devices,omitempty"`
	IssuedAt       string `json:"issued_at,omitempty"`
	ExpiresAt      string `json:"expires_at,omitempty"`
	ValidatedAt    string `json:"validated_at,omitempty"`
	DaysRemaining  *int   `json:"days_remaining,omitempty"`
	Status         string `json:"status"`
	Reason         string `json:"reason,omitempty"`
}

const dateOnlyLayout = "2006-01-02"

// handleAdminLicense mengembalikan status lisensi instalasi ini. SENGAJA
// tidak dibungkus requireLicense — admin harus selalu bisa melihat kenapa
// lisensinya tidak aktif, bukan cuma saat aktif. Read-only sepenuhnya:
// tidak ada endpoint untuk mengetik license key di sini, itu lewat
// LICENSE_KEY di .env (lihat internal/licenseclient dan §5 spec).
func (a *API) handleAdminLicense(w http.ResponseWriter, r *http.Request) {
	lic := a.loadLicense()
	out := licenseJSON{Status: string(lic.Status), Reason: lic.Reason}

	// Detail cuma ditampilkan kalau ada sesuatu yang benar-benar valid
	// untuk ditunjukkan (signature dan installation sudah lolos verifikasi)
	// — missing dan invalid tidak punya data yang bisa dipercaya.
	if lic.Status != licensecheck.StatusMissing && lic.Status != licensecheck.StatusInvalid {
		out.LicenseID = lic.LicenseID
		out.InstallationID = lic.InstallationID
		out.Customer = lic.Customer
		out.Plan = lic.Plan
		out.MaxDevices = lic.MaxDevices
		out.IssuedAt = lic.IssuedAt.Format(dateOnlyLayout)
		out.ExpiresAt = lic.ExpiresAt.Format(dateOnlyLayout)
		out.ValidatedAt = lic.ValidatedAt.Format(time.RFC3339)
		days := lic.DaysRemaining(a.now())
		out.DaysRemaining = &days
	}

	writeJSON(w, http.StatusOK, map[string]any{"success": true, "license": out})
}
