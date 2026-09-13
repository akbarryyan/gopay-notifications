// Command seedtool mengisi database dev (gopay_dev) dengan banyak account
// customer sekaligus data pendukungnya (device, invoice, event, API key,
// webhook) -- supaya seluruh halaman Customer Dashboard kelihatan terisi
// wajar saat dikerjakan/di-demo lokal, bukan kosong melompong.
//
// HANYA untuk database dev/local. Jangan pernah dijalankan ke gopay_test
// (make test men-TRUNCATE-nya setiap kali jalan, seed ini akan hilang) atau
// ke database UAT/produksi manapun -- tidak ada pengaman di sini yang
// mencegah itu selain kamu sendiri mengarahkan DATABASE_URL yang benar.
//
// Pakai lewat Store yang sama seperti server sungguhan (bukan SQL mentah
// buat bagian yang sensitif) supaya password/secret device/API
// key/webhook secret ikut terenkripsi & ter-hash persis seperti aliran
// asli -- account/device seed ini bisa langsung dipakai login sungguhan.
//
// Contoh:
//
//	set -a; . ./.env.dev; set +a
//	go run ./cmd/seedtool -accounts 8
package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"math/big"
	mrand "math/rand"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/akbarryyan/gopay-notifications/backend/internal/config"
	"github.com/akbarryyan/gopay-notifications/backend/internal/store"
)

// planPresets meniru internal/httpapi/vendor_accounts.go -- disalin, bukan
// diimpor, karena seedtool sengaja tidak bergantung pada package httpapi
// (cmd tools cuma bergantung ke store+config, pola yang sama dengan
// devicetool/admintool).
var planPresets = []struct {
	name       string
	maxDevices int
}{
	{"Starter", 3},
	{"Business", 10},
	{"Enterprise", -1},
}

var businessNames = []string{
	"Warung Kopi Senja", "Toko Kelontong Makmur", "Laundry Bersih Cepat",
	"Kedai Mie Ayam Pak Budi", "Distro Anak Muda", "Bengkel Motor Jaya",
	"Apotek Sehat Selalu", "Salon Cantik Selalu", "Toko Bangunan Sejahtera",
	"Kios Pulsa Barokah", "Rumah Makan Padang Sederhana", "Percetakan Digital Prima",
	"Toko Sembako Rejeki", "Barbershop Kece", "Toko Oleh-Oleh Khas Daerah",
}

var deviceNamePool = []string{"Kasir Depan", "Kasir Belakang", "Kasir Lantai 2", "HP Cadangan", "Meja Kasir Utama"}

const (
	// Judul allowlist satu-satunya yang valid untuk arah "uang masuk" --
	// lihat CLAUDE.md "Arah transaksi ditentukan allowlist judul".
	notifTitle       = "Pembayaran QRIS statis diterima"
	notifPackageName = "com.gojek.gopaymerchant"
	notifSource      = "gopay"
)

func main() {
	accountCount := flag.Int("accounts", 8, "jumlah account customer yang dibuat")
	password := flag.String("password", "password123", "password sama untuk semua account seed (dev only, minimal 8 karakter)")
	flag.Parse()

	if len(*password) < 8 {
		fail("-password minimal 8 karakter")
	}
	if *accountCount < 1 {
		fail("-accounts minimal 1")
	}

	cfg, err := config.Load()
	if err != nil {
		fail("%v", err)
	}

	ctx := context.Background()
	s, err := store.New(ctx, cfg.DatabaseURL)
	if err != nil {
		fail("%v", err)
	}
	defer s.Close()

	type seededAccount struct {
		business string
		username string
		plan     string
		devices  int
		invoices int
	}
	var results []seededAccount

	for i := 0; i < *accountCount; i++ {
		business := businessNameFor(i)
		username := usernameFor(business, i)
		email := username + "@seed.test"
		accountID := "acc_" + randHex(8)
		plan := planPresets[i%len(planPresets)]

		err := s.CreateAccount(ctx, store.CreateAccountInput{
			ID:                accountID,
			BusinessName:      business,
			Email:             email,
			Username:          username,
			PlaintextPassword: *password,
			Plan:              plan.name,
			MaxDevices:        plan.maxDevices,
			ExpiresAt:         time.Now().Add(180 * 24 * time.Hour), // jauh dari expiring (>30 hari)
		})
		if err != nil {
			fail("buat account %q: %v", business, err)
		}

		deviceIDs, err := seedDevices(ctx, s, cfg, accountID)
		if err != nil {
			fail("seed device %q: %v", business, err)
		}

		invoiceCount := 40 + mrand.Intn(41) // 40..80
		if err := seedInvoicesAndEvents(ctx, s, accountID, deviceIDs, invoiceCount); err != nil {
			fail("seed invoice %q: %v", business, err)
		}

		if err := seedAPIKeys(ctx, s, accountID); err != nil {
			fail("seed api key %q: %v", business, err)
		}

		if err := seedWebhook(ctx, s, cfg, accountID, i); err != nil {
			fail("seed webhook %q: %v", business, err)
		}

		results = append(results, seededAccount{
			business: business, username: username, plan: plan.name,
			devices: len(deviceIDs), invoices: invoiceCount,
		})
		fmt.Fprintf(os.Stderr, "seed: %-32s selesai (%d device, %d invoice)\n", business, len(deviceIDs), invoiceCount)
	}

	fmt.Println()
	fmt.Println("Selesai. Semua account pakai password yang sama:", *password)
	fmt.Println()
	tw := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
	fmt.Fprintln(tw, "BUSINESS\tUSERNAME\tPLAN\tDEVICE\tINVOICE")
	for _, r := range results {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%d\t%d\n", r.business, r.username, r.plan, r.devices, r.invoices)
	}
	tw.Flush()
	fmt.Println()
	fmt.Println("Login lewat dashboard/ dengan salah satu username di atas + password di atas.")
}

func businessNameFor(i int) string {
	name := businessNames[i%len(businessNames)]
	round := i / len(businessNames)
	if round > 0 {
		name = fmt.Sprintf("%s %d", name, round+1)
	}
	return name
}

func usernameFor(business string, i int) string {
	slug := strings.ToLower(business)
	slug = strings.ReplaceAll(slug, " ", "_")
	replacer := strings.NewReplacer("-", "_", ".", "", ",", "")
	slug = replacer.Replace(slug)
	return fmt.Sprintf("%s_%d", slug, i+1)
}

func randHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		fail("acak id: %v", err)
	}
	return hex.EncodeToString(b)
}

func randAmount() int64 {
	// Rp 15.000 - Rp 2.000.000, kelipatan 500 biar kelihatan wajar.
	n, _ := rand.Int(rand.Reader, big.NewInt(3970))
	return 15_000 + n.Int64()*500
}

// seedDevices membuat 1-3 device per account: sebagian besar online (baru
// mengirim heartbeat), sebagian belum pernah heartbeat sama sekali
// (PENDING), dan satu offline (heartbeat_at sengaja ditulis mundur lewat
// SQL langsung -- RecordHeartbeat selalu memakai now(), tidak bisa dipakai
// buat mensimulasikan device yang sudah lama mati).
func seedDevices(ctx context.Context, s *store.Store, cfg config.Config, accountID string) ([]string, error) {
	count := 1 + mrand.Intn(3) // 1..3
	ids := make([]string, 0, count)

	for i := 0; i < count; i++ {
		deviceID := "dev_" + randHex(8)
		name := deviceNamePool[i%len(deviceNamePool)]
		secret := make([]byte, 32)
		if _, err := rand.Read(secret); err != nil {
			return nil, err
		}
		if err := s.CreateDevice(ctx, cfg.DeviceSecretKey, accountID, deviceID, name, secret); err != nil {
			return nil, fmt.Errorf("create device: %w", err)
		}
		ids = append(ids, deviceID)

		switch i % 3 {
		case 0, 1: // online: heartbeat baru saja
			if err := s.RecordHeartbeat(ctx, deviceID, store.Heartbeat{
				AndroidVersion:    "1.4.0",
				ListenerConnected: true,
				PendingCount:      0,
				FailedCount:       0,
			}); err != nil {
				return nil, fmt.Errorf("record heartbeat: %w", err)
			}
		case 2: // offline: heartbeat lama, ditulis langsung lewat SQL
			stale := time.Now().Add(-2 * store.HeartbeatTolerance)
			if _, err := s.Pool().Exec(ctx,
				`UPDATE devices SET heartbeat_at = $2, last_seen_at = $2, android_version = '1.3.2',
				 listener_connected = true, pending_count = 0, failed_count = 0 WHERE device_id = $1`,
				deviceID, stale); err != nil {
				return nil, fmt.Errorf("backdate heartbeat: %w", err)
			}
		}
	}
	return ids, nil
}

// seedInvoicesAndEvents membuat invoice tersebar 60 hari ke belakang dengan
// campuran status: sebagian besar PAID (event pencocokan ikut dibuat),
// sebagian EXPIRED (dibiarkan tidak dicocokkan, lalu benar-benar
// kedaluwarsa lewat efek samping CreateInvoice/MatchEvent berikutnya yang
// memakai now() sungguhan), dan beberapa PENDING yang masih hidup (dibuat
// dengan now() sungguhan, dalam jendela 15 menit). Ditambah beberapa event
// "pengecualian" (amount_hint tidak cocok invoice manapun).
func seedInvoicesAndEvents(ctx context.Context, s *store.Store, accountID string, deviceIDs []string, count int) error {
	if len(deviceIDs) == 0 {
		return fmt.Errorf("tidak ada device untuk dipasangkan event")
	}
	pickDevice := func() string { return deviceIDs[mrand.Intn(len(deviceIDs))] }

	stillPending := 1 + mrand.Intn(3) // 1..3 invoice yang sengaja dibiarkan hidup
	for i := 0; i < count; i++ {
		amount := randAmount()
		externalRef := fmt.Sprintf("ORDER-%06d", mrand.Intn(999999))

		var created time.Time
		leaveLivePending := i < stillPending
		if leaveLivePending {
			created = time.Now()
		} else {
			daysAgo := mrand.Intn(60)
			created = time.Now().AddDate(0, 0, -daysAgo).Add(-time.Duration(mrand.Intn(1440)) * time.Minute)
		}

		inv, _, err := s.CreateInvoice(ctx, created, accountID, externalRef, amount)
		if err != nil {
			return fmt.Errorf("create invoice: %w", err)
		}

		if leaveLivePending {
			continue // dibiarkan PENDING sungguhan, tidak dicocokkan
		}

		willPay := mrand.Intn(100) < 72 // ~72% berujung lunas
		if !willPay {
			continue // dibiarkan PENDING historis -- akan jadi EXPIRED via efek samping
		}

		eventID := "evt_seed_" + randHex(10)
		paidAt := created.Add(time.Duration(30+mrand.Intn(600)) * time.Second) // 30 detik - 10 menit setelah invoice dibuat
		amountHint := inv.UniqueAmount

		e := store.Event{
			EventID:     eventID,
			AccountID:   accountID,
			DeviceID:    pickDevice(),
			Source:      notifSource,
			PackageName: notifPackageName,
			Title:       strPtr(notifTitle),
			BodyText:    strPtr(fmt.Sprintf("Rp %s diterima dari QRIS", formatRupiah(amountHint))),
			AmountHint:  &amountHint,
			PostedAt:    paidAt,
			ReceivedAt:  paidAt,
			RawPayload:  []byte(`{"seed":true}`),
		}
		if _, err := s.InsertEvent(ctx, e); err != nil {
			return fmt.Errorf("insert event: %w", err)
		}
		if _, err := s.MatchEvent(ctx, paidAt, accountID, eventID, &amountHint); err != nil {
			return fmt.Errorf("match event: %w", err)
		}
	}

	// Beberapa event "pengecualian" -- amount_hint terisi tapi sengaja
	// tidak cocok invoice manapun (nominal acak yang hampir pasti tidak
	// pernah dipakai unique_amount di atas).
	exceptionCount := 6 + mrand.Intn(7) // 6..12
	for i := 0; i < exceptionCount; i++ {
		amount := randAmount() + 1_234_567 // digeser jauh, hampir mustahil bentrok
		receivedAt := time.Now().AddDate(0, 0, -mrand.Intn(30)).Add(-time.Duration(mrand.Intn(1440)) * time.Minute)
		e := store.Event{
			EventID:     "evt_seed_exc_" + randHex(10),
			AccountID:   accountID,
			DeviceID:    pickDevice(),
			Source:      notifSource,
			PackageName: notifPackageName,
			Title:       strPtr(notifTitle),
			BodyText:    strPtr(fmt.Sprintf("Rp %s diterima dari QRIS", formatRupiah(amount))),
			AmountHint:  &amount,
			PostedAt:    receivedAt,
			ReceivedAt:  receivedAt,
			RawPayload:  []byte(`{"seed":true,"exception":true}`),
		}
		if _, err := s.InsertEvent(ctx, e); err != nil {
			return fmt.Errorf("insert exception event: %w", err)
		}
	}
	return nil
}

func seedAPIKeys(ctx context.Context, s *store.Store, accountID string) error {
	names := []string{"Integrasi Toko Online", "Integrasi Kasir"}
	for _, name := range names {
		id, err := store.NewAPIKeyID()
		if err != nil {
			return err
		}
		_, hash, err := store.GenerateAPIKeySecret()
		if err != nil {
			return err
		}
		if err := s.CreateAPIKey(ctx, accountID, id, name, hash); err != nil {
			return fmt.Errorf("create api key: %w", err)
		}
	}
	return nil
}

// seedWebhook membuat satu endpoint per account (URL contoh, domain
// example.test -- RFC 2606, tidak pernah mengarah ke server sungguhan) dan
// menuliskan riwayat delivery langsung lewat SQL (bukan
// EnqueueWebhookDeliveries, yang mencari endpoint lintas SEMUA account
// tanpa filter account_id -- tidak cocok dipakai seed per-account terisolasi).
func seedWebhook(ctx context.Context, s *store.Store, cfg config.Config, accountID string, seq int) error {
	id, err := store.NewWebhookID()
	if err != nil {
		return err
	}
	_, secretBytes, err := store.GenerateWebhookSecret()
	if err != nil {
		return err
	}
	url := fmt.Sprintf("https://sistem-toko-%d.example.test/webhooks/payment-bridge", seq+1)
	events := []string{store.WebhookEventInvoicePaid, store.WebhookEventInvoiceExpired}
	if err := s.CreateWebhookEndpoint(ctx, cfg.WebhookSecretKey, accountID, id, "Sistem Toko", url, events, secretBytes); err != nil {
		return fmt.Errorf("create webhook endpoint: %w", err)
	}

	deliveryCount := 15 + mrand.Intn(11) // 15..25
	for i := 0; i < deliveryCount; i++ {
		createdAt := time.Now().AddDate(0, 0, -mrand.Intn(30)).Add(-time.Duration(mrand.Intn(1440)) * time.Minute)
		event := store.WebhookEventInvoicePaid
		if mrand.Intn(4) == 0 {
			event = store.WebhookEventInvoiceExpired
		}
		payload, _ := json.Marshal(map[string]any{
			"event":   event,
			"sent_at": createdAt.Format(time.RFC3339),
		})

		roll := mrand.Intn(100)
		var (
			status      string
			attempt     int
			httpStatus  *int
			durationMs  *int
			deliveredAt *time.Time
			nextAttempt *time.Time
		)
		switch {
		case roll < 78: // DELIVERED
			status = store.WebhookDeliveryDelivered
			attempt = 1
			hs := 200
			httpStatus = &hs
			dm := 80 + mrand.Intn(600)
			durationMs = &dm
			t := createdAt.Add(time.Duration(200+mrand.Intn(2000)) * time.Millisecond)
			deliveredAt = &t
		case roll < 92: // RETRYING
			status = store.WebhookDeliveryRetrying
			attempt = 1 + mrand.Intn(3)
			hs := 503
			httpStatus = &hs
			dm := 1000 + mrand.Intn(4000)
			durationMs = &dm
			t := createdAt.Add(5 * time.Minute)
			nextAttempt = &t
		default: // FAILED (habis percobaan)
			status = store.WebhookDeliveryFailed
			attempt = 5
			hs := 500
			httpStatus = &hs
			dm := 1000 + mrand.Intn(4000)
			durationMs = &dm
		}

		deliveryID := "whd_" + randHex(10)
		if _, err := s.Pool().Exec(ctx,
			`INSERT INTO webhook_deliveries
			   (id, account_id, endpoint_id, event, invoice_id, payload, status, attempt,
			    next_attempt_at, http_status, duration_ms, created_at, delivered_at)
			 VALUES ($1, $2, $3, $4, NULL, $5, $6, $7, $8, $9, $10, $11, $12)`,
			deliveryID, accountID, id, event, payload, status, attempt,
			nextAttempt, httpStatus, durationMs, createdAt, deliveredAt); err != nil {
			return fmt.Errorf("insert webhook delivery: %w", err)
		}
	}
	return nil
}

func strPtr(s string) *string { return &s }

func formatRupiah(n int64) string {
	s := fmt.Sprintf("%d", n)
	var out []byte
	for i, c := range []byte(s) {
		if i > 0 && (len(s)-i)%3 == 0 {
			out = append(out, '.')
		}
		out = append(out, c)
	}
	return string(out)
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "seedtool: "+format+"\n", args...)
	os.Exit(1)
}
