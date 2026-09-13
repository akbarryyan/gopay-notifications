// Package licensecheck memverifikasi local license state yang ditandatangani
// Ed25519 oleh License Server. Tidak menyentuh database maupun jaringan
// sendiri — sepenuhnya pure logic, sejalan dengan internal/auth dan
// internal/secretbox. Dipakai ulang oleh DUA sisi:
//
//   - backend/internal/licenseserver menandatangani (lihat issue.go) tiap
//     kali menjawab /activate atau /validate.
//   - backend customer (paket ini sendiri, lewat Load) memverifikasi file
//     lokal yang ditulis oleh internal/licenseclient.
//
// Lihat docs/superpowers/specs/2026-09-13-online-license-platform-design.md
// untuk rancangan lengkap (menggantikan versi offline murni di
// 2026-09-13-license-system-design.md).
package licensecheck

import (
	"bufio"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"os"
	"strings"
	"time"
)

// licensePublicKeyBase64 di-hardcode, BUKAN env var atau file yang bisa
// diedit operator server. Private key pasangannya HANYA ada di konfigurasi
// License Server (LICENSE_SIGNING_PRIVATE_KEY), tidak pernah masuk repo.
// Kalau kunci ini bisa dikonfigurasi lewat env/file di server customer,
// customer bisa membuat key pair sendiri dan memalsukan state aktif sendiri
// — meniadakan seluruh gunanya tanda tangan.
const licensePublicKeyBase64 = "HOdMzYphVIuVD6GXvSfOHcGYntPP94LhT4EujgBCfv0="

// WarningThresholdDays: ambang status berubah dari "active" ke "expiring".
const WarningThresholdDays = 30

// GracePeriodDays: berapa lama local state boleh dipakai tanpa validasi
// baru yang berhasil sebelum dianggap "unreachable" (§12 license-spec.md).
const GracePeriodDays = 7

type Status string

const (
	StatusActive      Status = "active"
	StatusExpiring    Status = "expiring"
	StatusExpired     Status = "expired"
	StatusSuspended   Status = "suspended"
	StatusRevoked     Status = "revoked"
	StatusMissing     Status = "missing"     // belum pernah aktivasi
	StatusInvalid     Status = "invalid"     // signature/format tidak valid
	StatusUnreachable Status = "unreachable" // basi, License Server tak terjangkau > grace period
)

// License adalah hasil pembacaan dan verifikasi satu local license state.
type License struct {
	LicenseID      string
	InstallationID string
	Customer       string
	Plan           string
	MaxDevices     int
	IssuedAt       time.Time
	ExpiresAt      time.Time
	// ValidatedAt: kapan License Server menghasilkan state ini (bukan kapan
	// file ini dibaca). Dipakai menghitung grace period.
	ValidatedAt time.Time
	Status      Status
	// Reason menjelaskan status non-active secara spesifik, untuk
	// ditampilkan di halaman dashboard /license. Kosong bila Status ==
	// StatusActive.
	Reason string
}

// DaysRemaining menghitung selisih hari ke ExpiresAt dari now, dibulatkan ke
// bawah. Bisa negatif kalau sudah lewat.
func (l License) DaysRemaining(now time.Time) int {
	return int(l.ExpiresAt.Sub(now).Hours() / 24)
}

// adminStatus adalah status administratif mentah dari database License
// Server — cuma tiga nilai, terpisah dari Status di atas yang juga mencakup
// turunan waktu (expiring/expired) dan turunan lokal (missing/invalid/
// unreachable).
type adminStatus string

const (
	adminStatusActive    adminStatus = "active"
	adminStatusSuspended adminStatus = "suspended"
	adminStatusRevoked   adminStatus = "revoked"
)

type payload struct {
	LicenseID      string `json:"license_id"`
	InstallationID string `json:"installation_id"`
	Customer       string `json:"customer"`
	Plan           string `json:"plan"`
	AdminStatus    string `json:"admin_status"`
	MaxDevices     int    `json:"max_devices"`
	IssuedAt       string `json:"issued_at"`
	ExpiresAt      string `json:"expires_at"`
	ValidatedAt    string `json:"validated_at"`
}

const dateLayout = "2006-01-02"

// Load membaca dan memverifikasi local license state di path tersebut.
// TIDAK PERNAH mengembalikan error — kegagalan apa pun (file tak ada,
// signature salah, JSON korup, basi lewat grace period) menghasilkan
// License dengan Status yang sesuai, supaya server tetap bisa start dan
// menampilkan status itu di dashboard alih-alih crash.
func Load(path string, now time.Time) License {
	pubKey, err := base64.StdEncoding.DecodeString(licensePublicKeyBase64)
	if err != nil {
		// Tidak mungkin terjadi di build yang benar — konstanta di source.
		return License{Status: StatusInvalid, Reason: "kunci publik lisensi tidak valid"}
	}
	return load(path, now, ed25519.PublicKey(pubKey))
}

// load melakukan pekerjaan sesungguhnya, menerima public key sebagai
// parameter supaya test dapat memverifikasi jalur "signature valid" dengan
// key pair miliknya sendiri — private key produksi sengaja tidak pernah ada
// di repo ini.
func load(path string, now time.Time, pubKey ed25519.PublicKey) License {
	raw, err := os.ReadFile(path)
	if err != nil {
		return License{Status: StatusMissing, Reason: "belum pernah aktivasi"}
	}

	lines := splitLines(string(raw))
	if len(lines) < 2 {
		return License{Status: StatusInvalid, Reason: "format local license state tidak dikenali"}
	}

	payloadBytes, err := base64.StdEncoding.DecodeString(strings.TrimSpace(lines[0]))
	if err != nil {
		return License{Status: StatusInvalid, Reason: "baris payload bukan base64 yang sah"}
	}
	sig, err := base64.StdEncoding.DecodeString(strings.TrimSpace(lines[1]))
	if err != nil {
		return License{Status: StatusInvalid, Reason: "baris signature bukan base64 yang sah"}
	}
	if !ed25519.Verify(pubKey, payloadBytes, sig) {
		return License{Status: StatusInvalid, Reason: "tanda tangan local license state tidak valid"}
	}

	var p payload
	if err := json.Unmarshal(payloadBytes, &p); err != nil {
		return License{Status: StatusInvalid, Reason: "isi local license state tidak dapat dibaca"}
	}

	issuedAt, err1 := time.Parse(dateLayout, p.IssuedAt)
	expiresRaw, err2 := time.Parse(dateLayout, p.ExpiresAt)
	validatedAt, err3 := time.Parse(time.RFC3339, p.ValidatedAt)
	if err1 != nil || err2 != nil || err3 != nil {
		return License{Status: StatusInvalid, Reason: "tanggal pada local license state tidak valid"}
	}
	// expires_at berlaku sampai akhir hari itu (23:59:59), bukan awal hari.
	expiresAt := expiresRaw.Add(24*time.Hour - time.Second)

	lic := License{
		LicenseID:      p.LicenseID,
		InstallationID: p.InstallationID,
		Customer:       p.Customer,
		Plan:           p.Plan,
		MaxDevices:     p.MaxDevices,
		IssuedAt:       issuedAt,
		ExpiresAt:      expiresAt,
		ValidatedAt:    validatedAt,
	}

	switch adminStatus(p.AdminStatus) {
	case adminStatusSuspended:
		lic.Status = StatusSuspended
		lic.Reason = "lisensi disuspend oleh vendor"
		return lic
	case adminStatusRevoked:
		lic.Status = StatusRevoked
		lic.Reason = "lisensi dicabut oleh vendor"
		return lic
	case adminStatusActive:
		// lanjut ke pengecekan waktu di bawah
	default:
		return License{Status: StatusInvalid, Reason: "admin_status pada local license state tidak dikenali"}
	}

	if now.After(expiresAt) {
		lic.Status = StatusExpired
		lic.Reason = "lisensi sudah kedaluwarsa"
		return lic
	}

	// Grace period: state ini bilang "active", tapi kalau sudah lama tidak
	// berhasil divalidasi ulang (License Server tak terjangkau berkepanjangan),
	// jangan dipercaya selamanya.
	if now.Sub(validatedAt) > GracePeriodDays*24*time.Hour {
		lic.Status = StatusUnreachable
		lic.Reason = "License Server tidak terjangkau lebih dari 7 hari"
		return lic
	}

	if lic.DaysRemaining(now) <= WarningThresholdDays {
		lic.Status = StatusExpiring
		return lic
	}
	lic.Status = StatusActive
	return lic
}

// PeekInstallationID membaca installation_id dari local license state TANPA
// verifikasi signature — dipakai internal/licenseclient sekadar untuk tahu
// installation_id mana yang harus dikirim ke /validate berikutnya. Ini
// BUKAN keputusan trust: kalau isi file dipalsukan, License Server yang
// menolaknya sendiri (installation_id harus benar-benar terikat ke license
// yang valid di database vendor — lihat store.GetActiveInstallation). Load
// (dengan verifikasi penuh) tetap satu-satunya sumber kebenaran untuk
// requireLicense memutuskan boleh/tidaknya request lain.
func PeekInstallationID(path string) (id string, ok bool) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	lines := splitLines(string(raw))
	if len(lines) < 1 {
		return "", false
	}
	payloadBytes, err := base64.StdEncoding.DecodeString(strings.TrimSpace(lines[0]))
	if err != nil {
		return "", false
	}
	var p payload
	if err := json.Unmarshal(payloadBytes, &p); err != nil || p.InstallationID == "" {
		return "", false
	}
	return p.InstallationID, true
}

// Operational melaporkan apakah status ini mengizinkan endpoint
// device/admin/API key beroperasi normal — dipakai requireLicense.
func (s Status) Operational() bool {
	return s == StatusActive || s == StatusExpiring
}

func splitLines(s string) []string {
	scanner := bufio.NewScanner(strings.NewReader(s))
	var out []string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		out = append(out, line)
	}
	return out
}
