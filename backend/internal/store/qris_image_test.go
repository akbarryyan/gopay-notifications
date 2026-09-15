package store_test

import (
	"context"
	"testing"

	"github.com/akbarryyan/gopay-notifications/backend/internal/store"
)

func TestUpsertDanGetQRISImage(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	seedAccount(t, s, "acc_1")

	png := []byte{0x89, 0x50, 0x4E, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}
	if err := s.UpsertQRISImage(ctx, "acc_1", png, "image/png"); err != nil {
		t.Fatalf("UpsertQRISImage: %v", err)
	}

	img, err := s.GetQRISImage(ctx, "acc_1")
	if err != nil {
		t.Fatalf("GetQRISImage: %v", err)
	}
	if string(img.ImageData) != string(png) || img.ContentType != "image/png" || img.AccountID != "acc_1" {
		t.Fatalf("img = %+v, mau data/tipe/account cocok dengan yang di-upload", img)
	}
}

func TestUpsertQRISImageMenimpaBukanMenambah(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	seedAccount(t, s, "acc_1")

	if err := s.UpsertQRISImage(ctx, "acc_1", []byte("gambar-lama"), "image/png"); err != nil {
		t.Fatalf("upsert pertama: %v", err)
	}
	if err := s.UpsertQRISImage(ctx, "acc_1", []byte("gambar-baru"), "image/jpeg"); err != nil {
		t.Fatalf("upsert kedua: %v", err)
	}

	img, err := s.GetQRISImage(ctx, "acc_1")
	if err != nil {
		t.Fatalf("GetQRISImage: %v", err)
	}
	if string(img.ImageData) != "gambar-baru" || img.ContentType != "image/jpeg" {
		t.Fatalf("img = %+v, mau menimpa jadi gambar-baru/image/jpeg", img)
	}

	var count int
	if err := s.Pool().QueryRow(ctx, "SELECT COUNT(*) FROM account_qris_images WHERE account_id = $1", "acc_1").Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 1 {
		t.Fatalf("count = %d, mau 1 baris (upsert, bukan insert baru)", count)
	}
}

func TestGetQRISImageTidakAdaMengembalikanErrNotFound(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	seedAccount(t, s, "acc_1")

	_, err := s.GetQRISImage(ctx, "acc_1")
	if err != store.ErrQRISImageNotFound {
		t.Fatalf("err = %v, mau ErrQRISImageNotFound", err)
	}
}

func TestHasQRISImage(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	seedAccount(t, s, "acc_1")

	has, err := s.HasQRISImage(ctx, "acc_1")
	if err != nil || has {
		t.Fatalf("has = %v, err = %v, mau false sebelum upload", has, err)
	}

	if err := s.UpsertQRISImage(ctx, "acc_1", []byte("x"), "image/png"); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	has, err = s.HasQRISImage(ctx, "acc_1")
	if err != nil || !has {
		t.Fatalf("has = %v, err = %v, mau true setelah upload", has, err)
	}
}

func TestDeleteQRISImageIdempotent(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	seedAccount(t, s, "acc_1")

	if err := s.DeleteQRISImage(ctx, "acc_1"); err != nil {
		t.Fatalf("delete pertama (belum ada): %v", err)
	}

	if err := s.UpsertQRISImage(ctx, "acc_1", []byte("x"), "image/png"); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if err := s.DeleteQRISImage(ctx, "acc_1"); err != nil {
		t.Fatalf("delete kedua: %v", err)
	}
	if err := s.DeleteQRISImage(ctx, "acc_1"); err != nil {
		t.Fatalf("delete ketiga (sudah tidak ada): %v", err)
	}

	if _, err := s.GetQRISImage(ctx, "acc_1"); err != store.ErrQRISImageNotFound {
		t.Fatalf("err = %v, mau ErrQRISImageNotFound setelah delete", err)
	}
}

func TestDeleteAccountIkutMenghapusQRISImage(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	seedAccount(t, s, "acc_1")
	if err := s.UpsertQRISImage(ctx, "acc_1", []byte("x"), "image/png"); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	if _, err := s.Pool().Exec(ctx, "DELETE FROM accounts WHERE id = $1", "acc_1"); err != nil {
		t.Fatalf("delete account: %v", err)
	}

	var count int
	if err := s.Pool().QueryRow(ctx, "SELECT COUNT(*) FROM account_qris_images WHERE account_id = $1", "acc_1").Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 0 {
		t.Fatalf("count = %d, mau 0 (ON DELETE CASCADE)", count)
	}
}
