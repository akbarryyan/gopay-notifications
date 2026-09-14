// Command server menjalankan layanan ingestion event notifikasi GoPay.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/akbarryyan/gopay-notifications/backend/internal/config"
	"github.com/akbarryyan/gopay-notifications/backend/internal/devicealert"
	"github.com/akbarryyan/gopay-notifications/backend/internal/httpapi"
	"github.com/akbarryyan/gopay-notifications/backend/internal/notify"
	"github.com/akbarryyan/gopay-notifications/backend/internal/reminder"
	"github.com/akbarryyan/gopay-notifications/backend/internal/store"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		slog.Error("konfigurasi tidak sah", "err", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	s, err := store.New(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("koneksi database gagal", "err", err)
		os.Exit(1)
	}
	defer s.Close()

	api := httpapi.New(s, cfg.DeviceSecretKey, cfg.AdminSessionKey, cfg.WebhookSecretKey, cfg.VendorSessionKey, cfg.SettingsSecretKey, time.Now).
		WithDashboardURL(cfg.DashboardURL)
	if cfg.DashboardURL == "" {
		slog.Warn("DASHBOARD_URL kosong -- lupa password lewat email tidak tersedia")
	}

	srv := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           api.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
	}

	go func() {
		slog.Info("server mulai", "addr", cfg.ListenAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server berhenti", "err", err)
			os.Exit(1)
		}
	}()

	// Worker webhook: mendeteksi invoice yang baru kedaluwarsa dan
	// mengeksekusi retry pengiriman yang jatuh tempo. Satu-satunya
	// goroutine berkala di backend ini sejak platform lisensi online
	// (License Server + licenseclient) dibongkar -- status akun sekarang
	// dicek langsung ke database tiap request (requireActiveAccount),
	// tidak ada lagi validasi berkala terpisah.
	webhookTicker := time.NewTicker(1 * time.Minute)
	defer webhookTicker.Stop()
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-webhookTicker.C:
				if err := api.ProcessDueWebhooks(ctx, time.Now()); err != nil {
					slog.Error("process due webhooks gagal", "err", err)
				}
			}
		}
	}()

	// Pengingat kedaluwarsa ke customer. Ticker TERPISAH dari worker
	// webhook di atas, bukan digabung: kadensinya beda jauh (sekali sejam
	// vs sekali semenit), dan menumpangkannya ke ticker 1 menit berarti
	// query pencarian akun jatuh tempo jalan 60x lebih sering tanpa
	// manfaat -- dedupe-nya ada di database (expiry_reminder_sent_for),
	// jadi satu jam sekali sudah cukup rapat dan aman terhadap restart.
	//
	// Ticker SELALU berjalan; aktif-tidaknya diputuskan tiap putaran dari
	// pengaturan SMTP di database (diatur lewat Vendor Dashboard), jadi
	// mengisi/mengubah SMTP langsung berlaku tanpa restart. Job sendiri yang
	// mencatat saat status aktif/tidak aktif berubah.
	senderFactory := notify.FromSettings(s, cfg.SettingsSecretKey)
	job := reminder.New(s, senderFactory, reminder.DefaultWithinDays)
	runReminder := func() {
		sent, err := job.Run(ctx, time.Now())
		if err != nil {
			slog.Error("pengingat kedaluwarsa gagal", "err", err)
			return
		}
		if sent > 0 {
			slog.Info("pengingat kedaluwarsa terkirim", "jumlah", sent)
		}
	}
	reminderTicker := time.NewTicker(1 * time.Hour)
	defer reminderTicker.Stop()
	go func() {
		// Satu putaran langsung saat start, tidak menunggu satu jam pertama
		// -- supaya status aktif/tidak aktif langsung terlihat di log.
		runReminder()
		for {
			select {
			case <-ctx.Done():
				return
			case <-reminderTicker.C:
				runReminder()
			}
		}
	}()

	// Peringatan HP offline/kembali online ke customer. Ticker sendiri lagi:
	// 5 menit, karena yang ditunggu customer adalah kabar secepatnya bahwa
	// pembayaran berhenti tercatat -- sejam terlalu lambat, semenit tidak
	// ada gunanya dibanding toleransi heartbeat 45 menit. Dedupe-nya di
	// devices.offline_alert_for, jadi aman terhadap restart.
	alertJob := devicealert.New(s, senderFactory)
	runDeviceAlert := func() {
		res, err := alertJob.Run(ctx, time.Now())
		if err != nil {
			slog.Error("peringatan status HP gagal", "err", err)
			return
		}
		if res.Offline > 0 || res.Online > 0 {
			slog.Info("peringatan status HP terkirim", "offline", res.Offline, "online", res.Online)
		}
	}
	deviceAlertTicker := time.NewTicker(devicealert.Interval)
	defer deviceAlertTicker.Stop()
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-deviceAlertTicker.C:
				runDeviceAlert()
			}
		}
	}()

	<-ctx.Done()
	slog.Info("shutdown dimulai")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("shutdown gagal", "err", err)
	}
}
