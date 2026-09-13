# Landing Page & Self-Service Signup — Spec

**Tanggal:** 2026-09-13
**Status:** Disetujui, siap implementasi

Ini sub-project #2 (signup publik) + #6 (landing page) dari
[`2026-09-13-multitenant-accounts-design.md`](2026-09-13-multitenant-accounts-design.md)
§0, digabung jadi satu — landing page tanpa signup fungsional tidak ada
gunanya, dan signup tanpa halaman yang mengarahkan ke situ juga tidak akan
pernah dipakai siapa pun.

## 1. Ruang lingkup

Ditambahkan ke `dashboard/` (app yang sudah ada, bukan app baru):

- `/` — landing page publik (marketing), tanpa perlu login.
- `/register` — form registrasi publik, langsung membuat account aktif.
- Overview (halaman data customer yang sekarang di `/`) **pindah ke
  `/overview`**.

Ditambahkan ke backend: satu endpoint publik baru,
`POST /api/v1/signup` — satu-satunya endpoint di seluruh backend yang
sengaja tidak butuh autentikasi apa pun (bukan sesi, bukan API key, bukan
HMAC) selain rate limit per IP.

Di luar cakupan: halaman harga (belum ada model harga pasca-trial),
verifikasi email, OAuth/social login, pembayaran otomatis.

## 2. Routing & `proxy.ts`

`src/proxy.ts` sekarang menolak SEMUA path tanpa cookie sesi kecuali
`/login`. Diubah supaya `/` dan `/register` ikut dikecualikan dari gerbang
sesi (publik, tidak butuh cookie) — persis pola yang sudah ada untuk
`/login`, cuma daftarnya bertambah dua path.

Halaman Overview (`src/app/(dashboard)/page.tsx`) dipindah jadi
`src/app/(dashboard)/overview/page.tsx`. `nav-items.ts` (`href: "/"` untuk
"Overview") diubah jadi `href: "/overview"`. `login/page.tsx` yang sekarang
`router.push("/")` setelah login berhasil diubah jadi
`router.push("/overview")`.

`src/app/page.tsx` (landing) dan `src/app/register/page.tsx` dibuat baru,
**di luar** route group `(dashboard)` — tidak memakai `AppShell`/sidebar
sama sekali, layout sendiri (halaman publik, bukan bagian dashboard).

## 3. Endpoint `POST /api/v1/signup`

Handler baru di `internal/httpapi/signup.go`, didaftarkan di `api.go`
TANPA middleware auth apa pun (beda dari seluruh endpoint lain):

```go
mux.HandleFunc("POST /api/v1/signup", a.handleSignup)
```

Request: `{business_name, email, username, password}` — semua wajib diisi,
`password` minimal 8 karakter (validasi sama seperti belum ada di endpoint
manapun sekarang; ini yang pertama menerima password dari input publik
langsung, bukan digenerate server seperti API key/webhook secret).

Proses:
1. Rate limit per IP — `signupThrottle *loginThrottle` (tipe yang sama
   persis dipakai `loginThrottle`/`vendorLoginThrottle`, cuma instance
   ketiga; endpoint publik tanpa auth adalah target abuse paling murah
   untuk diserang).
2. Cek `email`/`username` belum dipakai. Kolom `email`/`username` di
   tabel `accounts` sudah `UNIQUE` sejak migration `00008_accounts.sql`
   (dideklarasikan polos tanpa nama constraint eksplisit, jadi Postgres
   memberi nama otomatis `accounts_email_key`/`accounts_username_key`).
   `internal/store/account.go` diperluas: `CreateAccount` mendeteksi
   pelanggaran constraint itu (pola `isUniqueViolation` yang sudah ada di
   `invoice.go`, dipakai dengan dua nama constraint di atas) dan
   mengembalikan `store.ErrAccountEmailTaken`/`store.ErrAccountUsernameTaken`
   (dua sentinel baru) alih-alih error generik. `handleSignup` menerjemah-
   kannya jadi `409 email_taken`/`409 username_taken`. `handleVendorCreateAccount`
   (Vendor Dashboard, sudah ada) ikut diperbarui memakai sentinel yang sama
   — sebelumnya email/username bentrok di sana cuma menghasilkan `500`
   generik, ini sekalian memperbaikinya jadi `409` yang benar.
3. `store.CreateAccount(ctx, CreateAccountInput{ID: randomPrefixedAccountID(), BusinessName, Email, Username, PlaintextPassword: password, Plan: "Starter", MaxDevices: 3, ExpiresAt: now.Add(3*24*time.Hour)})`
   — `randomPrefixedAccountID` sudah ada (dipakai `vendor_accounts.go`,
   tinggal dipakai ulang, bukan dibuat baru).
4. `LogAudit(ctx, "signup", "ACCOUNT_CREATED", id, map[string]string{"business_name":..., "plan":"Starter"})`
   — `actor="signup"` (bukan username vendor manapun) supaya di Audit Log
   Vendor Dashboard kelihatan jelas ini dibuat sendiri oleh customer, bukan
   oleh Akbar.
5. Sukses → set cookie `admin_session` langsung (auto-login, kode sama
   persis blok `http.SetCookie` di `handleAdminLogin`) — response
   `{"success": true}`, sama bentuk dengan login biasa. Frontend redirect
   ke `/overview` setelah ini, tidak ada langkah tambahan.

Error response mengikuti `errorResponse` yang sudah ada
(`{success, error, message}`). Kode error baru: `invalid_payload` (field
kosong/password < 8 karakter), `email_taken`, `username_taken`,
`too_many_attempts` (rate limit, sama kode dengan login).

## 4. `dashboard/src/lib/api.ts`

Fungsi baru:

```ts
export function signup(input: {
  business_name: string;
  email: string;
  username: string;
  password: string;
}): Promise<{ success: true }> {
  return apiFetch("/api/v1/signup", { method: "POST", body: JSON.stringify(input) });
}
```

## 5. Halaman `/register`

Pola sama persis `login/page.tsx` (Card shadcn, form terkontrol, `ApiError`
handling) — field: Nama Bisnis, Email, Username, Password. Sukses →
`router.push("/overview")` (cookie sudah ter-set oleh response `signup()`,
sama seperti login). Error `email_taken`/`username_taken` ditampilkan
sebagai pesan spesifik ("Email/Username sudah dipakai"), bukan disamakan
generik seperti error login (di sini TIDAK ada alasan keamanan untuk
menyembunyikan "sudah dipakai" — ini pendaftaran baru, bukan percobaan
masuk ke akun orang lain).

## 6. Halaman `/` (landing page)

Layout sendiri (bukan `AppShell` dashboard), satu kolom, section dari atas
ke bawah:

1. **Hero** — judul + subjudul singkat, dua tombol: "Daftar Gratis 3 Hari"
   (→ `/register`) dan "Masuk" (→ `/login`).
2. **Cara kerja** — 3 langkah singkat dengan ikon: pasang app Android →
   notifikasi GoPay otomatis tercatat → invoice otomatis lunas + webhook
   ke website merchant.
3. **Fitur** — grid kartu singkat, HANYA fitur yang sungguhan sudah ada
   (nominal unik anti-salah-hitung, webhook otomatis invoice.paid/expired,
   dashboard realtime Overview/Devices/Events/Transactions, konsol
   pengecualian transaksi ganjil, API key untuk integrasi backend
   merchant) — tidak boleh menyebut fitur yang belum ada (harga, laporan
   lanjutan, dll).
4. **CTA penutup** — ulang tombol "Daftar Gratis 3 Hari".
5. Footer minimal — nama produk + tahun.

Tidak ada section harga atau testimoni — belum ada model harga
pasca-trial atau customer nyata untuk dikutip, mengarang keduanya lebih
buruk daripada tidak menampilkannya sama sekali.

## 7. Testing

Backend (`internal/httpapi/signup_test.go`):
- Signup berhasil → 200, cookie `admin_session` ter-set, account
  tersimpan dengan plan Starter/max_devices 3/expires_at ≈ now+3 hari.
- Email sudah dipakai → 409 `email_taken`.
- Username sudah dipakai → 409 `username_taken`.
- Password < 8 karakter → 400 `invalid_payload`.
- Rate limit: percobaan ke-6 dalam window → 429 `too_many_attempts`.
- Audit log tercatat dengan `actor="signup"`.
- Endpoint TIDAK menerima cookie/API key/HMAC apa pun — dipanggil tanpa
  header auth sama sekali dan tetap berhasil (membuktikan memang sengaja
  publik, bukan lupa memasang middleware).

Frontend (`dashboard/`): `npx tsc --noEmit`, `npx eslint .`,
`npx next build` — tidak ada test unit terpisah untuk halaman baru,
konsisten dengan pola dashboard yang sudah ada (verifikasi manual di
browser untuk halaman UI, lihat qa-rules.md).

## 8. Risiko yang sengaja diterima

Signup publik tanpa verifikasi email berarti siapa pun bisa membuat
banyak account dengan email palsu — dibatasi rate limit per IP (5 per 15
menit, sama seperti login), bukan diblokir total. Trial 3 hari membuat
dampak penyalahgunaan kecil (bukan akses setahun gratis). Verifikasi email
sungguhan ditunda sampai ada bukti penyalahgunaan nyata terjadi — YAGNI.
