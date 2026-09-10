package secretbox_test

import (
	"bytes"
	"testing"

	"github.com/akbarryyan/gopay-notifications/backend/internal/secretbox"
)

func key32() []byte {
	k := make([]byte, 32)
	for i := range k {
		k[i] = byte(i)
	}
	return k
}

func TestSealOpenRoundTrip(t *testing.T) {
	plain := []byte("rahasia-device-01")

	sealed, err := secretbox.Seal(key32(), plain)
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}
	if bytes.Contains(sealed, plain) {
		t.Fatal("ciphertext masih memuat plaintext")
	}

	got, err := secretbox.Open(key32(), sealed)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if !bytes.Equal(got, plain) {
		t.Fatalf("Open = %q, mau %q", got, plain)
	}
}

func TestSealProducesDifferentCiphertextEachTime(t *testing.T) {
	a, err := secretbox.Seal(key32(), []byte("sama"))
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}
	b, err := secretbox.Seal(key32(), []byte("sama"))
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}
	if bytes.Equal(a, b) {
		t.Fatal("dua Seal atas plaintext sama menghasilkan ciphertext identik — nonce tidak acak")
	}
}

func TestOpenRejectsWrongKey(t *testing.T) {
	sealed, err := secretbox.Seal(key32(), []byte("rahasia"))
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}

	wrong := key32()
	wrong[0] ^= 0xFF

	if _, err := secretbox.Open(wrong, sealed); err == nil {
		t.Fatal("mau error untuk kunci salah, dapat nil")
	}
}

func TestOpenRejectsTamperedCiphertext(t *testing.T) {
	sealed, err := secretbox.Seal(key32(), []byte("rahasia"))
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}
	sealed[len(sealed)-1] ^= 0x01

	if _, err := secretbox.Open(key32(), sealed); err == nil {
		t.Fatal("mau error untuk ciphertext yang diubah, dapat nil")
	}
}

func TestOpenRejectsShortInput(t *testing.T) {
	if _, err := secretbox.Open(key32(), []byte{1, 2, 3}); err == nil {
		t.Fatal("mau error untuk input lebih pendek dari nonce, dapat nil")
	}
}

func TestSealRejectsWrongKeySize(t *testing.T) {
	if _, err := secretbox.Seal([]byte("pendek"), []byte("x")); err == nil {
		t.Fatal("mau error untuk kunci bukan 32 byte, dapat nil")
	}
}
