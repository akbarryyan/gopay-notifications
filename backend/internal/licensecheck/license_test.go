package licensecheck

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// testKeyPair menghasilkan key pair BARU khusus untuk test ini, terpisah
// dari licensePublicKeyBase64 yang di-hardcode untuk produksi. Private key
// produksi sengaja tidak pernah ada di repo, jadi test memverifikasi
// jalur "signature valid" lewat fungsi tak-diekspor `load` yang menerima
// public key sebagai parameter, bukan lewat `Load` yang dipakai server
// sungguhan.
func testKeyPair(t *testing.T) (pub ed25519.PublicKey, privBase64 string) {
	t.Helper()
	p, s, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	return p, base64.StdEncoding.EncodeToString(s)
}

func writeLicenseFile(t *testing.T, dir string, content string) string {
	t.Helper()
	path := filepath.Join(dir, "license.lic")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("tulis file lisensi: %v", err)
	}
	return path
}

func TestLoadFileTidakAdaMenghasilkanMissing(t *testing.T) {
	pub, _ := testKeyPair(t)
	lic := load(filepath.Join(t.TempDir(), "tidak-ada.lic"), "whuzpay.com", time.Now(), pub)
	if lic.Status != StatusMissing {
		t.Fatalf("status = %q, mau %q", lic.Status, StatusMissing)
	}
}

func TestLoadFormatKorupMenghasilkanInvalid(t *testing.T) {
	pub, _ := testKeyPair(t)
	dir := t.TempDir()
	path := writeLicenseFile(t, dir, "cuma-satu-baris\n")
	lic := load(path, "whuzpay.com", time.Now(), pub)
	if lic.Status != StatusInvalid {
		t.Fatalf("status = %q, mau %q", lic.Status, StatusInvalid)
	}
}

func TestLoadSignatureValidDomainCocokMenghasilkanActive(t *testing.T) {
	pub, priv := testKeyPair(t)
	now := time.Date(2026, 9, 13, 12, 0, 0, 0, time.UTC)
	content, err := Issue(priv, "Toko Contoh", "whuzpay.com", "Business",
		now, now.AddDate(1, 0, 0))
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}

	dir := t.TempDir()
	path := writeLicenseFile(t, dir, content)
	lic := load(path, "whuzpay.com", now, pub)
	if lic.Status != StatusActive {
		t.Fatalf("status = %q (reason=%q), mau %q", lic.Status, lic.Reason, StatusActive)
	}
	if lic.Customer != "Toko Contoh" || lic.Domain != "whuzpay.com" || lic.Plan != "Business" {
		t.Fatalf("field lisensi tidak sesuai: %+v", lic)
	}
}

func TestLoadTepatDiHariTerakhirMasihActive(t *testing.T) {
	pub, priv := testKeyPair(t)
	issuedAt := time.Date(2025, 9, 13, 0, 0, 0, 0, time.UTC)
	expiresAt := time.Date(2026, 9, 13, 0, 0, 0, 0, time.UTC)
	content, err := Issue(priv, "Toko Contoh", "whuzpay.com", "Business", issuedAt, expiresAt)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	dir := t.TempDir()
	path := writeLicenseFile(t, dir, content)

	// Jam 23:00 di hari expires_at itu sendiri — masih harus active.
	now := time.Date(2026, 9, 13, 23, 0, 0, 0, time.UTC)
	lic := load(path, "whuzpay.com", now, pub)
	if lic.Status != StatusActive {
		t.Fatalf("status = %q, mau %q (masih di hari expires_at)", lic.Status, StatusActive)
	}
}

func TestLoadSetelahExpiresAtMenghasilkanExpired(t *testing.T) {
	pub, priv := testKeyPair(t)
	issuedAt := time.Date(2025, 9, 13, 0, 0, 0, 0, time.UTC)
	expiresAt := time.Date(2026, 9, 13, 0, 0, 0, 0, time.UTC)
	content, err := Issue(priv, "Toko Contoh", "whuzpay.com", "Business", issuedAt, expiresAt)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	dir := t.TempDir()
	path := writeLicenseFile(t, dir, content)

	now := time.Date(2026, 9, 14, 0, 0, 1, 0, time.UTC)
	lic := load(path, "whuzpay.com", now, pub)
	if lic.Status != StatusExpired {
		t.Fatalf("status = %q, mau %q", lic.Status, StatusExpired)
	}
}

func TestLoadDomainTidakCocokMenghasilkanInvalid(t *testing.T) {
	pub, priv := testKeyPair(t)
	now := time.Date(2026, 9, 13, 0, 0, 0, 0, time.UTC)
	content, err := Issue(priv, "Toko Contoh", "domain-lain.com", "Business",
		now, now.AddDate(1, 0, 0))
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	dir := t.TempDir()
	path := writeLicenseFile(t, dir, content)

	lic := load(path, "whuzpay.com", now, pub)
	if lic.Status != StatusInvalid {
		t.Fatalf("status = %q, mau %q", lic.Status, StatusInvalid)
	}
}

func TestLoadSignatureDitandatanganiKunciLainDitolak(t *testing.T) {
	pubA, _ := testKeyPair(t)
	_, privB := testKeyPair(t)

	now := time.Date(2026, 9, 13, 0, 0, 0, 0, time.UTC)
	// Ditandatangani dengan privB, tapi diverifikasi terhadap pubA —
	// mensimulasikan file lisensi yang ditandatangani kunci privat siapa
	// pun selain yang public key-nya dipercaya server.
	content, err := Issue(privB, "Toko Contoh", "whuzpay.com", "Business",
		now, now.AddDate(1, 0, 0))
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	dir := t.TempDir()
	path := writeLicenseFile(t, dir, content)

	lic := load(path, "whuzpay.com", now, pubA)
	if lic.Status != StatusInvalid {
		t.Fatalf("status = %q, mau %q", lic.Status, StatusInvalid)
	}
}

func TestLoadBarisPayloadBukanBase64(t *testing.T) {
	pub, _ := testKeyPair(t)
	dir := t.TempDir()
	path := writeLicenseFile(t, dir, "bukan-base64!!!\nYWJj\n")
	lic := load(path, "whuzpay.com", time.Now(), pub)
	if lic.Status != StatusInvalid {
		t.Fatalf("status = %q, mau %q", lic.Status, StatusInvalid)
	}
}

func TestDaysRemainingDihitungDariExpiresAt(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	lic := License{ExpiresAt: now.AddDate(0, 0, 10)}
	if got := lic.DaysRemaining(now); got != 10 {
		t.Fatalf("DaysRemaining = %d, mau 10", got)
	}

	expiredLic := License{ExpiresAt: now.AddDate(0, 0, -5)}
	if got := expiredLic.DaysRemaining(now); got >= 0 {
		t.Fatalf("DaysRemaining untuk lisensi lewat = %d, mau negatif", got)
	}
}

func TestIssueMenolakPrivateKeyBukanBase64(t *testing.T) {
	_, err := Issue("bukan-base64!!!", "Toko", "whuzpay.com", "Business",
		time.Now(), time.Now().AddDate(1, 0, 0))
	if err == nil {
		t.Fatal("mau error, dapat nil")
	}
}

func TestIssueMenolakPrivateKeySalahUkuran(t *testing.T) {
	_, err := Issue(base64.StdEncoding.EncodeToString([]byte("terlalu-pendek")),
		"Toko", "whuzpay.com", "Business", time.Now(), time.Now().AddDate(1, 0, 0))
	if err == nil {
		t.Fatal("mau error, dapat nil")
	}
}

func TestGenerateKeyPairMenghasilkanUkuranYangBenar(t *testing.T) {
	pub, priv, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair: %v", err)
	}
	if len(pub) != ed25519.PublicKeySize {
		t.Fatalf("len(pub) = %d, mau %d", len(pub), ed25519.PublicKeySize)
	}
	if len(priv) != ed25519.PrivateKeySize {
		t.Fatalf("len(priv) = %d, mau %d", len(priv), ed25519.PrivateKeySize)
	}
}
