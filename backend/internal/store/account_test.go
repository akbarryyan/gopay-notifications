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
		// Tidak ada status "expiring" (peringatan dini) -- dicabut atas
		// permintaan eksplisit Akbar: sisa waktu berminggu-minggu tidak
		// boleh ditandai "akan berakhir". Tetap "active" sampai PERSIS
		// melewati expires_at.
		{"akan berakhir dalam 10 hari, tetap aktif", store.Account{AdminStatus: "active", ExpiresAt: now.Add(10 * 24 * time.Hour)}, "active"},
		{"akan berakhir dalam 1 jam, tetap aktif", store.Account{AdminStatus: "active", ExpiresAt: now.Add(time.Hour)}, "active"},
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

func TestSetAccountPlan(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	if err := s.CreateAccount(ctx, store.CreateAccountInput{
		ID: "acc_plan", BusinessName: "Toko Plan", Email: "plan@uji.test",
		Username: "toko_plan", PlaintextPassword: "x", Plan: "Starter", MaxDevices: 3,
		ExpiresAt: time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := s.SetAccountPlan(ctx, "acc_plan", "Business", 10); err != nil {
		t.Fatalf("set plan: %v", err)
	}
	acc, err := s.GetAccountByID(ctx, "acc_plan")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if acc.Plan != "Business" || acc.MaxDevices != 10 {
		t.Fatalf("plan/max_devices = %q/%d, mau Business/10", acc.Plan, acc.MaxDevices)
	}
}

func TestSetAccountPlanTidakDitemukan(t *testing.T) {
	s := testStore(t)
	err := s.SetAccountPlan(context.Background(), "tidak-ada", "Business", 10)
	if err != store.ErrAccountNotFound {
		t.Fatalf("err = %v, mau ErrAccountNotFound", err)
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

func TestAccountsNeedingExpiryReminder(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	now := time.Now()

	mustAccount := func(id string, expiresAt time.Time) {
		t.Helper()
		if err := s.CreateAccount(ctx, store.CreateAccountInput{
			ID: id, BusinessName: id, Email: id + "@uji.test", Username: id,
			PlaintextPassword: "rahasia123", Plan: "Starter", MaxDevices: 3,
			ExpiresAt: expiresAt,
		}); err != nil {
			t.Fatalf("create %s: %v", id, err)
		}
	}

	mustAccount("acc_besok", now.Add(24*time.Hour))        // masuk
	mustAccount("acc_seminggu", now.Add(6*24*time.Hour))   // masuk
	mustAccount("acc_sebulan", now.Add(30*24*time.Hour))   // terlalu jauh
	mustAccount("acc_kedaluwarsa", now.Add(-24*time.Hour)) // sudah lewat
	mustAccount("acc_suspended", now.Add(2*24*time.Hour))  // disuspend
	if err := s.SetAccountAdminStatus(ctx, "acc_suspended", "suspended"); err != nil {
		t.Fatalf("suspend: %v", err)
	}

	got, err := s.AccountsNeedingExpiryReminder(ctx, now, 7)
	if err != nil {
		t.Fatalf("AccountsNeedingExpiryReminder: %v", err)
	}
	ids := make([]string, 0, len(got))
	for _, a := range got {
		ids = append(ids, a.ID)
	}
	if len(ids) != 2 || ids[0] != "acc_besok" || ids[1] != "acc_seminggu" {
		t.Fatalf("ids = %v, mau [acc_besok acc_seminggu] (terurut expires_at)", ids)
	}
}

func TestExpiryReminderTidakDikirimDuaKaliTapiTerbukaLagiSetelahRenew(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	now := time.Now()
	expiresAt := now.Add(3 * 24 * time.Hour).Truncate(time.Microsecond)

	if err := s.CreateAccount(ctx, store.CreateAccountInput{
		ID: "acc_1", BusinessName: "Toko", Email: "t@uji.test", Username: "toko",
		PlaintextPassword: "rahasia123", Plan: "Starter", MaxDevices: 3, ExpiresAt: expiresAt,
	}); err != nil {
		t.Fatalf("create: %v", err)
	}

	acc, err := s.GetAccountByID(ctx, "acc_1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if err := s.MarkExpiryReminderSent(ctx, "acc_1", acc.ExpiresAt); err != nil {
		t.Fatalf("mark: %v", err)
	}

	got, err := s.AccountsNeedingExpiryReminder(ctx, now, 7)
	if err != nil {
		t.Fatalf("query kedua: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("got = %+v, mau kosong -- pengingat sudah dikirim untuk periode ini", got)
	}

	// Setelah diperpanjang, pengingat periode BARU harus terbuka lagi.
	newExpiry := now.Add(5 * 24 * time.Hour).Truncate(time.Microsecond)
	if err := s.RenewAccount(ctx, "acc_1", newExpiry); err != nil {
		t.Fatalf("renew: %v", err)
	}
	got, err = s.AccountsNeedingExpiryReminder(ctx, now, 7)
	if err != nil {
		t.Fatalf("query setelah renew: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got = %+v, mau 1 -- perpanjangan membuka pengingat periode berikutnya", got)
	}
}

func TestSetAccountTelegramChatID(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	seedAccount(t, s, "acc_1")

	chatID := "123456789"
	if err := s.SetAccountTelegramChatID(ctx, "acc_1", &chatID); err != nil {
		t.Fatalf("set: %v", err)
	}
	acc, _ := s.GetAccountByID(ctx, "acc_1")
	if acc.TelegramChatID == nil || *acc.TelegramChatID != chatID {
		t.Fatalf("TelegramChatID = %v, mau %q", acc.TelegramChatID, chatID)
	}

	if err := s.SetAccountTelegramChatID(ctx, "acc_1", nil); err != nil {
		t.Fatalf("hapus: %v", err)
	}
	acc, _ = s.GetAccountByID(ctx, "acc_1")
	if acc.TelegramChatID != nil {
		t.Fatalf("TelegramChatID = %v, mau nil setelah dicabut", acc.TelegramChatID)
	}
}
