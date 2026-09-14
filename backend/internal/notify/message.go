// Package notify mengirim pemberitahuan ke customer lewat email (jalur
// utama) dan Telegram (tambahan, cuma untuk customer yang mengisinya).
//
// Penyusunan pesan sengaja dipisah dari pengirimannya: bagian penyusunan
// murni fungsi (bisa diuji tanpa jaringan, dan itulah bagian yang paling
// mungkin salah -- salah hitung sisa hari, salah sebut nama), sedangkan
// bagian pengiriman cuma membungkus net/smtp dan Bot API Telegram.
package notify

import (
	"fmt"
	"strings"
	"time"
)

// ExpiryReminder adalah data satu pengingat kedaluwarsa untuk satu akun.
type ExpiryReminder struct {
	BusinessName string
	Email        string
	// TelegramChatID kosong berarti customer ini tidak punya/tidak mengisi
	// Telegram -- pengingatnya cukup lewat email saja.
	TelegramChatID string
	ExpiresAt      time.Time
}

// DaysRemaining membulatkan KE ATAS, bukan ke bawah: sisa 0,4 hari harus
// dibaca "1 hari lagi", bukan "0 hari lagi" yang terbaca seperti sudah
// berakhir padahal belum.
func (r ExpiryReminder) DaysRemaining(now time.Time) int {
	d := r.ExpiresAt.Sub(now)
	if d <= 0 {
		return 0
	}
	days := int(d / (24 * time.Hour))
	if d%(24*time.Hour) > 0 {
		days++
	}
	return days
}

func (r ExpiryReminder) sisaHariFrasa(now time.Time) string {
	days := r.DaysRemaining(now)
	if days <= 0 {
		return "hari ini"
	}
	return fmt.Sprintf("%d hari lagi", days)
}

// EmailSubject dan EmailBody sengaja teks polos, bukan HTML: isinya cuma
// beberapa kalimat, dan teks polos tidak pernah bermasalah di klien email
// manapun (tidak ada gambar diblokir, tidak ada CSS yang dibuang).
func (r ExpiryReminder) EmailSubject(now time.Time) string {
	return fmt.Sprintf("Akun Payment Bridge kamu berakhir %s", r.sisaHariFrasa(now))
}

func (r ExpiryReminder) EmailBody(now time.Time) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Halo %s,\n\n", r.BusinessName)
	fmt.Fprintf(&b, "Akun Payment Bridge kamu berakhir %s (%s).\n\n",
		r.sisaHariFrasa(now), r.ExpiresAt.Format("2 January 2006"))
	b.WriteString("Setelah masa aktif berakhir, pengiriman event dari HP dan " +
		"pembuatan invoice lewat API akan berhenti sampai akun diperpanjang. " +
		"Data kamu tidak dihapus.\n\n")
	b.WriteString("Balas email ini untuk memperpanjang.\n\n")
	b.WriteString("-- Payment Bridge\n")
	return b.String()
}

// TelegramText versi ringkas dari email yang sama -- Telegram dibaca
// sambil lalu, jadi kalimatnya dipendekkan, bukan disalin mentah.
func (r ExpiryReminder) TelegramText(now time.Time) string {
	return fmt.Sprintf(
		"Akun Payment Bridge %s berakhir %s (%s). Perpanjang sebelum pengiriman event berhenti.",
		r.BusinessName, r.sisaHariFrasa(now), r.ExpiresAt.Format("2 January 2006"))
}

// Message menyatukan ketiga bentuk pengingat untuk Deliver.
func (r ExpiryReminder) Message(now time.Time) Message {
	return Message{Subject: r.EmailSubject(now), EmailBody: r.EmailBody(now), TelegramText: r.TelegramText(now)}
}
