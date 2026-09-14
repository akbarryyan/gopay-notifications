package notify

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"mime"
	"net"
	"net/http"
	"net/smtp"
	"strings"
	"time"
)

// SMTPConfig adalah konfigurasi pengirim email. From wajib; Username/
// Password boleh kosong untuk server SMTP internal yang tidak menuntut
// autentikasi.
type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

// Sender adalah jalur kirim per channel. Dipisah jadi interface supaya
// pekerjaan berkala bisa diuji tanpa benar-benar menyentuh SMTP atau
// Telegram. Urutan kirim, pencatatan riwayat, dan aturan "Telegram cuma
// tambahan" ada di Deliver, bukan di sini.
type Sender interface {
	SendEmail(ctx context.Context, to, subject, body string) error
	SendTelegram(ctx context.Context, chatID, text string) error
	// TelegramEnabled false berarti token bot belum diisi -- Telegram
	// dilewati tanpa dicatat sebagai gagal.
	TelegramEnabled() bool
}

// smtpTimeout membatasi SELURUH percakapan SMTP (dial sampai QUIT), bukan
// cuma dial -- server yang menerima koneksi lalu diam tidak boleh bisa
// menggantung pekerjaan pengingat atau request "kirim uji" selamanya.
const smtpTimeout = 30 * time.Second

// Notifier adalah Sender sungguhan: SMTP + Bot API Telegram.
type Notifier struct {
	smtp       SMTPConfig
	botToken   string
	httpClient *http.Client
}

func New(smtpCfg SMTPConfig, telegramBotToken string) *Notifier {
	return &Notifier{
		smtp:       smtpCfg,
		botToken:   telegramBotToken,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (n *Notifier) TelegramEnabled() bool { return n.botToken != "" }

// TestEmailSubject dan pesan uji lainnya dipakai tombol "kirim uji" di
// Vendor Dashboard -- supaya konfigurasi yang salah ketahuan saat disimpan,
// bukan saat customer pertama sudah perlu diingatkan.
const TestEmailSubject = "Email uji Payment Bridge"

const testEmailBody = "Ini email uji dari pengaturan notifikasi Vendor Dashboard.\n\n" +
	"Kalau email ini sampai, konfigurasi SMTP sudah benar dan notifikasi " +
	"ke customer akan terkirim lewat jalur yang sama.\n"

// TestTelegramText adalah isi pesan uji Telegram.
const TestTelegramText = "Pesan uji dari Payment Bridge — token bot Telegram sudah benar."

func (n *Notifier) SendTestEmail(ctx context.Context, to string) error {
	return n.SendEmail(ctx, to, TestEmailSubject, testEmailBody)
}

func (n *Notifier) SendTestTelegram(ctx context.Context, chatID string) error {
	return n.SendTelegram(ctx, chatID, TestTelegramText)
}

// TelegramError menandai kegagalan yang HANYA menyentuh jalur Telegram --
// emailnya sudah terkirim. Pemanggil memakai ini untuk memutuskan tetap
// menganggap notifikasi terkirim (dan cuma mencatat peringatan), bukan
// mencoba ulang seluruhnya.
type TelegramError struct{ Err error }

func (e *TelegramError) Error() string { return "notify: kirim telegram: " + e.Err.Error() }
func (e *TelegramError) Unwrap() error { return e.Err }

// SendEmail mengirim satu email teks polos.
func (n *Notifier) SendEmail(ctx context.Context, to, subject, body string) error {
	addr := net.JoinHostPort(n.smtp.Host, fmt.Sprint(n.smtp.Port))
	deadline := time.Now().Add(smtpTimeout)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}
	dialer := &net.Dialer{Deadline: deadline}
	tlsCfg := &tls.Config{ServerName: n.smtp.Host}

	// Port 465 memakai TLS implisit (TLS sejak byte pertama), port lain
	// (587/25) memulai polos lalu naik lewat STARTTLS. Keduanya didial
	// manual dengan tenggat -- smtp.SendMail tidak punya timeout sama sekali.
	var conn net.Conn
	var err error
	if n.smtp.Port == 465 {
		conn, err = tls.DialWithDialer(dialer, "tcp", addr, tlsCfg)
	} else {
		conn, err = dialer.DialContext(ctx, "tcp", addr)
	}
	if err != nil {
		return fmt.Errorf("sambung ke %s: %w", addr, err)
	}
	defer conn.Close()
	if err := conn.SetDeadline(deadline); err != nil {
		return fmt.Errorf("pasang tenggat: %w", err)
	}

	client, err := smtp.NewClient(conn, n.smtp.Host)
	if err != nil {
		return fmt.Errorf("smtp client: %w", err)
	}
	defer client.Close()

	if n.smtp.Port != 465 {
		if ok, _ := client.Extension("STARTTLS"); ok {
			if err := client.StartTLS(tlsCfg); err != nil {
				return fmt.Errorf("starttls: %w", err)
			}
		}
	}
	if n.smtp.Username != "" {
		// smtp.PlainAuth sengaja menolak mengirim password lewat koneksi tak
		// terenkripsi (kecuali ke localhost) -- server yang tidak mendukung
		// STARTTLS di port 587/25 akan gagal di sini dengan pesan jelas,
		// bukan diam-diam membocorkan password.
		if err := client.Auth(smtp.PlainAuth("", n.smtp.Username, n.smtp.Password, n.smtp.Host)); err != nil {
			return fmt.Errorf("autentikasi: %w", err)
		}
	}
	if err := client.Mail(senderAddress(n.smtp.From)); err != nil {
		return fmt.Errorf("mail from: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("rcpt to: %w", err)
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("data: %w", err)
	}
	if _, err := w.Write(buildRFC822(n.smtp.From, to, subject, body)); err != nil {
		return fmt.Errorf("tulis body: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("tutup body: %w", err)
	}
	return client.Quit()
}

// senderAddress mengambil bagian alamat saja dari From yang boleh
// berbentuk "Nama <alamat@domain>" -- perintah MAIL FROM cuma menerima
// alamatnya, bukan nama tampilannya.
func senderAddress(from string) string {
	if i := strings.LastIndex(from, "<"); i >= 0 {
		if j := strings.Index(from[i:], ">"); j >= 0 {
			return from[i+1 : i+j]
		}
	}
	return strings.TrimSpace(from)
}

func buildRFC822(from, to, subject, body string) []byte {
	var b bytes.Buffer
	fmt.Fprintf(&b, "From: %s\r\n", from)
	fmt.Fprintf(&b, "To: %s\r\n", to)
	// mime.QEncoding meng-encode RFC 2047 hanya bila perlu (string ASCII
	// polos dikembalikan apa adanya) -- supaya karakter non-ASCII pada nama
	// bisnis tidak tampil rusak di klien email.
	fmt.Fprintf(&b, "Subject: %s\r\n", mime.QEncoding.Encode("UTF-8", subject))
	fmt.Fprintf(&b, "Date: %s\r\n", time.Now().Format(time.RFC1123Z))
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/plain; charset=\"UTF-8\"\r\n")
	b.WriteString("\r\n")
	b.WriteString(strings.ReplaceAll(body, "\n", "\r\n"))
	return b.Bytes()
}

// SendTelegram mengirim satu pesan lewat Bot API.
func (n *Notifier) SendTelegram(ctx context.Context, chatID, text string) error {
	if n.botToken == "" {
		return fmt.Errorf("token bot telegram belum diisi")
	}
	payload, err := json.Marshal(map[string]string{"chat_id": chatID, "text": text})
	if err != nil {
		return err
	}
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", n.botToken)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := n.httpClient.Do(req)
	if err != nil {
		// Galat dari http.Client memuat URL lengkap -- termasuk token bot.
		// Jangan pernah diteruskan apa adanya ke log atau respons HTTP.
		return fmt.Errorf("gagal menghubungi api telegram")
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var apiErr struct {
			Description string `json:"description"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&apiErr)
		if apiErr.Description != "" {
			return fmt.Errorf("telegram menjawab %d: %s", resp.StatusCode, apiErr.Description)
		}
		return fmt.Errorf("telegram menjawab %d", resp.StatusCode)
	}
	return nil
}
