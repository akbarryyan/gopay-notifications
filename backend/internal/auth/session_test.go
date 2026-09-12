package auth_test

import (
	"testing"
	"time"

	"github.com/akbarryyan/gopay-notifications/backend/internal/auth"
)

var sessionKey = []byte("session-key-untuk-test")

func TestSessionTokenValidLangsungSetelahDibuat(t *testing.T) {
	now := time.Unix(1789200000, 0)
	token := auth.NewSessionToken(sessionKey, now)

	if !auth.VerifySessionToken(sessionKey, token, now) {
		t.Fatal("token yang baru dibuat seharusnya valid")
	}
}

func TestSessionTokenValidSelamaBelumKedaluwarsa(t *testing.T) {
	now := time.Unix(1789200000, 0)
	token := auth.NewSessionToken(sessionKey, now)

	nantiTapiMasihDalamMasaBerlaku := now.Add(auth.SessionDuration - time.Minute)
	if !auth.VerifySessionToken(sessionKey, token, nantiTapiMasihDalamMasaBerlaku) {
		t.Fatal("token seharusnya masih valid sebelum kedaluwarsa")
	}
}

func TestSessionTokenKedaluwarsaDitolak(t *testing.T) {
	now := time.Unix(1789200000, 0)
	token := auth.NewSessionToken(sessionKey, now)

	setelahKedaluwarsa := now.Add(auth.SessionDuration + time.Minute)
	if auth.VerifySessionToken(sessionKey, token, setelahKedaluwarsa) {
		t.Fatal("token yang sudah kedaluwarsa seharusnya ditolak")
	}
}

func TestSessionTokenKunciSalahDitolak(t *testing.T) {
	now := time.Unix(1789200000, 0)
	token := auth.NewSessionToken(sessionKey, now)

	if auth.VerifySessionToken([]byte("kunci-lain"), token, now) {
		t.Fatal("token dengan kunci verifikasi yang salah seharusnya ditolak")
	}
}

func TestSessionTokenYangDiubahDitolak(t *testing.T) {
	now := time.Unix(1789200000, 0)
	token := auth.NewSessionToken(sessionKey, now)

	diubah := token[:len(token)-1] + "x"
	if auth.VerifySessionToken(sessionKey, diubah, now) {
		t.Fatal("token yang tanda tangannya diubah seharusnya ditolak")
	}
}

func TestSessionTokenSampahDitolakTanpaCrash(t *testing.T) {
	now := time.Unix(1789200000, 0)

	for _, sampah := range []string{"", "tanpa-titik", ".", "abc.def", "123", "123."} {
		if auth.VerifySessionToken(sessionKey, sampah, now) {
			t.Fatalf("token sampah %q seharusnya ditolak", sampah)
		}
	}
}
