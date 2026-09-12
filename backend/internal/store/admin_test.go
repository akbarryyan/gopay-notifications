package store_test

import (
	"context"
	"errors"
	"testing"

	"github.com/akbarryyan/gopay-notifications/backend/internal/store"
)

func TestUpsertAdminLaluVerifikasiPassword(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	if err := s.UpsertAdmin(ctx, "admin", "rahasia-awal"); err != nil {
		t.Fatalf("UpsertAdmin: %v", err)
	}

	got, err := s.GetAdminByUsername(ctx, "admin")
	if err != nil {
		t.Fatalf("GetAdminByUsername: %v", err)
	}
	if !got.VerifyPassword("rahasia-awal") {
		t.Fatal("password benar seharusnya diterima")
	}
	if got.VerifyPassword("salah") {
		t.Fatal("password salah seharusnya ditolak")
	}
}

func TestUpsertAdminMenggantiPasswordLama(t *testing.T) {
	// cmd/admintool dipakai baik untuk pembuatan awal maupun reset password —
	// keduanya operasi yang sama, bukan dua perintah berbeda.
	s := testStore(t)
	ctx := context.Background()

	if err := s.UpsertAdmin(ctx, "admin", "password-lama"); err != nil {
		t.Fatalf("upsert pertama: %v", err)
	}
	if err := s.UpsertAdmin(ctx, "admin", "password-baru"); err != nil {
		t.Fatalf("upsert kedua: %v", err)
	}

	got, err := s.GetAdminByUsername(ctx, "admin")
	if err != nil {
		t.Fatalf("GetAdminByUsername: %v", err)
	}
	if got.VerifyPassword("password-lama") {
		t.Fatal("password lama seharusnya tidak lagi berlaku")
	}
	if !got.VerifyPassword("password-baru") {
		t.Fatal("password baru seharusnya berlaku")
	}
}

func TestGetAdminUnknownReturnsSentinel(t *testing.T) {
	s := testStore(t)

	_, err := s.GetAdminByUsername(context.Background(), "tidak-ada")
	if !errors.Is(err, store.ErrAdminNotFound) {
		t.Fatalf("err = %v, mau ErrAdminNotFound", err)
	}
}

func TestPasswordTidakTersimpanSebagaiPlaintext(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	if err := s.UpsertAdmin(ctx, "admin", "rahasia-jangan-bocor"); err != nil {
		t.Fatalf("UpsertAdmin: %v", err)
	}

	var hash string
	err := s.Pool().QueryRow(ctx,
		"SELECT password_hash FROM admin_users WHERE username = $1", "admin").Scan(&hash)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if hash == "rahasia-jangan-bocor" {
		t.Fatal("password tersimpan sebagai plaintext")
	}
}
