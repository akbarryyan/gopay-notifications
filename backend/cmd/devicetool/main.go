// Command devicetool membuat device baru beserta secret-nya.
//
// Secret dicetak satu kali ke stdout dan tidak pernah dapat dibaca kembali
// dari database dalam bentuk yang mudah — salin langsung ke Settings aplikasi.
package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"flag"
	"fmt"
	"os"

	"github.com/akbarryyan/gopay-notifications/backend/internal/config"
	"github.com/akbarryyan/gopay-notifications/backend/internal/store"
)

func main() {
	name := flag.String("name", "", "nama device, contoh: \"HP GoPay Utama\"")
	accountID := flag.String("account", "", "account_id pemilik device ini (lihat Vendor Dashboard)")
	genKey := flag.Bool("genkey", false, "cetak DEVICE_SECRET_KEY baru lalu keluar")
	flag.Parse()

	if *genKey {
		key := make([]byte, 32)
		if _, err := rand.Read(key); err != nil {
			fail("acak kunci: %v", err)
		}
		fmt.Println(base64.StdEncoding.EncodeToString(key))
		return
	}

	if *name == "" {
		fail("-name wajib diisi")
	}
	if *accountID == "" {
		fail("-account wajib diisi -- satu-satunya cara membuat device sampai Customer Dashboard swalayan tersedia")
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

	idSuffix := make([]byte, 8)
	if _, err := rand.Read(idSuffix); err != nil {
		fail("acak device id: %v", err)
	}
	deviceID := "dev_" + hex.EncodeToString(idSuffix)

	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		fail("acak secret: %v", err)
	}
	secretB64 := base64.StdEncoding.EncodeToString(secret)

	if err := s.CreateDevice(ctx, cfg.DeviceSecretKey, *accountID, deviceID, *name, []byte(secretB64)); err != nil {
		fail("%v", err)
	}

	fmt.Println("Device berhasil dibuat. Salin dua nilai ini ke Settings aplikasi Android:")
	fmt.Println()
	fmt.Println("  Device ID     :", deviceID)
	fmt.Println("  Device Secret :", secretB64)
	fmt.Println()
	fmt.Println("Secret tidak akan ditampilkan lagi.")
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "devicetool: "+format+"\n", args...)
	os.Exit(1)
}
