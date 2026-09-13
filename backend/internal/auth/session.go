package auth

import (
	"crypto/hmac"
	"strconv"
	"strings"
	"time"
)

// SessionDuration adalah umur sesi dashboard sejak login (customer maupun vendor).
const SessionDuration = 12 * time.Hour

// NewSessionToken membuat token sesi bertanda tangan yang membawa identitas
// pemiliknya: "<subject>:<expiryUnix>.<hmac>".
//
// subject adalah account_id (sesi customer) atau username (sesi vendor) --
// dipisah pakai ":" dari expiry, lalu seluruh payload itu ditandatangani.
// subject TIDAK BOLEH mengandung karakter ":" (account_id dan username di
// proyek ini selalu alfanumerik+underscore, jadi ini aman).
//
// Stateless dengan sengaja — tidak ada tabel sesi di database. Kelebihannya:
// tidak perlu pembersihan sesi kedaluwarsa. Kekurangannya: sesi tidak dapat
// dicabut satu per satu sebelum kedaluwarsa — trade-off yang wajar untuk MVP.
func NewSessionToken(key []byte, now time.Time, subject string) string {
	expiry := now.Add(SessionDuration).Unix()
	return sessionToken(key, subject, expiry)
}

func sessionToken(key []byte, subject string, expiry int64) string {
	payload := subject + ":" + strconv.FormatInt(expiry, 10)
	sig := Sign(key, payload)
	return payload + "." + sig
}

// VerifySessionToken memeriksa tanda tangan dan masa berlaku token, lalu
// mengembalikan subject yang tersimpan di dalamnya.
//
// Split pakai LastIndex (bukan strings.Cut yang berhenti di titik PERTAMA)
// karena payload sendiri sudah mengandung karakter selain titik
// ("subject:expiry"), sig-nya baru ditempel setelah titik TERAKHIR.
func VerifySessionToken(key []byte, token string, now time.Time) (subject string, ok bool) {
	idx := strings.LastIndex(token, ".")
	if idx < 0 {
		return "", false
	}
	payload, sig := token[:idx], token[idx+1:]
	if payload == "" || sig == "" {
		return "", false
	}

	subj, expiryRaw, found := strings.Cut(payload, ":")
	if !found || subj == "" {
		return "", false
	}
	expiry, err := strconv.ParseInt(expiryRaw, 10, 64)
	if err != nil {
		return "", false
	}
	if now.Unix() > expiry {
		return "", false
	}

	want := Sign(key, payload)
	if !hmac.Equal([]byte(want), []byte(sig)) {
		return "", false
	}
	return subj, true
}
