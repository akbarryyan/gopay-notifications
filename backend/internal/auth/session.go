package auth

import (
	"crypto/hmac"
	"strconv"
	"strings"
	"time"
)

// SessionDuration adalah umur sesi dashboard admin sejak login.
const SessionDuration = 12 * time.Hour

// NewSessionToken membuat token sesi bertanda tangan: "<expiryUnix>.<hmac>".
//
// Stateless dengan sengaja — tidak ada tabel sesi di database. Kelebihannya:
// tidak perlu pembersihan sesi kedaluwarsa. Kekurangannya: sesi tidak dapat
// dicabut satu per satu sebelum kedaluwarsa (mis. saat logout paksa dari
// perangkat lain) — untuk MVP satu-admin, itu trade-off yang wajar.
func NewSessionToken(key []byte, now time.Time) string {
	expiry := now.Add(SessionDuration).Unix()
	return sessionToken(key, expiry)
}

func sessionToken(key []byte, expiry int64) string {
	payload := strconv.FormatInt(expiry, 10)
	sig := Sign(key, payload)
	return payload + "." + sig
}

// VerifySessionToken memeriksa tanda tangan dan masa berlaku token.
func VerifySessionToken(key []byte, token string, now time.Time) bool {
	payload, sig, ok := strings.Cut(token, ".")
	if !ok || payload == "" || sig == "" {
		return false
	}

	expiry, err := strconv.ParseInt(payload, 10, 64)
	if err != nil {
		return false
	}
	if now.Unix() > expiry {
		return false
	}

	want := Sign(key, payload)
	return hmac.Equal([]byte(want), []byte(sig))
}
