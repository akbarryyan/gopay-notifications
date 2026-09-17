# Distribusi APK Android Bridge Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Merchant mana pun bisa mengunduh dan memasang aplikasi Android bridge (varian production) tanpa butuh laptop/toolchain development Akbar.

**Architecture:** Build APK lewat EAS Build (cloud, bukan laptop), APK-nya sendiri di-hosting sebagai file statis permanen di Customer Dashboard (`dashboard/public/downloads/`), dan halaman publik baru di dashboard menjelaskan cara pasang + tombol unduh.

**Tech Stack:** Expo/EAS Build (`mobile/`), Next.js App Router (`dashboard/`).

## Global Constraints

- Varian yang didistribusikan HARUS `production` (`id.akbarryyan.gopaybridge`, nama "GoPay Bridge") — bukan `.dev`/`.uat`.
- Halaman unduh WAJIB publik (tanpa sesi) — merchant baru mengunduh app sebelum/sambil membuat akun.
- Tidak ada Play Store, tidak ada auto-update in-app, tidak ada sub-project onboarding streamlining di plan ini (lihat spec §3 — di luar cakupan, spec terpisah).
- Minimum Android 8.0 (API 26) — sudah dipatok di `mobile/app.config.ts` (`minSdkVersion: 26`), cuma perlu disebut di copy halaman unduh, tidak perlu diubah.
- File APK sungguhan TIDAK dibuat oleh plan ini — itu hasil `eas build` yang Akbar jalankan sendiri di Task 5. Jangan membuat file APK palsu/dummy.

---

### Task 1: `eas.json` — build profile production

**Files:**
- Create: `mobile/eas.json`

**Interfaces:**
- Konsumsi: `mobile/app.config.ts` (env `APP_VARIANT` dibaca dari sana, sudah ada — lihat Task 2).
- Tidak ada kode lain yang bergantung pada isi file ini; ini murni config CLI EAS.

- [ ] **Step 1: Tulis `mobile/eas.json`**

```json
{
  "cli": {
    "version": ">= 16.0.0",
    "appVersionSource": "remote"
  },
  "build": {
    "production": {
      "env": {
        "APP_VARIANT": "production"
      },
      "android": {
        "buildType": "apk"
      },
      "distribution": "internal",
      "autoIncrement": true
    }
  }
}
```

- [ ] **Step 2: Validasi JSON-nya sah dan field-nya sesuai skema EAS**

Run: `node -e "const c = require('./mobile/eas.json'); if (c.build.production.distribution !== 'internal' || c.build.production.android.buildType !== 'apk' || c.build.production.env.APP_VARIANT !== 'production') { throw new Error('field tidak sesuai') }; console.log('ok')"`

Expected: `ok`

Ini bukan pengganti build sungguhan (lihat Task 5) — cuma memastikan file-nya valid JSON dan field kuncinya benar sebelum Akbar mencoba `eas build` sungguhan.

- [ ] **Step 3: Commit**

```bash
git add mobile/eas.json
git commit -m "build(mobile): tambah profile EAS production untuk distribusi APK langsung"
```

---

### Task 2: `expo-dev-client` jadi kondisional per varian

**Files:**
- Modify: `mobile/app.config.ts:60-94`

**Interfaces:**
- Konsumsi: `VARIANT` (`'development' | 'uat' | 'production'`, sudah ada di file ini, baris 7) dan `VARIANTS`/`v` yang sudah ada.
- Tidak menghasilkan interface baru untuk task lain — perubahan murni internal ke file ini.

Isi `mobile/app.config.ts` SEKARANG (baris 43-97, untuk konteks — jangan disalin ulang, cuma acuan lokasi edit):

```ts
const config: ExpoConfig = {
  name: v.name,
  slug: 'gopay-bridge',
  version: '1.0.0',
  orientation: 'portrait',
  icon: './assets/icon.png',
  android: {
    package: v.package,
    adaptiveIcon: {
      backgroundColor: '#E6F4FE',
      foregroundImage: './assets/android-icon-foreground.png',
      backgroundImage: './assets/android-icon-background.png',
      monochromeImage: './assets/android-icon-monochrome.png',
    },
    predictiveBackGestureEnabled: false,
    permissions: ['android.permission.INTERNET', 'android.permission.ACCESS_NETWORK_STATE'],
  },
  plugins: [
    'expo-dev-client',
    // Menyuntikkan NotificationListenerService ke AndroidManifest.
    './plugins/withGopayListener',
    [
      'expo-camera',
      {
        cameraPermission: 'Dipakai untuk memindai QR pairing dari Dashboard.',
        // Tidak butuh audio sama sekali -- recordAudioAndroid: false
        // membuat plugin TIDAK menambahkan izin RECORD_AUDIO ke manifest,
        // yang tidak pernah dipakai di sini.
        recordAudioAndroid: false,
        barcodeScannerEnabled: true,
      },
    ],
    [
      'expo-build-properties',
      {
        android: {
          minSdkVersion: 26,
          usesCleartextTraffic: v.cleartext,
        },
      },
    ],
  ],
}

export default config
```

- [ ] **Step 1: Ganti literal array `plugins` jadi variabel yang dibangun kondisional**

Ganti seluruh blok `const config: ExpoConfig = { ... }` sampai `export default config` menjadi:

```ts
// expo-dev-client cuma perlu di development/uat -- ini yang dipakai Metro
// untuk terhubung ke laptop saat `expo run:android`. Production sengaja
// TIDAK menyertakannya: aplikasi yang dipasang merchant lewat APK hasil
// EAS Build (lihat mobile/eas.json) harus langsung masuk ke aplikasi itu
// sendiri, bukan layar dev-client yang minta connect ke dev server --
// dan dev-menu (bisa terpicu shake gesture) tidak semestinya ada di
// aplikasi yang dipegang pengguna akhir.
const plugins: ExpoConfig['plugins'] = [
  // Menyuntikkan NotificationListenerService ke AndroidManifest.
  './plugins/withGopayListener',
  [
    'expo-camera',
    {
      cameraPermission: 'Dipakai untuk memindai QR pairing dari Dashboard.',
      // Tidak butuh audio sama sekali -- recordAudioAndroid: false
      // membuat plugin TIDAK menambahkan izin RECORD_AUDIO ke manifest,
      // yang tidak pernah dipakai di sini.
      recordAudioAndroid: false,
      barcodeScannerEnabled: true,
    },
  ],
  [
    'expo-build-properties',
    {
      android: {
        minSdkVersion: 26,
        usesCleartextTraffic: v.cleartext,
      },
    },
  ],
]

if (VARIANT !== 'production') {
  plugins.unshift('expo-dev-client')
}

const config: ExpoConfig = {
  name: v.name,
  slug: 'gopay-bridge',
  version: '1.0.0',
  orientation: 'portrait',
  icon: './assets/icon.png',
  android: {
    package: v.package,
    adaptiveIcon: {
      backgroundColor: '#E6F4FE',
      foregroundImage: './assets/android-icon-foreground.png',
      backgroundImage: './assets/android-icon-background.png',
      monochromeImage: './assets/android-icon-monochrome.png',
    },
    predictiveBackGestureEnabled: false,
    permissions: ['android.permission.INTERNET', 'android.permission.ACCESS_NETWORK_STATE'],
  },
  plugins,
}

export default config
```

(Komentar penjelasan `minSdkVersion`/cleartext yang sudah ada di file asli boleh tetap dipertahankan persis di tempatnya — dihilangkan di sini cuma supaya diff mudah dibaca, BUKAN instruksi untuk menghapusnya.)

- [ ] **Step 2: Type-check**

Run: `cd mobile && npx tsc --noEmit`
Expected: tidak ada error.

- [ ] **Step 3: Verifikasi manual isi plugin per varian (tanpa build sungguhan)**

Run:
```bash
cd mobile
APP_VARIANT=production npx expo config --type public --json | node -e "const d=JSON.parse(require('fs').readFileSync(0,'utf8')); console.log(d.plugins.some(p => p === 'expo-dev-client' || (Array.isArray(p) && p[0] === 'expo-dev-client')))"
APP_VARIANT=development npx expo config --type public --json | node -e "const d=JSON.parse(require('fs').readFileSync(0,'utf8')); console.log(d.plugins.some(p => p === 'expo-dev-client' || (Array.isArray(p) && p[0] === 'expo-dev-client')))"
```

Expected: baris pertama (`production`) mencetak `false`, baris kedua (`development`) mencetak `true`.

- [ ] **Step 4: Commit**

```bash
git add mobile/app.config.ts
git commit -m "fix(mobile): expo-dev-client cuma untuk development/uat, tidak untuk production"
```

---

### Task 3: Route `/download-app` dikecualikan dari gerbang sesi

**Files:**
- Modify: `dashboard/src/proxy.ts:18-25`

**Interfaces:**
- Konsumsi: `PUBLIC_PATHS` (Set yang sudah ada di file ini).
- Produces: path `/download-app` sekarang lolos proxy tanpa cookie sesi — dikonsumsi Task 4 (halaman yang dipasang di path itu harus bisa diakses tanpa login).

- [ ] **Step 1: Tambah `/download-app` ke `PUBLIC_PATHS`**

Ganti:

```ts
const PUBLIC_PATHS = new Set([
  "/",
  "/register",
  "/login",
  "/forgot-password",
  "/reset-password",
  "/verify-email",
]);
```

Menjadi:

```ts
const PUBLIC_PATHS = new Set([
  "/",
  "/register",
  "/login",
  "/forgot-password",
  "/reset-password",
  "/verify-email",
  // Merchant baru perlu mengunduh aplikasi Android bridge sebelum atau
  // sambil membuat akun -- lihat docs/superpowers/specs/2026-09-17-android-bridge-apk-distribution-design.md.
  "/download-app",
]);
```

- [ ] **Step 2: Type-check + lint**

Run: `cd dashboard && npx tsc --noEmit && npx eslint .`
Expected: tidak ada error.

- [ ] **Step 3: Commit**

```bash
git add dashboard/src/proxy.ts
git commit -m "feat(dashboard): kecualikan /download-app dari gerbang sesi"
```

---

### Task 4: Halaman publik `/download-app`

**Files:**
- Create: `dashboard/src/app/download-app/page.tsx`
- Create: `dashboard/public/downloads/.gitkeep`

**Interfaces:**
- Konsumsi: `LandingNavbar` (`@/components/landing/navbar`), `LandingFooter` (`@/components/landing/pricing-faq-footer`) — sudah ada, dipakai persis seperti di `dashboard/src/app/page.tsx`. Tombol unduh memakai `Link` yang distyle langsung (bukan komponen `Button`) — mengikuti pola CTA yang sudah ada di `dashboard/src/components/landing/hero.tsx:44-50`, karena `Button` di proyek ini adalah base-ui (`@base-ui/react/button`, BUKAN Radix) dan tidak punya prop `asChild`; komposisi base-ui pakai prop `render`, tapi untuk sekadar CTA-link-yang-terlihat-seperti-tombol, styling `Link` langsung lebih sederhana dan sudah jadi pola mapan di file ini.
- Konsumsi: route dikecualikan dari sesi oleh Task 3.
- Produces: tombol unduh mengarah ke `/downloads/gopay-bridge.apk` — file itu SENGAJA belum ada (lihat Task 5), jadi tautannya akan 404 sampai Akbar mengunggah APK sungguhan. Ini diketahui dan didokumentasikan, bukan bug.

- [ ] **Step 1: Buat folder `public/downloads/` supaya ter-track git**

```bash
mkdir -p dashboard/public/downloads
touch dashboard/public/downloads/.gitkeep
```

- [ ] **Step 2: Tulis `dashboard/src/app/download-app/page.tsx`**

```tsx
import Link from "next/link";
import { Download, ShieldCheck, Smartphone } from "lucide-react";
import { LandingNavbar } from "@/components/landing/navbar";
import { LandingFooter } from "@/components/landing/pricing-faq-footer";

const STEPS = [
  {
    icon: Download,
    title: "1. Unduh APK",
    desc: "Klik tombol di bawah. Browser akan menandai file ini sebagai \"tidak dikenal\" -- itu wajar untuk aplikasi di luar Play Store.",
  },
  {
    icon: ShieldCheck,
    title: "2. Izinkan pemasangan",
    desc: "Saat memasang, Android akan meminta izin \"Pasang aplikasi tidak dikenal\". Aktifkan untuk browser yang kamu pakai mengunduh tadi.",
  },
  {
    icon: Smartphone,
    title: "3. Izinkan akses notifikasi",
    desc: "Setelah terpasang, buka aplikasinya dan izinkan Notification Access saat diminta -- ini yang dipakai aplikasi membaca notifikasi pembayaran GoPay.",
  },
];

export const metadata = {
  title: "Unduh Aplikasi Android — GoPay Notification Bridge",
};

export default function DownloadAppPage() {
  return (
    <div className="font-(--font-lp-body) flex min-h-screen flex-col">
      <LandingNavbar />
      <main className="flex-1 bg-white py-16 sm:py-20">
        <div className="mx-auto flex max-w-2xl flex-col items-center px-4 text-center">
          <span className="mb-4 flex size-11 items-center justify-center rounded-xl bg-slate-900 text-white ring-1 ring-slate-900/10">
            <Smartphone className="size-5" />
          </span>
          <h1 className="text-2xl font-semibold text-slate-900 sm:text-3xl">
            Unduh Aplikasi Android
          </h1>
          <p className="mt-3 text-sm text-slate-500 sm:text-base">
            Aplikasi ini yang memantau notifikasi GoPay di HP kamu dan melaporkannya secara
            otomatis. Wajib Android 8.0 (API 26) ke atas.
          </p>

          <Link
            href="/downloads/gopay-bridge.apk"
            className="group mt-8 flex h-12 items-center justify-center gap-1.5 rounded-lg bg-slate-900 px-8 text-base font-medium text-white transition-all duration-200 hover:-translate-y-0.5 hover:bg-slate-800 hover:shadow-lg hover:shadow-slate-900/20 active:translate-y-0 active:scale-[0.97]"
          >
            <Download className="size-4" />
            Unduh APK
          </Link>

          <div className="mt-14 grid w-full gap-6 text-left sm:grid-cols-3">
            {STEPS.map((step) => (
              <div key={step.title} className="flex flex-col gap-2">
                <span className="flex size-9 items-center justify-center rounded-lg bg-teal-50 text-teal-700 ring-1 ring-teal-600/20">
                  <step.icon className="size-4" />
                </span>
                <h2 className="text-sm font-semibold text-slate-900">{step.title}</h2>
                <p className="text-sm text-slate-500">{step.desc}</p>
              </div>
            ))}
          </div>
        </div>
      </main>
      <LandingFooter />
    </div>
  );
}
```

- [ ] **Step 3: Type-check, lint, build**

Run: `cd dashboard && npx tsc --noEmit && npx eslint . && npx next build`
Expected: tidak ada error, `/download-app` muncul di daftar route hasil `next build`.

- [ ] **Step 4: Commit**

```bash
git add dashboard/src/app/download-app dashboard/public/downloads/.gitkeep
git commit -m "feat(dashboard): halaman publik unduh aplikasi Android bridge"
```

---

### Task 5: Build sungguhan + verifikasi manual (`NEEDS-DEVICE`, Akbar menjalankan)

**Files:** tidak ada file kode diubah di task ini — murni menjalankan tooling dan mengunggah hasilnya.

**Interfaces:**
- Konsumsi: `mobile/eas.json` (Task 1), `mobile/app.config.ts` (Task 2), `dashboard/public/downloads/` (Task 4).

Task ini TIDAK bisa dijalankan Claude — butuh akun Expo milik Akbar dan HP fisik. Ikuti persis, tempel buktinya (output terminal, screenshot) sebelum menandai selesai.

- [ ] **Step 1: Login/buat akun Expo (sekali saja)**

```bash
cd mobile
npx eas-cli@latest login
```

- [ ] **Step 2: Build APK production**

```bash
npx eas-cli@latest build --platform android --profile production
```

Expected: build selesai dengan status `finished`, EAS mencetak link halaman hasil build (bukan link unduh langsung — itu didapat di step berikutnya).

- [ ] **Step 3: Unduh APK dari halaman hasil build**

Buka link dari Step 2 di browser, klik tombol unduh di halaman itu. Simpan sebagai `gopay-bridge.apk`.

- [ ] **Step 4: Taruh APK di folder yang sudah disiapkan Task 4**

```bash
cp ~/Downloads/gopay-bridge.apk dashboard/public/downloads/gopay-bridge.apk
cd dashboard && npx next build   # pastikan tetap bersih dengan file baru ini
```

(File APK ini besar dan biner — TIDAK di-commit ke git kalau repo dashboard sudah punya `.gitignore` untuk pola serupa; kalau belum, tambahkan `public/downloads/*.apk` ke `dashboard/.gitignore` supaya tidak membengkakkan repo. File ini diupload manual ke server produksi lewat proses deploy biasa, bukan lewat `git push`.)

- [ ] **Step 5: Pasang ke HP yang belum pernah menjalankan Metro/`expo run:android` untuk aplikasi ini**

Transfer `gopay-bridge.apk` ke HP itu (lewat kabel, cloud storage, atau langsung unduh dari `/download-app` setelah deploy — lihat Step 7), pasang.

Expected: aplikasi terbuka LANGSUNG ke layar aplikasi itu sendiri (mis. Settings/Dashboard tab) — BUKAN layar "development client" yang meminta scan QR/connect ke dev server.

- [ ] **Step 6: Alur pairing penuh**

Di HP itu: buka Pengaturan → izinkan Notification Access → "Scan QR dari Dashboard" → pindai QR device baru dari Customer Dashboard (`whuzpay.com`, halaman Devices) → Test Connection.

Expected: Test Connection sukses ("Terhubung sebagai ...").

- [ ] **Step 7: Deploy `dashboard/` ke VPS (langkah U2 di `backend/deploy/README.md`) supaya `/download-app` dan file APK live di produksi**

Ikuti §"Update rutin setelah ada perubahan kode" di `backend/deploy/README.md`, bagian dashboard. Setelah selesai:

```bash
curl -s -o /dev/null -w '%{http_code}\n' https://whuzpay.com/download-app        # mau 200, tanpa cookie sesi
curl -s -o /dev/null -w '%{http_code}\n' https://whuzpay.com/downloads/gopay-bridge.apk   # mau 200
```

- [ ] **Step 8: Update `docs/qa/qa-report.md`**

Tambahkan section baru untuk sub-project ini (ikuti pola section-section sebelumnya: ringkasan PASS/FAIL/NEEDS-DEVICE, tabel butir+bukti). Tempel output curl dari Step 7 dan konfirmasi hasil Step 5-6 sebagai bukti `PASS` (bukan `NEEDS-DEVICE` lagi setelah dilakukan dan dibuktikan).

---

## Self-review (sudah dilakukan penulis plan)

**Cakupan spec:** §4.1 (eas.json) → Task 1. §4.2 (expo-dev-client kondisional) → Task 2. §4.3 (hosting + halaman unduh, termasuk pengecualian dari gerbang sesi) → Task 3+4. §5 (pengujian otomatis) → step type-check/lint/build di tiap task; §5 (pengujian manual NEEDS-DEVICE) → Task 5. §6 (keputusan yang tidak boleh dibalik) tidak butuh task kode — sudah tercatat di spec, plan ini tidak menyentuhnya.

**Placeholder scan:** tidak ada "TBD"/"tulis nanti" — setiap step berisi kode/perintah lengkap yang bisa langsung dijalankan.

**Konsistensi tipe/nama:** `PUBLIC_PATHS` (Task 3) dan path `/download-app` (Task 4) cocok persis. `plugins` sebagai `ExpoConfig['plugins']` (Task 2) konsisten dengan tipe `ExpoConfig` yang sudah diimpor di baris 1 file itu. Nama file APK (`gopay-bridge.apk`) konsisten dipakai di Task 4 (link) dan Task 5 (nama file saat disalin).
