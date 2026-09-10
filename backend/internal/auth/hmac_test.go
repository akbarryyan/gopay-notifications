package auth_test

import (
	"testing"
	"time"

	"github.com/akbar/gopay-notifications/backend/internal/auth"
)

var secret = []byte("secret-device-01")

func TestSigningStringFormat(t *testing.T) {
	// sha256("") = e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
	got := auth.SigningString("dev_01ABC", 1789036200, nil)
	want := "dev_01ABC\n1789036200\n" +
		"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	if got != want {
		t.Fatalf("SigningString =\n%q\nmau\n%q", got, want)
	}
}

func TestVerifyAcceptsCorrectSignature(t *testing.T) {
	s := auth.SigningString("dev_01ABC", 1789036200, []byte(`{"a":1}`))
	sig := auth.Sign(secret, s)

	if !auth.Verify(secret, s, sig) {
		t.Fatal("Verify menolak tanda tangan yang benar")
	}
}

func TestVerifyRejectsModifiedBody(t *testing.T) {
	orig := auth.SigningString("dev_01ABC", 1789036200, []byte(`{"a":1}`))
	sig := auth.Sign(secret, orig)

	tampered := auth.SigningString("dev_01ABC", 1789036200, []byte(`{"a":2}`))

	if auth.Verify(secret, tampered, sig) {
		t.Fatal("Verify menerima body yang sudah diubah")
	}
}

func TestVerifyRejectsWrongSecret(t *testing.T) {
	s := auth.SigningString("dev_01ABC", 1789036200, []byte(`{"a":1}`))
	sig := auth.Sign(secret, s)

	if auth.Verify([]byte("secret-lain"), s, sig) {
		t.Fatal("Verify menerima secret yang salah")
	}
}

func TestVerifyRejectsWrongDeviceID(t *testing.T) {
	s := auth.SigningString("dev_01ABC", 1789036200, []byte(`{"a":1}`))
	sig := auth.Sign(secret, s)

	other := auth.SigningString("dev_LAIN", 1789036200, []byte(`{"a":1}`))
	if auth.Verify(secret, other, sig) {
		t.Fatal("Verify menerima device id yang berbeda")
	}
}

func TestVerifyRejectsGarbageSignature(t *testing.T) {
	s := auth.SigningString("dev_01ABC", 1789036200, []byte(`{"a":1}`))

	for _, bad := range []string{"", "zzzz", "00"} {
		if auth.Verify(secret, s, bad) {
			t.Fatalf("Verify menerima tanda tangan sampah %q", bad)
		}
	}
}

func TestCheckSkewBoundaries(t *testing.T) {
	now := time.Unix(1789036200, 0)

	cases := []struct {
		name   string
		offset time.Duration
		want   bool
	}{
		{"tepat sekarang", 0, true},
		{"299 detik lampau", -299 * time.Second, true},
		{"300 detik lampau", -300 * time.Second, true},
		{"301 detik lampau", -301 * time.Second, false},
		{"299 detik depan", 299 * time.Second, true},
		{"300 detik depan", 300 * time.Second, true},
		{"301 detik depan", 301 * time.Second, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ts := now.Add(tc.offset).Unix()
			if got := auth.CheckSkew(ts, now); got != tc.want {
				t.Fatalf("CheckSkew(%s) = %v, mau %v", tc.name, got, tc.want)
			}
		})
	}
}
