package licensecheck

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"
)

// GenerateKeyPair menghasilkan key pair Ed25519 baru untuk `licensetool
// -genkey`. Dipanggil sekali saja seumur hidup produk ini kecuali terjadi
// rotasi darurat (lihat spec §3) — bukan operasi yang dilakukan tiap
// menerbitkan lisensi.
func GenerateKeyPair() (pub, priv []byte, err error) {
	p, s, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	return p, s, nil
}

// Issue membangun isi berkas lisensi (dua baris: payload lalu signature)
// yang ditandatangani dengan privKey. Dipakai oleh `licensetool -issue`,
// dijalankan hanya di mesin Akbar.
func Issue(privKeyBase64, customer, domain, plan string, issuedAt, expiresAt time.Time) (string, error) {
	privKey, err := base64.StdEncoding.DecodeString(privKeyBase64)
	if err != nil {
		return "", fmt.Errorf("licensecheck: private key bukan base64 yang sah: %w", err)
	}
	if len(privKey) != ed25519.PrivateKeySize {
		return "", fmt.Errorf("licensecheck: private key harus %d byte, dapat %d",
			ed25519.PrivateKeySize, len(privKey))
	}

	p := payload{
		Customer:  customer,
		Domain:    domain,
		Plan:      plan,
		IssuedAt:  issuedAt.Format(dateLayout),
		ExpiresAt: expiresAt.Format(dateLayout),
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
