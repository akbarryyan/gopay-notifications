package auth_test

import (
	"testing"
	"time"

	"github.com/akbarryyan/gopay-notifications/backend/internal/auth"
)

var sessionKey = []byte("session-key-untuk-test")

func TestSessionTokenValidLangsungSetelahDibuat(t *testing.T) {
	now := time.Unix(1789200000, 0)
	token := auth.NewSessionToken(sessionKey, now, "acc_test")

	subject, ok := auth.VerifySessionToken(sessionKey, token, now)
	if !ok {
		t.Fatal("token yang baru dibuat seharusnya valid")
	}
	if subject != "acc_test" {
		t.Fatalf("subject = %q, mau %q", subject, "acc_test")
	}
}

func TestSessionTokenValidSelamaBelumKedaluwarsa(t *testing.T) {
	now := time.Unix(1789200000, 0)
	token := auth.NewSessionToken(sessionKey, now, "acc_test")

	nantiTapiMasihDalamMasaBerlaku := now.Add(auth.SessionDuration - time.Minute)
	if _, ok := auth.VerifySessionToken(sessionKey, token, nantiTapiMasihDalamMasaBerlaku); !ok {
		t.Fatal("token seharusnya masih valid sebelum kedaluwarsa")
	}
}

func TestSessionTokenKedaluwarsaDitolak(t *testing.T) {
	now := time.Unix(1789200000, 0)
	token := auth.NewSessionToken(sessionKey, now, "acc_test")

	setelahKedaluwarsa := now.Add(auth.SessionDuration + time.Minute)
	if _, ok := auth.VerifySessionToken(sessionKey, token, setelahKedaluwarsa); ok {
		t.Fatal("token yang sudah kedaluwarsa seharusnya ditolak")
	}
}

func TestSessionTokenKunciSalahDitolak(t *testing.T) {
	now := time.Unix(1789200000, 0)
	token := auth.NewSessionToken(sessionKey, now, "acc_test")

	if _, ok := auth.VerifySessionToken([]byte("kunci-lain"), token, now); ok {
		t.Fatal("token dengan kunci verifikasi yang salah seharusnya ditolak")
	}
}

func TestSessionTokenYangDiubahDitolak(t *testing.T) {
	now := time.Unix(1789200000, 0)
	token := auth.NewSessionToken(sessionKey, now, "acc_test")

	diubah := token[:len(token)-1] + "x"
	if _, ok := auth.VerifySessionToken(sessionKey, diubah, now); ok {
		t.Fatal("token yang tanda tangannya diubah seharusnya ditolak")
	}
}

func TestSessionTokenSampahDitolakTanpaCrash(t *testing.T) {
	now := time.Unix(1789200000, 0)

	for _, sampah := range []string{"", "tanpa-titik", ".", "abc.def", "123", "123.", "acc_x:abc.sig"} {
		if _, ok := auth.VerifySessionToken(sessionKey, sampah, now); ok {
			t.Fatalf("token sampah %q seharusnya ditolak", sampah)
		}
	}
}

func TestSessionTokenDuaSubjectBerbeda(t *testing.T) {
	now := time.Unix(1789200000, 0)
	tokenA := auth.NewSessionToken(sessionKey, now, "acc_a")
	tokenB := auth.NewSessionToken(sessionKey, now, "acc_b")

	subjectA, ok := auth.VerifySessionToken(sessionKey, tokenA, now)
	if !ok || subjectA != "acc_a" {
		t.Fatalf("tokenA: subject=%q ok=%v, mau acc_a/true", subjectA, ok)
	}
	subjectB, ok := auth.VerifySessionToken(sessionKey, tokenB, now)
	if !ok || subjectB != "acc_b" {
		t.Fatalf("tokenB: subject=%q ok=%v, mau acc_b/true", subjectB, ok)
	}
}

func TestSessionIssuedAtDariMasaBerlaku(t *testing.T) {
	issued := time.Unix(1789036200, 0)
	token := auth.NewSessionToken([]byte("kunci"), issued, "acc_1")

	got, ok := auth.SessionIssuedAt(token)
	if !ok || !got.Equal(issued) {
		t.Fatalf("SessionIssuedAt = %v, %v; mau %v", got, ok, issued)
	}
	if _, ok := auth.SessionIssuedAt("rusak"); ok {
		t.Fatal("token rusak dianggap sah")
	}
}
