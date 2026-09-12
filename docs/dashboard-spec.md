# Dashboard Specification

## 1. Document Information

| Item               | Value                        |
| ------------------ | ---------------------------- |
| Product            | Payment Notification Bridge  |
| Document           | Dashboard Specification      |
| Version            | 1.0                          |
| Status             | Draft                        |
| Deployment Model   | Self-Hosted + Annual License |
| Primary Platform   | Web                          |
| Customer Dashboard | Self-Hosted                  |
| Vendor Dashboard   | Vendor Infrastructure        |

---

# 2. Dashboard Architecture

Produk memiliki dua dashboard yang terpisah.

```text
                         PAYMENT BRIDGE
                               │
              ┌────────────────┴────────────────┐
              │                                 │
              ▼                                 ▼
       CUSTOMER SIDE                       VENDOR SIDE
              │                                 │
     Customer Dashboard                  Vendor Dashboard
              │                                 │
       Customer Backend                  License Server
              │                                 │
       Customer Database                Vendor Database
```

## 2.1 Customer Dashboard

Customer Dashboard berjalan pada infrastructure customer.

Contoh:

```text
https://payment.customer.com
```

Digunakan untuk:

* monitoring system
* device management
* payment source management
* event monitoring
* transaction monitoring
* webhook management
* API key management
* license information
* system settings

---

## 2.2 Vendor Dashboard

Vendor Dashboard berjalan pada infrastructure vendor.

Contoh:

```text
https://admin.paymentbridge.com
```

Digunakan untuk:

* customer management
* license management
* installation management
* activation monitoring
* renewal
* release management
* support
* audit

Vendor Dashboard tidak digunakan untuk mengelola transaction data customer.

---

# 3. Dashboard Design Principles

Dashboard harus mengikuti prinsip:

1. Clean
2. Modern
3. Professional
4. Information-dense tetapi tidak overwhelming
5. Mobile-friendly untuk monitoring
6. Desktop-first untuk administration
7. Clear status indication
8. Consistent component
9. Minimal unnecessary animation
10. Fokus pada operational information

Dashboard bukan landing page.

Dashboard harus terasa seperti **production infrastructure management system**.

---

# 4. User Roles

## 4.1 Customer Roles

### Owner

Memiliki akses penuh terhadap customer installation.

Permissions:

* Dashboard
* Devices
* Payment Sources
* Transactions
* Events
* Webhooks
* API Keys
* License
* Settings
* User Management

---

### Admin

Memiliki akses operasional.

Permissions:

* Dashboard
* Devices
* Payment Sources
* Transactions
* Events
* Webhooks
* API Keys
* Settings

Tidak dapat:

* transfer ownership
* mengubah license
* menghapus installation

---

### Operator

Digunakan untuk user operasional.

Permissions:

* Dashboard
* Transactions
* Events
* Devices

Tidak dapat:

* License
* API Keys
* Settings
* User Management

---

## 4.2 Vendor Roles

### Super Admin

Full access.

### Support

Akses:

* Customers
* Installations
* License status
* Version
* Support information

Tidak dapat:

* melihat raw transaction data customer
* mengubah billing tanpa authorization

### License Manager

Akses:

* Licenses
* Activations
* Renewals
* Installations

---

# 5. Customer Dashboard Navigation

Sidebar utama:

```text
Dashboard

MONITORING
├── Overview
├── Transactions
├── Events
└── Logs

INTEGRATION
├── Devices
├── Payment Sources
├── Webhooks
└── API Keys

SYSTEM
├── License
├── Settings
└── About
```

Jika role tidak memiliki permission tertentu, menu tersebut tidak ditampilkan.

---

# 6. Customer Dashboard — Overview

Route:

```text
/
```

Purpose:

Memberikan gambaran kondisi sistem secara cepat.

---

## 6.1 Header

Menampilkan:

```text
Good afternoon, Customer Name

Production
● System Operational
```

Environment switcher:

```text
[ Production ▼ ]
```

Jika UAT tersedia:

```text
[ UAT ▼ ]
[ Production ▼ ]
```

Environment harus selalu terlihat untuk mencegah kesalahan konfigurasi.

---

# 7. Overview — License Card

Card:

```text
License

Business
● Active

Expires:
31 Dec 2027

183 days remaining

[View License]
```

Status:

* Active
* Expiring Soon
* Expired
* Suspended
* Revoked
* Validation Error

---

# 8. Overview — Device Card

```text
Devices

8 / 10

● 7 Online
○ 1 Offline

[Manage Devices]
```

Informasi:

* total devices
* online devices
* offline devices
* disabled devices

---

# 9. Overview — Payment Events Card

```text
Payment Events

Today
128

Successful
124

Failed
4
```

Time range:

* Today
* 7 Days
* 30 Days

---

# 10. Overview — Webhook Card

```text
Webhooks

Delivered
126

Failed
2

Success Rate
98.4%

[View Deliveries]
```

---

# 11. Overview — Recent Events

Table:

| Time  | Source |   Amount | Device     | Status    |
| ----- | ------ | -------: | ---------- | --------- |
| 17:30 | GoPay  | Rp50.000 | Device #01 | Delivered |
| 17:28 | GoPay  | Rp25.000 | Device #01 | Matched   |
| 17:21 | GoPay  | Rp10.000 | Device #02 | Pending   |

Actions:

* View event
* View transaction
* View webhook delivery

---

# 12. Overview — System Health

System health card:

```text
System Health

Backend          ● Operational
Database         ● Operational
Queue Worker     ● Operational
Webhook          ● Operational
License          ● Active
```

Possible statuses:

* Operational
* Warning
* Degraded
* Offline
* Error

---

# 13. Devices

Route:

```text
/devices
```

Purpose:

Mengelola Android Bridge devices.

---

## 13.1 Device List

Columns:

| Device      | Source | Environment | Status  | Last Seen | Version |
| ----------- | ------ | ----------- | ------- | --------- | ------- |
| Kasir Utama | GoPay  | Production  | Online  | 2 min ago | 1.2.0   |
| Backup      | GoPay  | Production  | Online  | 5 min ago | 1.2.0   |
| Testing     | GoPay  | UAT         | Offline | 2 hrs ago | 1.2.0   |

Actions:

* View
* Rename
* Disable
* Delete/Unregister

---

# 14. Device Detail

Route:

```text
/devices/:id
```

Sections:

### Device Information

```text
Device Name
Device ID
Environment
App Version
Android Version
Created At
Last Seen
```

### Connection

```text
Status
Last Seen
Last Event
```

### Payment Sources

```text
GoPay
● Connected
```

### Recent Events

Menampilkan event yang diterima device.

### Actions

```text
Rename Device
Disable Device
Unregister Device
```

Dangerous actions membutuhkan confirmation modal.

---

# 15. Device Status

Status:

```text
ONLINE
OFFLINE
DISABLED
PENDING
UNKNOWN
```

Definition:

### ONLINE

Device melakukan heartbeat/communication dalam threshold yang ditentukan.

### OFFLINE

Tidak ada communication dalam threshold.

### DISABLED

Device dinonaktifkan secara manual.

### PENDING

Device belum selesai melakukan registration.

### UNKNOWN

Status belum dapat ditentukan.

---

# 16. Payment Sources

Route:

```text
/payment-sources
```

Purpose:

Mengelola sumber payment notification.

---

## 16.1 Payment Source Cards

Contoh:

```text
GoPay

● Active

3 devices connected

[Manage]
```

Future:

```text
DANA

○ Available

Requires Business plan

[Upgrade]
```

```text
OVO

Coming Soon
```

Payment source harus menggunakan generic architecture.

Jangan membuat seluruh dashboard hanya berfokus pada GoPay.

---

# 17. Payment Source Detail

Route:

```text
/payment-sources/:source
```

Informasi:

```text
Source
GoPay

Status
Active

Connected Devices
3

Events Today
128

Last Event
17:30:21
```

Sections:

* Overview
* Devices
* Events
* Configuration

---

# 18. Transactions

Route:

```text
/transactions
```

Purpose:

Menampilkan transaksi yang sudah diproses oleh system.

---

## 18.1 Transaction Filters

Filter:

* Date range
* Payment source
* Device
* Status
* Amount
* Transaction ID

Search:

```text
Search transaction ID...
```

---

## 18.2 Transaction Table

| Transaction ID |   Amount | Source | Status  | Matched At |
| -------------- | -------: | ------ | ------- | ---------- |
| ORD-001        | Rp50.000 | GoPay  | Paid    | 17:30      |
| ORD-002        | Rp25.000 | GoPay  | Pending | -          |
| ORD-003        | Rp10.000 | GoPay  | Paid    | 17:21      |

Status:

```text
PAID
PENDING
FAILED
EXPIRED
REFUNDED
```

---

# 19. Transaction Detail

Route:

```text
/transactions/:id
```

Sections:

### Transaction

```text
Transaction ID
Amount
Status
Created At
Updated At
```

### Matching

```text
Matched Event
EVT-ABC123

Matched Source
GoPay

Matched Device
Device #01
```

### Webhook

```text
Webhook
payment.received

Status
Delivered

Response
200 OK
```

### Timeline

```text
17:29:58
Transaction Created

17:30:12
Payment Event Received

17:30:13
Transaction Matched

17:30:14
Webhook Delivered
```

---

# 20. Events

Route:

```text
/events
```

Purpose:

Menampilkan raw/normalized notification events yang diterima dari Android Bridge.

---

## 20.1 Event Table

| Event ID | Source |   Amount | Device     | Status    | Received |
| -------- | ------ | -------: | ---------- | --------- | -------- |
| EVT-001  | GoPay  | Rp50.000 | Device #01 | Delivered | 17:30    |
| EVT-002  | GoPay  | Rp25.000 | Device #01 | Matched   | 17:28    |

---

# 21. Event Detail

Route:

```text
/events/:id
```

Sections:

### Event Metadata

```text
Event ID
Source
Device
Environment
Received At
Processed At
```

### Notification

```text
Title
Message
Timestamp
```

### Parsed Payment

```text
Amount
Currency
Payment Type
```

### Processing Status

```text
Received ✓
Parsed ✓
Persisted ✓
Matched ✓
Webhook ✓
```

### Retry

```text
Retry Count
Last Retry
Next Retry
```

---

# 22. Webhooks

Route:

```text
/webhooks
```

Purpose:

Mengelola endpoint tujuan webhook.

---

## 22.1 Webhook List

| Name       | Endpoint                                 | Events | Status | Last Delivery |
| ---------- | ---------------------------------------- | ------ | ------ | ------------- |
| Production | https://customer.com/api/payment         | 3      | Active | 17:30         |
| Staging    | https://staging.customer.com/api/payment | 3      | Active | 15:20         |

Actions:

* View
* Edit
* Test
* Disable
* Delete

---

# 23. Create Webhook

Fields:

```text
Name
Endpoint URL
Events
Secret
Active
```

Events:

```text
payment.received
payment.matched
payment.failed
payment.updated
```

Secret:

```text
Generate Secret
```

Secret harus ditampilkan secara aman.

---

# 24. Webhook Deliveries

Route:

```text
/webhooks/:id/deliveries
```

Table:

| Time  | Event            | Status  | HTTP | Duration |
| ----- | ---------------- | ------- | ---: | -------: |
| 17:30 | payment.received | Success |  200 |     42ms |
| 17:29 | payment.received | Failed  |  500 |     81ms |

Status:

```text
DELIVERED
FAILED
RETRYING
PENDING
```

---

# 25. Webhook Delivery Detail

Menampilkan:

### Request

```text
Method
URL
Headers
Body
```

### Response

```text
HTTP Status
Headers
Body
Duration
```

### Retry

```text
Attempt
Retry Count
Next Retry
```

Actions:

```text
Retry Delivery
```

Retry harus membutuhkan confirmation jika delivery berpotensi membuat duplicate action pada customer system.

---

# 26. API Keys

Route:

```text
/api-keys
```

Table:

| Name        | Environment | Created | Last Used | Status |
| ----------- | ----------- | ------- | --------- | ------ |
| Production  | Production  | Jan 01  | Today     | Active |
| Development | UAT         | Jan 02  | Yesterday | Active |

Actions:

* Create
* Revoke
* Rotate

Secret hanya ditampilkan sekali saat creation.

---

# 27. License

Route:

```text
/license
```

Purpose:

Menampilkan informasi license installation.

---

## 27.1 License Summary

```text
Business

● Active

License ID
PB-BUS-001

Activated
01 Jan 2027

Expires
31 Dec 2027

183 days remaining
```

---

## 27.2 Entitlements

```text
Devices
8 / 10

Installations
1 / 1

Payment Sources
3 / Unlimited

Webhooks
8 / 20
```

---

## 27.3 License Timeline

```text
01 Jan 2027
License Activated

01 Jun 2027
License Validated

17 Dec 2027
Renewal Reminder

31 Dec 2027
License Expiry
```

---

# 28. License Warning States

### Normal

```text
● License Active
```

### Expiring Soon

```text
⚠ License expires in 14 days.
[Renew License]
```

### Expired

```text
License expired.

Some operational features may be restricted.

[Renew License]
```

### Suspended

```text
License suspended.

Contact administrator.
```

### Revoked

```text
License revoked.

Contact support.
```

---

# 29. Settings

Route:

```text
/settings
```

Sections:

```text
General
Environment
Security
Notifications
System
```

---

## 29.1 General

Fields:

```text
Organization Name
Timezone
Date Format
Currency
```

---

## 29.2 Environment

```text
Environment
Production

Installation ID
INS-001

Version
1.4.2

Backend URL
https://payment.customer.com
```

Environment harus read-only jika dikontrol oleh deployment configuration.

---

## 29.3 Security

Options:

```text
Session Timeout
Two-Factor Authentication
Allowed Origins
Webhook Security
API Security
```

---

## 29.4 Notifications

Customer dapat mengatur:

```text
License Expiration
Device Offline
Webhook Failure
System Error
```

Notification channel dapat dikembangkan:

* Dashboard
* Email
* Webhook
* Future: Slack/WhatsApp/etc.

---

# 30. Logs

Route:

```text
/logs
```

Purpose:

Technical troubleshooting.

Categories:

```text
SYSTEM
DEVICE
EVENT
WEBHOOK
LICENSE
AUTH
```

Table:

| Time  | Type    | Severity | Message             |
| ----- | ------- | -------- | ------------------- |
| 17:30 | WEBHOOK | INFO     | Delivery successful |
| 17:29 | DEVICE  | WARNING  | Device offline      |
| 17:28 | LICENSE | INFO     | License validated   |

---

# 31. Log Detail

Menampilkan:

```text
Timestamp
Severity
Category
Message
Request ID
Device ID
Event ID
Metadata
```

Sensitive information harus di-redact.

Jangan menampilkan:

* passwords
* API secrets
* license private secrets
* unnecessary personal information

---

# 32. About

Route:

```text
/about
```

Information:

```text
Payment Notification Bridge

Version
1.4.2

API Version
v1

License
Business

Documentation
Support
Release Notes
```

---

# 33. UAT / Production Environment

Environment switcher harus tersedia jika customer memiliki kedua environment.

```text
┌───────────────┐
│ Production ▼  │
└───────────────┘
```

Dropdown:

```text
● Production
○ UAT
```

Saat berpindah environment:

* API endpoint berubah
* data berubah
* devices berubah
* events berubah
* webhooks berubah

UAT dan Production tidak boleh bercampur.

---

# 34. UAT Banner

Jika sedang berada di UAT:

```text
┌─────────────────────────────────────────────────────┐
│ ⚠ UAT ENVIRONMENT                                  │
│ You are currently viewing test data.               │
└─────────────────────────────────────────────────────┘
```

Banner harus selalu terlihat.

Production tidak menggunakan banner yang sama.

---

# 35. Empty States

Setiap halaman harus memiliki empty state yang jelas.

Contoh Devices:

```text
No devices connected

Connect an Android Bridge device to start
receiving payment notifications.

[Connect Device]
```

Webhook:

```text
No webhook endpoints

Create a webhook endpoint to receive
payment events.

[Create Webhook]
```

Transactions:

```text
No transactions found

Transactions will appear here after
payment events are processed.
```

---

# 36. Loading States

Gunakan skeleton loading untuk:

* cards
* tables
* detail pages
* dashboard statistics

Hindari blank screen.

Contoh:

```text
┌──────────────┐
│ ░░░░░░░░░░░ │
│ ░░░░░░░░    │
└──────────────┘
```

---

# 37. Error States

Error harus menjelaskan:

1. Apa yang terjadi
2. Dampaknya
3. Apa yang bisa dilakukan

Contoh:

```text
Unable to load devices

The backend could not retrieve device information.

[Try Again]
```

Jangan hanya menampilkan:

```text
Error 500
```

---

# 38. Confirmation Modal

Action berbahaya harus menggunakan confirmation.

Contoh:

```text
Disable Device?

Device "Kasir Utama" will stop sending
payment events.

[Cancel] [Disable Device]
```

Untuk delete:

```text
Delete Webhook?

This action cannot be undone.

[Cancel] [Delete Webhook]
```

---

# 39. Toast Notification

Gunakan toast untuk feedback ringan.

Success:

```text
✓ Webhook created successfully.
```

Error:

```text
✕ Failed to save webhook.
```

Warning:

```text
⚠ Device is currently offline.
```

---

# 40. Status System

Gunakan semantic status yang konsisten.

### Success

```text
ACTIVE
ONLINE
DELIVERED
PAID
HEALTHY
```

### Warning

```text
EXPIRING
PENDING
RETRYING
DEGRADED
```

### Error

```text
FAILED
OFFLINE
EXPIRED
SUSPENDED
REVOKED
```

Status tidak hanya dibedakan menggunakan warna.

Gunakan:

* icon
* text
* color
* tooltip jika diperlukan

Agar accessible.

---

# 41. Responsive Design

Dashboard harus responsive.

## Desktop

Target:

```text
≥ 1280px
```

Sidebar permanent.

## Tablet

```text
768px – 1279px
```

Sidebar dapat collapse.

## Mobile

```text
< 768px
```

Gunakan:

* compact navigation
* horizontally scrollable table jika diperlukan
* stacked cards
* bottom action / sticky action untuk action penting

Dashboard tidak harus memberikan seluruh kemampuan administration secara sempurna di mobile, tetapi monitoring harus tetap nyaman.

---

# 42. Dashboard Priority

Feature priority:

## P0 — Critical

* Overview
* Devices
* Events
* Webhooks
* License
* System status

## P1 — Important

* Transactions
* Payment Sources
* Logs
* API Keys
* Settings

## P2 — Future

* Analytics
* Advanced reporting
* Team management
* Advanced audit
* Export
* Notifications integration

---

# 43. Vendor Dashboard Navigation

Vendor Dashboard:

```text
Overview

CUSTOMERS
├── Customers
├── Installations
└── Support

LICENSING
├── Licenses
├── Activations
├── Renewals
└── Entitlements

PRODUCT
├── Releases
├── Versions
└── Connectors

SYSTEM
├── Audit Logs
└── Settings
```

---

# 44. Vendor Overview

Cards:

```text
Customers
127

Active Licenses
113

Expiring Soon
8

Expired
6

Active Installations
119
```

Charts:

* new customers
* license activations
* renewals
* expiration trend

---

# 45. Vendor Customers

Table:

| Customer | Plan     | Installations | Devices | License | Expiry   |
| -------- | -------- | ------------: | ------: | ------- | -------- |
| PT ABC   | Business |             2 |       8 | Active  | Dec 2027 |
| PT XYZ   | Starter  |             1 |       2 | Active  | Mar 2027 |

Actions:

* View
* View licenses
* View installations
* Support

Vendor tidak dapat melihat raw payment data.

---

# 46. Vendor Customer Detail

Information:

```text
Customer
PT ABC

Plan
Business

License
PB-BUS-001

Installations
2

Devices
8

Version
1.4.2

License Expiry
31 Dec 2027
```

Tabs:

```text
Overview
Licenses
Installations
Versions
Support
```

---

# 47. Vendor License Management

Features:

* create license
* activate
* suspend
* revoke
* renew
* extend
* change entitlement
* reset installation

License table:

| License | Customer | Plan     | Status  | Expiry   |
| ------- | -------- | -------- | ------- | -------- |
| PB-001  | PT ABC   | Business | Active  | Dec 2027 |
| PB-002  | PT XYZ   | Starter  | Expired | Mar 2026 |

---

# 48. Vendor Installation Management

Information:

```text
Installation ID
Customer
Environment
Version
First Activated
Last Validation
Status
```

Statuses:

```text
ACTIVE
OFFLINE
SUSPENDED
REVOKED
UNKNOWN
```

Vendor hanya menerima metadata yang diperlukan untuk licensing dan support.

---

# 49. Vendor Release Management

Dashboard dapat mencatat:

```text
Version
Release Date
Release Type
Minimum Compatible Version
Changelog
```

Example:

```text
v1.4.2

Stable

Released:
10 Jan 2027

Changes:
- Improved webhook retry
- Fixed device heartbeat
- Improved GoPay parser
```

Future:

```text
[Publish Release]
```

---

# 50. Vendor Audit Logs

Audit log:

```text
Who
Action
Resource
Timestamp
IP
Result
```

Contoh:

```text
Admin
Renewed license PB-001
10 Jan 2027 10:30
Success
```

Audit log tidak boleh mudah dihapus oleh ordinary admin.

---

# 51. Dashboard Security

Customer Dashboard:

* authentication
* session management
* CSRF protection jika applicable
* rate limiting
* role-based access control
* secure password storage
* optional 2FA
* secure API tokens
* audit logging

Vendor Dashboard:

* stronger authentication
* mandatory 2FA
* strict RBAC
* IP restriction optional
* audit log
* session timeout
* sensitive action confirmation

---

# 52. Data Boundary

Customer Dashboard:

```text
Can access:
- Transactions
- Events
- Webhooks
- Devices
- Customer data
```

Vendor Dashboard:

```text
Can access:
- Customer metadata
- License
- Installation
- Version
- Activation
- Renewal
```

Vendor Dashboard:

```text
Must NOT access by default:
- Raw payment notification
- Transaction amount
- Webhook payload
- Customer database
- Customer API secrets
```

---

# 53. Recommended Dashboard UX Flow

Customer pertama kali membuka dashboard:

```text
Login
  ↓
Environment Setup
  ↓
License Activation
  ↓
Connect Android Device
  ↓
Configure Payment Source
  ↓
Configure Webhook
  ↓
Test Connection
  ↓
UAT
  ↓
Production
```

Dashboard harus membantu customer mengikuti flow tersebut.

---

# 54. First-Time Setup Wizard

Pada installation baru:

```text
Welcome to Payment Bridge

Step 1
Activate License

Step 2
Connect Android Device

Step 3
Connect Payment Source

Step 4
Configure Webhook

Step 5
Test Integration

Step 6
Complete
```

Progress:

```text
●────●────○────○────○────○
```

Customer dapat melewati step tertentu jika diperlukan, tetapi recommended flow tetap tersedia.

---

# 55. System Health

Dashboard harus menyediakan satu tempat untuk mengetahui apakah seluruh pipeline bekerja.

```text
Android Device
      ↓
Notification Listener
      ↓
Backend API
      ↓
Event Processor
      ↓
Database
      ↓
Transaction Matching
      ↓
Webhook
      ↓
Customer Website
```

Health check harus memvisualisasikan pipeline tersebut.

Contoh:

```text
Android Device      ●
Backend API         ●
Event Processor     ●
Database            ●
Webhook             ●
```

Jika error:

```text
Android Device      ●
Backend API         ●
Event Processor     ●
Database            ●
Webhook             ✕
```

User langsung mengetahui bahwa masalah berada di webhook layer.

---

# 56. Dashboard Success Criteria

Dashboard dianggap selesai apabila:

* [ ] Customer dapat login
* [ ] Customer dapat melihat environment
* [ ] Customer dapat melihat license
* [ ] Customer dapat melihat device
* [ ] Customer dapat melihat payment source
* [ ] Customer dapat melihat event
* [ ] Customer dapat melihat transaction
* [ ] Customer dapat mengelola webhook
* [ ] Customer dapat melihat webhook delivery
* [ ] Customer dapat melihat logs
* [ ] Customer dapat melihat system health
* [ ] Customer dapat mengelola API keys
* [ ] Customer dapat melihat license expiry
* [ ] UAT dan Production terpisah
* [ ] Role/permission berjalan
* [ ] Empty state tersedia
* [ ] Loading state tersedia
* [ ] Error state tersedia
* [ ] Confirmation modal tersedia
* [ ] Responsive layout tersedia

Vendor:

* [ ] Customer management
* [ ] License management
* [ ] Installation management
* [ ] Activation management
* [ ] Renewal management
* [ ] Release management
* [ ] Audit log
* [ ] RBAC
* [ ] Vendor security controls

---

# 57. Final Dashboard Architecture

Final architecture:

```text
                           VENDOR
                             │
                   ┌─────────┴─────────┐
                   │                   │
            Vendor Dashboard      License Server
                   │                   │
                   └─────────┬─────────┘
                             │
                       License Only
                             │
                             ▼
                    CUSTOMER SERVER
                             │
                  ┌──────────┴──────────┐
                  │                     │
          Customer Dashboard      Backend API
                  │                     │
                  └──────────┬──────────┘
                             │
                    ┌────────┴────────┐
                    │                 │
                Database            Queue
                    │
                    ▼
              Payment Events
                    ▲
                    │
             Android Bridge
                    │
          ┌─────────┼─────────┐
          ▼         ▼         ▼
        GoPay      DANA      OVO
```

---

# 58. Product Philosophy

Dashboard harus membuat customer merasa bahwa mereka memiliki sebuah **payment infrastructure platform**, bukan sekadar aplikasi Android.

Customer harus dapat menjawab lima pertanyaan hanya dari dashboard:

1. **Apakah sistem saya aktif?**
2. **Apakah Android device saya online?**
3. **Apakah payment notification diterima?**
4. **Apakah webhook berhasil dikirim?**
5. **Apakah license saya masih aktif?**

Jika kelima hal tersebut dapat diketahui dengan cepat, dashboard sudah memenuhi fungsi utamanya.

---

# 59. Recommended MVP Dashboard

Untuk versi pertama, jangan implementasikan semua halaman sekaligus.

Prioritas MVP:

```text
Customer Dashboard
│
├── Overview
├── Devices
├── Payment Sources
├── Events
├── Webhooks
├── License
└── Settings
```

Kemudian Phase 2:

```text
├── Transactions
├── Webhook Deliveries
├── Logs
└── API Keys
```

Phase 3:

```text
├── Analytics
├── Team Management
├── Advanced Audit
└── Reporting
```

Vendor Dashboard MVP:

```text
Vendor Dashboard
│
├── Overview
├── Customers
├── Licenses
├── Installations
├── Activations
└── Audit Logs
```

---

# 60. Final Principle

**Customer Dashboard = operational control.**

**Vendor Dashboard = licensing and product management.**

**Customer owns payment data.**

**Vendor owns software licensing.**

Dengan separation tersebut, model **Self-Hosted + Annual License** tetap scalable, profesional, dan dapat berkembang dari satu connector seperti GoPay menjadi platform dengan berbagai payment notification sources.
