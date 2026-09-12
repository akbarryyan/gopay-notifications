// Command admintool membuat atau mereset password akun admin dashboard.
//
// Satu instalasi self-hosted melayani satu merchant, sehingga satu akun admin
// sudah cukup untuk MVP — tidak ada manajemen banyak pengguna. Perintah ini
// dipakai baik untuk pembuatan awal maupun reset password; keduanya operasi
// yang sama (UpsertAdmin), bukan dua alur berbeda.
//
// Untuk menghasilkan ADMIN_SESSION_KEY, pakai `go run ./cmd/devicetool -genkey`
// yang sama seperti DEVICE_SECRET_KEY — keduanya sama-sama kunci acak 32 byte
// dan tidak butuh generator terpisah. Jangan memakai NILAI yang sama untuk
// keduanya; jalankan perintahnya dua kali agar dua kunci independen.
package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"

	"github.com/akbarryyan/gopay-notifications/backend/internal/config"
	"github.com/akbarryyan/gopay-notifications/backend/internal/store"
)

func main() {
	username := flag.String("username", "admin", "username akun admin")
	genPassword := flag.Bool("random", false, "hasilkan password acak alih-alih diketik interaktif")
	flag.Parse()

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

	var password string
	if *genPassword {
		buf := make([]byte, 16)
		if _, err := rand.Read(buf); err != nil {
			fail("acak password: %v", err)
		}
		password = hex.EncodeToString(buf)
	} else {
		password = promptPassword()
	}

	if len(password) < 8 {
		fail("password terlalu pendek, minimal 8 karakter")
	}

	if err := s.UpsertAdmin(ctx, *username, password); err != nil {
		fail("%v", err)
	}

	fmt.Println("Akun admin berhasil disimpan.")
	fmt.Println()
	fmt.Println("  Username :", *username)
	if *genPassword {
		fmt.Println("  Password :", password)
		fmt.Println()
		fmt.Println("Password tidak akan ditampilkan lagi.")
	}
}

// promptPassword membaca password dua kali tanpa menampilkannya di terminal,
// supaya tidak tertinggal di riwayat shell maupun terlihat orang di sekitar.
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
	fmt.Fprintf(os.Stderr, "admintool: "+format+"\n", args...)
	os.Exit(1)
}
