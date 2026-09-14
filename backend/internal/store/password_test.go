package store_test

import (
	"context"
	"crypto/sha256"
	"errors"
	"testing"
	"time"

	"github.com/akbarryyan/gopay-notifications/backend/internal/store"
)

func tokenHash(raw string) []byte {
	h := sha256.Sum256([]byte(raw))
	return h[:]
}

func TestGetAccountByEmailTidakPekaHurufBesar(t *testing.T) {
	s := testStore(t)
	seedAccount(t, s, "acc_1")

	acc, err := s.GetAccountByEmail(context.Background(), "ACC_1@Uji.Test")
	if err != nil || acc.ID != "acc_1" {
		t.Fatalf("GetAccountByEmail = %+v, %v; mau acc_1", acc, err)
	}
	if _, err := s.GetAccountByEmail(context.Background(), "tidak@ada.test"); !errors.Is(err, store.ErrAccountNotFound) {
		t.Fatalf("err = %v, mau ErrAccountNotFound", err)
	}
}

func TestResetPasswordWithTokenSekaliPakai(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	seedAccount(t, s, "acc_1")
	now := time.Now().UTC().Truncate(time.Microsecond)

	if err := s.CreatePasswordResetToken(ctx, "acc_1", tokenHash("token-a"), now); err != nil {
		t.Fatalf("CreatePasswordResetToken: %v", err)
	}
	acc, err := s.ResetPasswordWithToken(ctx, tokenHash("token-a"), "password-baru-123", now.Add(time.Minute))
	if err != nil {
		t.Fatalf("ResetPasswordWithToken: %v", err)
	}
	if acc.ID != "acc_1" || !acc.VerifyPassword("password-baru-123") || acc.VerifyPassword("rahasia123") {
		t.Fatalf("password tidak berganti: %+v", acc)
	}
	if acc.PasswordChangedAt == nil || !acc.PasswordChangedAt.Equal(now.Add(time.Minute)) {
		t.Fatalf("PasswordChangedAt = %v, mau waktu reset", acc.PasswordChangedAt)
	}

	if _, err := s.ResetPasswordWithToken(ctx, tokenHash("token-a"), "lagi-12345", now.Add(2*time.Minute)); !errors.Is(err, store.ErrResetTokenInvalid) {
		t.Fatalf("pakai ulang token: err = %v, mau ErrResetTokenInvalid", err)
	}
}

func TestResetPasswordTokenKedaluwarsaDanTokenLamaDigantikan(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	seedAccount(t, s, "acc_1")
	now := time.Now().UTC()

	if err := s.CreatePasswordResetToken(ctx, "acc_1", tokenHash("kedaluwarsa"), now); err != nil {
		t.Fatalf("CreatePasswordResetToken: %v", err)
	}
	if _, err := s.ResetPasswordWithToken(ctx, tokenHash("kedaluwarsa"), "password-baru-123",
		now.Add(store.PasswordResetTTL+time.Second)); !errors.Is(err, store.ErrResetTokenInvalid) {
		t.Fatalf("token kedaluwarsa: err = %v, mau ErrResetTokenInvalid", err)
	}

	// Permintaan baru membatalkan link dari email sebelumnya.
	if err := s.CreatePasswordResetToken(ctx, "acc_1", tokenHash("lama"), now); err != nil {
		t.Fatalf("token lama: %v", err)
	}
	if err := s.CreatePasswordResetToken(ctx, "acc_1", tokenHash("baru"), now); err != nil {
		t.Fatalf("token baru: %v", err)
	}
	if _, err := s.ResetPasswordWithToken(ctx, tokenHash("lama"), "password-baru-123", now); !errors.Is(err, store.ErrResetTokenInvalid) {
		t.Fatalf("token lama: err = %v, mau ErrResetTokenInvalid", err)
	}
	if _, err := s.ResetPasswordWithToken(ctx, tokenHash("baru"), "password-baru-123", now); err != nil {
		t.Fatalf("token terbaru: %v", err)
	}

	recent, err := s.PasswordResetRequestedSince(ctx, "acc_1", now.Add(-time.Minute))
	if err != nil || !recent {
		t.Fatalf("PasswordResetRequestedSince = %v, %v; mau true", recent, err)
	}
}

func TestChangeAccountPasswordMembatalkanTokenReset(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	seedAccount(t, s, "acc_1")
	now := time.Now().UTC()

	if err := s.CreatePasswordResetToken(ctx, "acc_1", tokenHash("tertinggal"), now); err != nil {
		t.Fatalf("CreatePasswordResetToken: %v", err)
	}
	if err := s.ChangeAccountPassword(ctx, "acc_1", "password-baru-123", now); err != nil {
		t.Fatalf("ChangeAccountPassword: %v", err)
	}
	acc, _ := s.GetAccountByID(ctx, "acc_1")
	if !acc.VerifyPassword("password-baru-123") || acc.PasswordChangedAt == nil {
		t.Fatalf("password tidak berganti: %+v", acc)
	}
	if _, err := s.ResetPasswordWithToken(ctx, tokenHash("tertinggal"), "punya-penyerang", now); !errors.Is(err, store.ErrResetTokenInvalid) {
		t.Fatalf("link reset lama masih berlaku setelah ganti password: err = %v", err)
	}
	if err := s.ChangeAccountPassword(ctx, "acc_tidak_ada", "password-baru-123", now); !errors.Is(err, store.ErrAccountNotFound) {
		t.Fatalf("err = %v, mau ErrAccountNotFound", err)
	}
}

func TestUpdateAccountProfile(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	seedAccount(t, s, "acc_1")
	seedAccount(t, s, "acc_2")

	if err := s.UpdateAccountProfile(ctx, "acc_1", "Toko Baru", "baru@uji.test"); err != nil {
		t.Fatalf("UpdateAccountProfile: %v", err)
	}
	acc, _ := s.GetAccountByID(ctx, "acc_1")
	if acc.BusinessName != "Toko Baru" || acc.Email != "baru@uji.test" {
		t.Fatalf("account = %+v", acc)
	}
	if err := s.UpdateAccountProfile(ctx, "acc_1", "Toko Baru", "acc_2@uji.test"); !errors.Is(err, store.ErrAccountEmailTaken) {
		t.Fatalf("err = %v, mau ErrAccountEmailTaken", err)
	}
}
