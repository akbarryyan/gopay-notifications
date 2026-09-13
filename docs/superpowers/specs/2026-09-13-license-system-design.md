# Sistem Lisensi — Spec

**Tanggal:** 2026-09-13
**Status:** Disetujui, siap implementasi

## 1. Latar belakang dan keputusan pokok

Model bisnis produk ini adalah **self-hosted + annual license**: tiap customer
men-deploy backend dan dashboard sendiri di server mereka sendiri (lihat
"Pemecahan scope" di `CLAUDE.md`). Sampai sekarang tidak ada mekanisme apa pun
yang menegakkan bagian "annual" itu — backend berjalan selamanya begitu
di-deploy, tidak peduli lisensinya "habis" atau tidak, karena memang belum ada
konsep lisensi di kode sama sekali.

Spec awal produk (`docs/dashboard-spec.md` §2, §27–28) membayangkan license
server online milik vendor (aktivasi, renewal lewat internet ke server
Akbar). Itu **tidak dipakai di sini** — lihat §7.

Keputusan pokok, sudah dikonfirmasi lewat brainstorming:

| Pertanyaan | Keputusan |
|---|---|
| Model validasi | **Offline signed file**, backend tidak pernah menghubungi server manapun |
| Entitlement yang ditegakkan | **Cuma tanggal kedaluwarsa** — tidak ada kuota device/webhook/dst |
| Efek lisensi tidak aktif | **Blokir semua API bisnis** (device, admin, API key), dashboard login + halaman License tetap bisa dibuka |
| Domain-lock | **Ya** — lisensi diikat ke satu domain |
| Lokasi file | Path tetap `/opt/gopay-ingestion/license.lic`, bisa di-override `LICENSE_FILE_PATH` |
| Ambang peringatan "akan berakhir" | **30 hari** |

## 2. Format file lisensi

File teks dua baris:

```
<base64 payload JSON>
<base64 signature Ed25519>
```

Payload JSON, field tetap dan lengkap (tidak ada field opsional):

```json
{
  "customer": "Toko Contoh",
  "domain": "whuzpay.com",
  "plan": "Business",
  "issued_at": "2026-09-13",
  "expires_at": "2027-09-13"
}
```

- `plan` murni label tampilan di dashboard. Tidak menegakkan kuota apa pun.
- `issued_at`/`expires_at` format `YYYY-MM-DD`. `expires_at` berlaku sampai
  akhir hari itu (`23:59:59 UTC`) — lisensi masih `active` sepanjang hari
  tanggal tersebut, baru `expired` keesokan harinya.
- Signature dihitung atas **byte JSON persis seperti yang disimpan** di baris
  pertama (base64-decode baris 1 → itulah pesan yang ditandatangani), bukan
  hasil re-encode saat verifikasi. Menghindari bug kanonikalisasi JSON
  (urutan key, whitespace) yang bisa membuat signature valid tampak tidak
  valid atau sebaliknya.

## 3. Kunci Ed25519

Dua kunci, peran benar-benar terpisah — **ini bukan sekadar konvensi seperti
tiga kunci secretbox lain di proyek ini, tapi properti keamanan inti dari
seluruh sistem lisensi:**

- **Private key**: dihasilkan sekali oleh Akbar lewat `licensetool -genkey`,
  disimpan hanya di mesin Akbar sendiri (mis. `~/.gopay-license/private.key`).
  **Tidak pernah** masuk ke repo, `.env` manapun, atau server customer.
- **Public key**: di-*hardcode* sebagai konstanta Go di source code backend
  (`internal/licensecheck`), **bukan** env var atau file yang bisa diedit
  operator server. Ini disengaja: kalau public key bisa dikonfigurasi lewat
  env/file di server customer, customer bisa membuat key pair sendiri dan
  menandatangani lisensinya sendiri tanpa sepengetahuan Akbar — meniadakan
  seluruh gunanya tanda tangan. Karena di-*hardcode*, kunci ikut otomatis ke
  setiap build binary `server`, tanpa langkah deploy tambahan.

Mengganti key pair (mis. kalau private key bocor) berarti: generate pasangan
baru, update konstanta public key di source, rebuild dan redeploy `server` ke
**seluruh** customer, dan terbitkan ulang lisensi seluruh customer dengan key
baru. Ini operasi darurat yang mahal secara sengaja — bukan sesuatu yang
didesain untuk sering terjadi.

## 4. Paket `internal/licensecheck`

Paket baru, tanpa dependensi ke `store` (tidak menyentuh database) — sejalan
dengan pola `internal/auth` dan `internal/secretbox` yang juga pure-logic.

```go
type Status string

const (
    StatusActive  Status = "active"
    StatusMissing Status = "missing" // file tidak ada
    StatusInvalid Status = "invalid" // signature salah, atau domain tak cocok
    StatusExpired Status = "expired"
)

type License struct {
    Customer  string
    Domain    string
    Plan      string
    IssuedAt  time.Time
    ExpiresAt time.Time
    Status    Status
    // Reason menjelaskan status non-active secara spesifik, buat ditampilkan
    // di halaman License — mis. "domain pada lisensi (x) tidak cocok dengan
    // PUBLIC_DOMAIN server ini (y)". Kosong bila Status == StatusActive.
    Reason string
}

// Load membaca dan memverifikasi file lisensi di path tersebut, dibandingkan
// terhadap domain server ini. Tidak pernah mengembalikan error — kegagalan
// apa pun (file tak ada, signature salah, JSON korup) menghasilkan License
// dengan Status yang sesuai, supaya server tetap bisa start dan menampilkan
// status itu di dashboard alih-alih crash.
func Load(path string, domain string, now time.Time) License
```

`Load` dipanggil sekali di `main.go` setelah `config.Load()`, hasilnya
disimpan sebagai field `*API` (mis. `a.license license.License`) — **tidak**
dibaca ulang per-request. Memperpanjang lisensi berarti timpa file lalu
`systemctl restart gopay-ingestion`, sama seperti mengganti kunci-kunci lain
di proyek ini.

## 5. Penegakan di backend

**Env var baru**, wajib diisi (`config.Load` menolak start kalau kosong,
sama seperti kunci-kunci lain):

- `PUBLIC_DOMAIN` — domain instalasi ini, mis. `whuzpay.com`. Dibaca dari
  config, **bukan** dari header `Host` request (bisa dipalsukan klien).
- `LICENSE_FILE_PATH` — opsional, default `/opt/gopay-ingestion/license.lic`.

**Middleware baru `requireLicense`**, ditulis di `internal/httpapi`, dipasang
paling luar (sebelum `requireDevice`/`requireAdmin`/`requireAPIKey`) supaya
permintaan ditolak murah — tinggal baca status di memori, tanpa perlu buka
sesi/DB dulu:

```go
func (a *API) requireLicense(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if a.license.Status != license.StatusActive {
            a.writeError(w, http.StatusPaymentRequired,
                "license_"+string(a.license.Status), pesanUntukStatus(a.license.Status))
            return
        }
        next.ServeHTTP(w, r)
    })
}
```

Dipasang membungkus route berikut (urutan wrap: `requireLicense` di luar):

- `POST /api/v1/events`, `POST /api/v1/devices/heartbeat`, `GET /api/v1/device/me`
- `POST /api/v1/invoices`, `GET /api/v1/invoices/{invoiceID}`
- Seluruh `/api/v1/admin/*` **kecuali** `POST .../login`, `POST .../logout`,
  dan `GET .../license` (endpoint baru, §6) — ketiganya harus tetap jalan
  walau lisensi tidak aktif, supaya admin bisa login dan melihat kenapa.

`GET /api/v1/health` tidak berubah — tetap publik, tidak menyebut lisensi
sama sekali (bukan permukaan yang tepat untuk membocorkan status bisnis).

**Ticker background** (`ProcessDueWebhooks` dan expire-invoice di
`cmd/server/main.go`) diberi pengecekan yang sama di baris pertama tiap tick:
kalau `a.license.Status != StatusActive`, `return` tanpa memproses apa pun.
Tidak di-log tiap menit — cukup diam, statusnya sudah terlihat di dashboard.

## 6. Endpoint `GET /api/v1/admin/license`

Dibungkus `requireAdmin` (butuh sesi valid) tapi **tidak** `requireLicense`.

Respons:

```json
{
  "success": true,
  "license": {
    "customer": "Toko Contoh",
    "domain": "whuzpay.com",
    "plan": "Business",
    "issued_at": "2026-09-13",
    "expires_at": "2027-09-13",
    "days_remaining": 365,
    "status": "active",
    "reason": ""
  }
}
```

`days_remaining` bisa negatif kalau `status: "expired"` (menunjukkan sudah
berapa hari lewat). Kalau `status: "missing"`, seluruh field lisensi lain
dikirim kosong/null.

## 7. `licensetool`

`backend/cmd/licensetool/` — pola sama seperti `admintool`/`devicetool` dari
segi struktur kode (`flag`, pesan bahasa Indonesia), tapi **beda kelas**:
dijalankan **hanya di mesin Akbar**, tidak pernah di-build atau dikirim ke
VPS customer. `backend/deploy/README.md` akan menyebut ini eksplisit di
bagian "Tiap rilis — backend" supaya tidak tercampur dengan
`scp server devicetool admintool ...` yang sudah ada.

```bash
# Sekali saja, di laptop Akbar:
go run ./cmd/licensetool -genkey
# Mencetak:
#   Private Key (base64): simpan sendiri, mis. ~/.gopay-license/private.key
#   Public Key  (base64): tempel ke konstanta licensePublicKey di
#                          internal/licensecheck/verify.go, lalu commit

# Tiap terbit/perpanjang lisensi customer:
go run ./cmd/licensetool -issue \
  -key ~/.gopay-license/private.key \
  -customer "Toko Contoh" -domain "whuzpay.com" \
  -plan "Business" -expires "2027-09-13" \
  -out license.lic
# issued_at diisi otomatis dengan tanggal hari ini.
# Lalu: scp license.lic <VPS>:/opt/gopay-ingestion/license.lic
#       ssh <VPS> sudo systemctl restart gopay-ingestion
```

Tidak butuh database — `licensetool` tidak mengimpor `internal/store` sama
sekali, berbeda dari `admintool`/`devicetool`.

## 8. Dashboard — halaman `/license`

Dipindah dari daftar "Segera" ke halaman aktif di sidebar (item lain di
"Segera" — Settings, Logs — tidak berubah, masih menunggu hal lain).

Isi:

- Kartu status: badge warna sesuai `status` (`active` hijau "Aktif",
  `<30 hari tersisa>` kuning "Akan berakhir", `expired` merah "Kedaluwarsa",
  `missing` abu-abu "Belum terpasang", `invalid` merah "Tidak valid"),
  ditambah `customer`, `domain`, `plan`, `issued_at`, `expires_at`,
  `days_remaining`.
- Banner peringatan saat `days_remaining <= 30` dan `status == "active"`:
  "Lisensi akan berakhir dalam N hari."
- Banner beda (lebih tegas) saat status bukan `active`: menjelaskan bahwa
  endpoint pengiriman event, invoice, dan sebagian besar halaman admin
  lain sedang diblokir, dan mengarahkan untuk menghubungi Akbar untuk
  memperpanjang — **tanpa** menyebut istilah internal (nama file, path
  server, dsb), sesuai aturan copywriting di `CLAUDE.md`.
- Kalau `status == "invalid"` atau `"missing"`, field detail lisensi
  (customer/domain/plan/tanggal) tidak ditampilkan sama sekali karena tidak
  ada yang valid untuk ditampilkan — hanya status dan pesan.

Halaman lain yang memanggil endpoint ter-gerbang `requireLicense` akan
menerima `402` dari backend saat lisensi tidak aktif. UI halaman-halaman itu
menampilkan pesan error generik yang sudah ada untuk kegagalan API (tidak
perlu penanganan khusus per halaman) — hanya halaman `/license` yang
mendapat perlakuan UI khusus.

## 9. Testing

- `internal/licensecheck`: unit test murni, tanpa DB — signature valid/tidak
  valid, domain cocok/tidak cocok, sebelum/tepat di hari terakhir/setelah
  `expires_at`, file tidak ada, baris base64 korup, JSON korup dalam payload
  yang signature-nya tetap valid (kasus edge: signature valid atas payload
  yang bentuknya salah).
- `internal/httpapi`: `requireLicense` menolak `402` untuk representasi tiap
  kategori route (device, admin, API key) saat status bukan `active`, dan
  `GET /admin/license` tetap `200` di setiap status termasuk `expired`/
  `missing`. Helper test butuh cara menyuntikkan `License` tertentu ke `*API`
  tanpa lewat file sungguhan — field `license.License` di-set langsung di
  test setup.
- `licensetool`: tidak ada unit test khusus (CLI internal sekali-pakai untuk
  Akbar sendiri), cukup `go build ./...` dan `go vet ./...` bersih — sama
  seperti `admintool`/`devicetool`.
- `NEEDS-DEVICE` (manual, di VPS `whuzpay.com`): generate lisensi asli,
  pasang, restart, verifikasi `/license` dashboard menampilkan data benar
  dan endpoint lain tetap `200`; lalu generate lisensi sengaja expired,
  ulangi, verifikasi endpoint device/admin lain menjawab `402` dan `/license`
  tetap bisa dibuka dengan pesan yang sesuai.

## 10. Eksplisit di luar lingkup fase ini

- Tidak ada kuota per device/webhook/payment source/installation — hanya
  tanggal kedaluwarsa yang ditegakkan.
- Tidak ada dashboard/portal vendor untuk melihat status semua customer
  sekaligus — proyek terpisah, baru masuk akal saat customer lebih dari satu
  (lihat catatan di `CLAUDE.md` §"Pemecahan scope").
- Tidak ada mekanisme online: tidak ada phone-home, tidak ada aktivasi lewat
  internet, tidak ada auto-renewal. Sepenuhnya file manual yang dikirim
  Akbar ke customer di luar sistem (email, chat, dsb.).
- Tidak ada reminder email/notifikasi keluar saat mendekati kedaluwarsa —
  hanya banner di dashboard yang admin lihat sendiri saat membuka halaman.
- Tidak ada mekanisme revoke/blacklist untuk lisensi yang sudah terlanjur
  diterbitkan. Kalau perlu mencabut sebelum `expires_at`, satu-satunya jalan
  adalah mengganti `PUBLIC_DOMAIN` di server itu (memutus domain-lock) —
  limitasi yang disengaja untuk skema offline murni.
- Rotasi key pair Ed25519 (§3) bukan bagian dari alur normal — hanya
  didokumentasikan sebagai prosedur darurat, tidak dibangun tooling
  khusus untuknya di fase ini.
