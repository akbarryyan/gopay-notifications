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
	"github.com/akbarryyan/gopay-notifications/backend/internal/httpapi"
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

	api := httpapi.New(s, cfg.DeviceSecretKey, cfg.AdminSessionKey, cfg.WebhookSecretKey, time.Now)

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

	<-ctx.Done()
	slog.Info("shutdown dimulai")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("shutdown gagal", "err", err)
	}
}
