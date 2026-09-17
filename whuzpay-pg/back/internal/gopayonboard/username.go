package gopayonboard

import "strings"

// SanitizeUsername menurunkan username dari bagian sebelum "@" di email --
// huruf kecil, cuma huruf+angka, dipotong maks 20 karakter. Dipakai
// AuthService.RegisterMerchant sebagai basis username akun gopay-notifications
// yang dibuat otomatis (lihat spec 2026-09-17-whuzpay-pg-unified-onboarding-design.md §4.1).
// Kalau bentrok 409 username_taken, pemanggil menambah akhiran angka --
// fungsi ini sendiri tidak tahu apa pun soal collision.
func SanitizeUsername(email string) string {
	local, _, _ := strings.Cut(email, "@")
	var b strings.Builder
	for _, r := range strings.ToLower(local) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	sanitized := b.String()
	if sanitized == "" {
		sanitized = "merchant"
	}
	if len(sanitized) > 20 {
		sanitized = sanitized[:20]
	}
	return sanitized
}
