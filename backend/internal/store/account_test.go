package store_test

import (
	"context"
	"testing"
	"time"

	"github.com/akbarryyan/gopay-notifications/backend/internal/store"
)

func TestCreateAccountDanGetByUsername(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	err := s.CreateAccount(ctx, store.CreateAccountInput{
		ID: "acc_1", BusinessName: "Toko Uji", Email: "toko@uji.test",
		Username: "toko_uji", PlaintextPassword: "rahasia123",
		Plan: "Business", MaxDevices: 10, ExpiresAt: time.Now().Add(30 * 24 * time.Hour),
	})
	if err != nil {
		t.Fatalf("create account: %v", err)
	}

	acc, err := s.GetAccountByUsername(ctx, "toko_uji")
	if err != nil {
		t.Fatalf("get by username: %v", err)
	}
	if acc.BusinessName != "Toko Uji" || acc.Plan != "Business" || acc.MaxDevices != 10 {
		t.Fatalf("account salah: %+v", acc)
	}
	if !acc.VerifyPassword("rahasia123") {
		t.Fatal("password seharusnya cocok")
	}
	if acc.VerifyPassword("salah") {
		t.Fatal("password salah seharusnya ditolak")
	}
}

func TestGetAccountByUsernameTidakDitemukan(t *testing.T) {
	s := testStore(t)
	_, err := s.GetAccountByUsername(context.Background(), "tidak-ada")
	if err != store.ErrAccountNotFound {
		t.Fatalf("err = %v, mau ErrAccountNotFound", err)
	}
}

func TestAccountDerivedStatus(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	cases := []struct {
		name   string
		acc    store.Account
		expect string
	}{
		{"aktif jauh dari kedaluwarsa", store.Account{AdminStatus: "active", ExpiresAt: now.Add(200 * 24 * time.Hour)}, "active"},
		{"akan berakhir dalam 10 hari", store.Account{AdminStatus: "active", ExpiresAt: now.Add(10 * 24 * time.Hour)}, "expiring"},
		{"sudah lewat", store.Account{AdminStatus: "active", ExpiresAt: now.Add(-time.Hour)}, "expired"},
		{"disuspend walau belum kedaluwarsa", store.Account{AdminStatus: "suspended", ExpiresAt: now.Add(200 * 24 * time.Hour)}, "suspended"},
		{"dicabut walau belum kedaluwarsa", store.Account{AdminStatus: "revoked", ExpiresAt: now.Add(200 * 24 * time.Hour)}, "revoked"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.acc.DerivedStatus(now); got != c.expect {
				t.Fatalf("DerivedStatus = %q, mau %q", got, c.expect)
			}
		})
	}
}

func TestAccountOperational(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	active := store.Account{AdminStatus: "active", ExpiresAt: now.Add(365 * 24 * time.Hour)}
	if !active.Operational(now) {
		t.Fatal("account active seharusnya operational")
	}
	expired := store.Account{AdminStatus: "active", ExpiresAt: now.Add(-time.Hour)}
	if expired.Operational(now) {
		t.Fatal("account expired seharusnya tidak operational")
	}
	suspended := store.Account{AdminStatus: "suspended", ExpiresAt: now.Add(365 * 24 * time.Hour)}
	if suspended.Operational(now) {
		t.Fatal("account suspended seharusnya tidak operational")
	}
}

func TestRenewAccount(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	newExpiry := time.Now().Add(365 * 24 * time.Hour).Truncate(time.Microsecond)

	err := s.CreateAccount(ctx, store.CreateAccountInput{
		ID: "acc_2", BusinessName: "Toko B", Email: "b@uji.test",
		Username: "toko_b", PlaintextPassword: "x", Plan: "Starter", MaxDevices: 3,
		ExpiresAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := s.RenewAccount(ctx, "acc_2", newExpiry); err != nil {
		t.Fatalf("renew: %v", err)
	}
	acc, err := s.GetAccountByID(ctx, "acc_2")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if !acc.ExpiresAt.Equal(newExpiry) {
		t.Fatalf("expires_at tidak berubah: %v, mau %v", acc.ExpiresAt, newExpiry)
	}
}

func TestRenewAccountTidakDitemukan(t *testing.T) {
	s := testStore(t)
	err := s.RenewAccount(context.Background(), "tidak-ada", time.Now())
	if err != store.ErrAccountNotFound {
		t.Fatalf("err = %v, mau ErrAccountNotFound", err)
	}
}

func TestSetAccountAdminStatus(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	if err := s.CreateAccount(ctx, store.CreateAccountInput{
		ID: "acc_3", BusinessName: "Toko C", Email: "c@uji.test",
		Username: "toko_c", PlaintextPassword: "x", Plan: "Starter", MaxDevices: 3,
		ExpiresAt: time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := s.SetAccountAdminStatus(ctx, "acc_3", "suspended"); err != nil {
		t.Fatalf("suspend: %v", err)
	}
	acc, _ := s.GetAccountByID(ctx, "acc_3")
	if acc.AdminStatus != "suspended" {
		t.Fatalf("admin_status = %q, mau suspended", acc.AdminStatus)
	}
}

func TestListAccounts(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	seedAccount(t, s, "acc_x")
	seedAccount(t, s, "acc_y")

	list, err := s.ListAccounts(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("len(list) = %d, mau 2", len(list))
	}
}

func TestCreateAccountEmailBentrokDitolak(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	if err := s.CreateAccount(ctx, store.CreateAccountInput{
		ID: "acc_a", BusinessName: "Toko A", Email: "sama@uji.test",
		Username: "toko_a", PlaintextPassword: "rahasia123",
		Plan: "Starter", MaxDevices: 3, ExpiresAt: time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatalf("create pertama: %v", err)
	}

	err := s.CreateAccount(ctx, store.CreateAccountInput{
		ID: "acc_b", BusinessName: "Toko B", Email: "sama@uji.test",
		Username: "toko_b", PlaintextPassword: "rahasia123",
		Plan: "Starter", MaxDevices: 3, ExpiresAt: time.Now().Add(time.Hour),
	})
	if err != store.ErrAccountEmailTaken {
		t.Fatalf("err = %v, mau ErrAccountEmailTaken", err)
	}
}

func TestCreateAccountUsernameBentrokDitolak(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	if err := s.CreateAccount(ctx, store.CreateAccountInput{
		ID: "acc_a", BusinessName: "Toko A", Email: "a@uji.test",
		Username: "sama_username", PlaintextPassword: "rahasia123",
		Plan: "Starter", MaxDevices: 3, ExpiresAt: time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatalf("create pertama: %v", err)
	}

	err := s.CreateAccount(ctx, store.CreateAccountInput{
		ID: "acc_b", BusinessName: "Toko B", Email: "b@uji.test",
		Username: "sama_username", PlaintextPassword: "rahasia123",
		Plan: "Starter", MaxDevices: 3, ExpiresAt: time.Now().Add(time.Hour),
	})
	if err != store.ErrAccountUsernameTaken {
		t.Fatalf("err = %v, mau ErrAccountUsernameTaken", err)
	}
}
