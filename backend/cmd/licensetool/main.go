// Command licensetool menerbitkan file lisensi offline untuk instalasi
// customer.
//
// PERINGATAN: alat ini HANYA dijalankan di mesin Akbar sendiri. Berbeda dari
// devicetool/admintool, licensetool tidak pernah di-build atau dikirim ke
// VPS customer manapun — lihat backend/deploy/README.md. Ia juga tidak
// membutuhkan koneksi database sama sekali; seluruh operasinya (genkey,
// issue) murni kriptografi lokal.
//
// Lihat docs/superpowers/specs/2026-09-13-license-system-design.md.
package main

import (
	"encoding/base64"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/akbarryyan/gopay-notifications/backend/internal/licensecheck"
)

func main() {
	genKey := flag.Bool("genkey", false, "cetak key pair Ed25519 baru lalu keluar")
	issue := flag.Bool("issue", false, "terbitkan file lisensi baru")
	keyPath := flag.String("key", "", "path file berisi private key base64 (dari -genkey)")
	customer := flag.String("customer", "", "nama customer")
	domain := flag.String("domain", "", "domain instalasi, mis. whuzpay.com")
	plan := flag.String("plan", "", "label plan, mis. Business (cuma tampilan, tidak menegakkan kuota)")
	expires := flag.String("expires", "", "tanggal berakhir, format YYYY-MM-DD")
	out := flag.String("out", "license.lic", "path file lisensi hasil terbit")
	flag.Parse()

	switch {
	case *genKey:
		runGenKey()
	case *issue:
		runIssue(*keyPath, *customer, *domain, *plan, *expires, *out)
	default:
		fail("pilih salah satu: -genkey atau -issue")
	}
}

func runGenKey() {
	pub, priv, err := licensecheck.GenerateKeyPair()
	if err != nil {
		fail("generate key pair: %v", err)
	}
	fmt.Println("Key pair Ed25519 baru dibuat.")
	fmt.Println()
	fmt.Println("Public Key  (base64):", base64.StdEncoding.EncodeToString(pub))
	fmt.Println("Private Key (base64):", base64.StdEncoding.EncodeToString(priv))
	fmt.Println()
	fmt.Println("Tempel Public Key ke konstanta licensePublicKeyBase64 di")
	fmt.Println("backend/internal/licensecheck/license.go, lalu commit.")
	fmt.Println()
	fmt.Println("SIMPAN Private Key hanya di mesin ini (mis. ~/.gopay-license/private.key).")
	fmt.Println("Jangan pernah commit atau kirim ke server customer manapun — siapa pun")
	fmt.Println("yang memegangnya dapat menerbitkan lisensi palsu untuk produk ini.")
}

func runIssue(keyPath, customer, domain, plan, expiresStr, out string) {
	if keyPath == "" || customer == "" || domain == "" || plan == "" || expiresStr == "" {
		fail("-key, -customer, -domain, -plan, dan -expires wajib diisi")
	}

	keyBytes, err := os.ReadFile(keyPath)
	if err != nil {
		fail("baca private key: %v", err)
	}
	privBase64 := trimNewline(string(keyBytes))

	expiresAt, err := time.Parse("2006-01-02", expiresStr)
	if err != nil {
		fail("-expires harus format YYYY-MM-DD: %v", err)
	}

	content, err := licensecheck.Issue(privBase64, customer, domain, plan, time.Now(), expiresAt)
	if err != nil {
		fail("%v", err)
	}

	if err := os.WriteFile(out, []byte(content), 0o600); err != nil {
		fail("tulis %s: %v", out, err)
	}

	fmt.Println("Lisensi berhasil diterbitkan:", out)
	fmt.Println()
	fmt.Println("  Customer :", customer)
	fmt.Println("  Domain   :", domain)
	fmt.Println("  Plan     :", plan)
	fmt.Println("  Berakhir :", expiresStr)
	fmt.Println()

	if domain == "localhost" {
		// Lisensi dev lokal — dipakai langsung di mesin ini lewat
		// LICENSE_FILE_PATH di .env.dev, tidak dikirim kemana-mana.
		fmt.Printf("Ini lisensi LOKAL (domain \"localhost\"), bukan untuk customer.\n")
		fmt.Printf("Pastikan .env.dev berisi LICENSE_FILE_PATH=%s (relatif dari backend/),\n", out)
		fmt.Println("lalu jalankan make run-dev seperti biasa — tidak perlu scp/ssh apa pun.")
		return
	}

	fmt.Printf("Kirim %s ke customer, lalu di VPS mereka:\n", out)
	fmt.Println("  scp", out, "<VPS>:/opt/gopay-ingestion/license.lic")
	fmt.Println("  ssh <VPS> sudo systemctl restart gopay-ingestion")
}

func trimNewline(s string) string {
	for len(s) > 0 && (s[len(s)-1] == '\n' || s[len(s)-1] == '\r' || s[len(s)-1] == ' ') {
		s = s[:len(s)-1]
	}
	return s
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "licensetool: "+format+"\n", args...)
	os.Exit(1)
}
