package httpapi_test

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"os"
	"testing"
	"time"

	"github.com/akbarryyan/gopay-notifications/backend/internal/licenseserver/httpapi"
	"github.com/akbarryyan/gopay-notifications/backend/internal/licenseserver/store"
)

var fixedNow = time.Date(2026, 9, 13, 12, 0, 0, 0, time.UTC)

func adminSessionKey() []byte {
	k := make([]byte, 32)
	for i := range k {
		k[i] = byte(i*5 + 1)
	}
	return k
}

// testSigningPrivateKey menghasilkan key pair Ed25519 baru khusus test ini
// — TERPISAH dari kunci produksi milik License Server sungguhan. Response
// /activate dan /validate tetap bisa diperiksa isinya (base64-decode +
// JSON-unmarshal payload langsung di test), cuma tidak diverifikasi
// terhadap public key produksi (itu tanggung jawab internal/licensecheck,
// sudah diuji tuntas di paketnya sendiri).
func testSigningPrivateKey(t *testing.T) string {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate signing key: %v", err)
	}
	return base64.StdEncoding.EncodeToString(priv)
}

// newTestStore membuka koneksi ke database test License Server dan
// mengosongkan seluruh tabel.
func newTestStore(t *testing.T) *store.Store {
	t.Helper()
	url := os.Getenv("LICENSE_TEST_DATABASE_URL")
	if url == "" {
		t.Fatal("LICENSE_TEST_DATABASE_URL belum diset. Jalankan: make db-up migrate-license")
	}
	ctx := context.Background()
	s, err := store.New(ctx, url)
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	t.Cleanup(s.Close)
	_, err = s.Pool().Exec(ctx,
		"TRUNCATE audit_log, installations, licenses, customers, admin_users RESTART IDENTITY CASCADE")
	if err != nil {
		t.Fatalf("truncate: %v", err)
	}
	return s
}

func newTestAPI(t *testing.T) (*httpapi.API, *store.Store) {
	t.Helper()
	s := newTestStore(t)
	api := httpapi.New(s, adminSessionKey(), testSigningPrivateKey(t), func() time.Time { return fixedNow })
	return api, s
}

// createTestVendorAdmin membuat akun vendor untuk test login.
func createTestVendorAdmin(t *testing.T, s *store.Store, username, password string) {
	t.Helper()
	if err := s.UpsertAdmin(context.Background(), username, password); err != nil {
		t.Fatalf("UpsertAdmin: %v", err)
	}
}

// createTestLicenseWithKey membuat customer + license aktif, mengembalikan
// license key MENTAH (dipakai memanggil /activate dan /validate di test).
func createTestLicenseWithKey(t *testing.T, s *store.Store) (rawKey string, licenseID string) {
	t.Helper()
	ctx := context.Background()

	custID, err := store.NewCustomerID()
	if err != nil {
		t.Fatalf("NewCustomerID: %v", err)
	}
	if err := s.CreateCustomer(ctx, custID, "Toko Contoh"); err != nil {
		t.Fatalf("CreateCustomer: %v", err)
	}

	licID, err := store.NewLicenseID()
	if err != nil {
		t.Fatalf("NewLicenseID: %v", err)
	}
	raw, keyHash, err := store.GenerateLicenseKey("Business")
	if err != nil {
		t.Fatalf("GenerateLicenseKey: %v", err)
	}
	in := store.CreateLicenseInput{
		ID: licID, CustomerID: custID, KeyHash: keyHash, Plan: "Business", MaxDevices: 10,
		ProductionInstallations: 1, UATInstallations: 1,
		IssuedAt: fixedNow, ExpiresAt: fixedNow.AddDate(1, 0, 0),
	}
	if err := s.CreateLicense(ctx, in); err != nil {
		t.Fatalf("CreateLicense: %v", err)
	}
	return raw, licID
}
