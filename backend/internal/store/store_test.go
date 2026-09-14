package store_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/akbarryyan/gopay-notifications/backend/internal/store"
)

// testStore membuka koneksi ke database test dan mengosongkan seluruh tabel.
// Dipakai ulang oleh test lain di paket ini.
func testStore(t *testing.T) *store.Store {
	t.Helper()

	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Fatal("TEST_DATABASE_URL belum diset. Jalankan: make db-up migrate")
	}

	ctx := context.Background()
	s, err := store.New(ctx, url)
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	t.Cleanup(s.Close)

	_, err = s.Pool().Exec(ctx,
		"TRUNCATE notification_events, event_reviews, invoices, api_keys, webhook_deliveries, webhook_endpoints, devices, accounts, vendor_admins, audit_log, notification_settings RESTART IDENTITY CASCADE")
	if err != nil {
		t.Fatalf("truncate: %v", err)
	}
	return s
}

// seedAccount membuat account uji minimal, dipakai test yang butuh
// account_id valid buat foreign key devices/invoices/dst.
func seedAccount(t *testing.T, s *store.Store, id string) {
	t.Helper()
	err := s.CreateAccount(context.Background(), store.CreateAccountInput{
		ID: id, BusinessName: id, Email: id + "@uji.test", Username: id,
		PlaintextPassword: "rahasia123", Plan: "Business", MaxDevices: 10,
		ExpiresAt: time.Now().Add(365 * 24 * time.Hour),
	})
	if err != nil {
		t.Fatalf("seed account %s: %v", id, err)
	}
}

func TestNewConnectsAndTablesExist(t *testing.T) {
	s := testStore(t)

	var n int
	err := s.Pool().QueryRow(context.Background(),
		`SELECT count(*) FROM information_schema.tables
		 WHERE table_schema = 'public'
		   AND table_name IN ('devices', 'notification_events')`).Scan(&n)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if n != 2 {
		t.Fatalf("jumlah tabel = %d, mau 2 — migrasi belum dijalankan?", n)
	}
}

func TestNewRejectsBadURL(t *testing.T) {
	_, err := store.New(context.Background(), "postgres://nobody@127.0.0.1:1/none?sslmode=disable")
	if err == nil {
		t.Fatal("mau error untuk URL yang tidak bisa dihubungi, dapat nil")
	}
}
