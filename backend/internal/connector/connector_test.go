package connector_test

import (
	"testing"

	"github.com/akbarryyan/gopay-notifications/backend/internal/connector"
)

func TestGoPayDikenal(t *testing.T) {
	if !connector.IsKnown("gopay") {
		t.Fatal("gopay seharusnya terdaftar")
	}
	info, ok := connector.Get("gopay")
	if !ok {
		t.Fatal("Get(gopay) = !ok")
	}
	if info.Name != "GoPay" {
		t.Fatalf("Name = %q", info.Name)
	}
	if len(info.Packages) == 0 || info.Packages[0] != "com.gojek.gopaymerchant" {
		t.Fatalf("Packages = %v", info.Packages)
	}
}

func TestSumberTakDikenalDitolak(t *testing.T) {
	// Sumber yang belum diimplementasikan harus ditolak, bukan diterima
	// lalu diam-diam tidak pernah dicocokkan.
	for _, s := range []string{"dana", "ovo", "shopeepay", "", "GOPAY", " gopay"} {
		if connector.IsKnown(s) {
			t.Fatalf("%q seharusnya tidak dikenal", s)
		}
	}
}

func TestAllStabilDanTidakKosong(t *testing.T) {
	all := connector.All()
	if len(all) == 0 {
		t.Fatal("All() kosong")
	}
	for i := 1; i < len(all); i++ {
		if all[i-1].ID >= all[i].ID {
			t.Fatal("All() harus urut berdasarkan ID agar UI stabil")
		}
	}
}
