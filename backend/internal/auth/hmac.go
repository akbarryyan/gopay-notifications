// Package auth mengurus autentikasi HMAC antara perangkat Android dan backend.
package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"time"
)

// SkewTolerance adalah selisih waktu maksimum yang masih diterima
// antara jam perangkat dan jam server.
const SkewTolerance = 5 * time.Minute

// SigningString membentuk string yang ditandatangani, sesuai api-contract.md §3.2.
//
// Yang di-hash adalah byte mentah body. Jangan pernah memanggil fungsi ini
// dengan hasil re-encode JSON — urutan field dan spasi akan berbeda dari
// yang dikirim perangkat, dan tanda tangan akan gagal secara acak.
func SigningString(deviceID string, timestamp int64, body []byte) string {
	sum := sha256.Sum256(body)
	return deviceID + "\n" +
		strconv.FormatInt(timestamp, 10) + "\n" +
		hex.EncodeToString(sum[:])
}

// Sign menghasilkan tanda tangan heksadesimal untuk signingString.
func Sign(secret []byte, signingString string) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(signingString))
	return hex.EncodeToString(mac.Sum(nil))
}

// Verify membandingkan tanda tangan dengan waktu konstan.
func Verify(secret []byte, signingString, gotSignature string) bool {
	want := Sign(secret, signingString)
	return hmac.Equal([]byte(want), []byte(gotSignature))
}

// CheckSkew melaporkan apakah timestamp (Unix detik) masih dalam toleransi.
func CheckSkew(timestamp int64, now time.Time) bool {
	diff := now.Sub(time.Unix(timestamp, 0))
	if diff < 0 {
		diff = -diff
	}
	return diff <= SkewTolerance
}
