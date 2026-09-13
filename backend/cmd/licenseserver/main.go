// Command licenseserver menjalankan License Server — layanan terpisah
// milik vendor (Akbar), bukan bagian dari instalasi customer manapun.
// Database sendiri (gopay_license), tidak pernah menyentuh database
// customer.
//
// Lihat docs/superpowers/specs/2026-09-13-online-license-platform-design.md.
package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"golang.org/x/term"

	"github.com/akbarryyan/gopay-notifications/backend/internal/licensecheck"
	"github.com/akbarryyan/gopay-notifications/backend/internal/licenseserver/httpapi"
	"github.com/akbarryyan/gopay-notifications/backend/internal/licenseserver/store"
)

func main() {
	genKey := flag.Bool("genkey", false, "cetak ADMIN_SESSION_KEY + key pair Ed25519 signing baru lalu keluar")
	createAdmin := flag.String("create-admin", "", "buat/reset akun vendor dengan username ini, interaktif")
	flag.Parse()

	if *genKey {
		runGenKey()
		return
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg, err := loadConfig()
	if err != nil {
		slog.Error("konfigurasi tidak sah", "err", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	s, err := store.New(ctx, cfg.databaseURL)
	if err != nil {
		slog.Error("koneksi database gagal", "err", err)
		os.Exit(1)
	}
	defer s.Close()

	if *createAdmin != "" {
		runCreateAdmin(ctx, s, *createAdmin)
		return
	}

	api := httpapi.New(s, cfg.adminSessionKey, cfg.signingPrivateKey, time.Now)

	srv := &http.Server{
		Addr:              cfg.listenAddr,
		Handler:           api.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
	}

	go func() {
		slog.Info("license server mulai", "addr", cfg.listenAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server berhenti", "err", err)
			os.Exit(1)
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

type config struct {
	databaseURL       string
	listenAddr        string
	adminSessionKey   []byte
	signingPrivateKey string
}

func loadConfig() (config, error) {
	var c config
	c.databaseURL = os.Getenv("DATABASE_URL")
	if c.databaseURL == "" {
		return config{}, fmt.Errorf("DATABASE_URL wajib diisi")
	}
	c.listenAddr = os.Getenv("LISTEN_ADDR")
	if c.listenAddr == "" {
		c.listenAddr = ":8080"
	}

	rawSession := os.Getenv("ADMIN_SESSION_KEY")
	if rawSession == "" {
		return config{}, fmt.Errorf("ADMIN_SESSION_KEY wajib diisi")
	}
	sessionKey, err := base64.StdEncoding.DecodeString(rawSession)
	if err != nil || len(sessionKey) != 32 {
		return config{}, fmt.Errorf("ADMIN_SESSION_KEY harus base64 32 byte")
	}
	c.adminSessionKey = sessionKey

	c.signingPrivateKey = os.Getenv("LICENSE_SIGNING_PRIVATE_KEY")
	if c.signingPrivateKey == "" {
		return config{}, fmt.Errorf("LICENSE_SIGNING_PRIVATE_KEY wajib diisi")
	}

	return c, nil
}

// runGenKey mencetak SEMUA kunci yang dibutuhkan License Server sekali
// jalan: ADMIN_SESSION_KEY (kunci acak 32 byte, sama pola dengan
// devicetool -genkey) dan key pair Ed25519 untuk menandatangani local
// license state. Public key hasilnya harus ditempel manual ke
// internal/licensecheck/license.go (licensePublicKeyBase64) dan di-commit;
// private key HANYA masuk .env License Server, tidak pernah ke repo.
func runGenKey() {
	sessionKey := make([]byte, 32)
	if _, err := rand.Read(sessionKey); err != nil {
		fail("acak ADMIN_SESSION_KEY: %v", err)
	}
	pub, priv, err := licensecheck.GenerateKeyPair()
	if err != nil {
		fail("generate signing key pair: %v", err)
	}

	fmt.Println("ADMIN_SESSION_KEY (.env License Server):")
	fmt.Println(" ", base64.StdEncoding.EncodeToString(sessionKey))
	fmt.Println()
	fmt.Println("LICENSE_SIGNING_PRIVATE_KEY (.env License Server, JANGAN commit):")
	fmt.Println(" ", base64.StdEncoding.EncodeToString(priv))
	fmt.Println()
	fmt.Println("Signing Public Key (tempel ke licensePublicKeyBase64 di")
	fmt.Println("backend/internal/licensecheck/license.go, lalu commit):")
	fmt.Println(" ", base64.StdEncoding.EncodeToString(pub))
}

func runCreateAdmin(ctx context.Context, s *store.Store, username string) {
	password := promptPassword()
	if err := s.UpsertAdmin(ctx, username, password); err != nil {
		fail("%v", err)
	}
	fmt.Println("Akun vendor", username, "berhasil dibuat/direset.")
}

// promptPassword membaca password dua kali tanpa menampilkannya di
// terminal — pola sama persis cmd/admintool.
func promptPassword() string {
	fmt.Fprint(os.Stderr, "Password baru: ")
	pw1, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr)
	if err != nil {
		fail("baca password: %v", err)
	}

	fmt.Fprint(os.Stderr, "Ulangi password: ")
	pw2, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr)
	if err != nil {
		fail("baca password: %v", err)
	}

	if string(pw1) != string(pw2) {
		fail("kedua password tidak sama")
	}
	return strings.TrimSpace(string(pw1))
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "licenseserver: "+format+"\n", args...)
	os.Exit(1)
}
