# Android Notification Bridge Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Aplikasi Expo yang menangkap notifikasi GoPay lewat `NotificationListenerService`, menyimpannya ke Room, dan mengirimkannya ke backend dengan tanda tangan HMAC serta retry — semuanya tetap bekerja saat UI aplikasi tertutup.

**Architecture:** Kotlin memiliki seluruh pipeline data (tangkap → simpan → kirim → retry lewat WorkManager). TypeScript hanya UI dan konfigurasi, membaca data lewat Expo local module. React Native tidak pernah membuka database dan tidak pernah mengirim event.

**Tech Stack:** Expo SDK 57 (RN 0.86, React 19) dengan prebuild dan development build, Expo Modules API (Kotlin), Room, WorkManager, OkHttp, EncryptedSharedPreferences, React Navigation.

**Milestone:** M2–M6 dari [spec §8](../specs/2026-09-10-ingestion-and-android-bridge-design.md). Membutuhkan backend dari [plan backend](2026-09-10-backend-ingestion.md) sudah jalan di VPS.

## Global Constraints

- Kontrak API yang mengikat: [`docs/api-contract.md`](../../api-contract.md). Bila plan ini dan kontrak berbeda, kontrak yang menang.
- **Expo Go tidak dapat dipakai.** Seluruh pengujian memakai development build hasil `npx expo run:android`.
- `android/` **tidak** di-commit dan **tidak** diedit manual — seluruh perubahan manifest ditulis sebagai config plugin.
- Package sumber: **`com.gojek.gopaymerchant`**, tepat satu entri. Disimpan sebagai **daftar yang dapat diedit di Settings**, tidak pernah sebagai konstanta di kode. Jangan pernah menambahkan `com.gojek.gopay` — kedua aplikasi melaporkan pembayaran yang sama, dan karena `packageName` ikut jadi bahan `event_id`, satu pembayaran akan terhitung dua kali.
- `event_id = "evt_" + sha256(packageName | title | text | when)[:32]`. `notificationKey` dan `postTime` **tidak** dipakai — alasannya di [spec §4.1](../specs/2026-09-10-ingestion-and-android-bridge-design.md).
- Toleransi timestamp ±300 detik; jam yang meleset harus tampil sebagai pesan berbeda dari kredensial salah.
- Parsing nominal **display-only**. Payload selalu memuat `title` dan `text` mentah.
- Dilarang menulis `device_secret`, tanda tangan, atau isi notifikasi ke Logcat.
- Bahasa UI: Indonesia.
- Perangkat target menjalankan **ColorOS**. Setiap klaim soal ketahanan service wajib diuji di sana.
- **Penyimpanan secret:** [spec §3.3](../specs/2026-09-10-ingestion-and-android-bridge-design.md) menyebut `expo-secure-store`, tetapi `expo-secure-store` hanya dapat dibaca dari JS — sementara `EventUploadWorker` berjalan saat JS mati. Secret karena itu disimpan di **`EncryptedSharedPreferences`** di sisi native, yang memakai Android Keystore yang sama seperti `expo-secure-store` di baliknya. Konsekuensi langsung dari keputusan "native memiliki pipeline data".

---

### Task 1: Scaffold Expo dan development build di perangkat

**Files:**
- Create: `mobile/package.json`, `mobile/app.config.ts`, `mobile/tsconfig.json`
- Create: `mobile/App.tsx`
- Modify: `.gitignore`

**Interfaces:**
- Consumes: tidak ada.
- Produces: aplikasi terpasang di perangkat dengan nama paket `id.akbarryyan.gopaybridge`.

- [ ] **Step 1: Buat project Expo**

```bash
cd /home/akbar/Kerjaan/personal/gopay-notifications
npx create-expo-app@latest mobile --template blank-typescript
cd mobile
npx expo install expo-dev-client react-native-safe-area-context react-native-screens \
  @react-navigation/native @react-navigation/bottom-tabs expo-build-properties
```

- [ ] **Step 2: Tulis `mobile/app.config.ts`**

```ts
import type { ExpoConfig } from 'expo/config'

const IS_DEV = process.env.APP_VARIANT === 'development'

const config: ExpoConfig = {
  name: IS_DEV ? 'GoPay Bridge (Dev)' : 'GoPay Bridge',
  slug: 'gopay-bridge',
  version: '1.0.0',
  orientation: 'portrait',
  userInterfaceStyle: 'automatic',
  android: {
    package: IS_DEV ? 'id.akbarryyan.gopaybridge.dev' : 'id.akbarryyan.gopaybridge',
    permissions: ['android.permission.INTERNET', 'android.permission.ACCESS_NETWORK_STATE'],
  },
  plugins: [
    'expo-dev-client',
    [
      'expo-build-properties',
      {
        android: {
          // Hanya minSdkVersion yang dipatok. compileSdk dan targetSdk
          // dibiarkan mengikuti bawaan Expo SDK 57 — memaksanya ke versi
          // lebih rendah justru merusak pustaka yang menuntut versi baru.
          minSdkVersion: 26,
        },
      },
    ],
  ],
}

export default config
```

`minSdkVersion: 26` dipilih karena `EncryptedSharedPreferences` mensyaratkan API 23+, `requestRebind` mensyaratkan API 24+, dan `java.time` (dipakai untuk `received_at` ISO 8601) mensyaratkan API 26+.

Perangkat target menjalankan **Android 13**, jauh di atas batas itu.

- [ ] **Step 3: Hapus `app.json` bawaan**

```bash
rm -f mobile/app.json
```

Dua sumber konfigurasi sekaligus membuat nilai yang menang sulit ditebak.

- [ ] **Step 4: Tambahkan pola abaikan untuk mobile**

Tambahkan ke `.gitignore` di root repo:

```
mobile/node_modules/
mobile/.expo/
mobile/android/
mobile/ios/
mobile/*.apk
```

- [ ] **Step 5: Prebuild dan pasang ke perangkat**

Sambungkan HP lewat USB dengan USB debugging aktif, lalu:

```bash
cd mobile
adb devices
APP_VARIANT=development npx expo run:android
```

- [ ] **Step 6: Verifikasi aplikasi benar-benar terpasang**

```bash
adb shell pm list packages | grep gopaybridge
```

Expected: `package:id.akbarryyan.gopaybridge.dev`

Layar aplikasi harus terbuka sendiri di HP dan menampilkan teks bawaan template.

- [ ] **Step 7: Commit**

```bash
git add mobile/ .gitignore
git commit -m "feat(mobile): scaffold Expo dengan development build"
```

---

### Task 2: Local module, config plugin, dan status Notification Access

**Files:**
- Create: `mobile/modules/gopay-listener/expo-module.config.json`
- Create: `mobile/modules/gopay-listener/index.ts`
- Create: `mobile/modules/gopay-listener/android/build.gradle`
- Create: `mobile/modules/gopay-listener/android/src/main/java/expo/modules/gopaylistener/GopayListenerModule.kt`
- Create: `mobile/modules/gopay-listener/android/src/main/java/expo/modules/gopaylistener/GoPayListenerService.kt`
- Create: `mobile/plugins/withGopayListener.js`
- Create: `mobile/plugins/withDevCleartext.js`
- Modify: `mobile/app.config.ts`
- Modify: `mobile/App.tsx`

**Interfaces:**
- Consumes: tidak ada.
- Produces:
  - `GopayListener.isNotificationAccessGranted(): boolean`
  - `GopayListener.openNotificationAccessSettings(): void`
  - `GopayListener.isListenerConnected(): boolean`
  - `GoPayListenerService.isConnected` (companion, `@Volatile var`)

- [ ] **Step 1: Buat kerangka local module**

```bash
cd mobile
npx create-expo-module@latest --local gopay-listener
```

Jawab prompt dengan: nama `gopay-listener`, package Android `expo.modules.gopaylistener`, platform Android saja.

- [ ] **Step 2: Tulis `mobile/modules/gopay-listener/expo-module.config.json`**

```json
{
  "platforms": ["android"],
  "android": {
    "modules": ["expo.modules.gopaylistener.GopayListenerModule"]
  }
}
```

- [ ] **Step 3: Tulis `GoPayListenerService.kt`**

```kotlin
package expo.modules.gopaylistener

import android.content.ComponentName
import android.service.notification.NotificationListenerService
import android.service.notification.StatusBarNotification
import android.util.Log

/**
 * Menerima notifikasi dari sistem. Berjalan di proses aplikasi, tetapi
 * lepas dari lifecycle React Native — tetap dipanggil saat UI tertutup.
 */
class GoPayListenerService : NotificationListenerService() {

    companion object {
        private const val TAG = "GoPayListener"

        /**
         * Menandai apakah sistem sedang benar-benar terikat ke service ini.
         * Berbeda dari "izin sudah diberikan": ColorOS dapat mencabut ikatan
         * tanpa mencabut izin, dan selisih itu justru gejala yang perlu terlihat.
         */
        @Volatile
        var isConnected: Boolean = false
            private set

        internal fun setConnected(value: Boolean) {
            isConnected = value
        }
    }

    override fun onListenerConnected() {
        super.onListenerConnected()
        setConnected(true)
        Log.i(TAG, "listener terikat")
    }

    override fun onListenerDisconnected() {
        super.onListenerDisconnected()
        setConnected(false)
        Log.w(TAG, "listener terlepas, meminta rebind")
        requestRebind(ComponentName(this, GoPayListenerService::class.java))
    }

    override fun onNotificationPosted(sbn: StatusBarNotification) {
        // Pipeline penangkapan ditambahkan pada Task 8.
    }
}
```

- [ ] **Step 4: Tulis `GopayListenerModule.kt`**

```kotlin
package expo.modules.gopaylistener

import android.content.Context
import android.content.Intent
import android.provider.Settings
import expo.modules.kotlin.modules.Module
import expo.modules.kotlin.modules.ModuleDefinition

class GopayListenerModule : Module() {

    private val context: Context
        get() = requireNotNull(appContext.reactContext) { "reactContext tidak tersedia" }

    override fun definition() = ModuleDefinition {
        Name("GopayListener")

        Function("isNotificationAccessGranted") {
            val enabled = Settings.Secure.getString(
                context.contentResolver,
                "enabled_notification_listeners"
            ) ?: return@Function false
            enabled.split(":").any { it.startsWith(context.packageName + "/") }
        }

        Function("openNotificationAccessSettings") {
            val intent = Intent(Settings.ACTION_NOTIFICATION_LISTENER_SETTINGS)
            intent.addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
            context.startActivity(intent)
        }

        Function("isListenerConnected") {
            GoPayListenerService.isConnected
        }
    }
}
```

- [ ] **Step 5: Tulis `mobile/modules/gopay-listener/index.ts`**

```ts
import { requireNativeModule } from 'expo-modules-core'

interface GopayListenerModule {
  isNotificationAccessGranted(): boolean
  openNotificationAccessSettings(): void
  isListenerConnected(): boolean
}

export default requireNativeModule<GopayListenerModule>('GopayListener')
```

- [ ] **Step 6: Tulis `mobile/plugins/withGopayListener.js`**

```js
const { AndroidConfig, withAndroidManifest } = require('expo/config-plugins')

const SERVICE_NAME = 'expo.modules.gopaylistener.GoPayListenerService'
const LISTENER_ACTION = 'android.service.notification.NotificationListenerService'

/**
 * Mendaftarkan NotificationListenerService ke AndroidManifest.
 *
 * Ditulis sebagai config plugin, bukan edit manual, karena android/
 * dibuat ulang setiap prebuild.
 */
module.exports = function withGopayListener(config) {
  return withAndroidManifest(config, (config) => {
    const app = AndroidConfig.Manifest.getMainApplicationOrThrow(config.modResults)

    app.service = app.service ?? []
    const already = app.service.some((s) => s.$['android:name'] === SERVICE_NAME)
    if (!already) {
      app.service.push({
        $: {
          'android:name': SERVICE_NAME,
          'android:label': 'GoPay Notification Bridge',
          'android:exported': 'false',
          'android:permission': 'android.permission.BIND_NOTIFICATION_LISTENER_SERVICE',
        },
        'intent-filter': [{ action: [{ $: { 'android:name': LISTENER_ACTION } }] }],
      })
    }

    return config
  })
}
```

- [ ] **Step 7: Tulis `mobile/plugins/withDevCleartext.js`**

```js
const { withAndroidManifest, withDangerousMod, AndroidConfig } = require('expo/config-plugins')
const fs = require('fs')
const path = require('path')

/**
 * Mengizinkan HTTP polos HANYA untuk host development yang disebut eksplisit.
 * Build production tidak memuat plugin ini sama sekali, sehingga tetap HTTPS-only.
 *
 * @param {object} config
 * @param {{ hosts: string[] }} props daftar host development, contoh ['192.168.1.10', '10.0.2.2']
 */
module.exports = function withDevCleartext(config, props) {
  const hosts = props?.hosts ?? []
  if (hosts.length === 0) {
    throw new Error('withDevCleartext: props.hosts wajib berisi minimal satu host')
  }

  config = withDangerousMod(config, [
    'android',
    async (config) => {
      const xmlDir = path.join(config.modRequest.platformProjectRoot, 'app/src/main/res/xml')
      fs.mkdirSync(xmlDir, { recursive: true })

      const domains = hosts
        .map((h) => `        <domain includeSubdomains="true">${h}</domain>`)
        .join('\n')

      const xml = `<?xml version="1.0" encoding="utf-8"?>
<network-security-config>
    <base-config cleartextTrafficPermitted="false" />
    <domain-config cleartextTrafficPermitted="true">
${domains}
    </domain-config>
</network-security-config>
`
      fs.writeFileSync(path.join(xmlDir, 'network_security_config.xml'), xml)
      return config
    },
  ])

  return withAndroidManifest(config, (config) => {
    const app = AndroidConfig.Manifest.getMainApplicationOrThrow(config.modResults)
    app.$['android:networkSecurityConfig'] = '@xml/network_security_config'
    return config
  })
}
```

- [ ] **Step 8: Daftarkan kedua plugin — `mobile/app.config.ts`**

Ganti array `plugins`:

```ts
  plugins: [
    'expo-dev-client',
    './plugins/withGopayListener',
    ...(IS_DEV
      ? [['./plugins/withDevCleartext', { hosts: ['192.168.1.10', '10.0.2.2'] }] as const]
      : []),
    [
      'expo-build-properties',
      {
        android: {
          minSdkVersion: 26,
          compileSdkVersion: 35,
          targetSdkVersion: 35,
        },
      },
    ],
  ],
```

Ganti `192.168.1.10` dengan alamat LAN laptop kamu (`ip addr show | grep 'inet 192'`).

- [ ] **Step 9: Tulis `mobile/App.tsx` sementara untuk memverifikasi jembatan**

```tsx
import { useState } from 'react'
import { Button, SafeAreaView, StyleSheet, Text, View } from 'react-native'
import GopayListener from './modules/gopay-listener'

export default function App() {
  const [granted, setGranted] = useState(GopayListener.isNotificationAccessGranted())
  const [connected, setConnected] = useState(GopayListener.isListenerConnected())

  function refresh() {
    setGranted(GopayListener.isNotificationAccessGranted())
    setConnected(GopayListener.isListenerConnected())
  }

  return (
    <SafeAreaView style={styles.root}>
      <View style={styles.box}>
        <Text style={styles.line}>Notification Access: {granted ? 'aktif' : 'belum aktif'}</Text>
        <Text style={styles.line}>Listener terikat: {connected ? 'ya' : 'tidak'}</Text>
        <Button title="Buka pengaturan" onPress={() => GopayListener.openNotificationAccessSettings()} />
        <Button title="Muat ulang status" onPress={refresh} />
      </View>
    </SafeAreaView>
  )
}

const styles = StyleSheet.create({
  root: { flex: 1, justifyContent: 'center' },
  box: { padding: 24, gap: 12 },
  line: { fontSize: 16 },
})
```

- [ ] **Step 10: Build ulang dan verifikasi manifest**

```bash
cd mobile
APP_VARIANT=development npx expo prebuild --clean
APP_VARIANT=development npx expo run:android
adb shell dumpsys package id.akbarryyan.gopaybridge.dev | grep -A 3 GoPayListenerService
```

Expected: keluaran memuat `expo.modules.gopaylistener.GoPayListenerService` dengan permission `BIND_NOTIFICATION_LISTENER_SERVICE`.

- [ ] **Step 11: `NEEDS-DEVICE` — beri Notification Access dan verifikasi status**

Di HP:
1. Buka aplikasi. Layar harus menampilkan `Notification Access: belum aktif`.
2. Tekan **Buka pengaturan**, aktifkan izin untuk **GoPay Bridge (Dev)**, setujui dialog.
3. Kembali ke aplikasi, tekan **Muat ulang status**.

Expected: `Notification Access: aktif` **dan** `Listener terikat: ya`.

Bila izin aktif tetapi listener tidak terikat, itu gejala ColorOS membunuh service — catat sebagai `FAIL` dan lanjutkan ke Step 12 sebelum menandai task ini selesai.

- [ ] **Step 12: `NEEDS-DEVICE` — setup anti-ColorOS**

Di HP, kerjakan keempat langkah ini:

1. Settings → Baterai → Manajemen baterai aplikasi → **GoPay Bridge (Dev)** → **Izinkan aktivitas latar belakang**, jangan dioptimalkan
2. Settings → Apps → Manajemen aplikasi → **GoPay Bridge (Dev)** → **Izinkan mulai otomatis**
3. Layar recent apps → tahan kartu aplikasi → **kunci** (ikon gembok)
4. Settings → Baterai → **Optimasi siaga tidur** → matikan

Lalu verifikasi ulang Step 11.

- [ ] **Step 13: Commit**

```bash
git add mobile/modules mobile/plugins mobile/app.config.ts mobile/App.tsx
git commit -m "feat(mobile): local module, config plugin, dan status Notification Access"
```

---

### Task 3: Parser nominal

**Files:**
- Create: `mobile/modules/gopay-listener/android/src/main/java/expo/modules/gopaylistener/AmountParser.kt`
- Create: `mobile/modules/gopay-listener/android/src/test/java/expo/modules/gopaylistener/AmountParserTest.kt`
- Modify: `mobile/modules/gopay-listener/android/build.gradle`

**Interfaces:**
- Consumes: tidak ada.
- Produces: `AmountParser.parse(text: String?): Long?`

- [ ] **Step 1: Tambahkan dependensi test — `mobile/modules/gopay-listener/android/build.gradle`**

Tambahkan di blok `dependencies`:

```gradle
  testImplementation 'junit:junit:4.13.2'
```

- [ ] **Step 2: Tulis test yang gagal — `AmountParserTest.kt`**

```kotlin
package expo.modules.gopaylistener

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test

class AmountParserTest {

    @Test
    fun `mengurai notifikasi transfer masuk sungguhan`() {
        assertEquals(
            1L,
            AmountParser.parse("Rp1 dari icaangg udah masuk ke GoPay kamu.")
        )
    }

    @Test
    fun `mengurai pemisah ribuan bergaya Indonesia`() {
        assertEquals(25_000L, AmountParser.parse("Pembayaran diterima Rp25.000"))
        assertEquals(1_234_567L, AmountParser.parse("Saldo kamu Rp1.234.567"))
    }

    @Test
    fun `mengurai varian spasi dan koma`() {
        assertEquals(25_000L, AmountParser.parse("Rp 25.000"))
        assertEquals(25_000L, AmountParser.parse("Rp25,000"))
    }

    @Test
    fun `mengabaikan sen di belakang koma`() {
        assertEquals(25_000L, AmountParser.parse("Rp25.000,50"))
    }

    @Test
    fun `mengambil kemunculan pertama bila ada lebih dari satu`() {
        assertEquals(25_000L, AmountParser.parse("Rp25.000 dari Rp50.000"))
    }

    @Test
    fun `menghasilkan null bila tidak ada Rp`() {
        assertNull(AmountParser.parse("Transfer masuk"))
        assertNull(AmountParser.parse("Ref 12345678"))
        assertNull(AmountParser.parse("Cashback 50% untuk kamu"))
    }

    @Test
    fun `menghasilkan null untuk masukan kosong atau null`() {
        assertNull(AmountParser.parse(null))
        assertNull(AmountParser.parse(""))
        assertNull(AmountParser.parse("Rp"))
        assertNull(AmountParser.parse("Rp "))
    }

    @Test
    fun `tidak memperlakukan angka biasa sebagai nominal`() {
        assertNull(AmountParser.parse("Diskon 25.000 poin menanti"))
        assertNull(AmountParser.parse("Kode OTP kamu 123456"))
    }
}
```

- [ ] **Step 3: Cari nama gradle project modul lalu jalankan test**

```bash
cd mobile/android
./gradlew projects | grep -i gopay
```

Catat nama project yang muncul (biasanya `:gopay-listener`), lalu:

```bash
./gradlew :gopay-listener:testDebugUnitTest --tests '*AmountParserTest*'
```

Expected: FAIL — `Unresolved reference: AmountParser`.

- [ ] **Step 4: Tulis `AmountParser.kt`**

```kotlin
package expo.modules.gopaylistener

/**
 * Mengekstrak nominal dari teks notifikasi.
 *
 * Hasilnya bersifat DISPLAY-ONLY. Backend melakukan ekstraksi otoritatifnya
 * sendiri dari teks mentah; parser ini boleh salah tanpa merusak apa pun.
 *
 * Prinsipnya: jangan pernah menebak. Angka yang tidak didahului "Rp"
 * bukan nominal.
 */
object AmountParser {

    // Hanya menerima pengelompokan yang berbentuk wajar:
    // "1", "25.000", "1.234.567", "25,000".
    // Rangkaian seperti "25.00" atau "1.2345" tidak cocok, sehingga
    // angka yang kebetulan berdekatan dengan "Rp" tidak ikut terambil.
    private val PATTERN = Regex("""Rp\s?(\d{1,3}(?:[.,]\d{3})*|\d+)""")

    fun parse(text: String?): Long? {
        if (text.isNullOrEmpty()) return null

        val match = PATTERN.find(text) ?: return null
        val digits = match.groupValues[1].replace(".", "").replace(",", "")
        if (digits.isEmpty()) return null

        return digits.toLongOrNull()
    }
}
```

- [ ] **Step 5: Jalankan test, pastikan lulus**

```bash
cd mobile/android && ./gradlew :gopay-listener:testDebugUnitTest --tests '*AmountParserTest*'
```

Expected: PASS — delapan test.

- [ ] **Step 6: Commit**

```bash
git add mobile/modules/gopay-listener/android/
git commit -m "feat(mobile): parser nominal display-only"
```

---

### Task 4: Pembentuk event_id

**Files:**
- Create: `.../expo/modules/gopaylistener/EventIdBuilder.kt`
- Create: `.../expo/modules/gopaylistener/EventIdBuilderTest.kt` (di `src/test`)

**Interfaces:**
- Consumes: tidak ada.
- Produces: `EventIdBuilder.build(packageName: String, title: String?, text: String?, whenMs: Long): String`

- [ ] **Step 1: Tulis test yang gagal — `EventIdBuilderTest.kt`**

```kotlin
package expo.modules.gopaylistener

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotEquals
import org.junit.Assert.assertTrue
import org.junit.Test

class EventIdBuilderTest {

    private val pkg = "com.gojek.gopaymerchant"
    private val title = "Transfer masuk"
    private val text = "Rp1 dari icaangg udah masuk ke GoPay kamu."
    private val whenMs = 1_789_051_832_829L

    @Test
    fun `berbentuk evt_ diikuti 32 heksadesimal`() {
        val id = EventIdBuilder.build(pkg, title, text, whenMs)
        assertTrue("id = $id", Regex("^evt_[0-9a-f]{32}$").matches(id))
    }

    @Test
    fun `deterministik untuk masukan yang sama`() {
        assertEquals(
            EventIdBuilder.build(pkg, title, text, whenMs),
            EventIdBuilder.build(pkg, title, text, whenMs)
        )
    }

    @Test
    fun `berubah bila satu bahan berubah`() {
        val base = EventIdBuilder.build(pkg, title, text, whenMs)

        assertNotEquals(base, EventIdBuilder.build("com.gojek.app", title, text, whenMs))
        assertNotEquals(base, EventIdBuilder.build(pkg, "Transfer keluar", text, whenMs))
        assertNotEquals(base, EventIdBuilder.build(pkg, title, text + " ", whenMs))
        assertNotEquals(base, EventIdBuilder.build(pkg, title, text, whenMs + 1))
    }

    @Test
    fun `menangani title dan text null tanpa menyamakannya dengan string kosong`() {
        val nullTitle = EventIdBuilder.build(pkg, null, text, whenMs)
        val emptyTitle = EventIdBuilder.build(pkg, "", text, whenMs)

        assertTrue(Regex("^evt_[0-9a-f]{32}$").matches(nullTitle))
        assertEquals(
            "null dan string kosong sengaja diperlakukan sama — keduanya berarti tidak ada judul",
            nullTitle,
            emptyTitle
        )
    }

    @Test
    fun `dua transfer identik pada waktu berbeda menghasilkan id berbeda`() {
        val a = EventIdBuilder.build(pkg, title, text, whenMs)
        val b = EventIdBuilder.build(pkg, title, text, whenMs + 1000)
        assertNotEquals("dua transfer sungguhan tidak boleh dianggap satu", a, b)
    }
}
```

- [ ] **Step 2: Jalankan test, pastikan gagal**

```bash
cd mobile/android && ./gradlew :gopay-listener:testDebugUnitTest --tests '*EventIdBuilderTest*'
```

Expected: FAIL — `Unresolved reference: EventIdBuilder`.

- [ ] **Step 3: Tulis `EventIdBuilder.kt`**

```kotlin
package expo.modules.gopaylistener

import java.security.MessageDigest

/**
 * Membentuk event_id yang deterministik.
 *
 * Bahan sengaja TIDAK memuat:
 *   - notificationKey — GoPay memakai id -1 tanpa tag, sehingga key-nya
 *     identik untuk semua notifikasinya dan tidak menyumbang apa pun.
 *   - postTime — berubah setiap notifikasi di-posting ulang, sehingga satu
 *     transfer yang di-repost akan terkirim sebagai dua event.
 *
 * whenMs berasal dari Notification.when, yang bertahan lintas repost.
 */
object EventIdBuilder {

    fun build(packageName: String, title: String?, text: String?, whenMs: Long): String {
        val raw = listOf(
            packageName,
            title.orEmpty(),
            text.orEmpty(),
            whenMs.toString()
        ).joinToString("|")

        val digest = MessageDigest.getInstance("SHA-256")
            .digest(raw.toByteArray(Charsets.UTF_8))

        val hex = digest.joinToString("") { "%02x".format(it) }
        return "evt_" + hex.take(32)
    }
}
```

- [ ] **Step 4: Jalankan test, pastikan lulus**

```bash
cd mobile/android && ./gradlew :gopay-listener:testDebugUnitTest --tests '*EventIdBuilderTest*'
```

Expected: PASS — lima test.

- [ ] **Step 5: Commit**

```bash
git add mobile/modules/gopay-listener/android/
git commit -m "feat(mobile): pembentuk event_id deterministik"
```

---

### Task 5: Penanda tangan HMAC, dicocokkan dengan backend

**Files:**
- Create: `.../expo/modules/gopaylistener/Signer.kt`
- Create: `.../expo/modules/gopaylistener/SignerTest.kt` (di `src/test`)

**Interfaces:**
- Consumes: tidak ada.
- Produces: `Signer.signingString(deviceId: String, timestamp: Long, body: ByteArray): String`, `Signer.sign(secret: String, signingString: String): String`

Task ini punya bahaya khusus: dua implementasi HMAC di dua bahasa yang *terlihat* benar tetapi tidak sepakat akan gagal hanya saat request sungguhan, dengan pesan `invalid_signature` yang tidak menjelaskan apa pun. Karena itu vektor ujinya diambil dari implementasi Go, bukan dikarang.

- [ ] **Step 1: Hasilkan vektor uji dari implementasi Go**

```bash
cd backend
cat > /tmp/vector.go <<'EOF'
package main

import (
	"fmt"

	"github.com/akbarryyan/gopay-notifications/backend/internal/auth"
)

func main() {
	body := []byte(`{"event_id":"evt_abc"}`)
	s := auth.SigningString("dev_01ABC", 1789036200, body)
	fmt.Printf("signing_string = %q\n", s)
	fmt.Printf("signature      = %s\n", auth.Sign([]byte("secret-untuk-test"), s))
}
EOF
go run /tmp/vector.go
```

Catat kedua nilai yang tercetak. Nilai `signature` dipakai di Step 2.

- [ ] **Step 2: Tulis test yang gagal — `SignerTest.kt`**

Ganti `GANTI_DENGAN_HASIL_STEP_1` dengan nilai `signature` dari Step 1.

```kotlin
package expo.modules.gopaylistener

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotEquals
import org.junit.Test

class SignerTest {

    private val secret = "secret-untuk-test"

    @Test
    fun `signing string untuk body kosong cocok dengan backend`() {
        // sha256("") = e3b0c442...b855
        assertEquals(
            "dev_01ABC\n1789036200\n" +
                "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
            Signer.signingString("dev_01ABC", 1789036200L, ByteArray(0))
        )
    }

    @Test
    fun `tanda tangan cocok dengan vektor dari implementasi Go`() {
        val body = """{"event_id":"evt_abc"}""".toByteArray(Charsets.UTF_8)
        val s = Signer.signingString("dev_01ABC", 1789036200L, body)

        assertEquals("GANTI_DENGAN_HASIL_STEP_1", Signer.sign(secret, s))
    }

    @Test
    fun `body yang berubah satu byte menghasilkan tanda tangan berbeda`() {
        val a = Signer.sign(secret, Signer.signingString("dev_01ABC", 1789036200L, "{\"a\":1}".toByteArray()))
        val b = Signer.sign(secret, Signer.signingString("dev_01ABC", 1789036200L, "{\"a\":2}".toByteArray()))
        assertNotEquals(a, b)
    }

    @Test
    fun `secret yang berbeda menghasilkan tanda tangan berbeda`() {
        val s = Signer.signingString("dev_01ABC", 1789036200L, ByteArray(0))
        assertNotEquals(Signer.sign(secret, s), Signer.sign("secret-lain", s))
    }

    @Test
    fun `tanda tangan berupa 64 karakter heksadesimal huruf kecil`() {
        val sig = Signer.sign(secret, Signer.signingString("dev_01ABC", 1789036200L, ByteArray(0)))
        assertEquals(64, sig.length)
        assertEquals(sig, sig.lowercase())
    }
}
```

- [ ] **Step 3: Jalankan test, pastikan gagal**

```bash
cd mobile/android && ./gradlew :gopay-listener:testDebugUnitTest --tests '*SignerTest*'
```

Expected: FAIL — `Unresolved reference: Signer`.

- [ ] **Step 4: Tulis `Signer.kt`**

```kotlin
package expo.modules.gopaylistener

import java.security.MessageDigest
import javax.crypto.Mac
import javax.crypto.spec.SecretKeySpec

/**
 * Menandatangani request ke backend, sesuai api-contract.md §3.2.
 *
 * Yang di-hash adalah byte mentah body yang benar-benar dikirim.
 * Jangan pernah menyusun ulang JSON setelah menandatanganinya.
 */
object Signer {

    fun signingString(deviceId: String, timestamp: Long, body: ByteArray): String {
        val sha = MessageDigest.getInstance("SHA-256").digest(body)
        return deviceId + "\n" + timestamp + "\n" + sha.toHex()
    }

    fun sign(secret: String, signingString: String): String {
        val mac = Mac.getInstance("HmacSHA256")
        mac.init(SecretKeySpec(secret.toByteArray(Charsets.UTF_8), "HmacSHA256"))
        return mac.doFinal(signingString.toByteArray(Charsets.UTF_8)).toHex()
    }

    private fun ByteArray.toHex(): String = joinToString("") { "%02x".format(it) }
}
```

- [ ] **Step 5: Jalankan test, pastikan lulus**

```bash
cd mobile/android && ./gradlew :gopay-listener:testDebugUnitTest --tests '*SignerTest*'
```

Expected: PASS — lima test, termasuk kecocokan dengan vektor Go.

- [ ] **Step 6: Bersihkan berkas sementara dan commit**

```bash
rm -f /tmp/vector.go
git add mobile/modules/gopay-listener/android/
git commit -m "feat(mobile): penanda tangan HMAC, dicocokkan dengan vektor backend"
```

---

### Task 6: Database Room

**Files:**
- Create: `.../gopaylistener/db/EventEntity.kt`
- Create: `.../gopaylistener/db/EventDao.kt`
- Create: `.../gopaylistener/db/AppDatabase.kt`
- Create: `.../gopaylistener/db/EventDaoTest.kt` (di `src/test`)
- Modify: `mobile/modules/gopay-listener/android/build.gradle`

**Interfaces:**
- Consumes: tidak ada.
- Produces:
  - `EventStatus` (enum: `PENDING`, `SENDING`, `SENT`, `FAILED`, `IGNORED`)
  - `EventEntity` (data class, primary key `eventId`)
  - `EventDao.insertIgnoringDuplicate(e: EventEntity): Long` — mengembalikan `-1` bila `eventId` sudah ada
  - `EventDao.claimPending(limit: Int): List<EventEntity>`
  - `EventDao.markSent(eventId: String, backendStatus: String, sentAt: Long)`
  - `EventDao.markFailed(eventId: String, error: String)`
  - `EventDao.rescheduleForRetry(eventId: String, error: String)`
  - `EventDao.recent(limit: Int): List<EventEntity>`
  - `EventDao.countByStatus(status: EventStatus): Int`
  - `EventDao.purgeOlderThan(cutoffMs: Long): Int`
  - `AppDatabase.get(context: Context): AppDatabase`

- [ ] **Step 1: Tambahkan dependensi — `mobile/modules/gopay-listener/android/build.gradle`**

Di paling atas berkas, setelah plugin yang sudah ada:

```gradle
apply plugin: 'kotlin-kapt'
```

Di blok `dependencies`:

```gradle
  // Room 2.8.x wajib, bukan 2.6.x. Compiler 2.6.1 hanya memahami metadata
  // Kotlin sampai 2.0.0, sementara project ini memakai Kotlin 2.1.20 yang
  // menghasilkan metadata 2.1.0 — kapt gagal dengan
  // "Provided Metadata instance has version 2.1.0, while maximum supported
  // version is 2.0.0". Jangan turunkan versinya.
  implementation 'androidx.room:room-runtime:2.8.5'
  implementation 'androidx.room:room-ktx:2.8.5'
  kapt 'androidx.room:room-compiler:2.8.5'

  testImplementation 'junit:junit:4.13.2'
  testImplementation 'org.robolectric:robolectric:4.12.2'
  testImplementation 'androidx.test:core:1.5.0'
```

Dan di blok `android`:

```gradle
  testOptions {
    unitTests {
      includeAndroidResources = true
    }
  }
```

- [ ] **Step 2: Tulis test yang gagal — `EventDaoTest.kt`**

```kotlin
package expo.modules.gopaylistener.db

import androidx.room.Room
import androidx.test.core.app.ApplicationProvider
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotEquals
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner

@RunWith(RobolectricTestRunner::class)
class EventDaoTest {

    private lateinit var db: AppDatabase
    private lateinit var dao: EventDao

    @Before
    fun setUp() {
        db = Room.inMemoryDatabaseBuilder(
            ApplicationProvider.getApplicationContext(),
            AppDatabase::class.java
        ).allowMainThreadQueries().build()
        dao = db.events()
    }

    @After
    fun tearDown() = db.close()

    private fun event(id: String, status: EventStatus = EventStatus.PENDING, receivedAt: Long = 1000L) =
        EventEntity(
            eventId = id,
            packageName = "com.gojek.gopaymerchant",
            title = "Transfer masuk",
            text = "Rp1 dari icaangg udah masuk ke GoPay kamu.",
            bigText = null,
            amountHint = 1L,
            postedAt = 1_789_051_832_829L,
            receivedAt = receivedAt,
            status = status,
            attemptCount = 0,
            lastError = null,
            backendStatus = null,
            sentAt = null
        )

    @Test
    fun `menyisipkan event baru`() {
        assertNotEquals(-1L, dao.insertIgnoringDuplicate(event("evt_a")))
        assertEquals(1, dao.countByStatus(EventStatus.PENDING))
    }

    @Test
    fun `menolak event_id yang sama tanpa melempar exception`() {
        dao.insertIgnoringDuplicate(event("evt_a"))
        val second = dao.insertIgnoringDuplicate(event("evt_a"))

        assertEquals("penyisipan kedua harus dilaporkan sebagai duplikat", -1L, second)
        assertEquals(1, dao.countByStatus(EventStatus.PENDING))
    }

    @Test
    fun `claimPending memindahkan status ke SENDING`() {
        dao.insertIgnoringDuplicate(event("evt_a"))
        dao.insertIgnoringDuplicate(event("evt_b"))

        val claimed = dao.claimPending(10)

        assertEquals(2, claimed.size)
        assertEquals(0, dao.countByStatus(EventStatus.PENDING))
        assertEquals(2, dao.countByStatus(EventStatus.SENDING))
    }

    @Test
    fun `claimPending tidak mengambil event IGNORED atau SENT`() {
        dao.insertIgnoringDuplicate(event("evt_ignored", EventStatus.IGNORED))
        dao.insertIgnoringDuplicate(event("evt_sent", EventStatus.SENT))
        dao.insertIgnoringDuplicate(event("evt_pending"))

        val claimed = dao.claimPending(10)

        assertEquals(1, claimed.size)
        assertEquals("evt_pending", claimed[0].eventId)
    }

    @Test
    fun `markSent mengisi backendStatus dan sentAt`() {
        dao.insertIgnoringDuplicate(event("evt_a"))
        dao.claimPending(10)

        dao.markSent("evt_a", "accepted", 5000L)

        val got = dao.recent(10).single()
        assertEquals(EventStatus.SENT, got.status)
        assertEquals("accepted", got.backendStatus)
        assertEquals(5000L, got.sentAt)
    }

    @Test
    fun `rescheduleForRetry mengembalikan ke PENDING dan menaikkan attemptCount`() {
        dao.insertIgnoringDuplicate(event("evt_a"))
        dao.claimPending(10)

        dao.rescheduleForRetry("evt_a", "timeout")
        dao.claimPending(10)
        dao.rescheduleForRetry("evt_a", "timeout")

        val got = dao.recent(10).single()
        assertEquals(EventStatus.PENDING, got.status)
        assertEquals(2, got.attemptCount)
        assertEquals("timeout", got.lastError)
    }

    @Test
    fun `markFailed menghentikan event dari antrean`() {
        dao.insertIgnoringDuplicate(event("evt_a"))
        dao.claimPending(10)

        dao.markFailed("evt_a", "invalid_signature")

        assertEquals(1, dao.countByStatus(EventStatus.FAILED))
        assertTrue(dao.claimPending(10).isEmpty())
    }

    @Test
    fun `recent mengembalikan yang terbaru lebih dulu`() {
        dao.insertIgnoringDuplicate(event("evt_lama", receivedAt = 1000L))
        dao.insertIgnoringDuplicate(event("evt_baru", receivedAt = 2000L))

        assertEquals("evt_baru", dao.recent(10)[0].eventId)
    }

    @Test
    fun `purgeOlderThan hanya menghapus SENT dan IGNORED`() {
        dao.insertIgnoringDuplicate(event("evt_sent", EventStatus.SENT, receivedAt = 100L))
        dao.insertIgnoringDuplicate(event("evt_ignored", EventStatus.IGNORED, receivedAt = 100L))
        dao.insertIgnoringDuplicate(event("evt_failed", EventStatus.FAILED, receivedAt = 100L))
        dao.insertIgnoringDuplicate(event("evt_baru", EventStatus.SENT, receivedAt = 9999L))

        val deleted = dao.purgeOlderThan(1000L)

        assertEquals(2, deleted)
        val sisa = dao.recent(10).map { it.eventId }.toSet()
        assertEquals(setOf("evt_failed", "evt_baru"), sisa)
    }
}
```

- [ ] **Step 3: Jalankan test, pastikan gagal**

```bash
cd mobile/android && ./gradlew :gopay-listener:testDebugUnitTest --tests '*EventDaoTest*'
```

Expected: FAIL — `Unresolved reference: AppDatabase`.

- [ ] **Step 4: Tulis `EventEntity.kt`**

```kotlin
package expo.modules.gopaylistener.db

import androidx.room.Entity
import androidx.room.PrimaryKey

enum class EventStatus {
    PENDING,
    SENDING,
    SENT,
    FAILED,
    IGNORED,
}

@Entity(tableName = "events")
data class EventEntity(
    @PrimaryKey val eventId: String,
    val packageName: String,
    val title: String?,
    val text: String?,
    val bigText: String?,
    /** Display-only. Backend melakukan ekstraksi otoritatifnya sendiri. */
    val amountHint: Long?,
    /** Notification.when, atau postTime bila when bernilai 0. */
    val postedAt: Long,
    val receivedAt: Long,
    val status: EventStatus,
    val attemptCount: Int,
    /** Pesan singkat. Tidak boleh memuat secret atau tanda tangan. */
    val lastError: String?,
    val backendStatus: String?,
    val sentAt: Long?,
)
```

- [ ] **Step 5: Tulis `EventDao.kt`**

```kotlin
package expo.modules.gopaylistener.db

import androidx.room.Dao
import androidx.room.Insert
import androidx.room.OnConflictStrategy
import androidx.room.Query
import androidx.room.Transaction

@Dao
interface EventDao {

    /**
     * Menyisipkan event. Mengembalikan -1 bila eventId sudah ada.
     *
     * Pencegahan duplikat bersandar pada primary key, bukan pada pemeriksaan
     * lebih dulu — Android dapat memanggil onNotificationPosted berkali-kali
     * untuk notifikasi yang sama, kadang beriringan.
     */
    @Insert(onConflict = OnConflictStrategy.IGNORE)
    fun insertIgnoringDuplicate(event: EventEntity): Long

    @Transaction
    fun claimPending(limit: Int): List<EventEntity> {
        val batch = selectPending(limit)
        batch.forEach { setStatus(it.eventId, EventStatus.SENDING) }
        return batch
    }

    @Query("SELECT * FROM events WHERE status = 'PENDING' ORDER BY receivedAt ASC LIMIT :limit")
    fun selectPending(limit: Int): List<EventEntity>

    @Query("UPDATE events SET status = :status WHERE eventId = :eventId")
    fun setStatus(eventId: String, status: EventStatus)

    @Query(
        """UPDATE events
           SET status = 'SENT', backendStatus = :backendStatus, sentAt = :sentAt, lastError = NULL
           WHERE eventId = :eventId"""
    )
    fun markSent(eventId: String, backendStatus: String, sentAt: Long)

    @Query("UPDATE events SET status = 'FAILED', lastError = :error WHERE eventId = :eventId")
    fun markFailed(eventId: String, error: String)

    @Query(
        """UPDATE events
           SET status = 'PENDING', attemptCount = attemptCount + 1, lastError = :error
           WHERE eventId = :eventId"""
    )
    fun rescheduleForRetry(eventId: String, error: String)

    @Query("SELECT * FROM events ORDER BY receivedAt DESC LIMIT :limit")
    fun recent(limit: Int): List<EventEntity>

    @Query("SELECT * FROM events WHERE status != 'IGNORED' ORDER BY receivedAt DESC LIMIT 1")
    fun latest(): EventEntity?

    @Query("SELECT COUNT(*) FROM events WHERE status = :status")
    fun countByStatus(status: EventStatus): Int

    /** Hanya SENT dan IGNORED yang didaur ulang. FAILED tidak pernah dihapus otomatis. */
    @Query("DELETE FROM events WHERE receivedAt < :cutoffMs AND status IN ('SENT', 'IGNORED')")
    fun purgeOlderThan(cutoffMs: Long): Int
}
```

- [ ] **Step 6: Tulis `AppDatabase.kt`**

```kotlin
package expo.modules.gopaylistener.db

import android.content.Context
import androidx.room.Database
import androidx.room.Room
import androidx.room.RoomDatabase

@Database(entities = [EventEntity::class], version = 1, exportSchema = false)
abstract class AppDatabase : RoomDatabase() {

    abstract fun events(): EventDao

    companion object {
        @Volatile
        private var instance: AppDatabase? = null

        /**
         * Database dimiliki lapisan native. React Native tidak pernah membuka
         * berkas ini — dua penulis ke satu berkas adalah sumber bug.
         */
        fun get(context: Context): AppDatabase =
            instance ?: synchronized(this) {
                instance ?: Room.databaseBuilder(
                    context.applicationContext,
                    AppDatabase::class.java,
                    "gopay-bridge.db"
                ).build().also { instance = it }
            }
    }
}
```

- [ ] **Step 7: Jalankan test, pastikan lulus**

```bash
cd mobile/android && ./gradlew :gopay-listener:testDebugUnitTest --tests '*EventDaoTest*'
```

Expected: PASS — sembilan test.

- [ ] **Step 8: Commit**

```bash
git add mobile/modules/gopay-listener/android/
git commit -m "feat(mobile): database Room untuk antrean event"
```

---
### Task 7: Konfigurasi terenkripsi dan penyimpanan Discovery

**Files:**
- Create: `.../gopaylistener/config/Settings.kt`
- Create: `.../gopaylistener/discovery/DiscoveryLog.kt`
- Create: `.../gopaylistener/config/SettingsTest.kt` (di `src/test`)
- Modify: `mobile/modules/gopay-listener/android/build.gradle`
- Modify: `GopayListenerModule.kt`
- Modify: `mobile/modules/gopay-listener/index.ts`

**Interfaces:**
- Consumes: tidak ada.
- Produces:
  - `data class BridgeConfig(val backendUrl: String, val deviceId: String, val deviceSecret: String)`
  - `Settings(context)` dengan properti `backendUrl`, `deviceId`, `deviceSecret`, `monitoredPackages: List<String>`, `ignoreKeywords: List<String>`, `discoveryUntilMs: Long`
  - `Settings.config(): BridgeConfig?` — `null` bila salah satu dari tiga nilai wajib masih kosong
  - `Settings.discoveryActive(nowMs: Long): Boolean`
  - `DiscoveryLog.record(context, packageName, title)`, `DiscoveryLog.entries(context): List<DiscoveryEntry>`, `DiscoveryLog.clear(context)`
  - Fungsi module: `getSettings()`, `saveSettings(object)`, `startDiscovery(minutes)`, `stopDiscovery()`, `getDiscoveryEntries()`, `clearDiscovery()`

- [ ] **Step 1: Tambahkan dependensi — `build.gradle`**

```gradle
  implementation 'androidx.security:security-crypto:1.0.0'
```

- [ ] **Step 2: Tulis test yang gagal — `SettingsTest.kt`**

```kotlin
package expo.modules.gopaylistener.config

import androidx.test.core.app.ApplicationProvider
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner

@RunWith(RobolectricTestRunner::class)
class SettingsTest {

    private lateinit var settings: Settings

    @Before
    fun setUp() {
        settings = Settings(ApplicationProvider.getApplicationContext())
        settings.clearAll()
    }

    @Test
    fun `daftar package default memuat GoPay`() {
        assertEquals(listOf("com.gojek.gopaymerchant"), settings.monitoredPackages)
    }

    @Test
    fun `daftar kata-diabaikan default kosong`() {
        assertTrue(
            "default harus kosong — keamanan dijamin allowlist di backend, bukan filter di HP",
            settings.ignoreKeywords.isEmpty()
        )
    }

    @Test
    fun `config null selama salah satu nilai wajib kosong`() {
        assertNull(settings.config())

        settings.backendUrl = "https://contoh.test/api/v1"
        assertNull(settings.config())

        settings.deviceId = "dev_01ABC"
        assertNull(settings.config())

        settings.deviceSecret = "rahasia"
        assertNotNull(settings.config())
    }

    @Test
    fun `config mengembalikan nilai yang tersimpan`() {
        settings.backendUrl = "https://contoh.test/api/v1"
        settings.deviceId = "dev_01ABC"
        settings.deviceSecret = "rahasia"

        val cfg = settings.config()!!
        assertEquals("https://contoh.test/api/v1", cfg.backendUrl)
        assertEquals("dev_01ABC", cfg.deviceId)
        assertEquals("rahasia", cfg.deviceSecret)
    }

    @Test
    fun `backendUrl dinormalkan tanpa garis miring di belakang`() {
        settings.backendUrl = "https://contoh.test/api/v1/"
        assertEquals("https://contoh.test/api/v1", settings.backendUrl)
    }

    @Test
    fun `daftar package dapat diubah`() {
        settings.monitoredPackages = listOf("com.gojek.gopaymerchant", "com.example.sumberlain")
        assertEquals(listOf("com.gojek.gopaymerchant", "com.example.sumberlain"), settings.monitoredPackages)
    }

    @Test
    fun `discovery mati secara default`() {
        assertFalse(settings.discoveryActive(nowMs = 1_000L))
    }

    @Test
    fun `discovery aktif hanya sampai batas waktunya`() {
        settings.discoveryUntilMs = 10_000L

        assertTrue(settings.discoveryActive(nowMs = 9_999L))
        assertFalse("discovery harus mati sendiri setelah lewat batas", settings.discoveryActive(nowMs = 10_001L))
    }
}
```

- [ ] **Step 3: Jalankan test, pastikan gagal**

```bash
cd mobile/android && ./gradlew :gopay-listener:testDebugUnitTest --tests '*SettingsTest*'
```

Expected: FAIL — `Unresolved reference: Settings`.

- [ ] **Step 4: Tulis `Settings.kt`**

```kotlin
package expo.modules.gopaylistener.config

import android.content.Context
import android.content.SharedPreferences
import androidx.security.crypto.EncryptedSharedPreferences
import androidx.security.crypto.MasterKeys

data class BridgeConfig(
    val backendUrl: String,
    val deviceId: String,
    val deviceSecret: String,
)

/**
 * Konfigurasi bridge.
 *
 * Disimpan di sisi native, bukan lewat expo-secure-store, karena
 * EventUploadWorker membacanya saat runtime JavaScript sudah mati.
 * Mekanisme di baliknya sama: Android Keystore.
 */
class Settings(context: Context) {

    private val prefs: SharedPreferences = EncryptedSharedPreferences.create(
        PREFS_NAME,
        MasterKeys.getOrCreate(MasterKeys.AES256_GCM_SPEC),
        context.applicationContext,
        EncryptedSharedPreferences.PrefKeyEncryptionScheme.AES256_SIV,
        EncryptedSharedPreferences.PrefValueEncryptionScheme.AES256_GCM,
    )

    var backendUrl: String
        get() = prefs.getString(KEY_BACKEND_URL, "").orEmpty()
        set(value) = prefs.edit().putString(KEY_BACKEND_URL, value.trim().trimEnd('/')).apply()

    var deviceId: String
        get() = prefs.getString(KEY_DEVICE_ID, "").orEmpty()
        set(value) = prefs.edit().putString(KEY_DEVICE_ID, value.trim()).apply()

    var deviceSecret: String
        get() = prefs.getString(KEY_DEVICE_SECRET, "").orEmpty()
        set(value) = prefs.edit().putString(KEY_DEVICE_SECRET, value.trim()).apply()

    var monitoredPackages: List<String>
        get() = prefs.getString(KEY_PACKAGES, DEFAULT_PACKAGES)
            .orEmpty().split(",").map { it.trim() }.filter { it.isNotEmpty() }
        set(value) = prefs.edit().putString(KEY_PACKAGES, value.joinToString(",")).apply()

    /**
     * Default sengaja kosong. Keamanan dijamin allowlist judul di backend;
     * daftar ini hanya pengurang noise. Bila ragu, lebih baik tetap kirim.
     */
    var ignoreKeywords: List<String>
        get() = prefs.getString(KEY_IGNORE, "")
            .orEmpty().split(",").map { it.trim() }.filter { it.isNotEmpty() }
        set(value) = prefs.edit().putString(KEY_IGNORE, value.joinToString(",")).apply()

    var discoveryUntilMs: Long
        get() = prefs.getLong(KEY_DISCOVERY_UNTIL, 0L)
        set(value) = prefs.edit().putLong(KEY_DISCOVERY_UNTIL, value).apply()

    fun discoveryActive(nowMs: Long = System.currentTimeMillis()): Boolean =
        nowMs < discoveryUntilMs

    fun config(): BridgeConfig? {
        val url = backendUrl
        val id = deviceId
        val secret = deviceSecret
        if (url.isEmpty() || id.isEmpty() || secret.isEmpty()) return null
        return BridgeConfig(url, id, secret)
    }

    fun clearAll() = prefs.edit().clear().apply()

    private companion object {
        const val PREFS_NAME = "gopay-bridge-settings"
        const val KEY_BACKEND_URL = "backend_url"
        const val KEY_DEVICE_ID = "device_id"
        const val KEY_DEVICE_SECRET = "device_secret"
        const val KEY_PACKAGES = "monitored_packages"
        const val KEY_IGNORE = "ignore_keywords"
        const val KEY_DISCOVERY_UNTIL = "discovery_until"
        const val DEFAULT_PACKAGES = "com.gojek.gopaymerchant"
    }
}
```

- [ ] **Step 5: Tulis `DiscoveryLog.kt`**

```kotlin
package expo.modules.gopaylistener.discovery

import android.content.Context
import org.json.JSONArray
import org.json.JSONObject

data class DiscoveryEntry(
    val packageName: String,
    val title: String,
    val seenAt: Long,
)

/**
 * Menyimpan jejak Discovery secara lokal.
 *
 * HANYA nama package dan judul yang dicatat, dan TIDAK PERNAH dikirim keluar
 * perangkat. Isi notifikasi tidak disimpan. Disimpan di SharedPreferences
 * biasa, bukan Room, karena datanya sementara dan tidak layak ikut
 * mengubah skema database.
 */
object DiscoveryLog {

    private const val PREFS = "gopay-bridge-discovery"
    private const val KEY = "entries"
    private const val MAX_ENTRIES = 100

    fun record(context: Context, packageName: String, title: String) {
        val prefs = context.applicationContext.getSharedPreferences(PREFS, Context.MODE_PRIVATE)
        val arr = JSONArray(prefs.getString(KEY, "[]"))

        val item = JSONObject().apply {
            put("package_name", packageName)
            put("title", title)
            put("seen_at", System.currentTimeMillis())
        }

        val trimmed = JSONArray()
        trimmed.put(item)
        for (i in 0 until minOf(arr.length(), MAX_ENTRIES - 1)) {
            trimmed.put(arr.get(i))
        }

        prefs.edit().putString(KEY, trimmed.toString()).apply()
    }

    fun entries(context: Context): List<DiscoveryEntry> {
        val prefs = context.applicationContext.getSharedPreferences(PREFS, Context.MODE_PRIVATE)
        val arr = JSONArray(prefs.getString(KEY, "[]"))
        return (0 until arr.length()).map { i ->
            val o = arr.getJSONObject(i)
            DiscoveryEntry(
                packageName = o.getString("package_name"),
                title = o.optString("title"),
                seenAt = o.getLong("seen_at"),
            )
        }
    }

    fun clear(context: Context) {
        context.applicationContext.getSharedPreferences(PREFS, Context.MODE_PRIVATE)
            .edit().clear().apply()
    }
}
```

- [ ] **Step 6: Tambahkan fungsi konfigurasi ke `GopayListenerModule.kt`**

Tambahkan di dalam `ModuleDefinition`, setelah `Function("isListenerConnected")`:

```kotlin
        Function("getSettings") {
            val s = Settings(context)
            mapOf(
                "backendUrl" to s.backendUrl,
                "deviceId" to s.deviceId,
                // Secret tidak pernah dikembalikan ke JS. Hanya penandanya.
                "hasDeviceSecret" to s.deviceSecret.isNotEmpty(),
                "monitoredPackages" to s.monitoredPackages,
                "ignoreKeywords" to s.ignoreKeywords,
                "discoveryUntilMs" to s.discoveryUntilMs,
            )
        }

        Function("saveSettings") { patch: Map<String, Any?> ->
            val s = Settings(context)
            (patch["backendUrl"] as? String)?.let { s.backendUrl = it }
            (patch["deviceId"] as? String)?.let { s.deviceId = it }
            (patch["deviceSecret"] as? String)?.let { if (it.isNotEmpty()) s.deviceSecret = it }
            @Suppress("UNCHECKED_CAST")
            (patch["monitoredPackages"] as? List<String>)?.let { s.monitoredPackages = it }
            @Suppress("UNCHECKED_CAST")
            (patch["ignoreKeywords"] as? List<String>)?.let { s.ignoreKeywords = it }
        }

        Function("startDiscovery") { minutes: Int ->
            val capped = minutes.coerceIn(1, 10)
            Settings(context).discoveryUntilMs = System.currentTimeMillis() + capped * 60_000L
        }

        Function("stopDiscovery") {
            Settings(context).discoveryUntilMs = 0L
        }

        Function("getDiscoveryEntries") {
            DiscoveryLog.entries(context).map {
                mapOf("packageName" to it.packageName, "title" to it.title, "seenAt" to it.seenAt)
            }
        }

        Function("clearDiscovery") {
            DiscoveryLog.clear(context)
        }
```

Tambahkan import `expo.modules.gopaylistener.config.Settings` dan `expo.modules.gopaylistener.discovery.DiscoveryLog`.

Secret sengaja tidak pernah dibaca balik ke JS — hanya ditulis. Layar Settings menampilkan "tersimpan" atau "belum diisi", bukan nilainya.

- [ ] **Step 7: Perbarui `mobile/modules/gopay-listener/index.ts`**

```ts
import { requireNativeModule } from 'expo-modules-core'

export interface BridgeSettings {
  backendUrl: string
  deviceId: string
  hasDeviceSecret: boolean
  monitoredPackages: string[]
  ignoreKeywords: string[]
  discoveryUntilMs: number
}

export interface SettingsPatch {
  backendUrl?: string
  deviceId?: string
  deviceSecret?: string
  monitoredPackages?: string[]
  ignoreKeywords?: string[]
}

export interface DiscoveryEntry {
  packageName: string
  title: string
  seenAt: number
}

interface GopayListenerModule {
  isNotificationAccessGranted(): boolean
  openNotificationAccessSettings(): void
  isListenerConnected(): boolean
  getSettings(): BridgeSettings
  saveSettings(patch: SettingsPatch): void
  startDiscovery(minutes: number): void
  stopDiscovery(): void
  getDiscoveryEntries(): DiscoveryEntry[]
  clearDiscovery(): void
}

export default requireNativeModule<GopayListenerModule>('GopayListener')
```

- [ ] **Step 8: Jalankan test, pastikan lulus**

```bash
cd mobile/android && ./gradlew :gopay-listener:testDebugUnitTest --tests '*SettingsTest*'
```

Expected: PASS — delapan test.

- [ ] **Step 9: Commit**

```bash
git add mobile/modules/gopay-listener/
git commit -m "feat(mobile): konfigurasi terenkripsi dan penyimpanan Discovery"
```

---

### Task 8: Pipeline penangkapan notifikasi

**Files:**
- Modify: `GoPayListenerService.kt`
- Create: `.../gopaylistener/CaptureBus.kt`
- Modify: `GopayListenerModule.kt`
- Modify: `mobile/modules/gopay-listener/index.ts`

**Interfaces:**
- Consumes: `Settings`, `DiscoveryLog`, `AmountParser`, `EventIdBuilder`, `AppDatabase`, `EventDao`, `EventEntity`, `EventStatus`.
- Produces:
  - `CaptureBus.listener: ((Map<String, Any?>) -> Unit)?`
  - Event JS `onBridgeChanged`
  - Fungsi module: `getStatus()`, `getEvents(limit)`

- [ ] **Step 1: Tulis `CaptureBus.kt`**

```kotlin
package expo.modules.gopaylistener

import android.os.Bundle

/**
 * Jalur satu arah dari service ke module, dipakai hanya untuk menyegarkan UI
 * saat aplikasi sedang dibuka.
 *
 * Ini murni kosmetik. Bila UI mati tidak ada observer terdaftar dan tidak ada
 * yang hilang — pengiriman tetap berjalan lewat WorkManager.
 *
 * Observer disimpan di objek statis, sehingga module WAJIB mendaftarkannya
 * lewat WeakReference (lihat Task 8 Step 3) agar instance module tidak
 * tertahan di memori setelah UI ditutup.
 */
object CaptureBus {

    private val observers = mutableSetOf<(Bundle) -> Unit>()

    @Synchronized
    fun register(observer: (Bundle) -> Unit) {
        observers.add(observer)
    }

    @Synchronized
    fun unregister(observer: (Bundle) -> Unit) {
        observers.remove(observer)
    }

    @Synchronized
    fun emit(payload: Bundle) {
        observers.forEach { it(payload) }
    }
}
```

- [ ] **Step 2: Lengkapi `GoPayListenerService.kt`**

Ganti seluruh isi berkas:

```kotlin
package expo.modules.gopaylistener

import android.app.Notification
import android.content.ComponentName
import android.service.notification.NotificationListenerService
import androidx.core.os.bundleOf
import android.service.notification.StatusBarNotification
import android.util.Log
import expo.modules.gopaylistener.config.Settings
import expo.modules.gopaylistener.db.AppDatabase
import expo.modules.gopaylistener.db.EventEntity
import expo.modules.gopaylistener.db.EventStatus
import expo.modules.gopaylistener.discovery.DiscoveryLog
import expo.modules.gopaylistener.work.UploadScheduler
import java.util.concurrent.Executors

class GoPayListenerService : NotificationListenerService() {

    companion object {
        private const val TAG = "GoPayListener"

        @Volatile
        var isConnected: Boolean = false
            private set

        internal fun setConnected(value: Boolean) {
            isConnected = value
        }

        // onNotificationPosted dipanggil di main thread; Room tidak boleh
        // disentuh dari sana.
        private val io = Executors.newSingleThreadExecutor()
    }

    override fun onListenerConnected() {
        super.onListenerConnected()
        setConnected(true)
        Log.i(TAG, "listener terikat")
    }

    override fun onListenerDisconnected() {
        super.onListenerDisconnected()
        setConnected(false)
        Log.w(TAG, "listener terlepas, meminta rebind")
        requestRebind(ComponentName(this, GoPayListenerService::class.java))
    }

    override fun onNotificationPosted(sbn: StatusBarNotification) {
        val app = applicationContext
        val pkg = sbn.packageName

        val extras = sbn.notification.extras
        val title = extras.getCharSequence(Notification.EXTRA_TITLE)?.toString()
        val text = extras.getCharSequence(Notification.EXTRA_TEXT)?.toString()
        val bigText = extras.getCharSequence(Notification.EXTRA_BIG_TEXT)?.toString()

        // Notification.when bertahan lintas repost; postTime tidak.
        val rawWhen = sbn.notification.`when`
        val whenMs = if (rawWhen > 0L) rawWhen else sbn.postTime

        io.execute {
            try {
                val settings = Settings(app)

                if (settings.discoveryActive()) {
                    DiscoveryLog.record(app, pkg, title.orEmpty())
                }

                if (pkg !in settings.monitoredPackages) return@execute

                val haystack = (title.orEmpty() + " " + text.orEmpty())
                val ignored = settings.ignoreKeywords.any { haystack.contains(it, ignoreCase = true) }

                val eventId = EventIdBuilder.build(pkg, title, text, whenMs)
                val now = System.currentTimeMillis()

                val entity = EventEntity(
                    eventId = eventId,
                    packageName = pkg,
                    title = title,
                    text = text,
                    bigText = bigText,
                    amountHint = AmountParser.parse(text ?: bigText),
                    postedAt = whenMs,
                    receivedAt = now,
                    status = if (ignored) EventStatus.IGNORED else EventStatus.PENDING,
                    attemptCount = 0,
                    lastError = null,
                    backendStatus = null,
                    sentAt = null,
                )

                val dao = AppDatabase.get(app).events()
                if (dao.insertIgnoringDuplicate(entity) == -1L) {
                    // Android memang memanggil ulang untuk notifikasi yang sama.
                    Log.i(TAG, "event sudah ada, dilewati")
                    return@execute
                }

                Log.i(TAG, "event baru ditangkap, status=${entity.status}")

                if (!ignored) {
                    UploadScheduler.enqueue(app)
                }

                // sendEvent menuntut Bundle, bukan Map.
                CaptureBus.emit(
                    bundleOf(
                        "eventId" to entity.eventId,
                        "title" to entity.title,
                        "text" to entity.text,
                        "amountHint" to entity.amountHint,
                        "receivedAt" to entity.receivedAt,
                        "status" to entity.status.name,
                    )
                )
            } catch (t: Throwable) {
                // Sebuah notifikasi yang gagal diproses tidak boleh mematikan service.
                Log.e(TAG, "gagal memproses notifikasi", t)
            }
        }
    }
}
```

Perhatikan yang **tidak** ditulis ke Logcat: isi notifikasi, nominal, dan nama pengirim. Yang dicatat hanya bahwa sesuatu terjadi.

- [ ] **Step 3: Tambahkan fungsi pembacaan ke `GopayListenerModule.kt`**

Tambahkan di dalam `ModuleDefinition`:

Tambahkan properti pada kelas module, di luar `definition()`:

```kotlin
    private var captureObserver: ((android.os.Bundle) -> Unit)? = null
```

Lalu di dalam `ModuleDefinition`:

```kotlin
        Events("onBridgeChanged")

        // Nama event wajib disebut pada OnStartObserving/OnStopObserving.
        // WeakReference dipakai karena observer disimpan di objek statis
        // CaptureBus; tanpa itu instance module tertahan setelah UI ditutup.
        OnStartObserving("onBridgeChanged") {
            val weakModule = java.lang.ref.WeakReference(this@GopayListenerModule)
            val observer: (android.os.Bundle) -> Unit = { payload ->
                weakModule.get()?.sendEvent("onBridgeChanged", payload)
            }
            captureObserver = observer
            CaptureBus.register(observer)
        }

        OnStopObserving("onBridgeChanged") {
            captureObserver?.let { CaptureBus.unregister(it) }
            captureObserver = null
        }

        Function("getStatus") {
            val dao = AppDatabase.get(context).events()
            val latest = dao.latest()
            mapOf(
                "notificationAccessGranted" to isAccessGranted(),
                "listenerConnected" to GoPayListenerService.isConnected,
                "pendingCount" to dao.countByStatus(EventStatus.PENDING),
                "sendingCount" to dao.countByStatus(EventStatus.SENDING),
                "failedCount" to dao.countByStatus(EventStatus.FAILED),
                "lastEvent" to latest?.let {
                    mapOf(
                        "eventId" to it.eventId,
                        "title" to it.title,
                        "text" to it.text,
                        "amountHint" to it.amountHint,
                        "receivedAt" to it.receivedAt,
                        "status" to it.status.name,
                    )
                },
            )
        }

        Function("getEvents") { limit: Int ->
            AppDatabase.get(context).events().recent(limit.coerceIn(1, 500)).map {
                mapOf(
                    "eventId" to it.eventId,
                    "title" to it.title,
                    "text" to it.text,
                    "amountHint" to it.amountHint,
                    "postedAt" to it.postedAt,
                    "receivedAt" to it.receivedAt,
                    "status" to it.status.name,
                    "attemptCount" to it.attemptCount,
                    "lastError" to it.lastError,
                    "backendStatus" to it.backendStatus,
                    "sentAt" to it.sentAt,
                )
            }
        }

        Function("retryEvent") { eventId: String ->
            AppDatabase.get(context).events().rescheduleForRetry(eventId, "dikirim ulang manual")
            UploadScheduler.enqueue(context)
        }

        Function("clearHistory") {
            AppDatabase.get(context).events().purgeOlderThan(System.currentTimeMillis())
        }
```

Dan tambahkan helper privat di kelas module, lalu pakai juga di `Function("isNotificationAccessGranted")`:

```kotlin
    private fun isAccessGranted(): Boolean {
        val enabled = android.provider.Settings.Secure.getString(
            context.contentResolver,
            "enabled_notification_listeners"
        ) ?: return false
        return enabled.split(":").any { it.startsWith(context.packageName + "/") }
    }
```

`getStatus` dan `getEvents` menyentuh Room secara sinkron. Itu diterima di sini karena hanya dipanggil dari UI yang sedang terbuka dan datanya kecil; pipeline yang sebenarnya tidak pernah lewat jalur ini.

- [ ] **Step 4: Perbarui `index.ts` dengan tipe baru**

Tambahkan:

```ts
export type EventStatus = 'PENDING' | 'SENDING' | 'SENT' | 'FAILED' | 'IGNORED'

export interface BridgeEvent {
  eventId: string
  title: string | null
  text: string | null
  amountHint: number | null
  postedAt: number
  receivedAt: number
  status: EventStatus
  attemptCount: number
  lastError: string | null
  backendStatus: string | null
  sentAt: number | null
}

export interface BridgeStatus {
  notificationAccessGranted: boolean
  listenerConnected: boolean
  pendingCount: number
  sendingCount: number
  failedCount: number
  lastEvent: Pick<BridgeEvent, 'eventId' | 'title' | 'text' | 'amountHint' | 'receivedAt' | 'status'> | null
}
```

Tambahkan ke `interface GopayListenerModule`:

```ts
  getStatus(): BridgeStatus
  getEvents(limit: number): BridgeEvent[]
  retryEvent(eventId: string): void
  clearHistory(): void
  addListener(event: 'onBridgeChanged', handler: (e: BridgeStatus['lastEvent']) => void): { remove(): void }
```

- [ ] **Step 5: Build dan pasang**

```bash
cd mobile && APP_VARIANT=development npx expo run:android
```

Expected: build sukses. Task 9 masih diperlukan agar `UploadScheduler` ada — bila build gagal karena `UploadScheduler` belum ada, kerjakan Task 9 Step 4 lebih dulu lalu kembali ke sini.

- [ ] **Step 6: `NEEDS-DEVICE` — verifikasi mode Discovery**

1. Di aplikasi, jalankan `startDiscovery(10)` (sementara bisa lewat tombol darurat di `App.tsx`, atau tunggu layar Debug di Task 10).
2. Kirim notifikasi WhatsApp ke HP.
3. Panggil `getDiscoveryEntries()`.

Expected: entri memuat `com.whatsapp` — membuktikan listener menerima notifikasi dari sistem. Tidak ada apa pun yang terkirim ke backend.

- [ ] **Step 7: `NEEDS-DEVICE` — verifikasi penangkapan GoPay sungguhan**

Minta transfer masuk Rp1, lalu panggil `getEvents(10)`.

Expected: satu event dengan `title = "Transfer masuk"`, `amountHint = 1`, `status = "PENDING"` (backend belum dikonfigurasi) atau `"SENT"` (bila sudah).

Ini titik penentu M2. Bila event tidak muncul sama sekali padahal Notification Access aktif, catat sebagai `FAIL` — jangan lanjut ke Task 9 sebelum ini terbukti.

- [ ] **Step 8: Commit**

```bash
git add mobile/modules/gopay-listener/
git commit -m "feat(mobile): pipeline penangkapan notifikasi ke Room"
```

---

### Task 9: Pengiriman ke backend dengan retry

**Files:**
- Create: `.../gopaylistener/net/ResponseMapper.kt`
- Create: `.../gopaylistener/net/Uploader.kt`
- Create: `.../gopaylistener/work/EventUploadWorker.kt`
- Create: `.../gopaylistener/work/UploadScheduler.kt`
- Create: `.../gopaylistener/net/ResponseMapperTest.kt` (di `src/test`)
- Modify: `build.gradle`, `GopayListenerModule.kt`, `index.ts`

**Interfaces:**
- Consumes: `Signer`, `Settings`, `BridgeConfig`, `EventDao`, `EventEntity`.
- Produces:
  - `sealed interface UploadOutcome` dengan `Sent(backendStatus)`, `Failed(error)`, `Retry(error, retryAfterSeconds)`
  - `ResponseMapper.map(httpCode: Int, body: String?, retryAfterHeader: String?): UploadOutcome`
  - `Uploader.send(cfg: BridgeConfig, e: EventEntity): UploadOutcome`
  - `Uploader.health(backendUrl: String): Pair<Boolean, Long?>`
  - `UploadScheduler.enqueue(context: Context)`
  - Fungsi module: `checkBackend()`, `testConnection()`

- [ ] **Step 1: Tambahkan dependensi — `build.gradle`**

```gradle
  implementation 'androidx.work:work-runtime-ktx:2.9.1'
  implementation 'com.squareup.okhttp3:okhttp:4.12.0'
```

- [ ] **Step 2: Tulis test yang gagal — `ResponseMapperTest.kt`**

```kotlin
package expo.modules.gopaylistener.net

import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

class ResponseMapperTest {

    @Test
    fun `200 accepted menjadi Sent`() {
        val out = ResponseMapper.map(200, """{"success":true,"status":"accepted"}""", null)
        assertTrue(out is UploadOutcome.Sent)
        assertEquals("accepted", (out as UploadOutcome.Sent).backendStatus)
    }

    @Test
    fun `200 duplicate juga menjadi Sent`() {
        val out = ResponseMapper.map(200, """{"success":true,"status":"duplicate"}""", null)
        assertTrue("duplicate adalah sukses, bukan error", out is UploadOutcome.Sent)
        assertEquals("duplicate", (out as UploadOutcome.Sent).backendStatus)
    }

    @Test
    fun `200 tanpa status yang dikenal menjadi Failed`() {
        val out = ResponseMapper.map(200, """{"success":true}""", null)
        assertTrue(out is UploadOutcome.Failed)
    }

    @Test
    fun `200 dengan body bukan JSON menjadi Failed`() {
        val out = ResponseMapper.map(200, "<html>proxy error</html>", null)
        assertTrue(out is UploadOutcome.Failed)
    }

    @Test
    fun `400 menjadi Failed tanpa retry`() {
        val out = ResponseMapper.map(400, """{"error":"invalid_payload"}""", null)
        assertTrue(out is UploadOutcome.Failed)
        assertEquals("invalid_payload", (out as UploadOutcome.Failed).error)
    }

    @Test
    fun `401 invalid_signature menjadi Failed tanpa retry`() {
        val out = ResponseMapper.map(401, """{"error":"invalid_signature"}""", null)
        assertTrue(out is UploadOutcome.Failed)
        assertEquals("invalid_signature", (out as UploadOutcome.Failed).error)
    }

    @Test
    fun `401 clock_skew menjadi Failed dengan kode sendiri`() {
        val out = ResponseMapper.map(401, """{"error":"clock_skew","server_time":1789036200}""", null)
        assertTrue(out is UploadOutcome.Failed)
        assertEquals(
            "clock_skew harus dapat dibedakan dari invalid_signature",
            "clock_skew",
            (out as UploadOutcome.Failed).error
        )
    }

    @Test
    fun `403 device_disabled menjadi Failed tanpa retry`() {
        val out = ResponseMapper.map(403, """{"error":"device_disabled"}""", null)
        assertTrue(out is UploadOutcome.Failed)
    }

    @Test
    fun `429 menjadi Retry dan menghormati Retry-After`() {
        val out = ResponseMapper.map(429, null, "120")
        assertTrue(out is UploadOutcome.Retry)
        assertEquals(120L, (out as UploadOutcome.Retry).retryAfterSeconds)
    }

    @Test
    fun `429 tanpa Retry-After tetap Retry`() {
        val out = ResponseMapper.map(429, null, null)
        assertTrue(out is UploadOutcome.Retry)
        assertEquals(null, (out as UploadOutcome.Retry).retryAfterSeconds)
    }

    @Test
    fun `Retry-After yang tidak berupa angka diabaikan`() {
        val out = ResponseMapper.map(429, null, "Wed, 21 Oct 2026 07:28:00 GMT")
        assertTrue(out is UploadOutcome.Retry)
        assertEquals(null, (out as UploadOutcome.Retry).retryAfterSeconds)
    }

    @Test
    fun `5xx menjadi Retry`() {
        for (code in listOf(500, 502, 503, 504)) {
            val out = ResponseMapper.map(code, null, null)
            assertTrue("kode $code seharusnya Retry", out is UploadOutcome.Retry)
        }
    }

    @Test
    fun `kode tak terduga di keluarga 4xx menjadi Failed`() {
        val out = ResponseMapper.map(418, null, null)
        assertTrue(out is UploadOutcome.Failed)
    }
}
```

- [ ] **Step 3: Jalankan test, pastikan gagal**

```bash
cd mobile/android && ./gradlew :gopay-listener:testDebugUnitTest --tests '*ResponseMapperTest*'
```

Expected: FAIL — `Unresolved reference: ResponseMapper`.

- [ ] **Step 4: Tulis `ResponseMapper.kt`**

```kotlin
package expo.modules.gopaylistener.net

import org.json.JSONObject

sealed interface UploadOutcome {
    data class Sent(val backendStatus: String) : UploadOutcome
    data class Failed(val error: String) : UploadOutcome
    data class Retry(val error: String, val retryAfterSeconds: Long?) : UploadOutcome
}

/**
 * Menerjemahkan response backend menjadi tindakan, sesuai tabel di
 * api-contract.md §4.6. Dipisah sebagai fungsi murni agar seluruh baris
 * tabel itu dapat diuji tanpa jaringan.
 */
object ResponseMapper {

    fun map(httpCode: Int, body: String?, retryAfterHeader: String?): UploadOutcome = when {
        httpCode == 200 -> mapOk(body)

        // duplicate ditangani di mapOk — di sini hanya kegagalan.
        httpCode == 429 -> UploadOutcome.Retry("rate_limited", retryAfterHeader?.toLongOrNull())

        httpCode >= 500 -> UploadOutcome.Retry("server_error_$httpCode", null)

        httpCode in 400..499 -> UploadOutcome.Failed(errorCode(body) ?: "http_$httpCode")

        else -> UploadOutcome.Failed("http_$httpCode")
    }

    private fun mapOk(body: String?): UploadOutcome {
        val status = try {
            JSONObject(body ?: "").optString("status")
        } catch (_: Throwable) {
            return UploadOutcome.Failed("response_tidak_terbaca")
        }

        // duplicate diperlakukan sebagai sukses: bila response hilang di tengah
        // jalan padahal backend sudah menerima, percobaan berikutnya membalas
        // duplicate dan event beres — tidak ada pembayaran terhitung dua kali.
        return if (status == "accepted" || status == "duplicate") {
            UploadOutcome.Sent(status)
        } else {
            UploadOutcome.Failed("status_tidak_dikenal")
        }
    }

    private fun errorCode(body: String?): String? = try {
        JSONObject(body ?: "").optString("error").takeIf { it.isNotEmpty() }
    } catch (_: Throwable) {
        null
    }
}
```

- [ ] **Step 5: Jalankan test, pastikan lulus**

```bash
cd mobile/android && ./gradlew :gopay-listener:testDebugUnitTest --tests '*ResponseMapperTest*'
```

Expected: PASS — tiga belas test.

- [ ] **Step 6: Tulis `Uploader.kt`**

```kotlin
package expo.modules.gopaylistener.net

import expo.modules.gopaylistener.Signer
import expo.modules.gopaylistener.config.BridgeConfig
import expo.modules.gopaylistener.db.EventEntity
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import org.json.JSONObject
import java.io.IOException
import java.time.Instant
import java.time.OffsetDateTime
import java.time.ZoneId
import java.time.format.DateTimeFormatter
import java.util.concurrent.TimeUnit

object Uploader {

    private val JSON = "application/json".toMediaType()

    private val client = OkHttpClient.Builder()
        .connectTimeout(15, TimeUnit.SECONDS)
        .readTimeout(30, TimeUnit.SECONDS)
        .writeTimeout(30, TimeUnit.SECONDS)
        .build()

    fun send(cfg: BridgeConfig, e: EventEntity): UploadOutcome {
        // Byte ini yang ditandatangani DAN yang dikirim. Jangan pernah
        // menyusun ulang JSON setelah menandatanganinya.
        val payload = buildPayload(cfg.deviceId, e).toByteArray(Charsets.UTF_8)

        val timestamp = System.currentTimeMillis() / 1000
        val signature = Signer.sign(
            cfg.deviceSecret,
            Signer.signingString(cfg.deviceId, timestamp, payload)
        )

        val request = Request.Builder()
            .url(cfg.backendUrl + "/callback/gopay")
            .post(payload.toRequestBody(JSON))
            .header("X-Device-Id", cfg.deviceId)
            .header("X-Timestamp", timestamp.toString())
            .header("X-Signature", signature)
            .build()

        return try {
            client.newCall(request).execute().use { response ->
                ResponseMapper.map(
                    response.code,
                    response.body?.string(),
                    response.header("Retry-After")
                )
            }
        } catch (io: IOException) {
            // Jaringan mati atau timeout — selalu dapat dipulihkan.
            UploadOutcome.Retry("network", null)
        }
    }

    /** Mengembalikan (backend hidup, jam server dalam detik). */
    fun health(backendUrl: String): Pair<Boolean, Long?> {
        val request = Request.Builder().url("$backendUrl/health").get().build()
        return try {
            client.newCall(request).execute().use { response ->
                if (!response.isSuccessful) return false to null
                val json = JSONObject(response.body?.string() ?: "")
                true to json.optLong("server_time").takeIf { it > 0 }
            }
        } catch (_: Throwable) {
            false to null
        }
    }

    private fun buildPayload(deviceId: String, e: EventEntity): String {
        val notification = JSONObject().apply {
            put("package_name", e.packageName)
            put("title", e.title ?: JSONObject.NULL)
            put("text", e.text ?: JSONObject.NULL)
            put("big_text", e.bigText ?: JSONObject.NULL)
            put("posted_at", e.postedAt)
        }

        val receivedAt = OffsetDateTime
            .ofInstant(Instant.ofEpochMilli(e.receivedAt), ZoneId.systemDefault())
            .format(DateTimeFormatter.ISO_OFFSET_DATE_TIME)

        return JSONObject().apply {
            put("event_id", e.eventId)
            put("device_id", deviceId)
            put("source", "gopay")
            put("notification", notification)
            put("amount_hint", e.amountHint ?: JSONObject.NULL)
            put("received_at", receivedAt)
        }.toString()
    }
}
```

- [ ] **Step 7: Tulis `UploadScheduler.kt`**

```kotlin
package expo.modules.gopaylistener.work

import android.content.Context
import androidx.work.BackoffPolicy
import androidx.work.Constraints
import androidx.work.ExistingWorkPolicy
import androidx.work.NetworkType
import androidx.work.OneTimeWorkRequestBuilder
import androidx.work.WorkManager
import java.util.concurrent.TimeUnit

object UploadScheduler {

    private const val WORK_NAME = "event-upload"

    /**
     * Menjadwalkan pengiriman.
     *
     * Constraint NetworkType.CONNECTED berarti Android sendiri yang
     * membangunkan kita saat jaringan pulih — tidak ada polling sama sekali.
     */
    fun enqueue(context: Context) {
        val request = OneTimeWorkRequestBuilder<EventUploadWorker>()
            .setConstraints(
                Constraints.Builder()
                    .setRequiredNetworkType(NetworkType.CONNECTED)
                    .build()
            )
            .setBackoffCriteria(BackoffPolicy.EXPONENTIAL, 30, TimeUnit.SECONDS)
            .build()

        WorkManager.getInstance(context.applicationContext)
            .enqueueUniqueWork(WORK_NAME, ExistingWorkPolicy.APPEND_OR_REPLACE, request)
    }
}
```

- [ ] **Step 8: Tulis `EventUploadWorker.kt`**

```kotlin
package expo.modules.gopaylistener.work

import android.content.Context
import android.util.Log
import androidx.work.CoroutineWorker
import androidx.work.WorkerParameters
import expo.modules.gopaylistener.config.Settings
import expo.modules.gopaylistener.db.AppDatabase
import expo.modules.gopaylistener.net.UploadOutcome
import expo.modules.gopaylistener.net.Uploader

class EventUploadWorker(
    context: Context,
    params: WorkerParameters,
) : CoroutineWorker(context, params) {

    override suspend fun doWork(): Result {
        val app = applicationContext
        val cfg = Settings(app).config()
        if (cfg == null) {
            Log.i(TAG, "backend belum dikonfigurasi, pengiriman ditunda")
            return Result.success()
        }

        val dao = AppDatabase.get(app).events()
        val batch = dao.claimPending(BATCH_SIZE)
        if (batch.isEmpty()) return Result.success()

        var needRetry = false

        for (event in batch) {
            when (val outcome = Uploader.send(cfg, event)) {
                is UploadOutcome.Sent -> {
                    dao.markSent(event.eventId, outcome.backendStatus, System.currentTimeMillis())
                    Log.i(TAG, "event terkirim, status=${outcome.backendStatus}")
                }

                is UploadOutcome.Failed -> {
                    dao.markFailed(event.eventId, outcome.error)
                    Log.w(TAG, "event gagal permanen: ${outcome.error}")
                }

                is UploadOutcome.Retry -> {
                    val attempts = event.attemptCount + 1
                    if (attempts >= MAX_ATTEMPTS) {
                        dao.markFailed(
                            event.eventId,
                            "menyerah setelah $MAX_ATTEMPTS percobaan: ${outcome.error}"
                        )
                        Log.w(TAG, "event menyerah setelah $MAX_ATTEMPTS percobaan")
                    } else {
                        dao.rescheduleForRetry(event.eventId, outcome.error)
                        needRetry = true
                    }
                }
            }
        }

        return if (needRetry) Result.retry() else Result.success()
    }

    private companion object {
        const val TAG = "GoPayUpload"
        const val BATCH_SIZE = 50

        /**
         * WorkManager memakai backoff eksponensial dari 30 detik, dibatasi
         * 5 jam per jeda. Tiga belas percobaan menghasilkan total sekitar
         * 24 jam sebelum menyerah:
         *   30s, 1m, 2m, 4m, 8m, 16m, 32m, 64m, 2.1j, lalu 5j × 4.
         */
        const val MAX_ATTEMPTS = 13
    }
}
```

- [ ] **Step 9: Tambahkan fungsi jaringan ke `GopayListenerModule.kt`**

```kotlin
        AsyncFunction("checkBackend") {
            val url = Settings(context).backendUrl
            if (url.isEmpty()) {
                return@AsyncFunction mapOf("configured" to false, "reachable" to false)
            }

            val (reachable, serverTime) = Uploader.health(url)
            val skewSeconds = serverTime?.let { it - System.currentTimeMillis() / 1000 }

            mapOf(
                "configured" to true,
                "reachable" to reachable,
                "serverTime" to serverTime,
                "skewSeconds" to skewSeconds,
                // Batas yang sama dengan backend, agar peringatan muncul
                // sebelum request sungguhan ditolak.
                "clockOutOfSync" to (skewSeconds != null && kotlin.math.abs(skewSeconds) > 300),
            )
        }

        AsyncFunction("testConnection") {
            val cfg = Settings(context).config()
                ?: return@AsyncFunction mapOf("ok" to false, "error" to "belum_dikonfigurasi")

            val timestamp = System.currentTimeMillis() / 1000
            val signature = Signer.sign(
                cfg.deviceSecret,
                Signer.signingString(cfg.deviceId, timestamp, ByteArray(0))
            )
            Uploader.deviceMe(cfg, timestamp, signature)
        }
```

Tambahkan `Uploader.deviceMe` ke `Uploader.kt`:

```kotlin
    fun deviceMe(cfg: BridgeConfig, timestamp: Long, signature: String): Map<String, Any?> {
        val request = Request.Builder()
            .url(cfg.backendUrl + "/device/me")
            .get()
            .header("X-Device-Id", cfg.deviceId)
            .header("X-Timestamp", timestamp.toString())
            .header("X-Signature", signature)
            .build()

        return try {
            client.newCall(request).execute().use { response ->
                val body = response.body?.string()
                if (response.isSuccessful) {
                    val json = JSONObject(body ?: "{}")
                    mapOf("ok" to true, "deviceName" to json.optString("name"))
                } else {
                    val error = try {
                        JSONObject(body ?: "{}").optString("error")
                    } catch (_: Throwable) {
                        ""
                    }
                    mapOf("ok" to false, "error" to error.ifEmpty { "http_${response.code}" })
                }
            }
        } catch (_: IOException) {
            mapOf("ok" to false, "error" to "network")
        }
    }
```

- [ ] **Step 10: Perbarui `index.ts`**

```ts
export interface BackendHealth {
  configured: boolean
  reachable: boolean
  serverTime?: number | null
  skewSeconds?: number | null
  clockOutOfSync?: boolean
}

export interface ConnectionTest {
  ok: boolean
  deviceName?: string
  error?: string
}
```

Tambahkan ke `interface GopayListenerModule`:

```ts
  checkBackend(): Promise<BackendHealth>
  testConnection(): Promise<ConnectionTest>
```

- [ ] **Step 11: Jalankan seluruh unit test Kotlin**

```bash
cd mobile/android && ./gradlew :gopay-listener:testDebugUnitTest
```

Expected: PASS — seluruh test dari Task 3 sampai 9.

- [ ] **Step 12: `NEEDS-DEVICE` — pengiriman sungguhan ke VPS**

1. Isi Settings dengan `Backend URL` (`https://<domain>/api/v1`), `Device ID`, dan `Device Secret` dari `devicetool`.
2. Minta transfer masuk Rp1.
3. Di laptop: `curl -u admin:<password> https://<domain>/api/v1/events | jq '.events[0]'`

Expected: event muncul di backend dengan `title = "Transfer masuk"`. Di HP, `getEvents(10)` menunjukkan `status = "SENT"` dan `backendStatus = "accepted"`.

- [ ] **Step 13: `NEEDS-DEVICE` — verifikasi retry saat jaringan mati**

1. Nyalakan mode pesawat di HP.
2. Minta transfer masuk Rp1 (transfer tetap terjadi; notifikasi muncul saat HP kembali online — bila tidak, lakukan langkah 3 lebih dulu lalu kirim transfer).
3. Matikan mode pesawat.

Expected: event yang tadinya `PENDING` berpindah ke `SENT` tanpa campur tangan. Pastikan aplikasi **tertutup** selama percobaan ini.

- [ ] **Step 14: Commit**

```bash
git add mobile/modules/gopay-listener/
git commit -m "feat(mobile): pengiriman ber-HMAC dengan WorkManager dan retry"
```

---

### Task 10: Layar aplikasi

**Files:**
- Create: `mobile/src/navigation/index.tsx`
- Create: `mobile/src/screens/DashboardScreen.tsx`
- Create: `mobile/src/screens/HistoryScreen.tsx`
- Create: `mobile/src/screens/SettingsScreen.tsx`
- Create: `mobile/src/screens/DebugScreen.tsx`
- Create: `mobile/src/lib/format.ts`
- Create: `mobile/src/lib/useBridge.ts`
- Modify: `mobile/App.tsx`

**Interfaces:**
- Consumes: seluruh fungsi module dari Task 2, 7, 8, 9.
- Produces: aplikasi lengkap empat layar.

- [ ] **Step 1: Tulis `mobile/src/lib/format.ts`**

```ts
export function formatRupiah(amount: number | null): string {
  if (amount === null) return '—'
  return 'Rp' + amount.toLocaleString('id-ID')
}

export function formatWaktu(ms: number): string {
  return new Date(ms).toLocaleString('id-ID', {
    day: '2-digit',
    month: 'short',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}

/** Dipakai Dashboard untuk memperlihatkan service yang mati diam-diam. */
export function sejakKapan(ms: number, nowMs = Date.now()): string {
  const detik = Math.max(0, Math.floor((nowMs - ms) / 1000))
  if (detik < 60) return 'baru saja'
  if (detik < 3600) return `${Math.floor(detik / 60)} menit lalu`
  if (detik < 86400) return `${Math.floor(detik / 3600)} jam lalu`
  return `${Math.floor(detik / 86400)} hari lalu`
}
```

- [ ] **Step 2: Tulis `mobile/src/lib/useBridge.ts`**

```ts
import { useCallback, useEffect, useState } from 'react'
import { AppState } from 'react-native'
import GopayListener, { type BackendHealth, type BridgeStatus } from '../../modules/gopay-listener'

/**
 * Menyegarkan status saat layar dibuka, saat aplikasi kembali ke depan,
 * dan saat native melaporkan notifikasi baru. Tidak ada polling.
 */
export function useBridgeStatus() {
  const [status, setStatus] = useState<BridgeStatus>(() => GopayListener.getStatus())
  const [backend, setBackend] = useState<BackendHealth | null>(null)

  const refresh = useCallback(() => {
    setStatus(GopayListener.getStatus())
    GopayListener.checkBackend().then(setBackend).catch(() => setBackend(null))
  }, [])

  useEffect(() => {
    refresh()

    const sub = GopayListener.addListener('onBridgeChanged', () => {
      setStatus(GopayListener.getStatus())
    })
    const appSub = AppState.addEventListener('change', (s) => {
      if (s === 'active') refresh()
    })

    return () => {
      sub.remove()
      appSub.remove()
    }
  }, [refresh])

  return { status, backend, refresh }
}
```

- [ ] **Step 3: Tulis `mobile/src/screens/DashboardScreen.tsx`**

```tsx
import { RefreshControl, ScrollView, StyleSheet, Text, TouchableOpacity, View } from 'react-native'
import GopayListener from '../../modules/gopay-listener'
import { useBridgeStatus } from '../lib/useBridge'
import { formatRupiah, formatWaktu, sejakKapan } from '../lib/format'

function Baris({ label, ok, detail }: { label: string; ok: boolean; detail?: string }) {
  return (
    <View style={styles.baris}>
      <Text style={styles.label}>{label}</Text>
      <View style={styles.kanan}>
        <Text style={[styles.titik, { color: ok ? '#16a34a' : '#dc2626' }]}>●</Text>
        <Text style={styles.detail}>{detail}</Text>
      </View>
    </View>
  )
}

export default function DashboardScreen() {
  const { status, backend, refresh } = useBridgeStatus()
  const last = status.lastEvent

  return (
    <ScrollView
      contentContainerStyle={styles.root}
      refreshControl={<RefreshControl refreshing={false} onRefresh={refresh} />}
    >
      <Text style={styles.judul}>GoPay Notification Bridge</Text>

      <View style={styles.kartu}>
        <Baris
          label="Notification Access"
          ok={status.notificationAccessGranted}
          detail={status.notificationAccessGranted ? 'aktif' : 'belum aktif'}
        />
        <Baris
          label="Listener"
          ok={status.listenerConnected}
          detail={status.listenerConnected ? 'terikat' : 'tidak terikat'}
        />
        <Baris
          label="Backend"
          ok={backend?.reachable ?? false}
          detail={
            !backend?.configured ? 'belum dikonfigurasi' : backend.reachable ? 'terhubung' : 'tidak dapat dihubungi'
          }
        />
      </View>

      {status.notificationAccessGranted && !status.listenerConnected && (
        <View style={styles.peringatan}>
          <Text style={styles.peringatanTeks}>
            Izin sudah diberikan tetapi listener tidak terikat. Ini biasanya berarti sistem membunuh
            service. Periksa pengaturan baterai dan mulai otomatis untuk aplikasi ini.
          </Text>
        </View>
      )}

      {backend?.clockOutOfSync && (
        <View style={styles.peringatan}>
          <Text style={styles.peringatanTeks}>
            Jam HP meleset {backend.skewSeconds} detik dari server. Selama selisihnya lebih dari 5
            menit, semua pengiriman akan ditolak. Setel jam ke otomatis.
          </Text>
        </View>
      )}

      <View style={styles.kartu}>
        <Text style={styles.subjudul}>Event terakhir</Text>
        {last ? (
          <>
            <Text style={styles.nominal}>{formatRupiah(last.amountHint)}</Text>
            <Text style={styles.teks}>{last.title ?? '—'}</Text>
            <Text style={styles.teksKecil}>
              {formatWaktu(last.receivedAt)} • {sejakKapan(last.receivedAt)} • {last.status}
            </Text>
          </>
        ) : (
          <Text style={styles.teksKecil}>Belum ada event.</Text>
        )}
      </View>

      <View style={styles.kartu}>
        <Text style={styles.subjudul}>Antrean</Text>
        <Text style={styles.teks}>
          {status.pendingCount} menunggu • {status.sendingCount} dikirim • {status.failedCount} gagal
        </Text>
      </View>

      {!status.notificationAccessGranted && (
        <TouchableOpacity
          style={styles.tombol}
          onPress={() => GopayListener.openNotificationAccessSettings()}
        >
          <Text style={styles.tombolTeks}>Aktifkan Notification Access</Text>
        </TouchableOpacity>
      )}
    </ScrollView>
  )
}

const styles = StyleSheet.create({
  root: { padding: 16, gap: 12 },
  judul: { fontSize: 20, fontWeight: '600', marginBottom: 4 },
  subjudul: { fontSize: 13, color: '#6b7280', marginBottom: 6 },
  kartu: { backgroundColor: '#f9fafb', borderRadius: 12, padding: 16, gap: 4 },
  baris: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center', paddingVertical: 6 },
  kanan: { flexDirection: 'row', alignItems: 'center', gap: 6 },
  label: { fontSize: 15 },
  titik: { fontSize: 14 },
  detail: { fontSize: 13, color: '#6b7280' },
  nominal: { fontSize: 24, fontWeight: '600' },
  teks: { fontSize: 15 },
  teksKecil: { fontSize: 13, color: '#6b7280' },
  peringatan: { backgroundColor: '#fef3c7', borderRadius: 12, padding: 14 },
  peringatanTeks: { fontSize: 13, color: '#92400e', lineHeight: 19 },
  tombol: { backgroundColor: '#111827', borderRadius: 12, padding: 16, alignItems: 'center' },
  tombolTeks: { color: '#fff', fontSize: 15, fontWeight: '600' },
})
```

- [ ] **Step 4: Tulis `mobile/src/screens/HistoryScreen.tsx`**

```tsx
import { useCallback, useState } from 'react'
import { useFocusEffect } from '@react-navigation/native'
import { FlatList, StyleSheet, Text, TouchableOpacity, View } from 'react-native'
import GopayListener, { type BridgeEvent } from '../../modules/gopay-listener'
import { formatRupiah, formatWaktu } from '../lib/format'

const WARNA: Record<string, string> = {
  SENT: '#16a34a',
  PENDING: '#ca8a04',
  SENDING: '#2563eb',
  FAILED: '#dc2626',
  IGNORED: '#9ca3af',
}

export default function HistoryScreen() {
  const [events, setEvents] = useState<BridgeEvent[]>([])

  const muat = useCallback(() => setEvents(GopayListener.getEvents(200)), [])
  useFocusEffect(muat)

  function kirimUlang(eventId: string) {
    GopayListener.retryEvent(eventId)
    muat()
  }

  return (
    <FlatList
      contentContainerStyle={styles.root}
      data={events}
      keyExtractor={(e) => e.eventId}
      ListEmptyComponent={<Text style={styles.kosong}>Belum ada event.</Text>}
      renderItem={({ item }) => (
        <View style={styles.kartu}>
          <View style={styles.baris}>
            <Text style={styles.nominal}>{formatRupiah(item.amountHint)}</Text>
            <Text style={[styles.status, { color: WARNA[item.status] ?? '#6b7280' }]}>{item.status}</Text>
          </View>
          <Text style={styles.teks}>{item.title ?? '—'}</Text>
          <Text style={styles.teksKecil}>{formatWaktu(item.receivedAt)}</Text>

          {item.attemptCount > 0 && (
            <Text style={styles.teksKecil}>Percobaan: {item.attemptCount}</Text>
          )}
          {item.lastError && <Text style={styles.error}>{item.lastError}</Text>}

          {item.status === 'FAILED' && (
            <TouchableOpacity style={styles.tombolKecil} onPress={() => kirimUlang(item.eventId)}>
              <Text style={styles.tombolKecilTeks}>Kirim ulang</Text>
            </TouchableOpacity>
          )}
        </View>
      )}
    />
  )
}

const styles = StyleSheet.create({
  root: { padding: 16, gap: 10 },
  kosong: { textAlign: 'center', color: '#6b7280', marginTop: 40 },
  kartu: { backgroundColor: '#f9fafb', borderRadius: 12, padding: 14, gap: 3 },
  baris: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center' },
  nominal: { fontSize: 18, fontWeight: '600' },
  status: { fontSize: 12, fontWeight: '600' },
  teks: { fontSize: 14 },
  teksKecil: { fontSize: 12, color: '#6b7280' },
  error: { fontSize: 12, color: '#dc2626' },
  tombolKecil: { marginTop: 8, backgroundColor: '#111827', borderRadius: 8, padding: 10, alignItems: 'center' },
  tombolKecilTeks: { color: '#fff', fontSize: 13, fontWeight: '600' },
})
```

- [ ] **Step 5: Tulis `mobile/src/screens/SettingsScreen.tsx`**

```tsx
import { useState } from 'react'
import { Alert, ScrollView, StyleSheet, Text, TextInput, TouchableOpacity, View } from 'react-native'
import GopayListener from '../../modules/gopay-listener'

export default function SettingsScreen() {
  const awal = GopayListener.getSettings()

  const [backendUrl, setBackendUrl] = useState(awal.backendUrl)
  const [deviceId, setDeviceId] = useState(awal.deviceId)
  const [deviceSecret, setDeviceSecret] = useState('')
  const [packages, setPackages] = useState(awal.monitoredPackages.join(', '))
  const [keywords, setKeywords] = useState(awal.ignoreKeywords.join(', '))
  const [punyaSecret, setPunyaSecret] = useState(awal.hasDeviceSecret)

  function simpan() {
    GopayListener.saveSettings({
      backendUrl,
      deviceId,
      deviceSecret: deviceSecret || undefined,
      monitoredPackages: packages.split(',').map((s) => s.trim()).filter(Boolean),
      ignoreKeywords: keywords.split(',').map((s) => s.trim()).filter(Boolean),
    })
    if (deviceSecret) setPunyaSecret(true)
    setDeviceSecret('')
    Alert.alert('Tersimpan')
  }

  async function uji() {
    const hasil = await GopayListener.testConnection()
    if (hasil.ok) {
      Alert.alert('Berhasil', `Terhubung sebagai ${hasil.deviceName}`)
      return
    }
    const pesan: Record<string, string> = {
      belum_dikonfigurasi: 'Backend URL, Device ID, dan Device Secret harus diisi lebih dulu.',
      invalid_signature: 'Device ID atau Device Secret salah.',
      clock_skew: 'Jam HP meleset lebih dari 5 menit dari server. Setel jam ke otomatis.',
      device_disabled: 'Device ini dinonaktifkan di backend.',
      network: 'Backend tidak dapat dihubungi.',
    }
    Alert.alert('Gagal', pesan[hasil.error ?? ''] ?? `Kesalahan: ${hasil.error}`)
  }

  return (
    <ScrollView contentContainerStyle={styles.root}>
      <Text style={styles.label}>Backend URL</Text>
      <TextInput
        style={styles.input}
        value={backendUrl}
        onChangeText={setBackendUrl}
        autoCapitalize="none"
        placeholder="https://contoh.com/api/v1"
      />

      <Text style={styles.label}>Device ID</Text>
      <TextInput style={styles.input} value={deviceId} onChangeText={setDeviceId} autoCapitalize="none" />

      <Text style={styles.label}>Device Secret</Text>
      <TextInput
        style={styles.input}
        value={deviceSecret}
        onChangeText={setDeviceSecret}
        secureTextEntry
        autoCapitalize="none"
        placeholder={punyaSecret ? 'tersimpan — isi untuk mengganti' : 'belum diisi'}
      />
      <Text style={styles.bantuan}>
        Secret tidak pernah ditampilkan kembali. Kosongkan bila tidak ingin mengubahnya.
      </Text>

      <Text style={styles.label}>Package yang dipantau</Text>
      <TextInput style={styles.input} value={packages} onChangeText={setPackages} autoCapitalize="none" />

      <Text style={styles.label}>Kata yang diabaikan</Text>
      <TextInput style={styles.input} value={keywords} onChangeText={setKeywords} autoCapitalize="none" />
      <Text style={styles.bantuan}>
        Hanya pengurang noise. Keamanan dijamin backend — bila ragu, biarkan kosong.
      </Text>

      <TouchableOpacity style={styles.tombol} onPress={simpan}>
        <Text style={styles.tombolTeks}>Simpan</Text>
      </TouchableOpacity>

      <TouchableOpacity style={styles.tombolSekunder} onPress={uji}>
        <Text style={styles.tombolSekunderTeks}>Test Connection</Text>
      </TouchableOpacity>

      <TouchableOpacity
        style={styles.tombolSekunder}
        onPress={() => GopayListener.openNotificationAccessSettings()}
      >
        <Text style={styles.tombolSekunderTeks}>Buka Notification Access</Text>
      </TouchableOpacity>

      <View style={styles.pemisah} />

      <TouchableOpacity
        style={styles.tombolBahaya}
        onPress={() =>
          Alert.alert('Hapus semua riwayat?', 'Event yang belum terkirim ikut terhapus.', [
            { text: 'Batal', style: 'cancel' },
            {
              text: 'Hapus',
              style: 'destructive',
              onPress: () => {
                GopayListener.clearHistory()
                Alert.alert('Riwayat dihapus')
              },
            },
          ])
        }
      >
        <Text style={styles.tombolBahayaTeks}>Hapus riwayat event</Text>
      </TouchableOpacity>
    </ScrollView>
  )
}

const styles = StyleSheet.create({
  root: { padding: 16, gap: 8 },
  label: { fontSize: 13, color: '#6b7280', marginTop: 10 },
  input: { borderWidth: 1, borderColor: '#d1d5db', borderRadius: 10, padding: 12, fontSize: 15 },
  bantuan: { fontSize: 12, color: '#9ca3af', lineHeight: 17 },
  tombol: { backgroundColor: '#111827', borderRadius: 12, padding: 16, alignItems: 'center', marginTop: 16 },
  tombolTeks: { color: '#fff', fontSize: 15, fontWeight: '600' },
  tombolSekunder: { borderWidth: 1, borderColor: '#d1d5db', borderRadius: 12, padding: 14, alignItems: 'center' },
  tombolSekunderTeks: { fontSize: 15 },
  pemisah: { height: 24 },
  tombolBahaya: { borderWidth: 1, borderColor: '#fecaca', borderRadius: 12, padding: 14, alignItems: 'center' },
  tombolBahayaTeks: { color: '#dc2626', fontSize: 15 },
})
```

- [ ] **Step 6: Tulis `mobile/src/screens/DebugScreen.tsx`**

```tsx
import { useCallback, useState } from 'react'
import { useFocusEffect } from '@react-navigation/native'
import { Alert, FlatList, StyleSheet, Text, TouchableOpacity, View } from 'react-native'
import GopayListener, { type DiscoveryEntry } from '../../modules/gopay-listener'
import { formatWaktu } from '../lib/format'

export default function DebugScreen() {
  const [entries, setEntries] = useState<DiscoveryEntry[]>([])
  const [aktifSampai, setAktifSampai] = useState(0)

  const muat = useCallback(() => {
    setEntries(GopayListener.getDiscoveryEntries())
    setAktifSampai(GopayListener.getSettings().discoveryUntilMs)
  }, [])
  useFocusEffect(muat)

  const aktif = aktifSampai > Date.now()

  function mulai() {
    Alert.alert(
      'Nyalakan mode Discovery?',
      'Selama aktif, aplikasi mencatat nama package dan judul dari SEMUA notifikasi, ' +
        'termasuk aplikasi lain. Isi notifikasi tidak disimpan dan tidak ada yang dikirim keluar HP. ' +
        'Mati sendiri setelah 10 menit.',
      [
        { text: 'Batal', style: 'cancel' },
        {
          text: 'Nyalakan',
          onPress: () => {
            GopayListener.startDiscovery(10)
            muat()
          },
        },
      ]
    )
  }

  return (
    <FlatList
      contentContainerStyle={styles.root}
      data={entries}
      keyExtractor={(e, i) => `${e.packageName}-${e.seenAt}-${i}`}
      ListHeaderComponent={
        <View style={styles.header}>
          <Text style={styles.status}>
            Mode Discovery: {aktif ? `aktif sampai ${formatWaktu(aktifSampai)}` : 'mati'}
          </Text>

          {aktif ? (
            <TouchableOpacity
              style={styles.tombol}
              onPress={() => {
                GopayListener.stopDiscovery()
                muat()
              }}
            >
              <Text style={styles.tombolTeks}>Matikan sekarang</Text>
            </TouchableOpacity>
          ) : (
            <TouchableOpacity style={styles.tombol} onPress={mulai}>
              <Text style={styles.tombolTeks}>Nyalakan 10 menit</Text>
            </TouchableOpacity>
          )}

          <TouchableOpacity style={styles.tombolSekunder} onPress={muat}>
            <Text style={styles.tombolSekunderTeks}>Muat ulang</Text>
          </TouchableOpacity>

          <TouchableOpacity
            style={styles.tombolSekunder}
            onPress={() => {
              GopayListener.clearDiscovery()
              muat()
            }}
          >
            <Text style={styles.tombolSekunderTeks}>Bersihkan catatan</Text>
          </TouchableOpacity>
        </View>
      }
      ListEmptyComponent={<Text style={styles.kosong}>Belum ada catatan.</Text>}
      renderItem={({ item }) => (
        <View style={styles.kartu}>
          <Text style={styles.pkg}>{item.packageName}</Text>
          <Text style={styles.judul}>{item.title || '(tanpa judul)'}</Text>
          <Text style={styles.waktu}>{formatWaktu(item.seenAt)}</Text>
        </View>
      )}
    />
  )
}

const styles = StyleSheet.create({
  root: { padding: 16, gap: 8 },
  header: { gap: 8, marginBottom: 8 },
  status: { fontSize: 14, marginBottom: 4 },
  kosong: { textAlign: 'center', color: '#6b7280', marginTop: 24 },
  kartu: { backgroundColor: '#f9fafb', borderRadius: 10, padding: 12, gap: 2 },
  pkg: { fontSize: 14, fontWeight: '600' },
  judul: { fontSize: 13 },
  waktu: { fontSize: 12, color: '#6b7280' },
  tombol: { backgroundColor: '#111827', borderRadius: 12, padding: 14, alignItems: 'center' },
  tombolTeks: { color: '#fff', fontSize: 15, fontWeight: '600' },
  tombolSekunder: { borderWidth: 1, borderColor: '#d1d5db', borderRadius: 12, padding: 12, alignItems: 'center' },
  tombolSekunderTeks: { fontSize: 14 },
})
```

- [ ] **Step 7: Tulis `mobile/src/navigation/index.tsx` dan sederhanakan `App.tsx`**

`mobile/src/navigation/index.tsx`:

```tsx
import { NavigationContainer } from '@react-navigation/native'
import { createBottomTabNavigator } from '@react-navigation/bottom-tabs'
import DashboardScreen from '../screens/DashboardScreen'
import HistoryScreen from '../screens/HistoryScreen'
import SettingsScreen from '../screens/SettingsScreen'
import DebugScreen from '../screens/DebugScreen'

const Tab = createBottomTabNavigator()

export default function Navigation() {
  return (
    <NavigationContainer>
      <Tab.Navigator screenOptions={{ headerShown: true }}>
        <Tab.Screen name="Dashboard" component={DashboardScreen} />
        <Tab.Screen name="Riwayat" component={HistoryScreen} />
        <Tab.Screen name="Pengaturan" component={SettingsScreen} />
        <Tab.Screen name="Debug" component={DebugScreen} />
      </Tab.Navigator>
    </NavigationContainer>
  )
}
```

`mobile/App.tsx`:

```tsx
import { SafeAreaProvider } from 'react-native-safe-area-context'
import Navigation from './src/navigation'

export default function App() {
  return (
    <SafeAreaProvider>
      <Navigation />
    </SafeAreaProvider>
  )
}
```

- [ ] **Step 8: Periksa tipe TypeScript**

```bash
cd mobile && npx tsc --noEmit
```

Expected: tanpa error.

- [ ] **Step 9: Build, pasang, dan telusuri keempat layar**

```bash
cd mobile && APP_VARIANT=development npx expo run:android
```

Expected: keempat tab terbuka tanpa crash. Dashboard menampilkan tiga indikator status.

- [ ] **Step 10: Commit**

```bash
git add mobile/src mobile/App.tsx
git commit -m "feat(mobile): dashboard, riwayat, pengaturan, dan layar debug"
```

---

### Task 11: Retensi, uji ketahanan, dan QA

**Files:**
- Create: `.../gopaylistener/work/PurgeWorker.kt`
- Modify: `GopayListenerModule.kt`
- Modify: `docs/qa/qa-report.md`

**Interfaces:**
- Consumes: `EventDao.purgeOlderThan`.
- Produces: `PurgeWorker`, penjadwalan harian saat module dimuat.

- [ ] **Step 1: Tulis `PurgeWorker.kt`**

```kotlin
package expo.modules.gopaylistener.work

import android.content.Context
import androidx.work.CoroutineWorker
import androidx.work.ExistingPeriodicWorkPolicy
import androidx.work.PeriodicWorkRequestBuilder
import androidx.work.WorkManager
import androidx.work.WorkerParameters
import expo.modules.gopaylistener.db.AppDatabase
import java.util.concurrent.TimeUnit

/**
 * Menghapus event lama. Hanya SENT dan IGNORED yang didaur ulang —
 * FAILED tidak pernah dihapus otomatis karena ia menandai sesuatu yang
 * perlu dilihat manusia.
 *
 * Teks notifikasi transfer pribadi memuat nama pengirim, jadi retensi
 * ini juga soal privasi, bukan sekadar ruang penyimpanan.
 */
class PurgeWorker(context: Context, params: WorkerParameters) : CoroutineWorker(context, params) {

    override suspend fun doWork(): Result {
        val cutoff = System.currentTimeMillis() - RETENTION_MS
        AppDatabase.get(applicationContext).events().purgeOlderThan(cutoff)
        return Result.success()
    }

    companion object {
        private const val WORK_NAME = "event-purge"
        private const val RETENTION_MS = 30L * 24 * 60 * 60 * 1000

        fun schedule(context: Context) {
            val request = PeriodicWorkRequestBuilder<PurgeWorker>(1, TimeUnit.DAYS).build()
            WorkManager.getInstance(context.applicationContext).enqueueUniquePeriodicWork(
                WORK_NAME,
                ExistingPeriodicWorkPolicy.KEEP,
                request
            )
        }
    }
}
```

- [ ] **Step 2: Jadwalkan saat module dimuat — `GopayListenerModule.kt`**

Tambahkan di dalam `ModuleDefinition`, setelah `Name("GopayListener")`:

```kotlin
        OnCreate {
            PurgeWorker.schedule(context)
        }
```

- [ ] **Step 3: Verifikasi penjadwalan**

```bash
cd mobile && APP_VARIANT=development npx expo run:android
adb shell dumpsys jobscheduler | grep -i gopaybridge | head -20
```

Expected: keluaran memuat job milik `id.akbarryyan.gopaybridge.dev`.

- [ ] **Step 4: Jalankan seluruh test otomatis sekali lagi**

```bash
cd mobile/android && ./gradlew :gopay-listener:testDebugUnitTest
cd ../ && npx tsc --noEmit
cd ../backend && make test
```

Expected: seluruhnya PASS. Salin keluarannya — ini bukti untuk laporan QA.

- [ ] **Step 5: `NEEDS-DEVICE` — uji ketahanan M6**

Kerjakan berurutan, catat hasil tiap butir:

| # | Langkah | Hasil yang diharapkan |
|---|---|---|
| 1 | Tutup aplikasi dari recent apps, minta transfer Rp1 | Event sampai ke `/events` di VPS tanpa membuka aplikasi |
| 2 | Restart HP, tunggu 2 menit, minta transfer Rp1 | Event tetap sampai tanpa membuka aplikasi |
| 3 | Mode pesawat menyala, picu notifikasi, lalu matikan mode pesawat | Event berpindah `PENDING` → `SENT` sendiri |
| 4 | Matikan backend di VPS, minta transfer, nyalakan lagi | Event `PENDING` lalu `SENT` setelah backend hidup |
| 5 | Ubah Device Secret jadi salah, minta transfer | Event `FAILED` dengan `invalid_signature`, **tidak** ada retry berulang |
| 6 | Setel jam HP mundur 10 menit, minta transfer | Dashboard memperingatkan jam meleset; event `FAILED` dengan `clock_skew` |
| 7 | Kirim notifikasi WhatsApp | Tidak ada event tercatat sama sekali |
| 8 | Minta dua transfer Rp1 berturut-turut | Dua event terpisah, keduanya `accepted` — bukan satu |
| 9 | **Diamkan HP semalaman**, pagi harinya minta transfer Rp1 | Event tetap sampai. Ini uji ColorOS yang sebenarnya |
| 10 | Cabut Notification Access lalu berikan lagi | Dashboard menunjukkan perubahan; listener terikat kembali |

Butir 9 yang paling menentukan. Bila gagal, keputusan "tanpa foreground service" di [spec §6.3](../specs/2026-09-10-ingestion-and-android-bridge-design.md) tidak bertahan dan foreground service harus ditambahkan — catat sebagai `FAIL` dan laporkan, jangan diperbaiki diam-diam.

- [ ] **Step 6: Jalankan QA milestone M2–M6**

Ikuti [`docs/qa/qa-rules.md`](../../qa/qa-rules.md) dan perbarui [`docs/qa/qa-report.md`](../../qa/qa-report.md).

Butir yang harus berubah status pada siklus ini:

| Butir | Bukti |
|---|---|
| FR-01 Notification Access | Uji ketahanan butir 10 |
| FR-02 Notification Listener | Uji ketahanan butir 1 |
| FR-03 GoPay filtering | Uji ketahanan butir 7 |
| FR-04 Notification parsing | Keluaran `AmountParserTest` + `EventIdBuilderTest` |
| FR-05 Backend request HTTPS | Event muncul di `/events` lewat domain HTTPS |
| FR-07 Retry | Uji ketahanan butir 3 dan 4 |
| FR-09 Event logging | Layar Riwayat menampilkan status dan `attemptCount` |
| FR-10 Service status | Dashboard menampilkan status ikatan sebenarnya |
| Baris `429`, `5xx`, `timeout` di tabel kode → tindakan | Keluaran `ResponseMapperTest` |
| Listener berjalan di background | Uji ketahanan butir 1, 2, 9 |
| Dapat diuji saat UI tidak terbuka | Uji ketahanan butir 1 |
| Diuji pada physical Android device | Seluruh uji ketahanan |
| Channel "Promotions and Marketing" masih aktif | Screenshot pengaturan notifikasi GoPay |
| Setup ColorOS selesai | Task 2 Step 12 |
| Mode Discovery default mati, tidak mengirim keluar | Layar Debug + tidak ada event non-GoPay di `/events` |

- [ ] **Step 7: Commit**

```bash
git add mobile/modules/gopay-listener/ docs/qa/qa-report.md
git commit -m "feat(mobile): retensi 30 hari dan laporan QA M2-M6"
```

---

## Self-Review

**Spec coverage.** §2.1 Kotlin memiliki pipeline (Task 8, 9); §2.2 parsing display-only (Task 3); §2.3 daftar kata-diabaikan default kosong (Task 7); §2.4 package configurable + Discovery (Task 7, 8); §2.5 HMAC (Task 5, 9); §2.6 data lapangan dipakai sebagai kasus uji (Task 3, 4); §3.3 Expo + local module + config plugin (Task 1, 2); §4.1 formula `event_id` (Task 4); §4.2 tabel Room (Task 6); §4.3 lima status (Task 6); §4.5 retensi 30 hari dan batas retry (Task 6, 9, 11); §6.1 lima berkas Kotlin (Task 3–9); §6.3 tanpa foreground service dan tanpa `BOOT_COMPLETED` (Task 11 butir 2 dan 9 yang mengujinya); §6.4 ColorOS (Task 2 Step 12); §6.5 empat layar (Task 10); §6.6 cleartext hanya untuk host dev (Task 2 Step 7).

**Placeholder scan.** Satu-satunya nilai yang sengaja dikosongkan adalah `GANTI_DENGAN_HASIL_STEP_1` di `SignerTest`, dan itu memang harus dihasilkan dari implementasi Go — mengarangnya akan membuat test lulus terhadap nilai yang salah. Alamat LAN `192.168.1.10` disertai perintah untuk mencarinya.

**Type consistency.** `EventStatus` memakai nama yang sama di Kotlin, SQL literal di `EventDao`, dan union TypeScript. `amountHint` konsisten di Room, payload JSON (`amount_hint`), dan tipe TS. `Settings.config()` mengembalikan `BridgeConfig?` dan dipakai dengan pola yang sama di Task 9 Step 8 dan Step 9. `UploadScheduler.enqueue(context)` dipanggil dari `GoPayListenerService` (Task 8) dan `retryEvent` (Task 8) dengan tanda tangan yang sama.

**Ketergantungan silang yang perlu diperhatikan.** Task 8 memakai `UploadScheduler` yang baru ditulis di Task 9 Step 7. Bila dikerjakan berurutan, build di Task 8 Step 5 akan gagal — Task 8 Step 5 sudah memuat instruksi untuk mengerjakan Task 9 Step 7 lebih dulu lalu kembali. Alternatifnya, kerjakan Task 9 sebelum Task 8; keduanya sah.
