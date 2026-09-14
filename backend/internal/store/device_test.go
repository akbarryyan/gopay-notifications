package store_test

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/akbarryyan/gopay-notifications/backend/internal/store"
)

func encKey() []byte {
	k := make([]byte, 32)
	for i := range k {
		k[i] = byte(i * 7)
	}
	return k
}

func mustCreateDevice(t *testing.T, s *store.Store, accountID, deviceID, name string) {
	t.Helper()
	if err := s.CreateDevice(context.Background(), encKey(), accountID, deviceID, name, []byte("secret")); err != nil {
		t.Fatalf("create device: %v", err)
	}
}

func TestCreateAndGetDevice(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	secret := []byte("secret-abc")
	seedAccount(t, s, "acc_1")

	if err := s.CreateDevice(ctx, encKey(), "acc_1", "dev_01ABC", "HP GoPay", secret); err != nil {
		t.Fatalf("CreateDevice: %v", err)
	}

	got, err := s.GetDevice(ctx, encKey(), "dev_01ABC")
	if err != nil {
		t.Fatalf("GetDevice: %v", err)
	}
	if got.DeviceID != "dev_01ABC" || got.Name != "HP GoPay" {
		t.Fatalf("device = %+v", got)
	}
	if got.AccountID != "acc_1" {
		t.Fatalf("AccountID = %q, mau acc_1", got.AccountID)
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
	seedAccount(t, s, "acc_1")

	if err := s.CreateDevice(ctx, encKey(), "acc_1", "dev_01ABC", "HP", []byte("secret-abc")); err != nil {
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
	seedAccount(t, s, "acc_1")

	if err := s.CreateDevice(ctx, encKey(), "acc_1", "dev_01ABC", "HP", []byte("s")); err != nil {
		t.Fatalf("CreateDevice: %v", err)
	}
	if err := s.TouchDevice(ctx, "dev_01ABC", "1.0.0"); err != nil {
		t.Fatalf("TouchDevice: %v", err)
	}

	got, err := s.GetDevice(ctx, encKey(), "dev_01ABC")
	if err != nil {
		t.Fatalf("GetDevice: %v", err)
	}
	if got.LastSeenAt == nil {
		t.Fatal("LastSeenAt masih nil setelah TouchDevice")
	}
	if got.AppVersion == nil || *got.AppVersion != "1.0.0" {
		t.Fatalf("AppVersion = %v, mau 1.0.0", got.AppVersion)
	}
}

func TestTouchDeviceVersiKosongTidakMenimpa(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	seedAccount(t, s, "acc_1")

	if err := s.CreateDevice(ctx, encKey(), "acc_1", "dev_01ABC", "HP", []byte("s")); err != nil {
		t.Fatalf("CreateDevice: %v", err)
	}
	if err := s.TouchDevice(ctx, "dev_01ABC", "1.2.3"); err != nil {
		t.Fatalf("TouchDevice pertama: %v", err)
	}

	// Aplikasi versi lama tidak mengirim header X-App-Version. Menghapus
	// versi yang sudah diketahui justru membuang informasi.
	if err := s.TouchDevice(ctx, "dev_01ABC", ""); err != nil {
		t.Fatalf("TouchDevice kedua: %v", err)
	}

	got, err := s.GetDevice(ctx, encKey(), "dev_01ABC")
	if err != nil {
		t.Fatalf("GetDevice: %v", err)
	}
	if got.AppVersion == nil || *got.AppVersion != "1.2.3" {
		t.Fatalf("AppVersion = %v, mau tetap 1.2.3", got.AppVersion)
	}
}

func TestListDevicesHanyaMilikAccountSendiri(t *testing.T) {
	s := testStore(t)
	seedAccount(t, s, "acc_a")
	seedAccount(t, s, "acc_b")

	mustCreateDevice(t, s, "acc_a", "dev_a1", "HP A1")
	mustCreateDevice(t, s, "acc_b", "dev_b1", "HP B1")

	listA, err := s.ListDevices(context.Background(), "acc_a")
	if err != nil {
		t.Fatalf("list devices acc_a: %v", err)
	}
	if len(listA) != 1 || listA[0].DeviceID != "dev_a1" {
		t.Fatalf("acc_a seharusnya cuma lihat dev_a1, dapat: %+v", listA)
	}
}

func TestSetDeviceEnabledMilikAccountLainDitolak(t *testing.T) {
	s := testStore(t)
	seedAccount(t, s, "acc_a")
	seedAccount(t, s, "acc_b")
	mustCreateDevice(t, s, "acc_a", "dev_a1", "HP A1")

	err := s.SetDeviceEnabled(context.Background(), "acc_b", "dev_a1", false)
	if err != store.ErrDeviceNotFound {
		t.Fatalf("err = %v, mau ErrDeviceNotFound (device milik akun lain)", err)
	}
}

func TestSetDeviceEnabledMilikSendiri(t *testing.T) {
	s := testStore(t)
	seedAccount(t, s, "acc_a")
	mustCreateDevice(t, s, "acc_a", "dev_a1", "HP A1")

	if err := s.SetDeviceEnabled(context.Background(), "acc_a", "dev_a1", false); err != nil {
		t.Fatalf("set device enabled: %v", err)
	}
	got, err := s.GetDevice(context.Background(), encKey(), "dev_a1")
	if err != nil {
		t.Fatalf("get device: %v", err)
	}
	if got.Enabled {
		t.Fatal("device seharusnya disabled")
	}
}

func TestDeleteDeviceTanpaRiwayatBerhasil(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	seedAccount(t, s, "acc_a")
	mustCreateDevice(t, s, "acc_a", "dev_a1", "HP A1")

	if err := s.DeleteDevice(ctx, "acc_a", "dev_a1"); err != nil {
		t.Fatalf("delete device: %v", err)
	}
	if _, err := s.GetDevice(ctx, encKey(), "dev_a1"); !errors.Is(err, store.ErrDeviceNotFound) {
		t.Fatalf("err = %v, mau ErrDeviceNotFound setelah dihapus", err)
	}
}

func TestDeleteDeviceDenganRiwayatDitolak(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	seedAccount(t, s, "acc_1") // sampleEvent() hardcode account_id "acc_1"/device_id "dev_01ABC"
	mustCreateDevice(t, s, "acc_1", "dev_01ABC", "HP Terpakai")
	if _, err := s.InsertEvent(ctx, sampleEvent("evt_riwayat")); err != nil {
		t.Fatalf("insert event: %v", err)
	}

	err := s.DeleteDevice(ctx, "acc_1", "dev_01ABC")
	if !errors.Is(err, store.ErrDeviceHasEvents) {
		t.Fatalf("err = %v, mau ErrDeviceHasEvents", err)
	}
	// Device tetap ada -- DeleteDevice yang ditolak tidak boleh menghapus
	// apa pun.
	if _, err := s.GetDevice(ctx, encKey(), "dev_01ABC"); err != nil {
		t.Fatalf("device seharusnya masih ada: %v", err)
	}
}

func TestDeleteDeviceTidakDitemukan(t *testing.T) {
	s := testStore(t)
	err := s.DeleteDevice(context.Background(), "acc_1", "tidak-ada")
	if !errors.Is(err, store.ErrDeviceNotFound) {
		t.Fatalf("err = %v, mau ErrDeviceNotFound", err)
	}
}

func TestDeleteDeviceMilikAkunLainDitolak(t *testing.T) {
	s := testStore(t)
	seedAccount(t, s, "acc_a")
	seedAccount(t, s, "acc_b")
	mustCreateDevice(t, s, "acc_a", "dev_a1", "HP A1")

	err := s.DeleteDevice(context.Background(), "acc_b", "dev_a1")
	if !errors.Is(err, store.ErrDeviceNotFound) {
		t.Fatalf("err = %v, mau ErrDeviceNotFound (device milik akun lain)", err)
	}
}
