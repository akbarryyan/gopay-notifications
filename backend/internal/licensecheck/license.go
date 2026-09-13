// Package licensecheck memverifikasi file lisensi offline yang ditandatangani
// Ed25519. Tidak menyentuh database maupun jaringan — sepenuhnya pure logic,
// sejalan dengan internal/auth dan internal/secretbox.
//
// Lihat docs/superpowers/specs/2026-09-13-license-system-design.md untuk
// rancangan lengkap.
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

// licensePublicKey di-hardcode, BUKAN env var atau file yang bisa diedit
// operator server. Kalau kunci ini bisa dikonfigurasi lewat env/file di
// server customer, customer bisa membuat key pair sendiri dan
// menandatangani lisensinya sendiri — meniadakan seluruh gunanya tanda
// tangan. Private key pasangannya HANYA ada di mesin Akbar sendiri, dipakai
// lewat `licensetool -issue`, dan tidak pernah masuk repo ini.
const licensePublicKeyBase64 = "HOdMzYphVIuVD6GXvSfOHcGYntPP94LhT4EujgBCfv0="

// WarningThresholdDays adalah ambang "akan berakhir" yang ditampilkan
// dashboard — dipusatkan di sini karena backend (endpoint /admin/license)
// dan tidak ada tempat lain yang perlu tahu angka ini selain dashboard, yang
// membacanya lewat response endpoint tersebut.
const WarningThresholdDays = 30

type Status string

const (
	// StatusActive: signature valid, domain cocok, belum lewat expires_at.
	StatusActive Status = "active"
	// StatusMissing: file lisensi tidak ditemukan di path yang dikonfigurasi.
	StatusMissing Status = "missing"
	// StatusInvalid: signature tidak valid, payload tidak dapat dibaca, atau
	// domain di lisensi tidak cocok dengan domain server ini.
	StatusInvalid Status = "invalid"
	// StatusExpired: signature dan domain valid, tapi sudah lewat expires_at.
	StatusExpired Status = "expired"
)

// License adalah hasil pembacaan dan verifikasi satu file lisensi.
type License struct {
	Customer  string
	Domain    string
	Plan      string
	IssuedAt  time.Time
	ExpiresAt time.Time
	Status    Status
	// Reason menjelaskan status non-active secara spesifik, untuk
	// ditampilkan di halaman dashboard /license. Kosong bila Status ==
	// StatusActive.
	Reason string
}

// DaysRemaining menghitung selisih hari ke expires_at dari now, dibulatkan
// ke bawah. Bisa negatif kalau sudah lewat. Hanya bermakna bila lisensi
// pernah berhasil dibaca (Status != StatusMissing).
func (l License) DaysRemaining(now time.Time) int {
	d := l.ExpiresAt.Sub(now)
	return int(d.Hours() / 24)
}

type payload struct {
	Customer  string `json:"customer"`
	Domain    string `json:"domain"`
	Plan      string `json:"plan"`
	IssuedAt  string `json:"issued_at"`
	ExpiresAt string `json:"expires_at"`
}

const dateLayout = "2006-01-02"

// Load membaca dan memverifikasi file lisensi di path tersebut, dibandingkan
// terhadap domain server ini. TIDAK PERNAH mengembalikan error — kegagalan
// apa pun (file tak ada, signature salah, JSON korup) menghasilkan License
// dengan Status yang sesuai, supaya server tetap bisa start dan menampilkan
// status itu di dashboard alih-alih crash.
func Load(path string, domain string, now time.Time) License {
	pubKey, err := base64.StdEncoding.DecodeString(licensePublicKeyBase64)
	if err != nil {
		// Tidak mungkin terjadi di build yang benar — konstanta di source.
		return License{Status: StatusInvalid, Reason: "kunci publik lisensi tidak valid"}
	}
	return load(path, domain, now, ed25519.PublicKey(pubKey))
}

// load melakukan pekerjaan sesungguhnya, menerima public key sebagai
// parameter supaya test dapat memverifikasi jalur "signature valid" dengan
// key pair miliknya sendiri — private key produksi sengaja tidak pernah ada
// di repo ini, jadi test tidak mungkin membuat signature yang valid
// terhadap licensePublicKeyBase64.
func load(path string, domain string, now time.Time, pubKey ed25519.PublicKey) License {
	raw, err := os.ReadFile(path)
	if err != nil {
		return License{Status: StatusMissing, Reason: "berkas lisensi tidak ditemukan"}
	}

	lines := splitLines(string(raw))
	if len(lines) < 2 {
		return License{Status: StatusInvalid, Reason: "format berkas lisensi tidak dikenali"}
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
		return License{Status: StatusInvalid, Reason: "tanda tangan lisensi tidak valid"}
	}

	var p payload
	if err := json.Unmarshal(payloadBytes, &p); err != nil {
		// Signature valid tapi isi payload rusak — seharusnya tidak pernah
		// terjadi kalau licensetool yang menerbitkannya, tapi tetap ditolak
		// dengan aman alih-alih panic di tempat lain.
		return License{Status: StatusInvalid, Reason: "isi lisensi tidak dapat dibaca"}
	}

	issuedAt, err := time.Parse(dateLayout, p.IssuedAt)
	if err != nil {
		return License{Status: StatusInvalid, Reason: "tanggal terbit pada lisensi tidak valid"}
	}
	expiresAt, err := time.Parse(dateLayout, p.ExpiresAt)
	if err != nil {
		return License{Status: StatusInvalid, Reason: "tanggal berakhir pada lisensi tidak valid"}
	}
	// expires_at berlaku sampai akhir hari itu (23:59:59), bukan awal hari.
	expiresAt = expiresAt.Add(24*time.Hour - time.Second)

	if p.Domain != domain {
		return License{
			Customer: p.Customer, Domain: p.Domain, Plan: p.Plan,
			IssuedAt: issuedAt, ExpiresAt: expiresAt,
			Status: StatusInvalid,
			Reason: "domain pada lisensi tidak cocok dengan domain server ini",
		}
	}

	lic := License{
		Customer:  p.Customer,
		Domain:    p.Domain,
		Plan:      p.Plan,
		IssuedAt:  issuedAt,
		ExpiresAt: expiresAt,
	}
	if now.After(expiresAt) {
		lic.Status = StatusExpired
		lic.Reason = "lisensi sudah kedaluwarsa"
		return lic
	}
	lic.Status = StatusActive
	return lic
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
