package store_test

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/akbar/gopay-notifications/backend/internal/store"
)

func encKey() []byte {
	k := make([]byte, 32)
	for i := range k {
		k[i] = byte(i * 7)
	}
	return k
}

func TestCreateAndGetDevice(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	secret := []byte("secret-abc")

	if err := s.CreateDevice(ctx, encKey(), "dev_01ABC", "HP GoPay", secret); err != nil {
		t.Fatalf("CreateDevice: %v", err)
	}

	got, err := s.GetDevice(ctx, encKey(), "dev_01ABC")
	if err != nil {
		t.Fatalf("GetDevice: %v", err)
	}
	if got.DeviceID != "dev_01ABC" || got.Name != "HP GoPay" {
		t.Fatalf("device = %+v", got)
	}
	if !got.Enabled {
		t.Fatal("device baru seharusnya enabled")
	}
	if !bytes.Equal(got.Secret, secret) {
		t.Fatalf("Secret = %q, mau %q", got.Secret, secret)
	}
	if got.LastSeenAt != nil {
		t.Fatal("device baru seharusnya belum punya LastSeenAt")
	}
}

func TestSecretIsNotStoredInPlaintext(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	if err := s.CreateDevice(ctx, encKey(), "dev_01ABC", "HP", []byte("secret-abc")); err != nil {
		t.Fatalf("CreateDevice: %v", err)
	}

	var raw []byte
	err := s.Pool().QueryRow(ctx,
		"SELECT secret_enc FROM devices WHERE device_id = $1", "dev_01ABC").Scan(&raw)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if bytes.Contains(raw, []byte("secret-abc")) {
		t.Fatal("secret tersimpan sebagai plaintext di kolom secret_enc")
	}
}

func TestGetDeviceUnknownReturnsSentinel(t *testing.T) {
	s := testStore(t)

	_, err := s.GetDevice(context.Background(), encKey(), "dev_TIDAKADA")
	if !errors.Is(err, store.ErrDeviceNotFound) {
		t.Fatalf("err = %v, mau ErrDeviceNotFound", err)
	}
}

func TestTouchDeviceSetsLastSeenAt(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	if err := s.CreateDevice(ctx, encKey(), "dev_01ABC", "HP", []byte("s")); err != nil {
		t.Fatalf("CreateDevice: %v", err)
	}
	if err := s.TouchDevice(ctx, "dev_01ABC"); err != nil {
		t.Fatalf("TouchDevice: %v", err)
	}

	got, err := s.GetDevice(ctx, encKey(), "dev_01ABC")
	if err != nil {
		t.Fatalf("GetDevice: %v", err)
	}
	if got.LastSeenAt == nil {
		t.Fatal("LastSeenAt masih nil setelah TouchDevice")
	}
}
