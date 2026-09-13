package licensecheck

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"
)

// GenerateKeyPair menghasilkan key pair Ed25519 baru untuk License Server.
// Dipanggil sekali saat setup awal (atau rotasi darurat) lewat
// `licenseserver -genkey`, bukan operasi rutin.
func GenerateKeyPair() (pub, priv []byte, err error) {
	p, s, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	return p, s, nil
}

// IssueInput adalah data satu local license state yang akan ditandatangani.
// AdminStatus harus salah satu dari "active"/"suspended"/"revoked" — nilai
// lain ditolak Load() di sisi verifikasi.
type IssueInput struct {
	LicenseID      string
	InstallationID string
	Customer       string
	Plan           string
	AdminStatus    string
	MaxDevices     int
	IssuedAt       time.Time
	ExpiresAt      time.Time
	ValidatedAt    time.Time
}

// Issue membangun local license state (dua baris: payload lalu signature)
// yang ditandatangani dengan privKeyBase64. Dipakai HANYA oleh
// internal/licenseserver, tiap kali menjawab /activate atau /validate.
func Issue(privKeyBase64 string, in IssueInput) (string, error) {
	privKey, err := base64.StdEncoding.DecodeString(privKeyBase64)
	if err != nil {
		return "", fmt.Errorf("licensecheck: private key bukan base64 yang sah: %w", err)
	}
	if len(privKey) != ed25519.PrivateKeySize {
		return "", fmt.Errorf("licensecheck: private key harus %d byte, dapat %d",
			ed25519.PrivateKeySize, len(privKey))
	}

	p := payload{
		LicenseID:      in.LicenseID,
		InstallationID: in.InstallationID,
		Customer:       in.Customer,
		Plan:           in.Plan,
		AdminStatus:    in.AdminStatus,
		MaxDevices:     in.MaxDevices,
		IssuedAt:       in.IssuedAt.Format(dateLayout),
		ExpiresAt:      in.ExpiresAt.Format(dateLayout),
		ValidatedAt:    in.ValidatedAt.Format(time.RFC3339),
	}
	payloadBytes, err := json.Marshal(p)
	if err != nil {
		return "", fmt.Errorf("licensecheck: gagal membuat payload: %w", err)
	}

	sig := ed25519.Sign(ed25519.PrivateKey(privKey), payloadBytes)

	payloadLine := base64.StdEncoding.EncodeToString(payloadBytes)
	sigLine := base64.StdEncoding.EncodeToString(sig)
	return payloadLine + "\n" + sigLine + "\n", nil
}
