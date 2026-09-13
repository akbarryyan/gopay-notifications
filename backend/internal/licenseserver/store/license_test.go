package store_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/akbarryyan/gopay-notifications/backend/internal/licenseserver/store"
)

func TestCreateAndGetLicenseByKeyHash(t *testing.T) {
	s := testStore(t)
	now := time.Date(2026, 9, 13, 0, 0, 0, 0, time.UTC)
	l := createTestCustomerAndLicense(t, s, now)

	if l.Status != store.AdminStatusActive {
		t.Fatalf("status = %q, mau active", l.Status)
	}
	if l.MaxDevices != 10 || l.ProductionInstallations != 1 || l.UATInstallations != 1 {
		t.Fatalf("field tidak sesuai: %+v", l)
	}
}

func TestGetLicenseByKeyHashSalahMenghasilkanNotFound(t *testing.T) {
	s := testStore(t)
	_, err := s.GetLicenseByKeyHash(context.Background(), []byte("hash-salah-32-byte-xxxxxxxxxxxx"))
	if !errors.Is(err, store.ErrLicenseNotFound) {
		t.Fatalf("err = %v, mau ErrLicenseNotFound", err)
	}
}

func TestRenewLicenseMengubahExpiresAt(t *testing.T) {
	s := testStore(t)
	now := time.Date(2026, 9, 13, 0, 0, 0, 0, time.UTC)
	l := createTestCustomerAndLicense(t, s, now)

	newExpiry := now.AddDate(2, 0, 0)
	if err := s.RenewLicense(context.Background(), l.ID, newExpiry); err != nil {
		t.Fatalf("RenewLicense: %v", err)
	}

	got, err := s.GetLicenseByID(context.Background(), l.ID)
	if err != nil {
		t.Fatalf("GetLicenseByID: %v", err)
	}
	if !got.ExpiresAt.Equal(newExpiry) {
		t.Fatalf("ExpiresAt = %v, mau %v", got.ExpiresAt, newExpiry)
	}
}

func TestRenewLicenseTidakDitemukan(t *testing.T) {
	s := testStore(t)
	err := s.RenewLicense(context.Background(), "lic_tidak_ada", time.Now())
	if !errors.Is(err, store.ErrLicenseNotFound) {
		t.Fatalf("err = %v, mau ErrLicenseNotFound", err)
	}
}

func TestSetLicenseStatusSuspendDanRevoke(t *testing.T) {
	s := testStore(t)
	now := time.Date(2026, 9, 13, 0, 0, 0, 0, time.UTC)
	l := createTestCustomerAndLicense(t, s, now)

	if err := s.SetLicenseStatus(context.Background(), l.ID, store.AdminStatusSuspended); err != nil {
		t.Fatalf("SetLicenseStatus suspend: %v", err)
	}
	got, err := s.GetLicenseByID(context.Background(), l.ID)
	if err != nil {
		t.Fatalf("GetLicenseByID: %v", err)
	}
	if got.Status != store.AdminStatusSuspended {
		t.Fatalf("status = %q, mau suspended", got.Status)
	}

	if err := s.SetLicenseStatus(context.Background(), l.ID, store.AdminStatusRevoked); err != nil {
		t.Fatalf("SetLicenseStatus revoke: %v", err)
	}
	got, _ = s.GetLicenseByID(context.Background(), l.ID)
	if got.Status != store.AdminStatusRevoked {
		t.Fatalf("status = %q, mau revoked", got.Status)
	}
}

func TestDerivedStatusMatriks(t *testing.T) {
	now := time.Date(2026, 9, 13, 0, 0, 0, 0, time.UTC)

	cases := []struct {
		name string
		l    store.License
		want string
	}{
		{"suspended langsung dari kolom", store.License{Status: store.AdminStatusSuspended, ExpiresAt: now.AddDate(1, 0, 0)}, "suspended"},
		{"revoked langsung dari kolom", store.License{Status: store.AdminStatusRevoked, ExpiresAt: now.AddDate(1, 0, 0)}, "revoked"},
		{"active jauh dari expiry", store.License{Status: store.AdminStatusActive, ExpiresAt: now.AddDate(1, 0, 0)}, "active"},
		{"expiring dalam 10 hari", store.License{Status: store.AdminStatusActive, ExpiresAt: now.AddDate(0, 0, 10)}, "expiring"},
		{"expired kemarin", store.License{Status: store.AdminStatusActive, ExpiresAt: now.AddDate(0, 0, -1)}, "expired"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.l.DerivedStatus(now, 30); got != c.want {
				t.Errorf("DerivedStatus = %q, mau %q", got, c.want)
			}
		})
	}
}

func TestGenerateLicenseKeyFormatDanUnik(t *testing.T) {
	raw1, hash1, err := store.GenerateLicenseKey("Business")
	if err != nil {
		t.Fatalf("GenerateLicenseKey: %v", err)
	}
	raw2, hash2, err := store.GenerateLicenseKey("Business")
	if err != nil {
		t.Fatalf("GenerateLicenseKey: %v", err)
	}
	if raw1 == raw2 {
		t.Fatal("dua panggilan menghasilkan key mentah yang sama")
	}
	if string(hash1) == string(hash2) {
		t.Fatal("dua panggilan menghasilkan hash yang sama")
	}
	if len(raw1) < len("PB-BUSINESS-XXXX-XXXX-XXXX") {
		t.Fatalf("format key terlalu pendek: %q", raw1)
	}
}
