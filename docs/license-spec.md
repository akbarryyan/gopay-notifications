# Self-Hosted + Annual License Specification

## 1. Overview

Produk menggunakan model **Self-Hosted + Annual License**.

Customer menjalankan software pada infrastructure milik mereka sendiri, sedangkan vendor menyediakan software, license management, updates, documentation, dan support sesuai paket.

### Core Principle

> **Customer owns the infrastructure and data. Vendor owns the software license.**

Customer tidak perlu mengirim raw payment data ke infrastructure vendor.

---

# 2. Business Model

Customer membeli **hak penggunaan software** untuk periode tertentu, umumnya 1 tahun.

Contoh:

```text
Business License
Duration: 1 Year
```

License memberikan hak untuk:

* Menjalankan software.
* Menggunakan fitur sesuai plan.
* Menggunakan jumlah installation sesuai entitlement.
* Menghubungkan jumlah device sesuai entitlement.
* Menggunakan connector sesuai plan.
* Menggunakan bot/channel sesuai plan.
* Mendapatkan software updates sesuai license.
* Mendapatkan support sesuai paket.

License bukan berarti customer menyewa server vendor.

Customer tetap menjalankan software pada infrastructure mereka sendiri.

---

# 3. Ownership Boundary

## 3.1 Customer Owns

Customer memiliki dan mengelola:

* VPS/server.
* Domain.
* SSL.
* Database.
* Payment data.
* Transaction data.
* Android devices.
* Payment accounts.
* Customer Dashboard.
* Backend application.
* Queue/worker.
* Bot configuration.
* Webhook destinations.
* Customer users.

## 3.2 Vendor Owns

Vendor memiliki dan mengelola:

* Software product.
* License Server.
* Vendor Dashboard.
* License records.
* Product releases.
* Version management.
* Entitlements.
* License lifecycle.
* Documentation.
* Support system.

Vendor License Server secara default **tidak menerima raw payment transaction data**.

---

# 4. Architecture

```text
                         VENDOR
                           │
              ┌────────────┴────────────┐
              │                         │
       Vendor Dashboard          License Server
              │                         │
              └────────────┬────────────┘
                           │
                     License Metadata
                           │
                           ▼
                    CUSTOMER SERVER
                           │
              ┌────────────┼────────────┐
              │            │            │
       Customer Dashboard Backend    Database
                           │
                    ┌──────┴──────┐
                    │             │
                  Queue          Bot
                    │
              Android Bridge
                    │
             Payment Application
```

License Server hanya menjadi authority untuk:

* License status.
* License expiration.
* Installation binding.
* Entitlements.
* Product/version compatibility.

License Server tidak menjadi bagian dari real-time payment processing.

---

# 5. License Entity

Setiap license memiliki data utama:

```text
License ID
Customer ID
Plan
Status
Created At
Activated At
Expires At
Installations
Entitlements
Last Validation
```

Contoh:

```text
License ID
LIC-2026-000123

Customer
CUS-000123

Plan
Business

Status
ACTIVE

Created
2026-09-13

Activated
2026-09-13

Expires
2027-09-13
```

---

# 6. License Key

Customer menerima license key untuk melakukan activation awal.

Contoh:

```text
PB-BUSINESS-7K4X-92LM-AX81
```

License key digunakan untuk:

* Initial activation.
* Menghubungkan license dengan customer installation.

License key **bukan satu-satunya security mechanism**.

Setelah activation, software menggunakan Installation ID dan signed license state untuk validasi lokal.

---

# 7. Installation ID

Setiap deployment software memiliki Installation ID.

Contoh:

```text
INST-8F21A
```

Flow:

```text
Customer deploy software
        ↓
Open Customer Dashboard
        ↓
Enter License Key
        ↓
Generate Installation ID
        ↓
License Server validation
        ↓
Installation activated
```

License dapat memiliki batas installation.

Contoh:

```text
License
LIC-000123
│
├── Production
│   └── INST-PROD-001
│
└── UAT
    └── INST-UAT-001
```

---

# 8. Installation Binding

License tidak sebaiknya diikat secara permanen ke public IP.

Jangan menggunakan:

```text
License
    ↓
Public IP
```

sebagai primary identity.

Gunakan:

```text
License
├── Customer ID
├── Installation ID
├── Product
├── Environment
├── Version
└── Entitlements
```

Hardware fingerprint dapat digunakan sebagai tambahan security mechanism jika diperlukan.

Namun binding harus tetap memungkinkan:

* VPS migration.
* Server replacement.
* Disaster recovery.
* Reinstallation.

---

# 9. Environment

UAT dan Production harus dipisahkan.

Contoh:

```text
License
│
├── Production
│   └── INST-PROD-001
│
└── UAT
    └── INST-UAT-001
```

Recommended entitlement:

```text
production_installations: 1
uat_installations: 1
```

Data, database, environment configuration, dan operational state UAT tidak boleh bercampur dengan Production.

---

# 10. License Activation

## Initial Activation

```text
Customer VPS
      ↓
Customer Dashboard
      ↓
Enter License Key
      ↓
Customer Backend
      ↓
License Server
      ↓
Validate License
      ↓
Bind Installation
      ↓
Return Entitlements
      ↓
ACTIVE
```

License Server melakukan pemeriksaan:

* License exists.
* License status.
* License expiration.
* Installation limit.
* Environment entitlement.
* Product compatibility.
* Feature entitlement.

---

# 11. License Validation

Recommended model:

> **Online Validation + Local License Cache + Grace Period**

Customer Backend melakukan periodic validation terhadap License Server.

Contoh:

```text
Every 24 Hours

Customer Backend
       ↓
License Server
       ↓
Validation Result
       ↓
Update Local License State
```

License validation **tidak dilakukan pada setiap payment event**.

---

# 12. Grace Period

Grace period digunakan apabila License Server tidak dapat diakses sementara.

Contoh:

```text
License Validation
       ↓
License Server Unavailable
       ↓
Use Last Valid Local License State
       ↓
Grace Period
```

Recommended initial grace period:

```text
7 Days
```

Tujuan:

* Mencegah vendor outage mematikan customer.
* Menghindari dependency real-time.
* Memberikan waktu untuk memperbaiki koneksi.
* Menjaga reliability self-hosted system.

---

# 13. Local License State

Customer Backend menyimpan license state secara lokal.

Contoh:

```text
License ID
Installation ID
Plan
Status
Expires At
Entitlements
Last Validation
Grace Period
Product Version
```

Local license state sebaiknya memiliki cryptographic signature agar tidak mudah dimodifikasi secara manual.

---

# 14. License Validation Request

Contoh request:

```json
{
  "license_id": "LIC-000123",
  "installation_id": "INST-8F21A",
  "product_version": "1.4.2",
  "environment": "production"
}
```

License Server tidak membutuhkan raw transaction data untuk melakukan validasi.

---

# 15. License Validation Response

Contoh:

```json
{
  "status": "active",
  "expires_at": "2027-09-13",
  "plan": "business",
  "entitlements": {
    "max_devices": 10,
    "max_webhooks": 20,
    "telegram_bot": true,
    "whatsapp_bot": true
  }
}
```

---

# 16. Data Privacy Boundary

License Server hanya menerima **license metadata**.

## Allowed

```text
License ID
Customer ID
Installation ID
Product
Version
Environment
License Status
Expiration
Entitlements
Last Validation
```

## Not Allowed by Default

```text
Payment Transactions
Payment Amount
Raw Payment Notifications
Webhook Payload
Payment Credentials
API Keys
Customer Database Contents
Customer Transaction Details
```

Raw payment data tetap berada di customer infrastructure.

---

# 17. Entitlements

License dapat mengatur:

* Feature access.
* Resource limits.
* Number of devices.
* Number of webhooks.
* Number of installations.
* Connector availability.
* Bot/channel availability.
* Support level.

Contoh Starter:

```json
{
  "max_devices": 3,
  "max_webhooks": 5,
  "telegram_bot": true,
  "whatsapp_bot": false,
  "advanced_logs": false
}
```

Contoh Business:

```json
{
  "max_devices": 10,
  "max_webhooks": 20,
  "telegram_bot": true,
  "whatsapp_bot": true,
  "advanced_logs": true
}
```

Contoh Enterprise:

```json
{
  "max_devices": 50,
  "max_webhooks": -1,
  "telegram_bot": true,
  "whatsapp_bot": true,
  "advanced_logs": true,
  "custom_connector": true,
  "priority_support": true
}
```

`-1` dapat digunakan sebagai representasi unlimited jika disepakati dalam product policy.

---

# 18. License Status

Recommended states:

```text
CREATED
    ↓
ACTIVATED
    ↓
ACTIVE
    ↓
EXPIRING
    ↓
EXPIRED
```

Administrative states:

```text
SUSPENDED
REVOKED
```

## Status Definitions

### CREATED

License sudah dibuat tetapi belum diaktivasi.

### ACTIVATED

License sudah terhubung dengan installation.

### ACTIVE

License aktif dan software dapat digunakan secara normal.

### EXPIRING

License mendekati tanggal expiration.

### EXPIRED

Masa berlaku license telah berakhir.

### SUSPENDED

License sementara dinonaktifkan oleh vendor.

### REVOKED

License dicabut oleh vendor.

---

# 19. Expiration Policy

License expired tidak boleh langsung menyebabkan:

* Database dihapus.
* Transaction history dihapus.
* Customer kehilangan akses terhadap data.
* Customer tidak dapat melakukan renewal.

## Tetap tersedia

* Login.
* Melihat data lama.
* Melihat transaksi.
* Melihat logs.
* Melihat license.
* Renewal.
* Export data jika tersedia.

## Dapat dibatasi

* Payment event processing.
* Webhook delivery.
* Device registration.
* Operational processing.
* Premium features.

Exact restriction harus ditentukan berdasarkan product policy dan plan.

---

# 20. Renewal

Customer dapat melakukan renewal sebelum atau setelah expiration.

Contoh:

```text
Original License

13 Sep 2026
      ↓
13 Sep 2027
```

Setelah renewal:

```text
13 Sep 2026
      ↓
13 Sep 2028
```

Renewal tidak membutuhkan reinstall software.

Customer Backend melakukan synchronization dengan License Server.

---

# 21. Renewal Warning

Customer mendapatkan notification sebelum expiration.

Recommended warnings:

```text
30 Days Before
14 Days Before
7 Days Before
3 Days Before
1 Day Before
```

Contoh:

```text
⚠️ License Expiring

Your Business license will expire
in 14 days.

Expiry:
13 Sep 2027
```

Warning dapat ditampilkan melalui:

* Customer Dashboard.
* Telegram Bot.
* WhatsApp Bot.
* Email jika tersedia.

---

# 22. Installation Migration

Customer harus dapat mengganti VPS.

Contoh:

```text
Old VPS
INST-001
```

Customer pindah ke:

```text
New VPS
INST-002
```

Vendor Dashboard menyediakan action:

> Reset Installation

Flow:

```text
Vendor
   ↓
Customer
   ↓
License
   ↓
Reset Installation
   ↓
Old Installation Released
   ↓
New Installation Can Be Activated
```

Semua reset harus masuk audit log.

---

# 23. Installation Limit

Contoh Business License:

```text
Production:
1 Installation

UAT:
1 Installation
```

Jika installation limit sudah tercapai:

```text
Installation limit reached.

Please release an existing installation
or contact support.
```

Vendor dapat melihat seluruh installation yang terhubung dengan license.

---

# 24. Vendor Dashboard

Vendor Dashboard menyediakan:

```text
Customers
Installations
Licenses
Activations
Renewals
Entitlements
Releases
Versions
Audit Logs
```

License detail:

```text
License ID
Customer
Plan
Status
Created At
Activated At
Expires At
Installations
Entitlements
Last Validation
Audit History
```

---

# 25. Vendor License Actions

Vendor dapat melakukan:

* Create license.
* Activate license.
* Suspend license.
* Revoke license.
* Renew license.
* Extend expiration.
* Change entitlement.
* Reset installation.
* View activation history.
* View validation history.

Sensitive actions harus membutuhkan confirmation.

---

# 26. Audit Logging

Semua tindakan penting terhadap license harus dicatat.

Recommended events:

```text
LICENSE_CREATED
LICENSE_ACTIVATED
LICENSE_RENEWED
LICENSE_SUSPENDED
LICENSE_REVOKED
LICENSE_EXTENDED
ENTITLEMENT_CHANGED
INSTALLATION_CREATED
INSTALLATION_RELEASED
INSTALLATION_RESET
LICENSE_VALIDATED
```

Minimum audit data:

```text
Actor
Action
Resource
Timestamp
IP
Result
Metadata
```

---

# 27. Customer Dashboard — License Page

Customer dapat melihat:

```text
License

Plan
Business

Status
🟢 Active

License ID
LIC-000123

Production Installation
INST-PROD-001

UAT Installation
INST-UAT-001

Expires
13 Sep 2027

Days Remaining
365

Devices
4 / 10

Webhooks
8 / 20
```

Customer tidak dapat melakukan vendor-only actions seperti:

* Mengubah expiration.
* Mengubah plan.
* Mengubah entitlement.
* Membuat license.
* Reset installation tanpa policy yang sesuai.

---

# 28. Bot License Information

Operational Bot dapat menyediakan:

```text
/license
```

Contoh response:

```text
🔐 License

Plan
Business

Status
🟢 Active

Expires
13 Sep 2027

Days Remaining
365

Installation
INST-PROD-001
```

Bot tidak boleh menampilkan vendor-only information.

---

# 29. License Server Availability

License Server tidak boleh menjadi single point of failure untuk payment processing.

## Incorrect Architecture

```text
Payment Event
      ↓
License Server
      ↓
Allowed?
      ↓
Process Payment
```

Jika License Server down, payment processing ikut berhenti.

## Recommended Architecture

```text
License Server
      ↓
Periodic Validation
      ↓
Signed Local License State
      ↓
Customer Backend
      ↓
Payment Processing
```

Payment processing tetap dapat berjalan berdasarkan local license state selama license masih valid atau berada dalam grace period.

---

# 30. Security Requirements

License system harus memiliki:

* HTTPS.
* Authentication.
* Signed license state.
* Secure license key generation.
* Activation rate limiting.
* Validation rate limiting.
* Installation binding.
* Audit logging.
* Replay protection.
* Request timestamp/nonce jika diperlukan.
* Secret rotation.
* Secure credential storage.
* Strict RBAC pada Vendor Dashboard.

License key sebaiknya disimpan menggunakan secure representation dan tidak perlu disimpan plaintext jika architecture memungkinkan.

---

# 31. Version Compatibility

License validation dapat mempertimbangkan product version.

Contoh:

```text
License
Product Version:
1.x

Installed Version:
1.4.2
```

Vendor dapat menentukan minimum supported version:

```text
minimum_supported_version:
1.3.0
```

Jika software terlalu lama:

```text
Your software version is no longer supported.

Please update to a supported version.
```

Migration path harus tersedia agar customer tidak kehilangan operational access secara tiba-tiba.

---

# 32. Software Updates

Annual license dapat memberikan hak mendapatkan software updates sesuai paket.

Flow:

```text
Active License
      ↓
Eligible for Updates
      ↓
New Release
      ↓
Customer Deploys Update
```

Vendor Dashboard menyimpan:

```text
Version
Release Date
Release Type
Minimum Compatible Version
Changelog
```

---

# 33. Support Entitlement

License plan dapat menentukan level support.

Contoh:

```text
Starter
Standard Support

Business
Priority Support

Enterprise
Priority Support + SLA
```

Support entitlement menjadi bagian dari license metadata.

---

# 34. Graceful Degradation

Jika terjadi masalah license:

```text
VALID
  ↓
WARNING
  ↓
GRACE
  ↓
RESTRICTED
```

Jangan langsung menggunakan:

```text
INVALID
   ↓
Delete Data
```

Customer harus tetap dapat:

* Mengakses data.
* Melihat status license.
* Melakukan renewal.
* Memperbaiki installation.
* Menghubungi support.

---

# 35. End-to-End Example

Customer membeli:

```text
Business License
Duration: 1 Year
```

Vendor membuat:

```text
LIC-000123
```

Customer menerima:

```text
PB-BUSINESS-7K4X-92LM-AX81
```

Customer melakukan deployment ke VPS.

Backend membuat:

```text
INST-PROD-001
```

Customer memasukkan license key.

```text
Customer Backend
       ↓
License Server
       ↓
LIC-000123 Valid
       ↓
Bind INST-PROD-001
       ↓
Return Entitlements
       ↓
ACTIVE
```

Customer kemudian dapat menjalankan:

```text
Android Bridge
Payment Events
Webhooks
Telegram Bot
Customer Dashboard
```

Setiap 24 jam:

```text
Customer Backend
       ↓
License Validation
       ↓
License Server
       ↓
ACTIVE
```

Jika License Server tidak tersedia:

```text
Local License State
       ↓
Grace Period
       ↓
System Remains Operational
```

Jika license expired:

```text
ACTIVE
   ↓
EXPIRING
   ↓
EXPIRED
   ↓
RESTRICTED
```

Setelah customer melakukan renewal:

```text
RENEWED
   ↓
ACTIVE
```

---

# 36. Recommended Initial Policy

Untuk MVP, gunakan policy berikut:

```text
License Model:
Annual

Activation:
Online

Validation:
Periodic

Validation Interval:
24 Hours

Grace Period:
7 Days

Production Installation:
1

UAT Installation:
1

Installation Migration:
Supported

License Server:
Vendor Hosted

Customer Data:
Customer Hosted

Raw Transaction Data:
Never sent to License Server by default
```

---

# 37. Design Principles

1. Customer data tetap berada di customer infrastructure.
2. License Server hanya menangani licensing metadata.
3. License Server bukan dependency real-time payment processing.
4. License menggunakan Installation ID, bukan public IP sebagai identity utama.
5. UAT dan Production dipisahkan.
6. License memiliki entitlement yang jelas.
7. Customer dapat melakukan migration VPS.
8. Expired license tidak menghapus data.
9. Semua administrative actions diaudit.
10. Grace period mencegah vendor outage mematikan customer.
11. License key bukan satu-satunya security mechanism.
12. Renewal tidak membutuhkan reinstall.
13. Software updates mengikuti entitlement/license policy.
14. Vendor tidak memiliki akses default ke customer transaction data.

---

# 38. Final Business Model

```text
                    VENDOR
                       │
          ┌────────────┴────────────┐
          │                         │
   Vendor Dashboard          License Server
          │                         │
          └────────────┬────────────┘
                       │
                License Metadata
                       │
                       ▼
                CUSTOMER SERVER
                       │
          ┌────────────┼────────────┐
          │            │            │
      Dashboard     Backend      Database
                       │
              ┌────────┼────────┐
              │        │        │
           Queue      Bot     Webhooks
              │
       Android Bridge
              │
       Payment Sources
```

### Core Business Rule

> **Self-hosted means the customer controls where the system and data run. Annual licensing means the customer receives a time-limited right to use the software and its licensed features.**
