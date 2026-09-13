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
// produksi sengaja tidak pernah ada di repo, jadi test memverifikasi jalur
// "signature valid" lewat fungsi tak-diekspor `load` yang menerima public
// key sebagai parameter, bukan lewat `Load` yang dipakai server sungguhan.
func testKeyPair(t *testing.T) (pub ed25519.PublicKey, privBase64 string) {
	t.Helper()
	p, s, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	return p, base64.StdEncoding.EncodeToString(s)
}

func writeStateFile(t *testing.T, dir string, content string) string {
	t.Helper()
	path := filepath.Join(dir, "license-state.lic")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("tulis local license state: %v", err)
	}
	return path
}

func baseInput(now time.Time) IssueInput {
	return IssueInput{
		LicenseID:      "lic_test",
		InstallationID: "inst_test",
		Customer:       "Toko Contoh",
		Plan:           "Business",
		AdminStatus:    "active",
		MaxDevices:     10,
		IssuedAt:       now.AddDate(-1, 0, 0),
		ExpiresAt:      now.AddDate(1, 0, 0),
		ValidatedAt:    now,
	}
}

func TestLoadFileTidakAdaMenghasilkanMissing(t *testing.T) {
	pub, _ := testKeyPair(t)
	lic := load(filepath.Join(t.TempDir(), "tidak-ada.lic"), time.Now(), pub)
	if lic.Status != StatusMissing {
		t.Fatalf("status = %q, mau %q", lic.Status, StatusMissing)
	}
}

func TestLoadFormatKorupMenghasilkanInvalid(t *testing.T) {
	pub, _ := testKeyPair(t)
	dir := t.TempDir()
	path := writeStateFile(t, dir, "cuma-satu-baris\n")
	lic := load(path, time.Now(), pub)
	if lic.Status != StatusInvalid {
		t.Fatalf("status = %q, mau %q", lic.Status, StatusInvalid)
	}
}

func TestLoadSignatureValidMenghasilkanActive(t *testing.T) {
	pub, priv := testKeyPair(t)
	now := time.Date(2026, 9, 13, 12, 0, 0, 0, time.UTC)
	in := baseInput(now)
	content, err := Issue(priv, in)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}

	dir := t.TempDir()
	path := writeStateFile(t, dir, content)
	lic := load(path, now, pub)
	if lic.Status != StatusActive {
		t.Fatalf("status = %q (reason=%q), mau %q", lic.Status, lic.Reason, StatusActive)
	}
	if lic.LicenseID != "lic_test" || lic.InstallationID != "inst_test" || lic.Customer != "Toko Contoh" {
		t.Fatalf("field tidak sesuai: %+v", lic)
	}
	if lic.MaxDevices != 10 {
		t.Fatalf("MaxDevices = %d, mau 10", lic.MaxDevices)
	}
}

func TestLoadSisaKurangDariAmbangMenghasilkanExpiring(t *testing.T) {
	pub, priv := testKeyPair(t)
	now := time.Date(2026, 9, 13, 0, 0, 0, 0, time.UTC)
	in := baseInput(now)
	in.ExpiresAt = now.AddDate(0, 0, 10) // 10 hari, di bawah ambang 30
	content, err := Issue(priv, in)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	dir := t.TempDir()
	path := writeStateFile(t, dir, content)

	lic := load(path, now, pub)
	if lic.Status != StatusExpiring {
		t.Fatalf("status = %q, mau %q", lic.Status, StatusExpiring)
	}
}

func TestLoadSetelahExpiresAtMenghasilkanExpired(t *testing.T) {
	pub, priv := testKeyPair(t)
	now := time.Date(2026, 9, 13, 0, 0, 0, 0, time.UTC)
	in := baseInput(now)
	in.ExpiresAt = now.AddDate(0, 0, -1)
	content, err := Issue(priv, in)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	dir := t.TempDir()
	path := writeStateFile(t, dir, content)

	lic := load(path, now, pub)
	if lic.Status != StatusExpired {
		t.Fatalf("status = %q, mau %q", lic.Status, StatusExpired)
	}
}

func TestLoadAdminStatusSuspended(t *testing.T) {
	pub, priv := testKeyPair(t)
	now := time.Date(2026, 9, 13, 0, 0, 0, 0, time.UTC)
	in := baseInput(now)
	in.AdminStatus = "suspended"
	content, err := Issue(priv, in)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	dir := t.TempDir()
	path := writeStateFile(t, dir, content)

	lic := load(path, now, pub)
	if lic.Status != StatusSuspended {
		t.Fatalf("status = %q, mau %q", lic.Status, StatusSuspended)
	}
}

func TestLoadAdminStatusRevoked(t *testing.T) {
	pub, priv := testKeyPair(t)
	now := time.Date(2026, 9, 13, 0, 0, 0, 0, time.UTC)
	in := baseInput(now)
	in.AdminStatus = "revoked"
	content, err := Issue(priv, in)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	dir := t.TempDir()
	path := writeStateFile(t, dir, content)

	lic := load(path, now, pub)
	if lic.Status != StatusRevoked {
		t.Fatalf("status = %q, mau %q", lic.Status, StatusRevoked)
	}
}

func TestLoadValidatedAtBasiMenghasilkanUnreachable(t *testing.T) {
	pub, priv := testKeyPair(t)
	now := time.Date(2026, 9, 13, 0, 0, 0, 0, time.UTC)
	in := baseInput(now)
	in.ValidatedAt = now.Add(-8 * 24 * time.Hour) // lewat grace period 7 hari
	content, err := Issue(priv, in)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	dir := t.TempDir()
	path := writeStateFile(t, dir, content)

	lic := load(path, now, pub)
	if lic.Status != StatusUnreachable {
		t.Fatalf("status = %q, mau %q", lic.Status, StatusUnreachable)
	}
}

func TestLoadValidatedAtDalamGracePeriodTetapActive(t *testing.T) {
	pub, priv := testKeyPair(t)
	now := time.Date(2026, 9, 13, 0, 0, 0, 0, time.UTC)
	in := baseInput(now)
	in.ValidatedAt = now.Add(-6 * 24 * time.Hour) // masih dalam grace period 7 hari
	content, err := Issue(priv, in)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	dir := t.TempDir()
	path := writeStateFile(t, dir, content)

	lic := load(path, now, pub)
	if lic.Status != StatusActive {
		t.Fatalf("status = %q, mau %q", lic.Status, StatusActive)
	}
}

func TestLoadSignatureDitandatanganiKunciLainDitolak(t *testing.T) {
	pubA, _ := testKeyPair(t)
	_, privB := testKeyPair(t)

	now := time.Date(2026, 9, 13, 0, 0, 0, 0, time.UTC)
	content, err := Issue(privB, baseInput(now))
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	dir := t.TempDir()
	path := writeStateFile(t, dir, content)

	lic := load(path, now, pubA)
	if lic.Status != StatusInvalid {
		t.Fatalf("status = %q, mau %q", lic.Status, StatusInvalid)
	}
}

func TestLoadAdminStatusTidakDikenalDitolak(t *testing.T) {
	pub, priv := testKeyPair(t)
	now := time.Date(2026, 9, 13, 0, 0, 0, 0, time.UTC)
	in := baseInput(now)
	in.AdminStatus = "apa-saja"
	content, err := Issue(priv, in)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	dir := t.TempDir()
	path := writeStateFile(t, dir, content)

	lic := load(path, now, pub)
	if lic.Status != StatusInvalid {
		t.Fatalf("status = %q, mau %q", lic.Status, StatusInvalid)
	}
}

func TestOperationalHanyaActiveDanExpiring(t *testing.T) {
	cases := map[Status]bool{
		StatusActive:      true,
		StatusExpiring:    true,
		StatusExpired:     false,
		StatusSuspended:   false,
		StatusRevoked:     false,
		StatusMissing:     false,
		StatusInvalid:     false,
		StatusUnreachable: false,
	}
	for status, want := range cases {
		if got := status.Operational(); got != want {
			t.Errorf("Status(%q).Operational() = %v, mau %v", status, got, want)
		}
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
	_, err := Issue("bukan-base64!!!", baseInput(time.Now()))
	if err == nil {
		t.Fatal("mau error, dapat nil")
	}
}

func TestIssueMenolakPrivateKeySalahUkuran(t *testing.T) {
	_, err := Issue(base64.StdEncoding.EncodeToString([]byte("terlalu-pendek")), baseInput(time.Now()))
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
