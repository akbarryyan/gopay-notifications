package store_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/akbarryyan/gopay-notifications/backend/internal/licenseserver/store"
)

// testStore membuka koneksi ke database test License Server (TERPISAH dari
// TEST_DATABASE_URL milik backend customer) dan mengosongkan seluruh tabel.
func testStore(t *testing.T) *store.Store {
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

// createTestCustomerAndLicense adalah helper dipakai berulang oleh test
// installation/license — customer + satu license active, quota default
// 1 production + 1 uat.
func createTestCustomerAndLicense(t *testing.T, s *store.Store, now time.Time) store.License {
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
	_, keyHash, err := store.GenerateLicenseKey("Business")
	if err != nil {
		t.Fatalf("GenerateLicenseKey: %v", err)
	}
	in := store.CreateLicenseInput{
		ID: licID, CustomerID: custID, KeyHash: keyHash, Plan: "Business", MaxDevices: 10,
		ProductionInstallations: 1, UATInstallations: 1,
		IssuedAt: now, ExpiresAt: now.AddDate(1, 0, 0),
	}
	if err := s.CreateLicense(ctx, in); err != nil {
		t.Fatalf("CreateLicense: %v", err)
	}

	l, err := s.GetLicenseByID(ctx, licID)
	if err != nil {
		t.Fatalf("GetLicenseByID: %v", err)
	}
	return l
}
