# Self-Hosted + Annual License

## 1. Overview

Model bisnis **Self-Hosted + Annual License** memungkinkan customer menjalankan seluruh sistem Payment Notification Bridge pada infrastructure milik mereka sendiri, sementara hak penggunaan software dikontrol melalui lisensi tahunan.

Customer tidak bergantung pada server utama milik vendor untuk menjalankan sistem operasional sehari-hari.

Vendor menyediakan:

- Android Payment Notification Bridge
- Backend
- Dashboard
- Database schema/migration
- Deployment package
- License management
- Documentation
- Software updates
- Technical support sesuai paket

Customer menyediakan:

- VPS/server
- Domain/subdomain
- Android device
- Internet connection
- Payment application/account
- Infrastructure pendukung lainnya

---

# 2. Product Positioning

Produk tidak diposisikan sebagai:

> "APK pembaca notifikasi GoPay."

Produk diposisikan sebagai:

> **Payment Notification Automation Platform yang dapat di-deploy secara mandiri pada infrastructure customer.**

Fungsi utama:

```text
Payment Application
       │
       │ Android Notification
       ▼
Payment Notification Bridge
       │
       ▼
Customer Self-Hosted Backend
       │
       ├── Event Processing
       ├── Transaction Matching
       ├── Webhook
       ├── Logs
       └── Dashboard
       │
       ▼
Customer Application
```

GoPay merupakan connector awal.

Arsitektur harus dirancang agar connector lain dapat ditambahkan di masa depan.

Contoh:

```text
Payment Notification Bridge
├── GoPay
├── DANA
├── OVO
├── ShopeePay
├── Livin'
└── Other Payment Sources
```

---

# 3. Business Model

Customer membeli hak penggunaan software berdasarkan periode lisensi.

Contoh:

```text
License Period
01 Jan 2027
      │
      ▼
31 Dec 2027
```

Customer membayar biaya lisensi tahunan.

Selama license aktif, customer mendapatkan hak menggunakan fitur sesuai paket.

Contoh:

```text
Annual License
Rp5.000.000 / year

Includes:
- 1 production installation
- 5 Android devices
- GoPay connector
- Webhook API
- Dashboard
- Updates selama masa lisensi
- Standard support
```

Setelah periode berakhir, customer dapat melakukan renewal.

---

# 4. Kenapa Self-Hosted?

Self-hosted ditujukan untuk customer yang ingin memiliki kontrol penuh terhadap infrastructure dan data mereka.

Customer dapat menjalankan:

```text
Customer VPS
│
├── Payment Bridge Backend
├── Database
├── Queue
├── Dashboard
└── Reverse Proxy
```

Android device terhubung ke backend milik customer.

```text
Android Device
      │
      │ HTTPS
      ▼
customer-payment.example.com
      │
      ▼
Customer Server
```

Vendor tidak perlu menjadi tempat penyimpanan utama seluruh data transaksi customer.

---

# 5. Target Customer

Model ini cocok untuk:

### 5.1 Developer

Developer ingin menggunakan payment notification pada project mereka sendiri.

Contoh:

- SaaS
- marketplace
- ticketing
- membership system
- digital product
- payment confirmation system

### 5.2 Agency

Agency ingin menggunakan software untuk beberapa client.

### 5.3 Company

Perusahaan ingin menjalankan software di infrastructure internal.

### 5.4 Enterprise

Perusahaan yang memiliki requirement:

- data berada di infrastructure sendiri
- network restriction
- internal security policy
- custom integration
- SLA
- audit log

---

# 6. Komponen Produk

Produk terdiri dari beberapa komponen utama.

## 6.1 Android Bridge

Aplikasi Android bertugas menerima notification dari payment application.

Contoh:

```text
GoPay
   ↓
Android Notification
   ↓
Payment Bridge
```

Android Bridge tidak mengakses database internal atau API private payment application.

Android Bridge hanya memproses informasi yang tersedia melalui Android Notification Listener.

---

## 6.2 Backend

Backend merupakan pusat sistem.

Tanggung jawab:

- authentication
- device management
- notification event processing
- event validation
- duplicate prevention
- transaction matching
- webhook delivery
- retry
- logging
- license validation
- dashboard API

Contoh:

```text
Android
   ↓
POST /api/events
   ↓
Backend
   ↓
Event Processing
   ↓
Webhook
```

---

## 6.3 Dashboard

Dashboard digunakan untuk administrasi sistem.

Contoh module:

```text
Dashboard
├── Overview
├── Devices
├── Payment Sources
├── Events
├── Transactions
├── Webhooks
├── API Keys
├── Logs
├── License
└── Settings
```

---

# 7. License Server

Karena software berjalan pada server customer, vendor membutuhkan mekanisme untuk mengetahui apakah software masih memiliki license yang valid.

Dibutuhkan sebuah **License Server** milik vendor.

Arsitektur:

```text
                    VENDOR
              ┌─────────────────┐
              │ License Server  │
              └────────┬────────┘
                       │
                 License Check
                       │
                       ▼
CUSTOMER       ┌─────────────────┐
               │ Customer Backend │
               └─────────────────┘
```

License Server tidak menangani transaksi customer.

Fungsinya hanya:

- activation
- license validation
- renewal status
- feature entitlement
- installation binding
- license revocation
- license metadata

---

# 8. License Object

Setiap customer mendapatkan sebuah license.

Contoh data konseptual:

```json
{
  "license_key": "PB-XXXX-XXXX-XXXX",
  "customer_id": "CUS-001",
  "product": "payment-bridge",
  "plan": "business",
  "status": "active",
  "issued_at": "2027-01-01T00:00:00Z",
  "expires_at": "2027-12-31T23:59:59Z",
  "max_devices": 10,
  "max_installations": 1,
  "features": ["gopay", "webhook", "transaction_matching", "dashboard"]
}
```

License key sebaiknya tidak menjadi satu-satunya sumber keamanan.

Backend customer juga harus memiliki installation identity.

---

# 9. Installation ID

Setiap instalasi self-hosted mendapatkan identifier unik.

Contoh:

```text
Installation ID
INS-01HXABCDEFG
```

Ketika pertama kali diaktifkan:

```text
Customer Backend
      │
      │ Activation Request
      ▼
License Server
      │
      │ Validate License
      ▼
Installation Registered
```

License kemudian di-bind dengan installation tersebut.

Contoh:

```text
License
PB-XXXX
   │
   └── Installation
       INS-001
```

Tujuannya agar satu license tidak digunakan tanpa batas pada banyak server.

---

# 10. Activation Flow

Proses activation:

```text
1. Customer deploy backend
        ↓
2. Customer membuka dashboard
        ↓
3. Customer memasukkan License Key
        ↓
4. Backend membuat Installation ID
        ↓
5. Backend menghubungi License Server
        ↓
6. License Server melakukan validation
        ↓
7. License valid
        ↓
8. Installation diaktifkan
        ↓
9. Dashboard dapat digunakan
```

Contoh status:

```text
UNACTIVATED
     ↓
ACTIVATING
     ↓
ACTIVE
```

Jika gagal:

```text
ACTIVATION_FAILED
```

---

# 11. License Validation

Backend melakukan validation terhadap License Server secara berkala.

Contoh:

```text
Customer Backend
       │
       │ License Check
       ▼
License Server
       │
       ├── ACTIVE
       ├── EXPIRED
       ├── SUSPENDED
       └── REVOKED
```

Validation tidak harus dilakukan pada setiap request API.

Hal tersebut dapat menyebabkan:

- dependency terhadap internet
- latency
- License Server menjadi single point of failure

Lebih baik menggunakan local license state dengan periodic validation.

---

# 12. License Grace Period

Self-hosted software tidak boleh langsung berhenti hanya karena License Server tidak dapat diakses sementara.

Contoh:

```text
License valid
     │
     ▼
License Server temporarily unavailable
     │
     ▼
Grace Period
     │
     ├── Day 1
     ├── Day 2
     ├── Day 3
     └── ...
```

Contoh grace period:

```text
7 hari
```

Selama grace period:

```text
License Server unreachable
        ↓
Existing license masih dianggap valid
        ↓
System tetap berjalan
```

Jika grace period habis tanpa successful validation:

```text
GRACE PERIOD EXPIRED
```

Sistem dapat masuk ke mode terbatas.

---

# 13. Jangan Langsung Mematikan Semua Sistem

Ketika license expired, sebaiknya jangan langsung membuat database atau data customer tidak dapat diakses.

Lebih baik menggunakan status:

```text
ACTIVE
    ↓
EXPIRING
    ↓
EXPIRED
    ↓
RESTRICTED
```

Contoh behavior:

### Active

Semua fitur berjalan.

### Expiring

Dashboard memberikan warning.

### Expired

Sistem masih dapat dibuka untuk melihat data dan melakukan renewal.

### Restricted

Fitur operasional tertentu dapat dibatasi.

Contoh:

```text
Dashboard       → tetap bisa dibuka
Data lama       → tetap bisa dilihat
Settings        → tetap bisa dibuka
Export          → sesuai kebijakan
New events      → dapat dibatasi
Webhook         → dapat dibatasi
New devices     → dibatasi
```

Tujuannya agar customer tidak merasa datanya "disandera".

---

# 14. Renewal

Sebelum license berakhir, customer mendapatkan notification.

Contoh:

```text
License expires in:

30 days
14 days
7 days
3 days
1 day
```

Setelah customer melakukan pembayaran:

```text
Payment
   ↓
Renewal
   ↓
License expires_at updated
   ↓
Customer Backend
   ↓
License validation
   ↓
ACTIVE
```

Renewal dapat memperpanjang:

```text
Current expiry
+
12 months
```

bukan selalu mengganti tanggal berdasarkan tanggal pembayaran.

---

# 15. Feature Entitlement

License dapat menentukan fitur yang tersedia.

Contoh:

```text
Starter
├── GoPay
├── 3 devices
└── 2 webhooks

Business
├── GoPay
├── DANA
├── OVO
├── 10 devices
├── 20 webhooks
└── Transaction Matching

Enterprise
├── All connectors
├── Unlimited devices
├── Unlimited webhooks
├── White label
├── Custom integration
└── Priority support
```

Dengan demikian satu software dapat digunakan untuk beberapa paket.

---

# 16. Recommended Pricing Structure

Harga berikut merupakan contoh awal dan harus divalidasi berdasarkan target market dan biaya support.

## Starter

```text
Rp2.500.000 / tahun

Includes:
- 1 installation
- 3 Android devices
- 1 payment source
- 2 webhook endpoints
- Dashboard
- Basic logs
- Standard updates
- Standard support
```

## Business

```text
Rp5.000.000 / tahun

Includes:
- 1 installation
- 10 Android devices
- Multiple payment sources
- 20 webhook endpoints
- Transaction matching
- Advanced logs
- Updates
- Priority support
```

## Enterprise

```text
Custom pricing

Includes:
- Multiple installations
- Custom device limits
- All payment sources
- Custom integration
- White label option
- SLA
- Priority support
- Deployment assistance
```

Harga bukan merupakan bagian permanen dari architecture dan dapat berubah tanpa mengubah software utama.

---

# 17. Installation Limit

License dapat memiliki batas jumlah installation.

Contoh:

```text
Business License
Max Installations: 1
```

Jika customer ingin menjalankan:

```text
Production Server
+
Backup Server
+
Staging Server
```

maka kebijakan harus jelas.

Rekomendasi:

```text
Production
= counted installation

UAT/Staging
= optional separate installation
```

Untuk paket tertentu, UAT dapat diberikan sebagai bagian dari license.

---

# 18. UAT dan Production

Karena produk memiliki environment UAT dan Production, license harus mendukung keduanya.

Contoh:

```text
Customer License
│
├── UAT Installation
│
└── Production Installation
```

Namun keduanya tetap dapat dianggap sebagai satu deployment entitlement jika paket mengizinkannya.

Contoh:

```text
Business License

Production:
1 installation

UAT:
1 installation

Total:
2 environment
```

UAT tidak boleh menggunakan data production.

---

# 19. Environment Separation

Customer sebaiknya menjalankan:

```text
UAT
uat-payment.customer.com
```

dan:

```text
Production
payment.customer.com
```

Dengan database terpisah:

```text
UAT Database
Production Database
```

Android UAT:

```text
Android UAT
    ↓
UAT Backend
```

Android Production:

```text
Android Production
    ↓
Production Backend
```

---

# 20. Update Strategy

Self-hosted berarti customer tidak otomatis mendapatkan update seperti SaaS.

Karena itu perlu mekanisme release.

Contoh:

```text
Vendor
   ↓
Release v1.2.0
   ↓
Customer receives update
   ↓
Customer downloads package
   ↓
Customer deploys update
```

Distribusi dapat dilakukan melalui:

- private download portal
- customer dashboard
- release package
- Docker image
- deployment package

---

# 21. Version Compatibility

Backend dan Android Bridge harus memiliki compatibility information.

Contoh:

```text
Backend:
v1.5.0

Android Bridge:
v1.4.x - v1.5.x

API:
v1
```

Update tidak boleh menyebabkan Android versi lama langsung gagal tanpa warning.

---

# 22. Recommended Deployment Method

Untuk self-hosted product, deployment sebaiknya dibuat sesederhana mungkin.

Rekomendasi:

```text
Docker
```

Contoh:

```text
Customer VPS
│
├── Reverse Proxy
├── Payment Bridge Backend
├── Database
└── Queue Worker
```

Customer idealnya cukup melakukan:

```text
1. Upload configuration
2. Configure domain
3. Start containers
4. Open dashboard
5. Activate license
```

Semakin mudah deployment, semakin kecil support burden.

---

# 23. Customer Deployment Responsibility

Customer bertanggung jawab terhadap:

- VPS
- Operating System
- Domain
- SSL certificate
- Firewall
- Internet connection
- Android device
- Payment application
- Backup infrastructure

Vendor bertanggung jawab terhadap:

- software
- license
- documentation
- official release
- bug fixes sesuai support policy
- technical support sesuai paket

---

# 24. Data Ownership

Data yang diproses pada self-hosted installation pada prinsipnya berada di infrastructure customer.

Contoh:

```text
Payment Notification
        ↓
Customer Android
        ↓
Customer Backend
        ↓
Customer Database
```

Vendor tidak perlu menerima seluruh transaction data customer.

License Server hanya menangani informasi yang diperlukan untuk licensing, seperti:

```text
License ID
Customer ID
Installation ID
Product
Version
License status
Feature entitlement
Expiry
```

Hindari mengirim raw payment notification ke License Server.

---

# 25. Security Principles

License system harus dirancang agar tidak menjadi security vulnerability.

Prinsip:

- License key tidak disimpan plaintext jika tidak diperlukan.
- Communication dengan License Server menggunakan HTTPS.
- License validation menggunakan authentication.
- Installation identity dibuat secara aman.
- Jangan hardcode master license secret pada Android app.
- Jangan memasukkan vendor private key ke client.
- Gunakan signed license/token jika diperlukan.
- Batasi rate request ke License Server.
- Audit setiap activation dan deactivation.
- Jangan mengirim data transaksi customer ke License Server.

---

# 26. License Server Failure

Jika License Server mengalami downtime:

```text
Customer Backend
      ↓
License Check
      ↓
License Server unavailable
      ↓
Use cached license state
      ↓
Grace Period
```

Customer tidak boleh langsung kehilangan akses hanya karena server licensing vendor mengalami masalah.

Ini sangat penting untuk menjaga kepercayaan customer B2B.

---

# 27. License Revocation

Vendor dapat melakukan revoke terhadap license pada kondisi tertentu.

Contoh:

- fraud
- license abuse
- penggunaan di luar kontrak
- chargeback
- pelanggaran license agreement

Status:

```text
ACTIVE
   ↓
SUSPENDED
   ↓
REVOKED
```

Revoked berbeda dengan expired.

```text
Expired
= masa lisensi habis.

Revoked
= lisensi dihentikan oleh vendor.
```

---

# 28. Anti-License Abuse

Tujuannya bukan membuat software mustahil dibajak, tetapi mencegah penggunaan license secara tidak wajar.

Protection dapat berupa:

```text
License Key
+
Installation ID
+
Domain
+
Environment
+
Periodic Validation
```

Contoh:

```text
License PB-001
     │
     ├── Installation: INS-001
     ├── Domain: payment.customer.com
     └── Environment: production
```

Jika license dipindahkan ke server lain:

```text
New Installation Detected
```

Customer dapat melakukan deactivation dari dashboard/vendor portal sesuai kebijakan.

---

# 29. Transfer License

Customer dapat meminta transfer license jika server lama diganti.

Contoh:

```text
Old VPS
INS-001
   ↓
Deactivate
   ↓
New VPS
INS-002
   ↓
Activate
```

Jumlah transfer dapat dibatasi untuk mencegah abuse.

Contoh:

```text
Maximum:
3 installation transfers / year
```

Enterprise dapat memiliki kebijakan berbeda.

---

# 30. Support Model

Support merupakan bagian penting dari bisnis self-hosted.

### Standard Support

- Documentation
- Email/ticket
- Bug assistance
- Installation guide

### Priority Support

- Faster response
- Deployment assistance
- Troubleshooting

### Enterprise Support

- SLA
- Dedicated support
- Custom integration
- Migration assistance

Support level harus ditulis secara jelas dalam agreement.

---

# 31. Recommended Customer Journey

```text
1. Customer melihat product
          ↓
2. Customer memilih license
          ↓
3. Payment
          ↓
4. License generated
          ↓
5. Customer mendapatkan deployment package
          ↓
6. Customer deploy backend
          ↓
7. Customer activate license
          ↓
8. Setup Android Bridge
          ↓
9. Connect payment source
          ↓
10. Configure webhook
          ↓
11. UAT
          ↓
12. Production
          ↓
13. License active
          ↓
14. Renewal after 12 months
```

---

# 32. Recommended Vendor Dashboard

Vendor sendiri membutuhkan dashboard terpisah dari dashboard customer.

```text
Vendor Admin
├── Customers
├── Licenses
├── Installations
├── Activations
├── Renewals
├── Releases
├── Support
└── Audit Logs
```

Vendor dapat melihat:

```text
Customer
License
Installation
Version
Environment
Status
Expiry
Last Validation
```

Vendor tidak perlu melihat raw payment transaction customer.

---

# 33. Product Architecture

Secara keseluruhan:

```text
                         VENDOR
                           │
                 ┌─────────┴─────────┐
                 │                   │
          Vendor Dashboard      License Server
                 │                   │
                 └─────────┬─────────┘
                           │
                    License Management
                           │
                           ▼
              ┌────────────────────────┐
              │    CUSTOMER SERVER     │
              │                        │
              │ Payment Bridge Backend │
              │        │               │
              │   ┌────┴────┐          │
              │   │         │          │
              │ Database   Queue        │
              │   │                    │
              │ Dashboard               │
              └───────────┬────────────┘
                          │
                        HTTPS
                          │
                          ▼
                    Android Bridge
                          │
              ┌───────────┼───────────┐
              ▼           ▼           ▼
            GoPay        DANA        OVO
```

---

# 34. Separation of Concerns

Sistem harus memisahkan:

### Vendor Infrastructure

```text
License
Update
Customer management
```

### Customer Infrastructure

```text
Payment events
Transactions
Webhooks
Business data
```

### Android

```text
Notification collection
Parsing
Queue
Forwarding
```

Dengan separation ini, produk lebih aman dan mudah dikembangkan.

---

# 35. Advantages

## 35.1 Customer owns infrastructure

Customer memiliki kontrol terhadap:

- server
- database
- network
- data

## 35.2 Lower vendor infrastructure cost

Vendor tidak perlu menyediakan server untuk setiap customer.

## 35.3 Higher-value B2B product

Self-hosted lebih mudah diposisikan sebagai software infrastructure dibanding APK utility.

## 35.4 Recurring revenue

License tahunan menciptakan revenue berulang.

## 35.5 Suitable for enterprise

Model ini cocok untuk customer dengan security requirement tinggi.

## 35.6 Easier to scale operationally

Pertumbuhan customer tidak selalu berarti pertumbuhan infrastructure vendor secara linear.

---

# 36. Disadvantages

## 36.1 Deployment complexity

Customer harus melakukan deployment sendiri.

## 36.2 Support burden

Masalah dapat berasal dari:

- VPS
- Docker
- Nginx
- SSL
- DNS
- firewall
- Android
- network

Tidak semuanya merupakan bug software.

## 36.3 Updates are harder

Customer tidak otomatis menggunakan versi terbaru.

## 36.4 License system complexity

Vendor harus membangun:

- activation
- validation
- renewal
- revocation
- installation management

## 36.5 Revenue per customer may be lower than managed SaaS

Customer hanya membayar license, sementara vendor tidak mendapatkan margin infrastructure bulanan.

---

# 37. Recommended Business Strategy

Model utama:

```text
SELF-HOSTED
+
ANNUAL LICENSE
```

Revenue tambahan:

```text
Annual License
+
Setup Fee
+
Priority Support
+
Enterprise SLA
+
Custom Integration
+
White Label
```

Contoh:

```text
Software License
Rp5.000.000 / year

Optional:
Setup               Rp1.500.000
Migration           Rp2.000.000
Custom Integration  Rp3.000.000+
Priority Support    Rp1.000.000 / year
White Label         Custom
SLA                 Custom
```

---

# 38. Recommended Product Packages

## Starter

Target:

- individual developer
- small project
- small business

```text
1 production
1 UAT
3 devices
1 payment source
2 webhooks
Standard support
```

## Business

Target:

- company
- SaaS
- agency

```text
1 production
1 UAT
10 devices
Multiple payment sources
20 webhooks
Transaction matching
Priority support
```

## Enterprise

Target:

- large company
- agency with many clients
- enterprise

```text
Custom installations
Custom devices
All connectors
Custom integration
White label
SLA
Priority support
Dedicated deployment support
```

---

# 39. Important Product Principle

Produk ini bukan sekadar software yang dijual sekali.

Produk merupakan:

> **Licensed software infrastructure.**

Customer membeli:

```text
Software
+
Right to Use
+
Updates
+
Support
```

selama periode lisensi.

---

# 40. Recommended MVP

Untuk versi pertama, jangan membangun seluruh sistem licensing yang terlalu kompleks.

MVP:

```text
Customer Backend
├── License Activation
├── Installation ID
├── License Status
├── Expiry Date
└── Periodic Validation

Vendor Backend
├── Customers
├── Licenses
├── Installations
├── Activation
└── Renewal
```

Belum perlu:

- complicated DRM
- advanced anti-tampering
- complicated billing automation
- marketplace
- reseller management
- multi-level licensing

Fokus pada license lifecycle yang stabil.

---

# 41. License Lifecycle

```text
                 ┌───────────┐
                 │  CREATED  │
                 └─────┬─────┘
                       │
                       ▼
                 ┌───────────┐
                 │ ACTIVATED │
                 └─────┬─────┘
                       │
                       ▼
                   ┌───────┐
                   │ ACTIVE│
                   └───┬───┘
                       │
              ┌────────┴────────┐
              ▼                 ▼
          EXPIRING           SUSPENDED
              │
              ▼
           EXPIRED
              │
         ┌────┴────┐
         ▼         ▼
      RENEWED    RESTRICTED
         │
         ▼
       ACTIVE
```

---

# 42. Definition of Done — Licensing

## Customer Side

- [ ] License activation tersedia
- [ ] Installation ID dibuat
- [ ] License status tersedia
- [ ] Expiry date tersedia
- [ ] Feature entitlement tersedia
- [ ] Periodic validation tersedia
- [ ] Grace period tersedia
- [ ] License warning tersedia
- [ ] Renewal state tersedia
- [ ] UAT/Production dapat dibedakan
- [ ] License tidak menyimpan raw payment data ke vendor

## Vendor Side

- [ ] Customer management
- [ ] License creation
- [ ] License activation
- [ ] Installation management
- [ ] License validation
- [ ] License suspension
- [ ] License revocation
- [ ] Renewal
- [ ] Feature entitlement
- [ ] Audit log
- [ ] License expiry notification

## Security

- [ ] HTTPS
- [ ] Authentication
- [ ] Rate limiting
- [ ] Secure license validation
- [ ] Installation binding
- [ ] No vendor private secret on customer client
- [ ] No raw payment event sent to License Server
- [ ] Grace period
- [ ] Abuse detection

---

# 43. Long-Term Roadmap

## Phase 1

```text
Payment Notification Bridge
+
Self-hosted Backend
+
Annual License
+
GoPay
```

## Phase 2

```text
Multiple Payment Sources
├── GoPay
├── DANA
├── OVO
└── Bank
```

## Phase 3

```text
Payment Automation Platform
├── Payment Notification
├── Transaction Matching
├── Webhook
├── Reconciliation
├── Device Management
└── Analytics
```

## Phase 4

```text
Enterprise Platform
├── White Label
├── Multi-tenant
├── Reseller
├── SLA
├── Advanced Audit
└── Custom Integrations
```

---

# 44. Final Product Concept

Produk akhir dapat diposisikan sebagai:

> **Self-hosted payment notification infrastructure for automated payment confirmation.**

Customer memiliki infrastructure sendiri.

Vendor menyediakan software dan licensing.

Model bisnis:

```text
Customer
    │
    │ Annual License
    ▼
Vendor
    │
    ├── Software
    ├── Updates
    ├── Support
    └── License Management
```

Sedangkan operational payment data tetap berjalan di infrastructure customer:

```text
GoPay / DANA / OVO / Bank
            │
            ▼
      Android Bridge
            │
            ▼
    Customer Backend
            │
            ▼
       Customer DB
            │
            ▼
     Customer Website
```

Dengan model ini, produk dapat berkembang dari:

> **GoPay Notification Bridge**

menjadi:

> **Payment Notification Bridge**

dan akhirnya:

> **Payment Automation Infrastructure**

tanpa harus mengubah model bisnis dasarnya.
