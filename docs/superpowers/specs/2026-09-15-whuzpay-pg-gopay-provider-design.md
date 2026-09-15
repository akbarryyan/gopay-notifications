# Design Spec — Provider "gopay" di whuzpay-pg + Pencabutan Cashi

**Tanggal:** 2026-09-15
**Cakupan:** Sub-project 2 dari 2 rencana migrasi whuzpay-pg dari Cashi ke gopay-notifications. Sub-project 1 (QRIS statis per account, gopay-notifications) sudah selesai dan sudah di-push — spec:
[`docs/superpowers/specs/2026-09-15-account-qris-image-design.md`](2026-09-15-account-qris-image-design.md).
Sub-project ini sepenuhnya di `whuzpay-pg/` (folder terpisah dalam repo yang sama) — tidak ada perubahan lanjutan di gopay-notifications.
**Status:** Disetujui, siap masuk implementation plan

---

## 1. Konteks

`whuzpay-pg` (payment gateway aggregator multi-tenant) saat ini merutekan seluruh pembayaran produksi lewat Cashi — satu kredensial **global** (`CASHI_API_KEY`/`CASHI_SECRET_KEY` di env, didaftarkan sekali di `cmd/api/main.go`, dipakai SEMUA merchant). Keputusan dari sesi brainstorming sebelumnya (chat): ganti total ke gopay-notifications, bukan coexist — aman karena `whuzpay-pg` belum production (data yang ada cuma seed demo).

Perbedaan mendasar yang menentukan seluruh desain ini: **gopay-notifications butuh kredensial per merchant** (tiap merchant whuzpay-pg = satu account gopay-notifications, dengan API key dan webhook secret miliknya sendiri — lihat model "bring-your-own-device" dari brainstorming sebelumnya), bukan satu kredensial global seperti Cashi. Interface `PaymentProvider` yang ada sekarang didesain untuk model kredensial global; sub-project ini memperluasnya secukupnya.

Keputusan dari sesi brainstorming ini:

| Keputusan | Pilihan |
|---|---|
| Penyimpanan kredensial (API key + webhook secret gopay-notifications milik tiap merchant) | Polos (plaintext), sama pola dengan `merchants.webhook_secret` yang sudah ada (migrasi 013) — whuzpay-pg belum punya sistem kripto sama sekali, dan menambahnya di luar cakupan migrasi ini. |
| Cakupan pencabutan Cashi | Hapus total — kode adapter, config, env var, referensi dokumentasi — bukan sekadar berhenti didaftarkan. |

---

## 2. Perluasan interface `PaymentProvider`

Tiga method yang butuh tahu kredensial merchant mana yang dipakai, ditambah `merchantID`. Ini dicek dulu terhadap seluruh titik pemanggilan yang ada (`internal/service/payment_service.go`, reconciliation) — **baris `payment` yang sedang diproses selalu sudah ada di scope di titik itu**, jadi `merchantID` selalu tersedia tanpa query tambahan.

`internal/provider/adapter.go`:

```go
type PaymentProvider interface {
	GetName() string
	CreatePayment(ctx context.Context, req *domainProvider.ProviderPaymentRequest) (*domainProvider.ProviderPaymentResponse, error)
	GetPaymentStatus(ctx context.Context, providerReference string, merchantID uuid.UUID) (*domainProvider.NormalizedPaymentStatus, error)
	ValidateWebhook(rawPayload []byte, signature string, merchantID uuid.UUID) error
	ParseWebhook(rawPayload []byte) (*domainProvider.ProviderWebhookPayload, error)
	NormalizeStatus(providerStatus string) string
}
```

`internal/domain/provider/provider.go` — `ProviderPaymentRequest` dapat field baru, dan field lama `UseCustomMerchantName` **dihapus** (satu-satunya pemakainya adalah fitur "QRIS Custom" Cashi, ikut dihapus di §6):

```go
type ProviderPaymentRequest struct {
	InternalReference string
	Amount            int64
	Currency          string
	Description       string
	CustomerName      *string
	CustomerEmail     *string
	ExpiresAt         time.Time
	CallbackURL       string
	MerchantID        uuid.UUID // BARU -- dipakai adapter cari kredensial merchant ini
}
```

Konstanta provider (`ProviderCashi` dihapus, `ProviderGopay` ditambah; `ProviderMidtrans`/`ProviderXendit`/`ProviderDuitku` TETAP dibiarkan sebagai reservasi provider masa depan, tidak disentuh):

```go
const (
	ProviderGopay    = "gopay"
	ProviderMidtrans = "midtrans"
	ProviderXendit   = "xendit"
	ProviderDuitku   = "duitku"
)
```

**Pemanggil (`payment_service.go`, `payment_reconcile`) tinggal menyertakan `req.MerchantID = p.MerchantID` (create) atau `p.MerchantID` (reconcile/webhook) di titik yang sudah ada** — bukan perubahan struktural, cuma menambah satu argumen di panggilan yang sudah ada. Sandbox adapter (`internal/provider/sandbox/adapter.go`) menerima parameter baru tapi mengabaikannya (tetap mock in-memory, tidak butuh kredensial apa pun).

---

## 3. Kredensial per merchant

**Migrasi baru** `016_create_merchant_gopay_credentials.sql`:

```sql
CREATE TABLE IF NOT EXISTS merchant_gopay_credentials (
    merchant_id UUID PRIMARY KEY REFERENCES merchants(id) ON DELETE CASCADE,
    api_key TEXT,
    webhook_secret TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

COMMENT ON TABLE merchant_gopay_credentials IS 'Kredensial gopay-notifications milik tiap merchant -- API key untuk memanggil POST /invoices, webhook secret untuk verifikasi X-Webhook-Signature. Disimpan polos (plaintext), pola sama dengan merchants.webhook_secret (migrasi 013): keamanan mengandalkan akses database, bukan enkripsi aplikasi.';
COMMENT ON COLUMN merchant_gopay_credentials.api_key IS 'sk_... dari halaman API Keys gopay-notifications milik merchant ini.';
COMMENT ON COLUMN merchant_gopay_credentials.webhook_secret IS 'whsec_... dari webhook endpoint yang dibuat merchant di gopay-notifications, mengarah ke {APP_BASE_URL}/api/v1/provider-webhooks/gopay.';
```

`merchant_id` sebagai primary key (relasi 1:1, sama pola dengan `account_qris_images` di gopay-notifications sub-project 1). Kedua kolom **nullable** dan independen — merchant boleh mengisi `api_key` dulu (bisa langsung coba buat payment) sebelum sempat membuat webhook endpoint dan mengisi `webhook_secret`; tidak ada gerbang all-or-nothing.

`internal/repository/merchant_gopay_credentials_repository.go` (baru, pola sama `merchant_provider_config_repository.go`):

```go
type MerchantGopayCredentialsRepository struct {
	db *sql.DB
}

func (r *MerchantGopayCredentialsRepository) Get(ctx context.Context, merchantID uuid.UUID) (*gopaycreds.Credentials, error)
func (r *MerchantGopayCredentialsRepository) Upsert(ctx context.Context, merchantID uuid.UUID, apiKey, webhookSecret *string) error
```

`Upsert` menerima `*string` per field — pola tri-state sama dengan `NotificationSettings` gopay-notifications: `nil` = biarkan nilai lama, `""` = hapus, isi = ganti. `Get` mengembalikan `sql.ErrNoRows`-wrapped not-found kalau merchant belum pernah mengisi apa pun.

---

## 4. Adapter baru `internal/provider/gopay/`

`adapter.go`:

```go
// credentialsRepository -- interface lokal ke package ini (bukan tipe
// konkret repository), supaya adapter bisa diuji dengan fake tanpa
// database sungguhan. *repository.MerchantGopayCredentialsRepository
// (§3) memenuhi ini secara struktural, tidak perlu deklarasi eksplisit.
type credentialsRepository interface {
	Get(ctx context.Context, merchantID uuid.UUID) (*gopaycreds.Credentials, error)
}

type Adapter struct {
	baseURL    string // alamat gopay-notifications, mis. https://whuzpay.com
	credsRepo  credentialsRepository
	httpClient *http.Client
}

func NewAdapter(baseURL string, credsRepo credentialsRepository) *Adapter
func (a *Adapter) GetName() string // "gopay"
```

Deteksi `409 qris_not_configured` di `CreatePayment`: baca `error` field dari body JSON respons gopay-notifications saat status HTTP-nya `409` — sama pola dengan `CashiAdapter.doRequest` yang sudah ada (decode body error sebelum menyimpulkan jenis kegagalan), bukan cuma memeriksa status code mentah.

- **`CreatePayment(ctx, req)`**: ambil `api_key` merchant lewat `credsRepo.Get(ctx, req.MerchantID)`; kalau kosong → `ErrCredentialsNotConfigured`. `POST {baseURL}/api/v1/invoices` (`Authorization: Bearer <api_key>`, body `{external_ref: req.InternalReference, amount: req.Amount}`). Map response: `unique_amount`→`Amount`, `id`→`ProviderReference`, `qris_image`→`QRISData` (data URI apa adanya — halaman bayar whuzpay-pg **sudah generic**, `isImageQr` di `pay/[reference]/page.tsx` sudah mengenali prefix `data:image`, TIDAK PERLU perubahan frontend halaman bayar), `expires_at`→`ExpiresAt`, `status`→`"pending"` (selalu `PENDING` saat baru dibuat). `PaymentURL` dibiarkan nil -- gopay-notifications tidak punya halaman checkout ter-host, whuzpay-pg sendiri yang menampilkan QR.
  - gopay-notifications menolak `409 qris_not_configured` kalau account belum upload QRIS (sub-project 1) — dipetakan ke error lokal `ErrQRISNotConfigured`, beda pesan dari `ErrCredentialsNotConfigured` (kredensial ada tapi merchant belum setup QRIS di gopay-notifications-nya).
- **`GetPaymentStatus(ctx, providerReference, merchantID)`**: ambil `api_key` lewat `credsRepo`, `GET {baseURL}/api/v1/invoices/{providerReference}`, map `status` lewat `NormalizeStatus`.
- **`ValidateWebhook(rawPayload, signature, merchantID)`**: ambil `webhook_secret` merchant lewat `credsRepo.Get`; kalau kosong atau `signature` kosong → `providerPkg.ErrInvalidWebhookSignature` (dipakai ulang, sudah ada, dipakai `payment.ErrWebhookValidationFailed` di pemanggil — tidak perlu error baru). Hitung HMAC-SHA256 hex dari `rawPayload` pakai `webhook_secret`, `hmac.Equal` terhadap `signature` — persis kontrak yang didokumentasikan API Docs gopay-notifications (`X-Webhook-Signature`).
- **`ParseWebhook(rawPayload)`**: decode `{event, invoice: {id, external_ref, status, paid_at, ...}, sent_at}`, `invoice.id`→`ProviderReference` (HARUS `id`, bukan `external_ref` — `id` yang tersimpan sebagai `payments.provider_reference` sejak `CreatePayment`, dipakai `GetByProviderReference` mencari baris payment). Status `PAID`→dipetakan lewat `NormalizeStatus`.
- **`NormalizeStatus(s)`**: `PAID`→`"paid"`, `EXPIRED`→`"expired"`, default→`"pending"` (gopay-notifications tidak punya status `failed`).

`errors.go`:

```go
var (
	ErrCredentialsNotConfigured = errors.New("gopay: merchant belum mengatur API key gopay-notifications")
	ErrQRISNotConfigured        = errors.New("gopay: merchant belum mengatur QRIS di gopay-notifications")
)
```

---

## 5. Perbaikan alur terima webhook

`internal/handler/webhook_handler.go` — header signature yang dibaca sekarang `x-gateway-signature` (konvensi Cashi, dihapus bersama Cashi) → ganti jadi `X-Webhook-Signature` (nama header gopay-notifications, lihat API Docs gopay-notifications §Webhook).

`internal/service/payment_webhook_service.go`, `ProcessWebhook` — **urutan dibalik**. Sekarang: `ValidateWebhook` → `ParseWebhook` → cari payment. Masalahnya: `ValidateWebhook` butuh tahu `merchantID` (kredensial per merchant), tapi `merchantID` baru diketahui SETELAH payment ditemukan lewat `ParseWebhook`. Urutan baru:

```
1. Persist raw webhook event (tidak berubah)
2. selectedProvider := getProviderByName(providerName)
3. webhookPayload, err := selectedProvider.ParseWebhook(rawPayload)   // DIPINDAH ke atas
4. p, err := paymentRepo.GetByProviderReference(ctx, webhookPayload.ProviderReference)  // DIPINDAH ke atas
5. err := selectedProvider.ValidateWebhook(rawPayload, signature, p.MerchantID)  // p.MerchantID sekarang tersedia
6. (lanjut seperti sekarang: cek status terminal/duplikat, terapkan update)
```

**Ini aman**: yang dibaca sebelum tanda tangan terverifikasi cuma `invoice.id` (referensi publik, bukan rahasia) untuk pencarian baris payment (baca-saja) — tidak ada state yang berubah sebelum langkah 5 lulus. Kalau `ParseWebhook` gagal (JSON rusak) atau payment tidak ditemukan, ditolak sama seperti sekarang, sebelum sempat mencapai validasi.

---

## 6. Pencabutan Cashi (total)

| Dihapus | Keterangan |
|---|---|
| `internal/provider/cashi/` (adapter, types, test) | Satu-satunya konsumen field `UseCustomMerchantName` |
| `CashiConfig` di `internal/config/config.go`, pemanggilan `cfg.Cashi.*` | Termasuk validasi wajib-di-production |
| `CASHI_BASE_URL`/`CASHI_API_KEY`/`CASHI_SECRET_KEY` di `.env.example` | Ganti jadi `GOPAY_BASE_URL` (alamat gopay-notifications, default `https://whuzpay.com`) — tidak ada API-key/secret global karena per-merchant |
| Wiring `cashiAdapter` di `cmd/api/main.go` | Ganti `RegisterProvider(cashiAdapter)` → `RegisterProvider(gopayAdapter)`, `RegisterPaymentMethodProvider("qris", cashiAdapter.GetName())` → `...gopayAdapter.GetName())` |
| Referensi Cashi di `README.md`, `RUNNING.md` | Struktur folder, tabel env var, diagram alur — diperbarui menyebut "gopay" |
| Bruno request Cashi-spesifik (`02 - Create Payment (Production, QRIS Custom).bru`) | Field `QRIS_CUSTOM` di request tidak relevan lagi |
| Default `provider_name: "cashi"` di `MerchantProviderConfigSection.tsx` (admin frontend) | Ganti default jadi `"gopay"` |

**Tidak disentuh** (di luar cakupan sub-project ini, cuma disebutkan biar jelas kenapa dilewati): `paymentlink.PlatformMinAmount`/`PlatformMaxAmount` (2.000–10.000.000 IDR) — komentarnya menyebut "Cashi limits" tapi ini kebijakan batas platform yang berdiri sendiri, bukan constraint teknis dari API Cashi; mengubah angkanya butuh keputusan bisnis terpisah, bukan bagian dari migrasi provider ini. Komentar kode yang menyebut "Cashi" di situ dibiarkan apa adanya kecuali diminta lain kali.

---

## 7. UI merchant — input kredensial

Card baru **"Provider Pembayaran GoPay"** di `whuzpay-pg/front/app/dashboard/settings/page.tsx` (halaman Settings milik merchant sendiri, di sebelah card "Webhook Secret" yang sudah ada — pola komponen sama: `Card`, form terkontrol, `AlertDialog` konfirmasi untuk aksi yang mengganti nilai tersimpan).

Isi:

- Instruksi 4 langkah singkat (teks statis): 1) Daftar/login ke gopay-notifications, 2) Upload QRIS di halaman Settings gopay-notifications, 3) Buat API key di halaman API Keys, 4) Buat webhook endpoint mengarah ke `{APP_BASE_URL}/api/v1/provider-webhooks/gopay` dengan event `invoice.paid` di halaman Webhooks gopay-notifications.
- Dua field password-masked: **API Key** dan **Webhook Secret**, masing-masing dengan status "Sudah diatur"/"Belum diatur" (TIDAK PERNAH menampilkan nilai asli balik, pola sama `smtp_password_set` gopay-notifications) + tombol simpan per field.

Backend baru:

```
GET /api/v1/merchant/gopay-credentials    -- {api_key_configured: bool, webhook_secret_configured: bool}
PUT /api/v1/merchant/gopay-credentials    -- {api_key?: string, webhook_secret?: string} (tri-state: absen=biarkan, ""=hapus, isi=ganti)
```

Keduanya di belakang middleware auth merchant yang sudah ada (`merchantAuth`/sesi dashboard, BUKAN API-key aggregator — ini halaman Settings merchant, bukan endpoint publik).

---

## 8. Testing

**Go (`internal/provider/gopay/adapter_test.go`, pola sama `sandbox/adapter_test.go`/`cashi/client_test.go` yang dihapus):**
- `CreatePayment` sukses: `httptest.Server` mock gopay-notifications, verifikasi `Authorization` header berisi API key merchant yang benar, mapping field response ke `ProviderPaymentResponse` benar (`QRISData` persis `qris_image`, `Amount` persis `unique_amount`).
- `CreatePayment` tanpa kredensial → `ErrCredentialsNotConfigured`, TIDAK memanggil HTTP sama sekali.
- `CreatePayment` menerima `409 qris_not_configured` dari upstream → `ErrQRISNotConfigured`.
- `GetPaymentStatus` sukses, mapping status.
- `ValidateWebhook`: signature benar → nil; signature salah → `ErrInvalidWebhookSignature`; `webhook_secret` belum diatur → `ErrInvalidWebhookSignature` juga (bukan panic/nil-pointer).
- `ParseWebhook`: `invoice.id` (bukan `external_ref`) yang jadi `ProviderReference`; status `PAID`→`"paid"`.

**Go (`internal/service/payment_webhook_service_test.go`, tambahan):**
- Webhook gopay untuk merchant A tervalidasi pakai secret merchant A meski ada merchant B dengan secret berbeda terdaftar juga (bukti urutan baru §5 benar-benar mengambil secret merchant yang TEPAT, bukan yang pertama ditemukan).
- Signature salah → payment TIDAK berubah status (masih pending), event tercatat `rejected`.

**Go (`internal/handler/payment_handler_test.go`, tambahan):** `CreatePayment` untuk merchant tanpa kredensial gopay → response HTTP `400` dengan pesan jelas (lewat `respondCreatePaymentError`, kasus baru `errors.Is(err, gopay.ErrCredentialsNotConfigured)`).

**Frontend:** `npm run build` (Next.js) bersih untuk `front/`; tidak ada perubahan di halaman `pay/[reference]` (sudah generic) jadi tidak perlu test baru di situ.

**Manual (`NEEDS-DEVICE`, tidak bisa diuji otomatis):** end-to-end sungguhan — merchant asli isi kredensial gopay-notifications sungguhan di whuzpay-pg, buat payment, scan QR sungguhan, bayar, dan webhook gopay-notifications benar-benar sampai ke `whuzpay-pg` (butuh whuzpay-pg bisa diakses dari internet publik supaya gopay-notifications bisa mengirim webhook — di lokal ini berarti tunnel semacam ngrok, atau diuji langsung setelah whuzpay-pg di-deploy).

---

## 9. Yang sengaja di luar cakupan spec ini

- Deploy `whuzpay-pg` ke VPS (belum ada infrastruktur produksi untuk `whuzpay-pg` sama sekali — di luar migrasi provider ini).
- Provider Midtrans/Xendit/Duitku (konstanta sudah direservasi di `domain/provider/provider.go`, tidak diimplementasikan sekarang).
- Mengubah `payment_links.PlatformMinAmount`/`PlatformMaxAmount` (lihat §6).
- Menyederhanakan/menghapus UI routing admin (`MerchantProviderConfigSection.tsx`, priority/weight/failover) meski dengan satu provider fiturnya jadi kurang berguna — dibiarkan apa adanya (cuma ganti default provider), berguna kembali begitu provider kedua (Midtrans dkk) ditambahkan nanti.
- Sistem enkripsi-at-rest untuk kredensial (lihat keputusan §1).
