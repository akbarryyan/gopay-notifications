package notify

import (
	"fmt"
	"strings"
)

// EmailVerification HANYA email, sama seperti PasswordResetEmail --
// verifikasi memang membuktikan alamat EMAIL itu sendiri, jadi mengirimnya
// ke Telegram tidak masuk akal.
type EmailVerification struct {
	BusinessName string
	Link         string
}

func (e EmailVerification) Subject() string { return "Verifikasi email Payment Bridge kamu" }

func (e EmailVerification) Body() string {
	var b strings.Builder
	fmt.Fprintf(&b, "Halo %s,\n\n", e.BusinessName)
	b.WriteString("Terima kasih sudah mendaftar di Payment Bridge. Konfirmasi alamat email ini " +
		"supaya kami bisa mengirimkan pengingat masa aktif, peringatan HP offline, dan link reset " +
		"password ke tempat yang benar.\n\n")
	b.WriteString("Buka link berikut untuk memverifikasi:\n\n")
	fmt.Fprintf(&b, "%s\n\n", e.Link)
	b.WriteString("Link ini berlaku 24 jam. Kalau kamu tidak mendaftar, abaikan email ini.\n\n")
	b.WriteString("-- Payment Bridge\n")
	return b.String()
}
