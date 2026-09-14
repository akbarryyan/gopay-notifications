package notify_test

import (
	"strings"
	"testing"
	"time"

	"github.com/akbarryyan/gopay-notifications/backend/internal/notify"
)

func TestDaysRemainingMembulatkanKeAtas(t *testing.T) {
	now := time.Date(2026, 9, 14, 10, 0, 0, 0, time.UTC)
	cases := []struct {
		nama      string
		expiresAt time.Time
		mau       int
	}{
		// Sisa 10 jam harus terbaca "1 hari lagi", bukan "0 hari lagi"
		// yang terbaca seperti sudah berakhir padahal belum.
		{"kurang dari sehari", now.Add(10 * time.Hour), 1},
		{"tepat sehari", now.Add(24 * time.Hour), 1},
		{"sehari lewat sedikit", now.Add(25 * time.Hour), 2},
		{"tujuh hari", now.Add(7 * 24 * time.Hour), 7},
		{"sudah lewat", now.Add(-time.Hour), 0},
	}
	for _, c := range cases {
		t.Run(c.nama, func(t *testing.T) {
			r := notify.ExpiryReminder{ExpiresAt: c.expiresAt}
			if got := r.DaysRemaining(now); got != c.mau {
				t.Fatalf("DaysRemaining = %d, mau %d", got, c.mau)
			}
		})
	}
}

func TestEmailMemuatNamaBisnisDanTanggal(t *testing.T) {
	now := time.Date(2026, 9, 14, 10, 0, 0, 0, time.UTC)
	r := notify.ExpiryReminder{
		BusinessName: "Warung Kopi Senja",
		Email:        "kopi@contoh.test",
		ExpiresAt:    time.Date(2026, 9, 18, 10, 0, 0, 0, time.UTC),
	}

	subject := r.EmailSubject(now)
	if !strings.Contains(subject, "4 hari lagi") {
		t.Fatalf("subject = %q, mau memuat sisa hari", subject)
	}

	body := r.EmailBody(now)
	for _, mau := range []string{"Warung Kopi Senja", "4 hari lagi", "18 September 2026"} {
		if !strings.Contains(body, mau) {
			t.Errorf("body tidak memuat %q:\n%s", mau, body)
		}
	}
}

func TestTelegramTextLebihRingkasDariEmail(t *testing.T) {
	now := time.Date(2026, 9, 14, 10, 0, 0, 0, time.UTC)
	r := notify.ExpiryReminder{
		BusinessName: "Toko Uji",
		ExpiresAt:    now.Add(3 * 24 * time.Hour),
	}
	text := r.TelegramText(now)
	if !strings.Contains(text, "Toko Uji") || !strings.Contains(text, "3 hari lagi") {
		t.Fatalf("TelegramText = %q", text)
	}
	if len(text) >= len(r.EmailBody(now)) {
		t.Fatal("pesan Telegram seharusnya lebih pendek daripada email")
	}
}
