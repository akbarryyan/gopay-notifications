package notify

import (
	"fmt"
	"strings"
	"time"
)

// PasswordResetEmail HANYA untuk email -- sengaja tidak punya versi
// Telegram. Link di dalamnya adalah kunci akun selama 30 menit; mengirimnya
// juga ke Telegram berarti menggandakan tempat token itu bisa bocor
// (riwayat chat, notifikasi di layar kunci HP).
type PasswordResetEmail struct {
	BusinessName string
	Username     string
	Link         string
	ValidFor     time.Duration
}

func (p PasswordResetEmail) Subject() string { return "Reset password Payment Bridge" }

func (p PasswordResetEmail) Body() string {
	var b strings.Builder
	fmt.Fprintf(&b, "Halo %s,\n\n", p.BusinessName)
	b.WriteString("Kami menerima permintaan untuk mengatur ulang password akun Payment Bridge kamu.\n\n")
	// Login memakai username, bukan email -- orang yang lupa password
	// sering juga lupa username-nya.
	fmt.Fprintf(&b, "Username: %s\n\n", p.Username)
	b.WriteString("Buka link berikut untuk membuat password baru:\n\n")
	fmt.Fprintf(&b, "%s\n\n", p.Link)
	fmt.Fprintf(&b, "Link ini berlaku %d menit dan hanya bisa dipakai sekali.\n\n", int(p.ValidFor.Minutes()))
	b.WriteString("Kalau kamu tidak meminta reset password, abaikan email ini -- password kamu tidak berubah.\n\n")
	b.WriteString("-- Payment Bridge\n")
	return b.String()
}

// PasswordChanged memberi tahu pemilik akun bahwa passwordnya baru saja
// diganti -- satu-satunya cara pemilik sungguhan tahu kalau yang mengganti
// ternyata orang lain.
type PasswordChanged struct {
	BusinessName string
	Username     string
	At           time.Time
	ViaReset     bool
}

func (p PasswordChanged) Message() Message {
	how := "lewat halaman Settings"
	if p.ViaReset {
		how = "lewat link reset password"
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Halo %s,\n\n", p.BusinessName)
	fmt.Fprintf(&b, "Password akun Payment Bridge kamu (username %s) baru saja diganti %s pada %s.\n\n",
		p.Username, how, formatWIB(p.At))
	b.WriteString("Semua sesi login lain sudah otomatis dikeluarkan.\n\n")
	b.WriteString("Kalau bukan kamu yang menggantinya, segera atur ulang password lewat \"Lupa password?\" " +
		"di halaman login, lalu balas email ini.\n\n")
	b.WriteString("-- Payment Bridge\n")

	return Message{
		Subject:   "Password Payment Bridge kamu baru saja diganti",
		EmailBody: b.String(),
		TelegramText: fmt.Sprintf(
			"🔐 Password akun Payment Bridge %s (username %s) baru saja diganti pada %s. Kalau bukan kamu, segera reset password.",
			p.BusinessName, p.Username, formatWIB(p.At)),
	}
}
