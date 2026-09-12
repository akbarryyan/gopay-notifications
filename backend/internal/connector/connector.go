// Package connector mendaftarkan sumber notifikasi pembayaran yang dikenal.
//
// Satu tempat untuk menambah sumber baru. Tanpa registry ini, nama sumber
// tersebar sebagai literal di validasi, rute, dan UI — dan menambah DANA
// berarti menyentuh semuanya.
package connector

import "sort"

// Source adalah pengenal sumber pembayaran yang dipakai di payload event.
type Source string

const (
	GoPay Source = "gopay"
)

type Info struct {
	ID   Source `json:"id"`
	Name string `json:"name"`
	// Package aplikasi Android yang memasang notifikasinya. Dipakai dokumen
	// dan dashboard; perangkat tetap memakai daftar yang dapat diedit di
	// Settings, karena nama package dapat berubah antar versi aplikasi.
	Packages []string `json:"packages"`
}

var known = map[Source]Info{
	GoPay: {
		ID:       GoPay,
		Name:     "GoPay",
		Packages: []string{"com.gojek.gopaymerchant"},
	},
}

// IsKnown melaporkan apakah sumber terdaftar.
func IsKnown(s string) bool {
	_, ok := known[Source(s)]
	return ok
}

// Get mengembalikan informasi sumber.
func Get(s string) (Info, bool) {
	info, ok := known[Source(s)]
	return info, ok
}

// All mengembalikan seluruh sumber yang dikenal, urut berdasarkan ID.
func All() []Info {
	out := make([]Info, 0, len(known))
	for _, info := range known {
		out = append(out, info)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}
