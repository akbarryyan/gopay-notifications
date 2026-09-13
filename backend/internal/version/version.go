// Package version menyimpan versi produk saat ini, dikirim ke License
// Server saat aktivasi/validasi (lihat internal/licenseclient). Bukan
// bagian dari mekanisme rilis terkelola — belum ada satu, ini cuma string
// statis yang di-bump manual tiap rilis. License Server v1 sengaja tidak
// menegakkan versi minimum (lihat
// docs/superpowers/specs/2026-09-13-online-license-platform-design.md §9).
package version

// Current adalah versi produk backend saat ini.
const Current = "1.0.0"
