package store_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/akbarryyan/gopay-notifications/backend/internal/licenseserver/store"
)

func TestActivateBerhasilDanKuotaPenuhSetelahnya(t *testing.T) {
	s := testStore(t)
	now := time.Date(2026, 9, 13, 0, 0, 0, 0, time.UTC)
	l := createTestCustomerAndLicense(t, s, now) // production_installations = 1

	inst, err := s.Activate(context.Background(), l, "production", "1.0.0")
	if err != nil {
		t.Fatalf("Activate: %v", err)
	}
	if inst.LicenseID != l.ID || inst.Environment != "production" {
		t.Fatalf("field tidak sesuai: %+v", inst)
	}

	// Kuota production = 1, sudah terpakai -- aktivasi kedua harus ditolak.
	_, err = s.Activate(context.Background(), l, "production", "1.0.0")
	if !errors.Is(err, store.ErrInstallationLimitReached) {
		t.Fatalf("err = %v, mau ErrInstallationLimitReached", err)
	}

	// UAT punya kuota sendiri, terpisah dari production -- masih boleh.
	if _, err := s.Activate(context.Background(), l, "uat", "1.0.0"); err != nil {
		t.Fatalf("Activate uat: %v", err)
	}
}

func TestActivateLicenseTidakAktifDitolak(t *testing.T) {
	s := testStore(t)
	now := time.Date(2026, 9, 13, 0, 0, 0, 0, time.UTC)
	l := createTestCustomerAndLicense(t, s, now)
	if err := s.SetLicenseStatus(context.Background(), l.ID, store.AdminStatusSuspended); err != nil {
		t.Fatalf("SetLicenseStatus: %v", err)
	}
	l.Status = store.AdminStatusSuspended

	_, err := s.Activate(context.Background(), l, "production", "1.0.0")
	if !errors.Is(err, store.ErrLicenseNotActive) {
		t.Fatalf("err = %v, mau ErrLicenseNotActive", err)
	}
}

// TestActivateRaceHanyaSatuYangMenang membuktikan row lock FOR UPDATE
// benar-benar mencegah dua aktivasi bersamaan sama-sama lolos kuota 1 —
// pola sama seperti test race MatchEvent/ManualMatchEvent di backend
// customer. Jalankan dengan -race.
func TestActivateRaceHanyaSatuYangMenang(t *testing.T) {
	s := testStore(t)
	now := time.Date(2026, 9, 13, 0, 0, 0, 0, time.UTC)
	l := createTestCustomerAndLicense(t, s, now)

	const attempts = 5
	var wg sync.WaitGroup
	results := make([]error, attempts)
	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, err := s.Activate(context.Background(), l, "production", "1.0.0")
			results[i] = err
		}(i)
	}
	wg.Wait()

	successCount := 0
	for _, err := range results {
		if err == nil {
			successCount++
		} else if !errors.Is(err, store.ErrInstallationLimitReached) {
			t.Fatalf("error tak terduga: %v", err)
		}
	}
	if successCount != 1 {
		t.Fatalf("successCount = %d, mau tepat 1 (kuota production = 1)", successCount)
	}
}

func TestGetActiveInstallationSetelahReset(t *testing.T) {
	s := testStore(t)
	now := time.Date(2026, 9, 13, 0, 0, 0, 0, time.UTC)
	l := createTestCustomerAndLicense(t, s, now)

	inst, err := s.Activate(context.Background(), l, "production", "1.0.0")
	if err != nil {
		t.Fatalf("Activate: %v", err)
	}

	if _, err := s.GetActiveInstallation(context.Background(), l.ID, inst.ID); err != nil {
		t.Fatalf("GetActiveInstallation sebelum reset: %v", err)
	}

	if err := s.ResetInstallation(context.Background(), inst.ID); err != nil {
		t.Fatalf("ResetInstallation: %v", err)
	}

	if _, err := s.GetActiveInstallation(context.Background(), l.ID, inst.ID); !errors.Is(err, store.ErrInstallationNotFound) {
		t.Fatalf("err setelah reset = %v, mau ErrInstallationNotFound", err)
	}

	// Kuota terbuka lagi setelah reset.
	if _, err := s.Activate(context.Background(), l, "production", "1.0.1"); err != nil {
		t.Fatalf("Activate setelah reset: %v", err)
	}
}

func TestResetInstallationTidakDitemukan(t *testing.T) {
	s := testStore(t)
	err := s.ResetInstallation(context.Background(), "inst_tidak_ada")
	if !errors.Is(err, store.ErrInstallationNotFound) {
		t.Fatalf("err = %v, mau ErrInstallationNotFound", err)
	}
}

func TestResetInstallationDuaKaliDitolakKeduaKalinya(t *testing.T) {
	s := testStore(t)
	now := time.Date(2026, 9, 13, 0, 0, 0, 0, time.UTC)
	l := createTestCustomerAndLicense(t, s, now)
	inst, err := s.Activate(context.Background(), l, "production", "1.0.0")
	if err != nil {
		t.Fatalf("Activate: %v", err)
	}

	if err := s.ResetInstallation(context.Background(), inst.ID); err != nil {
		t.Fatalf("reset pertama: %v", err)
	}
	if err := s.ResetInstallation(context.Background(), inst.ID); !errors.Is(err, store.ErrInstallationNotFound) {
		t.Fatalf("reset kedua err = %v, mau ErrInstallationNotFound", err)
	}
}
