/**
 * Klien API Vendor Dashboard -- memanggil endpoint vendor
 * (/api/v1/vendor/*) di backend utama. License Server yang dulu terpisah
 * sudah dibongkar total; backend utama sekarang satu-satunya service.
 *
 * Path relatif, sama pola dengan dashboard customer: next.config.ts
 * me-rewrite /api/* ke BACKEND_URL saat dev, Caddy yang merutekan saat
 * produksi. Autentikasi lewat cookie sesi HttpOnly ("vendor_session") --
 * terpisah total dari sesi customer (admin_session).
 */

export class ApiError extends Error {
  constructor(
    public readonly code: string,
    message: string,
    public readonly status: number,
  ) {
    super(message);
    this.name = "ApiError";
  }
}

type ErrorBody = { success: false; error: string; message: string };

async function apiFetch<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(path, {
    ...init,
    headers: { "Content-Type": "application/json", ...init?.headers },
  });

  if (!res.ok) {
    let body: ErrorBody | null = null;
    try {
      body = await res.json();
    } catch {
      // Body bukan JSON -- proxy di depan mungkin menjawab sendiri.
    }
    throw new ApiError(
      body?.error ?? "unknown_error",
      body?.message ?? `Permintaan gagal dengan status ${res.status}`,
      res.status,
    );
  }
  return res.json() as Promise<T>;
}

// --- Auth ----------------------------------------------------------------

export function login(username: string, password: string): Promise<{ success: true }> {
  return apiFetch("/api/v1/vendor/login", {
    method: "POST",
    body: JSON.stringify({ username, password }),
  });
}

export function logout(): Promise<{ success: true }> {
  return apiFetch("/api/v1/vendor/logout", { method: "POST" });
}

// --- Accounts --------------------------------------------------------------

export type AccountPlan = "Starter" | "Business" | "Enterprise";
export type AccountStatus = "active" | "expiring" | "expired" | "suspended" | "revoked";

export interface Account {
  id: string;
  business_name: string;
  email: string;
  username: string;
  plan: AccountPlan;
  max_devices: number;
  status: AccountStatus;
  expires_at: string;
  days_remaining: number;
  created_at: string;
}

export async function getAccounts(): Promise<Account[]> {
  const res = await apiFetch<{ accounts: Account[] }>("/api/v1/vendor/accounts");
  return res.accounts;
}

export function createAccount(input: {
  business_name: string;
  email: string;
  username: string;
  plan: AccountPlan;
  expires_at: string;
}): Promise<{ success: true; account: Account; initial_password: string }> {
  return apiFetch("/api/v1/vendor/accounts", {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export async function getAccount(id: string): Promise<Account> {
  const res = await apiFetch<{ account: Account }>(`/api/v1/vendor/accounts/${encodeURIComponent(id)}`);
  return res.account;
}

export function renewAccount(id: string, expiresAt: string): Promise<{ success: true }> {
  return apiFetch(`/api/v1/vendor/accounts/${encodeURIComponent(id)}/renew`, {
    method: "POST",
    body: JSON.stringify({ expires_at: expiresAt }),
  });
}

export function changeAccountPlan(id: string, plan: AccountPlan): Promise<{ success: true }> {
  return apiFetch(`/api/v1/vendor/accounts/${encodeURIComponent(id)}/plan`, {
    method: "POST",
    body: JSON.stringify({ plan }),
  });
}

export function suspendAccount(id: string): Promise<{ success: true }> {
  return apiFetch(`/api/v1/vendor/accounts/${encodeURIComponent(id)}/suspend`, { method: "POST" });
}

export function revokeAccount(id: string): Promise<{ success: true }> {
  return apiFetch(`/api/v1/vendor/accounts/${encodeURIComponent(id)}/revoke`, { method: "POST" });
}

// --- Audit log ---------------------------------------------------------------

export interface AuditEntry {
  id: number;
  actor: string;
  action: string;
  resource: string;
  created_at: string;
}

export async function getAuditLog(): Promise<AuditEntry[]> {
  const res = await apiFetch<{ entries: AuditEntry[] }>("/api/v1/vendor/audit-log");
  return res.entries;
}

// --- Dashboard (ringkasan lintas account) ------------------------------------

export interface VendorOverview {
  accounts: {
    total: number;
    active: number;
    expiring: number;
    expired: number;
    suspended: number;
    revoked: number;
    new_this_week: number;
  };
  devices: {
    total: number;
  };
  revenue: {
    total_paid_rp: number;
  };
  /** 14 hari terakhir, hari tertua lebih dulu, lintas SEMUA account. */
  daily: { date: string; new_accounts: number; paid_amount_rp: number }[];
  /** Account paling baru dibuat, terbanyak 5 baris -- lihat Accounts untuk daftar lengkap. */
  recent_accounts: Account[];
}

export async function getOverview(): Promise<VendorOverview> {
  const res = await apiFetch<{ overview: VendorOverview }>("/api/v1/vendor/overview");
  return res.overview;
}

// --- Devices (per account, read-only untuk vendor) ---------------------------

export type DeviceStatus = "PENDING" | "ONLINE" | "OFFLINE" | "DISABLED";

export interface Device {
  device_id: string;
  name: string;
  enabled: boolean;
  status: DeviceStatus;
  created_at: string;
  last_seen_at: string | null;
  heartbeat_at: string | null;
  android_version: string | null;
}

export async function getAccountDevices(accountId: string): Promise<Device[]> {
  const res = await apiFetch<{ devices: Device[] }>(
    `/api/v1/vendor/accounts/${encodeURIComponent(accountId)}/devices`,
  );
  return res.devices;
}

// --- Settings (vendor sendiri) ------------------------------------------------

export function changeVendorPassword(
  currentPassword: string,
  newPassword: string,
): Promise<{ success: true }> {
  return apiFetch("/api/v1/vendor/me/password", {
    method: "POST",
    body: JSON.stringify({ current_password: currentPassword, new_password: newPassword }),
  });
}

// --- Pengaturan notifikasi (pengingat kedaluwarsa ke customer) ---------------

/**
 * Backend TIDAK PERNAH mengirim balik password SMTP atau token bot --
 * cuma `*_set` yang menandakan apakah keduanya sudah tersimpan.
 */
export interface NotificationSettings {
  smtp_host: string;
  smtp_port: number;
  smtp_username: string;
  smtp_from: string;
  smtp_password_set: boolean;
  telegram_bot_token_set: boolean;
  email_configured: boolean;
  updated_at: string | null;
  updated_by: string | null;
}

/**
 * `smtp_password` dan `telegram_bot_token`: jangan disertakan (undefined)
 * untuk mempertahankan yang tersimpan, "" untuk menghapus, isi untuk
 * mengganti.
 */
export interface SaveNotificationSettingsInput {
  smtp_host: string;
  smtp_port: number;
  smtp_username: string;
  smtp_from: string;
  smtp_password?: string;
  telegram_bot_token?: string;
}

export async function getNotificationSettings(): Promise<NotificationSettings> {
  const res = await apiFetch<{ success: true; settings: NotificationSettings }>(
    "/api/v1/vendor/settings/notifications",
  );
  return res.settings;
}

export async function saveNotificationSettings(
  input: SaveNotificationSettingsInput,
): Promise<NotificationSettings> {
  const res = await apiFetch<{ success: true; settings: NotificationSettings }>(
    "/api/v1/vendor/settings/notifications",
    { method: "PUT", body: JSON.stringify(input) },
  );
  return res.settings;
}

export function sendTestNotification(
  channel: "email" | "telegram",
  to: string,
): Promise<{ success: true }> {
  return apiFetch("/api/v1/vendor/settings/notifications/test", {
    method: "POST",
    body: JSON.stringify({ channel, to }),
  });
}

// --- Transactions (lintas semua account) -------------------------------------

export type InvoiceStatus = "PENDING" | "PAID" | "EXPIRED";

export interface VendorInvoice {
  id: string;
  account_id: string;
  business_name: string;
  external_ref: string;
  requested_amount: number;
  unique_amount: number;
  status: InvoiceStatus;
  matched_event_id: string | null;
  created_at: string;
  expires_at: string;
  paid_at: string | null;
}

export async function getTransactions(
  limit: number,
  offset: number,
  filter: { q?: string; status?: InvoiceStatus; from?: string; to?: string },
): Promise<VendorInvoice[]> {
  const params = new URLSearchParams({ limit: String(limit), offset: String(offset) });
  if (filter.q) params.set("q", filter.q);
  if (filter.status) params.set("status", filter.status);
  if (filter.from) params.set("from", filter.from);
  if (filter.to) params.set("to", filter.to);
  const res = await apiFetch<{ invoices: VendorInvoice[] }>(`/api/v1/vendor/transactions?${params}`);
  return res.invoices;
}

// --- Webhook deliveries (lintas semua account, read-only) --------------------

export type DeliveryStatus = "PENDING" | "RETRYING" | "DELIVERED" | "FAILED";

export interface VendorDelivery {
  id: string;
  account_id: string;
  business_name: string;
  endpoint_id: string;
  endpoint_name: string;
  endpoint_url: string;
  event: string;
  invoice_id: string | null;
  status: DeliveryStatus;
  attempt: number;
  next_attempt_at: string | null;
  http_status: number | null;
  duration_ms: number | null;
  created_at: string;
  delivered_at: string | null;
}

export async function getWebhookDeliveries(
  limit: number,
  offset: number,
  filter: { q?: string; status?: DeliveryStatus; from?: string; to?: string },
): Promise<VendorDelivery[]> {
  const params = new URLSearchParams({ limit: String(limit), offset: String(offset) });
  if (filter.q) params.set("q", filter.q);
  if (filter.status) params.set("status", filter.status);
  if (filter.from) params.set("from", filter.from);
  if (filter.to) params.set("to", filter.to);
  const res = await apiFetch<{ deliveries: VendorDelivery[] }>(
    `/api/v1/vendor/webhook-deliveries?${params}`,
  );
  return res.deliveries;
}

// --- Riwayat notifikasi ke customer (lintas semua account, read-only) --------

export type NotificationKind =
  | "expiry_reminder"
  | "device_offline"
  | "device_online"
  | "password_reset"
  | "password_changed"
  | "test";
export type NotificationChannel = "email" | "telegram";
export type NotificationStatus = "sent" | "failed";

export interface NotificationLogEntry {
  id: number;
  /** null untuk pesan uji dari halaman Settings. */
  account_id: string | null;
  business_name: string | null;
  device_id: string | null;
  device_name: string | null;
  kind: NotificationKind;
  channel: NotificationChannel;
  recipient: string;
  subject: string;
  status: NotificationStatus;
  error: string | null;
  created_at: string;
}

export async function getNotificationLog(
  limit: number,
  offset: number,
  filter: {
    q?: string;
    kind?: NotificationKind;
    channel?: NotificationChannel;
    status?: NotificationStatus;
    from?: string;
    to?: string;
  },
): Promise<NotificationLogEntry[]> {
  const params = new URLSearchParams({ limit: String(limit), offset: String(offset) });
  for (const [k, v] of Object.entries(filter)) {
    if (v) params.set(k, v);
  }
  const res = await apiFetch<{ notifications: NotificationLogEntry[] }>(
    `/api/v1/vendor/notification-log?${params}`,
  );
  return res.notifications;
}
