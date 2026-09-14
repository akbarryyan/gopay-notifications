package store_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/akbarryyan/gopay-notifications/backend/internal/store"
)

func TestEmailVerificationTokenSekaliPakai(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	seedAccount(t, s, "acc_1")
	now := time.Now().UTC().Truncate(time.Microsecond)

	expires, err := s.CreateEmailVerificationToken(ctx, "acc_1", tokenHash("kode-a"), now)
	if err != nil || !expires.Equal(now.Add(store.EmailVerificationTTL)) {
		t.Fatalf("CreateEmailVerificationToken = %v, %v", expires, err)
	}

	before, _ := s.GetAccountByID(ctx, "acc_1")
	if before.EmailVerifiedAt != nil {
		t.Fatalf("EmailVerifiedAt = %v, mau nil sebelum diverifikasi", before.EmailVerifiedAt)
	}

	acc, err := s.ConsumeEmailVerificationToken(ctx, tokenHash("kode-a"), now.Add(time.Minute))
	if err != nil {
		t.Fatalf("ConsumeEmailVerificationToken: %v", err)
	}
	if acc.EmailVerifiedAt == nil || !acc.EmailVerifiedAt.Equal(now.Add(time.Minute)) {
		t.Fatalf("EmailVerifiedAt = %v, mau %v", acc.EmailVerifiedAt, now.Add(time.Minute))
	}

	if _, err := s.ConsumeEmailVerificationToken(ctx, tokenHash("kode-a"), now.Add(2*time.Minute)); !errors.Is(err, store.ErrEmailVerificationInvalid) {
		t.Fatalf("pakai ulang: err = %v, mau ErrEmailVerificationInvalid", err)
	}
}

func TestEmailVerificationTokenKedaluwarsaDanDigantikan(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	seedAccount(t, s, "acc_1")
	now := time.Now().UTC()

	if _, err := s.CreateEmailVerificationToken(ctx, "acc_1", tokenHash("basi"), now); err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := s.ConsumeEmailVerificationToken(ctx, tokenHash("basi"), now.Add(store.EmailVerificationTTL+time.Second)); !errors.Is(err, store.ErrEmailVerificationInvalid) {
		t.Fatalf("kedaluwarsa: err = %v", err)
	}

	if _, err := s.CreateEmailVerificationToken(ctx, "acc_1", tokenHash("lama"), now); err != nil {
		t.Fatalf("token lama: %v", err)
	}
	if _, err := s.CreateEmailVerificationToken(ctx, "acc_1", tokenHash("baru"), now); err != nil {
		t.Fatalf("token baru: %v", err)
	}
	if _, err := s.ConsumeEmailVerificationToken(ctx, tokenHash("lama"), now); !errors.Is(err, store.ErrEmailVerificationInvalid) {
		t.Fatalf("token lama masih berlaku: err = %v", err)
	}
	if _, err := s.ConsumeEmailVerificationToken(ctx, tokenHash("baru"), now); err != nil {
		t.Fatalf("token terbaru: %v", err)
	}

	recent, err := s.EmailVerificationRequestedSince(ctx, "acc_1", now.Add(-time.Minute))
	if err != nil || !recent {
		t.Fatalf("EmailVerificationRequestedSince = %v, %v; mau true", recent, err)
	}
}

func TestUpdateAccountProfileMenggantiEmailMengosongkanVerifikasi(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	seedAccount(t, s, "acc_1")
	now := time.Now().UTC()

	hash := tokenHash("verif")
	if _, err := s.CreateEmailVerificationToken(ctx, "acc_1", hash, now); err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := s.ConsumeEmailVerificationToken(ctx, hash, now); err != nil {
		t.Fatalf("consume: %v", err)
	}
	acc, _ := s.GetAccountByID(ctx, "acc_1")
	if acc.EmailVerifiedAt == nil {
		t.Fatal("email belum terverifikasi setelah consume")
	}

	// Simpan ulang dengan email yang SAMA -- verifikasi harus tetap.
	if err := s.UpdateAccountProfile(ctx, "acc_1", acc.BusinessName, acc.Email); err != nil {
		t.Fatalf("UpdateAccountProfile (email sama): %v", err)
	}
	acc, _ = s.GetAccountByID(ctx, "acc_1")
	if acc.EmailVerifiedAt == nil {
		t.Fatal("EmailVerifiedAt hilang padahal email tidak berubah")
	}

	// Ganti ke email BEDA -- verifikasi harus kembali kosong.
	if err := s.UpdateAccountProfile(ctx, "acc_1", acc.BusinessName, "baru@uji.test"); err != nil {
		t.Fatalf("UpdateAccountProfile (email beda): %v", err)
	}
	acc, _ = s.GetAccountByID(ctx, "acc_1")
	if acc.EmailVerifiedAt != nil {
		t.Fatal("EmailVerifiedAt masih terisi padahal email baru belum pernah diverifikasi")
	}
}
