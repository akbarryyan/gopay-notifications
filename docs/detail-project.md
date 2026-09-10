# Detail Project — GoPay Notification Bridge

## 1. Project Overview

**Project Name:** GoPay Notification Bridge
**Platform:** Android
**Framework:** React Native
**Language:** TypeScript
**Native Language:** Kotlin
**Architecture:** React Native + Native Android Service
**Target:** Android device yang menerima notifikasi GoPay

GoPay Notification Bridge adalah aplikasi Android yang menerima notifikasi dari aplikasi GoPay menggunakan Android `NotificationListenerService`, memproses informasi notifikasi menjadi event terstruktur, kemudian mengirimkan event tersebut ke backend melalui HTTPS API.

React Native digunakan sebagai framework utama untuk UI dan application layer, sedangkan fitur `NotificationListenerService` dibuat menggunakan native Android/Kotlin dan dijembatani ke React Native melalui Native Module/Event Emitter.

---

# 2. Technology Stack

## 2.1 Mobile Application

### React Native

Digunakan sebagai framework utama aplikasi Android.

Alasan:

* Development menggunakan TypeScript.
* UI lebih mudah dikembangkan.
* Struktur project dapat dipisahkan antara UI dan native functionality.
* Memungkinkan pengembangan platform lain di masa depan jika diperlukan.

### TypeScript

Digunakan sebagai bahasa utama pada React Native layer.

Digunakan untuk:

* UI.
* State management.
* API client.
* Type definitions.
* Notification event model.
* Configuration.
* Application logic.

### Kotlin

Digunakan untuk native Android functionality.

Kotlin bertanggung jawab terhadap:

* `NotificationListenerService`.
* Membaca notification event dari Android.
* Filtering package GoPay.
* Native-to-React Native communication.
* Background notification handling.
* Android-specific functionality.

---

# 3. Supporting Technologies

## 3.1 Navigation

Gunakan:

**React Navigation**

Untuk:

* Dashboard
* Event History
* Settings

---

## 3.2 HTTP Client

Gunakan:

**Axios**

Digunakan untuk komunikasi:

```text
React Native
      ↓
HTTPS
      ↓
Backend API
```

Request harus memiliki:

* Authentication credential.
* Device identifier.
* Event identifier.
* Timestamp.
* Notification payload.

---

## 3.3 Local Storage

Gunakan:

**MMKV**

Untuk data konfigurasi ringan seperti:

* Backend URL.
* Device ID.
* Authentication configuration.
* Application preferences.
* Listener settings.

Data sensitif tidak boleh disimpan secara sembarangan.

Untuk credential/token yang membutuhkan perlindungan lebih tinggi, gunakan secure storage Android yang sesuai.

---

## 3.4 Local Database

Gunakan:

**SQLite**

dengan abstraction layer yang sesuai untuk React Native.

Database digunakan untuk menyimpan event yang membutuhkan persistence, terutama event yang belum berhasil dikirim ke backend.

Contoh status:

```text
PENDING
SENDING
SENT
FAILED
```

---

# 4. Architecture

Arsitektur aplikasi menggunakan pendekatan:

```text
┌──────────────────────────────────────┐
│           React Native App           │
│                                      │
│  ┌────────────┐   ┌───────────────┐  │
│  │ UI Layer   │   │ Application   │  │
│  │            │   │ Logic         │  │
│  └────────────┘   └───────┬───────┘  │
│                           │          │
│  ┌────────────────────────▼───────┐  │
│  │ API / Repository / Local Store  │  │
│  └─────────────────────────────────┘  │
│                                      │
└──────────────────┬───────────────────┘
                   │
                   │ Native Module
                   ▼
┌──────────────────────────────────────┐
│          Android Native Layer        │
│                                      │
│  NotificationListenerService         │
│              │                       │
│              ▼                       │
│       GoPay Notification              │
│            Filter                     │
└──────────────────┬───────────────────┘
                   │
                   ▼
              Event Bridge
                   │
                   ▼
             React Native
                   │
                   ▼
              Backend API
```

---

# 5. Core Architecture Principle

Aplikasi dibagi menjadi dua layer utama:

### React Native Layer

Bertanggung jawab terhadap:

* UI.
* Configuration.
* API communication.
* Local event management.
* Event history.
* Application state.

### Native Android Layer

Bertanggung jawab terhadap:

* Notification Listener.
* Android service lifecycle.
* Notification extraction.
* Package filtering.
* Native Android event handling.

Prinsip:

> Native Android menangkap notifikasi, React Native mengelola aplikasi.

---

# 6. Native Android Notification Listener

Implementasikan:

```text
NotificationListenerService
```

Contoh konsep:

```text
Android System
      │
      │ onNotificationPosted()
      ▼
NotificationListenerService
      │
      ▼
Check package name
      │
      ├── Not GoPay → Ignore
      │
      └── GoPay
            │
            ▼
       Extract Payload
            │
            ▼
       Create Event
            │
            ▼
       Event Bridge
```

Service harus menangani event ketika notifikasi baru muncul.

---

# 7. GoPay Package Filtering

Filtering harus dilakukan berdasarkan package identifier aplikasi GoPay.

Konsep:

```text
notification.packageName
        ↓
Is package GoPay?
        ↓
     YES
        ↓
Process
```

Jangan menggunakan:

```text
title == "GoPay"
```

sebagai mekanisme utama filtering karena title notification dapat berubah.

Package identifier harus dikonfirmasi melalui perangkat Android target sebelum implementasi final.

Package identifier juga harus dibuat configurable pada native layer apabila diperlukan untuk maintenance.

---

# 8. Notification Extraction

Native service mengambil data yang tersedia dari Android notification.

Minimal data model:

```typescript
interface RawNotificationEvent {
  id: string;
  packageName: string;
  title?: string;
  text?: string;
  bigText?: string;
  timestamp: number;
}
```

Tidak semua field selalu tersedia.

Karena itu seluruh field yang berasal dari notification harus dianggap optional.

---

# 9. Notification Parser

Setelah native service menerima notification:

```text
Raw Android Notification
          ↓
Notification Parser
          ↓
Structured Payment Event
```

Parser harus dipisahkan dari Notification Listener agar perubahan format notifikasi tidak memengaruhi service utama.

Contoh:

```typescript
interface PaymentNotification {
  eventId: string;
  source: 'gopay';
  title?: string;
  message?: string;
  amount?: number;
  receivedAt: string;
}
```

Parser bertugas melakukan ekstraksi informasi yang memang tersedia.

---

# 10. Amount Parsing

Jika nominal tersedia pada notifikasi, parser dapat mencoba mengambil nominal.

Contoh:

```text
"Pembayaran diterima Rp25.000"
```

menjadi:

```text
amount: 25000
```

Parser harus menangani format nominal Indonesia seperti:

```text
Rp25.000
Rp 25.000
Rp25,000
```

Namun parser tidak boleh menganggap bahwa setiap angka yang ditemukan pasti merupakan nominal transaksi.

Jika nominal tidak dapat dipastikan:

```text
amount: null
```

Event tetap dapat diteruskan sebagai raw notification event.

---

# 11. Event ID / Idempotency

Setiap notification event harus mempunyai identifier.

Jika Android tidak memberikan identifier yang dapat diandalkan, aplikasi dapat menghasilkan deterministic event ID berdasarkan kombinasi data notification.

Contoh konsep:

```text
hash(
    packageName +
    title +
    message +
    timestamp
)
```

Tujuan:

```text
Notification
     ↓
Event ID
     ↓
Backend
     ↓
Already processed?
     ├── YES → Ignore
     └── NO  → Process
```

Backend tetap menjadi pihak terakhir yang menjamin idempotency.

---

# 12. React Native Native Module

Buat native module untuk komunikasi Kotlin ↔ React Native.

Konsep:

```text
Kotlin
NotificationListenerService
        │
        ▼
Native Module
        │
        ▼
React Native Event Emitter
        │
        ▼
TypeScript
```

React Native dapat menerima event seperti:

```typescript
NotificationReceived
```

Payload:

```typescript
interface NotificationReceivedEvent {
  eventId: string;
  packageName: string;
  title?: string;
  text?: string;
  timestamp: number;
}
```

---

# 13. Background Processing

Notification Listener harus tetap dapat menerima notification event ketika UI React Native tidak sedang terbuka.

Karena itu:

**NotificationListenerService tidak boleh bergantung pada React Native UI lifecycle.**

Flow:

```text
App UI CLOSED
      │
      ▼
Android Service tetap aktif
      │
      ▼
GoPay Notification
      │
      ▼
Native Service
      │
      ▼
Persist Event
      │
      ▼
Process / Forward
```

Ini merupakan bagian penting dari architecture.

---

# 14. Event Queue

Event yang diterima harus masuk ke local queue sebelum atau selama proses pengiriman ke backend.

Contoh:

```text
Notification
     ↓
Create Event
     ↓
Save Local DB
     ↓
PENDING
     ↓
Send Backend
     ↓
SUCCESS
     ↓
SENT
```

Dengan pendekatan ini, event tidak langsung hilang ketika jaringan bermasalah.

---

# 15. Retry Mechanism

Jika backend gagal diakses:

```text
PENDING
   ↓
Attempt #1
   ↓
FAILED
   ↓
Attempt #2
   ↓
FAILED
   ↓
Attempt #3
   ↓
SUCCESS
```

Gunakan exponential backoff.

Contoh konsep:

```text
1st retry → 5 seconds
2nd retry → 15 seconds
3rd retry → 30 seconds
4th retry → 60 seconds
```

Nilai final dapat disesuaikan pada tahap implementasi.

Retry hanya dilakukan untuk error yang bersifat temporary, seperti:

* Network unavailable.
* Timeout.
* Server unavailable.

Authentication error atau malformed request tidak perlu di-retry tanpa perubahan konfigurasi.

---

# 16. Backend API

Endpoint yang digunakan Android:

```text
POST /api/callback/gopay
```

Contoh request:

```json
{
  "event_id": "evt_01ABC",
  "device_id": "device_01ABC",
  "source": "gopay",
  "notification": {
    "title": "Pembayaran diterima",
    "text": "Rp25.000",
    "timestamp": 1789036200000
  },
  "payment": {
    "amount": 25000
  },
  "received_at": "2026-09-10T19:30:00+07:00"
}
```

Format final API harus disesuaikan dengan backend yang digunakan.

---

# 17. API Authentication

Android harus melakukan authentication ketika mengirim callback.

Minimal:

```text
Authorization
Device Identifier
Event Identifier
Timestamp
```

Backend harus memvalidasi:

1. Device terdaftar.
2. Credential valid.
3. Request memiliki format valid.
4. Timestamp masih dalam batas yang diperbolehkan.
5. Event ID belum pernah diproses.

Credential tidak boleh ditanam langsung sebagai hardcoded secret di source code.

---

# 18. HTTPS

Seluruh komunikasi backend wajib menggunakan:

```text
HTTPS
```

Jangan menggunakan:

```text
http://
```

untuk production.

Development dapat menggunakan local environment sesuai kebutuhan, tetapi credential production tidak boleh digunakan pada environment development.

---

# 19. Backend Response

Backend memberikan response yang jelas.

Contoh:

```json
{
  "success": true,
  "event_id": "evt_01ABC",
  "status": "accepted"
}
```

Jika event duplicate:

```json
{
  "success": true,
  "event_id": "evt_01ABC",
  "status": "duplicate"
}
```

Android dapat memperlakukan `duplicate` sebagai event yang sudah berhasil diproses.

---

# 20. Transaction Matching

Android **tidak bertanggung jawab** menentukan transaksi website mana yang dibayar.

Android hanya mengirim event.

Backend melakukan:

```text
GoPay Event
     ↓
Validation
     ↓
Amount Extraction
     ↓
Find Pending Transaction
     ↓
Transaction Matching
     ↓
Business Rule
     ↓
Update Payment
```

Contoh:

```text
Order #123
Expected Amount: Rp25.000
Status: PENDING

GoPay Notification
Amount: Rp25.000

        ↓

Backend Matching

        ↓

Order #123
Status: PAID
```

Jika terdapat lebih dari satu transaksi dengan nominal sama, backend harus memiliki mekanisme tambahan untuk menentukan transaksi yang benar.

---

# 21. Dashboard

Dashboard React Native terdiri dari beberapa status.

### Notification Access

```text
Notification Access
● Enabled
```

atau:

```text
Notification Access
○ Disabled

[Enable Notification Access]
```

Button akan membuka Android Settings untuk Notification Access.

---

### Listener Status

```text
Listener
● Running
```

Status harus menunjukkan kondisi sebenarnya dari native service.

---

### Backend Status

```text
Backend
● Connected
```

Status dapat diperoleh melalui health check endpoint.

---

### Last Event

Menampilkan event terakhir yang diterima:

```text
Last Event

GoPay
Rp25.000
10 Sep 2026, 19:30
```

---

# 22. Event History

Screen:

```text
Event History
```

Setiap event memiliki:

* Event ID.
* Waktu.
* Source.
* Notification title.
* Notification message.
* Amount jika tersedia.
* Status.
* Retry count.

Contoh:

```text
┌─────────────────────────────┐
│ Rp25.000                    │
│ Pembayaran diterima         │
│ 10 Sep 2026 • 19:30         │
│                             │
│ ✓ SENT                       │
└─────────────────────────────┘
```

---

# 23. Settings

Settings minimal:

### Backend

```text
Backend URL
[ https://example.com/api ]
```

### Device

```text
Device ID
[ device_01ABC ]
```

### Connection

```text
[ Test Connection ]
```

### Notification

```text
[ Open Notification Access ]
```

### Logs

```text
[ Clear Event History ]
```

---

# 24. Project Structure

Struktur awal project:

```text
gopay-notification-bridge/
│
├── android/
│   └── app/
│       └── src/
│           └── main/
│               ├── java/
│               │   └── ...
│               │
│               └── AndroidManifest.xml
│
├── src/
│   ├── components/
│   ├── screens/
│   │   ├── Dashboard/
│   │   ├── History/
│   │   └── Settings/
│   │
│   ├── navigation/
│   ├── services/
│   │   ├── api/
│   │   ├── notification/
│   │   └── sync/
│   │
│   ├── storage/
│   ├── hooks/
│   ├── types/
│   ├── utils/
│   └── constants/
│
├── docs/
│   ├── prd.md
│   └── detail-project.md
│
├── package.json
├── tsconfig.json
└── README.md
```

Native Kotlin dapat memiliki struktur seperti:

```text
android/app/src/main/java/.../
│
├── notification/
│   ├── GoPayNotificationListenerService.kt
│   ├── NotificationParser.kt
│   └── NotificationEvent.kt
│
└── bridge/
    └── NotificationModule.kt
```

Struktur final dapat disesuaikan dengan React Native architecture yang digunakan.

---

# 25. Application Layers

## Presentation Layer

```text
screens/
components/
hooks/
```

Bertanggung jawab terhadap UI.

---

## Application Layer

```text
services/
```

Bertanggung jawab terhadap:

* API communication.
* Notification event handling.
* Synchronization.
* Retry.

---

## Data Layer

```text
storage/
```

Bertanggung jawab terhadap:

* Local database.
* Event persistence.
* Configuration.

---

## Native Layer

```text
android/
```

Bertanggung jawab terhadap:

* Notification Listener.
* Android Service.
* Native module.

---

# 26. State Management

Untuk MVP, gunakan state management yang sederhana.

Global state hanya digunakan untuk data seperti:

```text
notificationAccessStatus
listenerStatus
backendStatus
lastEvent
```

Jangan memasukkan seluruh local database ke global state.

Event history tetap diambil dari local storage/database.

---

# 27. Notification Lifecycle

Lifecycle utama:

```text
RECEIVED
   ↓
PARSED
   ↓
PERSISTED
   ↓
QUEUED
   ↓
SENDING
   ↓
SENT
```

Jika gagal:

```text
SENDING
   ↓
FAILED
   ↓
RETRY_PENDING
   ↓
SENDING
```

Jika tidak dapat diproses:

```text
RECEIVED
   ↓
PARSE_FAILED
```

---

# 28. Duplicate Handling

Duplicate handling dilakukan pada dua level.

### Android

Mencegah event yang sama masuk ke queue berulang kali.

### Backend

Backend wajib memiliki idempotency protection sebagai lapisan terakhir.

Contoh:

```text
event_id = evt_123

Request #1 → ACCEPTED
Request #2 → DUPLICATE
Request #3 → DUPLICATE
```

Backend tidak boleh menjalankan business logic pembayaran lebih dari satu kali.

---

# 29. Device Identification

Setiap instalasi aplikasi memiliki:

```text
device_id
```

Contoh:

```text
device_01HXYZ...
```

Device ID digunakan untuk:

* Identifikasi Android device.
* Authentication.
* Monitoring.
* Debugging.
* Multi-device support di masa depan.

Jangan menggunakan IMEI atau hardware identifier sensitif yang tidak diperlukan.

---

# 30. Android Permission

Aplikasi membutuhkan permission dan konfigurasi Android yang diperlukan untuk:

* Notification Listener.
* Internet.
* Background functionality sesuai kebutuhan Android version.

Notification Access bukan permission runtime biasa dan harus diaktifkan user melalui Android Settings.

Aplikasi harus memberikan penjelasan kepada user sebelum meminta akses tersebut.

---

# 31. Android Boot Handling

Jika diperlukan, aplikasi harus dapat mendeteksi device restart dan memastikan konfigurasi service kembali berjalan sesuai mekanisme Android.

Flow:

```text
Device Restart
      ↓
Android Boot
      ↓
Application / Service Initialization
      ↓
Notification Listener Ready
```

Implementasi harus mengikuti restriction background Android pada versi target.

---

# 32. Battery Considerations

Aplikasi harus dibuat hemat battery.

Notification Listener tidak boleh melakukan:

* Polling notification secara terus-menerus.
* Network request berkala tanpa kebutuhan.
* Heavy processing.
* Background loop yang tidak diperlukan.

Event processing dilakukan ketika notification event diterima.

---

# 33. Security Requirements

### Mobile

* Jangan hardcode production secret.
* Gunakan HTTPS.
* Simpan credential menggunakan secure storage.
* Jangan mencatat credential ke log.
* Jangan menyimpan notification yang tidak diperlukan.
* Batasi data yang disimpan pada local database.

### Backend

* Authentication wajib.
* Validate payload.
* Rate limiting.
* Idempotency.
* Device authorization.
* Request timestamp validation.
* Audit log.

---

# 34. Logging

Gunakan log level:

```text
DEBUG
INFO
WARNING
ERROR
```

Contoh:

```text
INFO
GoPay notification received

INFO
Event evt_123 queued

INFO
Event evt_123 sent successfully

ERROR
Backend request failed
```

Jangan log:

* Authentication token.
* Secret key.
* Data sensitif yang tidak diperlukan.

---

# 35. Testing Strategy

## Unit Test

Test:

* Notification parser.
* Amount parser.
* Event ID generator.
* Duplicate detection.
* Retry calculation.

Contoh:

```text
"Pembayaran Rp25.000"
        ↓
25000
```

---

## Integration Test

Test:

```text
Notification
   ↓
Native Service
   ↓
React Native
   ↓
Local DB
   ↓
API
```

---

## Manual Android Test

Test pada perangkat Android nyata:

1. Notification Access disabled.
2. Notification Access enabled.
3. GoPay notification received.
4. Non-GoPay notification received.
5. Internet disconnected.
6. Internet reconnect.
7. Backend unavailable.
8. Backend returns authentication error.
9. Duplicate notification.
10. Device restart.
11. App UI closed.
12. App process restarted.

---

# 36. Development Environment

Recommended:

```text
Node.js
React Native CLI
TypeScript
Android Studio
Android SDK
JDK
Gradle
Kotlin
Git
```

Development utama menggunakan Android device/emulator.

Namun untuk pengujian notification listener, **physical Android device lebih disarankan** karena notifikasi GoPay harus benar-benar diterima oleh perangkat.

---

# 37. Development Phases

## Phase 1 — Project Initialization

* Create React Native project.
* Configure TypeScript.
* Configure Android.
* Setup project structure.
* Setup navigation.
* Setup basic UI.

---

## Phase 2 — Notification Listener

* Implement native Kotlin service.
* Register Notification Listener.
* Detect notification.
* Filter GoPay package.
* Extract notification payload.

---

## Phase 3 — React Native Bridge

* Implement Native Module.
* Implement event emitter.
* Receive notification event in TypeScript.
* Display last notification on dashboard.

---

## Phase 4 — Local Persistence

* Setup SQLite.
* Create event table.
* Save incoming events.
* Implement event history.

---

## Phase 5 — Backend Integration

* Create API client.
* Configure backend URL.
* Implement authentication.
* Send notification event.
* Handle response.

---

## Phase 6 — Retry & Reliability

* Implement queue.
* Implement retry.
* Implement exponential backoff.
* Handle network failure.
* Handle duplicate events.

---

## Phase 7 — Dashboard & Settings

* Notification Access status.
* Listener status.
* Backend status.
* Last event.
* Settings.
* Event history.

---

## Phase 8 — Testing

* Unit tests.
* Integration tests.
* Physical device testing.
* Background testing.
* Device restart testing.
* Network failure testing.

---

# 38. MVP Definition

MVP dianggap selesai apabila aplikasi dapat melakukan:

```text
GoPay Notification
       ↓
Android Notification Listener
       ↓
Filter GoPay
       ↓
Extract Data
       ↓
Create Event ID
       ↓
Save Local
       ↓
POST Backend
       ↓
Backend Accept
       ↓
Mark SENT
```

Dan ketika jaringan gagal:

```text
GoPay Notification
       ↓
Save Local
       ↓
PENDING
       ↓
Network Recovery
       ↓
Retry
       ↓
SENT
```

---

# 39. Important Technical Constraints

1. Aplikasi hanya dapat membaca informasi yang tersedia pada Android notification.
2. Aplikasi tidak mengakses database internal GoPay.
3. Aplikasi tidak mengakses API internal GoPay.
4. Notification format GoPay dapat berubah.
5. Notification Listener membutuhkan user authorization.
6. Android dapat menerapkan background restrictions.
7. Backend harus menjadi sumber keputusan final mengenai status pembayaran.
8. Notification event harus diproses secara idempotent.
9. Tidak semua notification event dapat dipastikan sebagai payment event.
10. Tidak boleh menganggap setiap nominal yang muncul dalam notification sebagai nominal pembayaran.

---

# 40. Recommended Implementation Principle

Gunakan prinsip:

```text
KEEP NATIVE PART SMALL
```

Sebisa mungkin:

```text
Kotlin
  ↓
Capture notification
  ↓
Send structured event
  ↓
React Native
```

Sedangkan:

```text
React Native
  ↓
Storage
  ↓
API
  ↓
Retry
  ↓
UI
  ↓
Configuration
```

Dengan begitu native Android code tidak menjadi terlalu besar dan business logic tetap mudah dikembangkan menggunakan TypeScript.

---

# 41. Future Multi-Source Architecture

Walaupun MVP hanya GoPay, arsitektur sebaiknya tidak terlalu mengunci seluruh sistem pada GoPay.

Gunakan konsep:

```typescript
type NotificationSource =
  | 'gopay';
```

Di masa depan dapat berkembang menjadi:

```typescript
type NotificationSource =
  | 'gopay'
  | 'other_payment_source';
```

Parser dapat menggunakan strategy berdasarkan source:

```text
Notification
     ↓
Source Detector
     ↓
GoPay Parser
     ↓
Structured Event
```

Namun fitur multi-source **tidak perlu diimplementasikan pada MVP**.

---

# 42. Definition of Done

Project MVP dianggap selesai apabila:

* [ ] React Native application berjalan pada Android.
* [ ] TypeScript digunakan sebagai application language.
* [ ] Kotlin digunakan untuk Notification Listener.
* [ ] Notification Access dapat diaktifkan.
* [ ] Notification Listener dapat berjalan di background.
* [ ] Notifikasi GoPay dapat terdeteksi.
* [ ] Notifikasi aplikasi lain diabaikan.
* [ ] Data notification dapat diekstrak.
* [ ] Event ID dibuat.
* [ ] Event tersimpan secara lokal.
* [ ] Event dapat dikirim ke backend.
* [ ] HTTPS digunakan untuk production.
* [ ] Authentication diterapkan.
* [ ] Retry mechanism berjalan.
* [ ] Duplicate event dapat ditangani.
* [ ] Dashboard menampilkan listener status.
* [ ] Dashboard menampilkan backend status.
* [ ] History event tersedia.
* [ ] Device ID tersedia.
* [ ] Credential tidak hardcoded.
* [ ] App dapat menangani network failure.
* [ ] App dapat diuji ketika UI tidak terbuka.
* [ ] App telah diuji pada physical Android device.
