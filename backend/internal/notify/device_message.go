package notify

import (
	"fmt"
	"strings"
	"time"
)

// wib dipakai untuk jam di pesan status HP. Jam penting di sini (beda
// dengan pengingat kedaluwarsa yang cuma menyebut tanggal), dan customer
// membacanya sebagai jam lokal. Zona tetap, bukan time.LoadLocation, supaya
// tidak bergantung pada tzdata di image server.
var wib = time.FixedZone("WIB", 7*60*60)

func formatWIB(t time.Time) string {
	return t.In(wib).Format("2 January 2006 15.04") + " WIB"
}

// DeviceOffline adalah pemberitahuan bahwa HP bridge berhenti mengirim
// kabar (heartbeat) melewati batas toleransi.
type DeviceOffline struct {
	BusinessName    string
	DeviceName      string
	LastHeartbeatAt time.Time
}

func (d DeviceOffline) Message() Message {
	subject := fmt.Sprintf("HP \"%s\" tidak mengirim kabar", d.DeviceName)

	var b strings.Builder
	fmt.Fprintf(&b, "Halo %s,\n\n", d.BusinessName)
	fmt.Fprintf(&b, "HP \"%s\" tidak mengirim kabar sejak %s.\n\n", d.DeviceName, formatWIB(d.LastHeartbeatAt))
	b.WriteString("Selama HP ini offline, pembayaran GoPay yang masuk belum akan tercatat " +
		"dan invoice belum akan ditandai lunas. Bila aplikasinya berhenti, pembayaran " +
		"yang masuk selama itu bisa terlewat sama sekali.\n\n")
	b.WriteString("Yang perlu dicek:\n")
	b.WriteString("- HP menyala dan tersambung ke internet\n")
	b.WriteString("- Aplikasi Payment Bridge masih berjalan (buka aplikasinya sekali)\n")
	b.WriteString("- Izin akses notifikasi untuk aplikasi masih aktif\n\n")
	b.WriteString("Kamu akan menerima pemberitahuan lagi saat HP kembali online.\n\n")
	b.WriteString("-- Payment Bridge\n")

	return Message{
		Subject:   subject,
		EmailBody: b.String(),
		TelegramText: fmt.Sprintf(
			"⚠️ HP \"%s\" (%s) tidak mengirim kabar sejak %s. Pembayaran belum akan tercatat sampai HP kembali online — cek koneksi internet dan aplikasi Payment Bridge.",
			d.DeviceName, d.BusinessName, formatWIB(d.LastHeartbeatAt)),
	}
}

// DeviceOnline adalah pasangan DeviceOffline: HP yang sebelumnya
// dikabarkan offline sudah mengirim kabar lagi.
type DeviceOnline struct {
	BusinessName string
	DeviceName   string
	// OfflineSince adalah heartbeat terakhir sebelum offline, BackAt
	// heartbeat pertama setelah kembali.
	OfflineSince time.Time
	BackAt       time.Time
}

func (d DeviceOnline) Message() Message {
	subject := fmt.Sprintf("HP \"%s\" kembali online", d.DeviceName)

	var b strings.Builder
	fmt.Fprintf(&b, "Halo %s,\n\n", d.BusinessName)
	fmt.Fprintf(&b, "HP \"%s\" kembali mengirim kabar sejak %s (sebelumnya terakhir %s).\n\n",
		d.DeviceName, formatWIB(d.BackAt), formatWIB(d.OfflineSince))
	b.WriteString("Pembayaran yang masuk sekarang kembali tercatat otomatis. Bila ada " +
		"pembayaran selama HP offline yang belum tercatat, cek halaman Transactions " +
		"dan Exceptions di dashboard.\n\n")
	b.WriteString("-- Payment Bridge\n")

	return Message{
		Subject:   subject,
		EmailBody: b.String(),
		TelegramText: fmt.Sprintf(
			"✅ HP \"%s\" (%s) kembali online sejak %s. Pembayaran kembali tercatat otomatis.",
			d.DeviceName, d.BusinessName, formatWIB(d.BackAt)),
	}
}
