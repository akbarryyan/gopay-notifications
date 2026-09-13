/**
 * Klien API dashboard admin.
 *
 * Seluruh permintaan memakai path relatif (/api/v1/admin/...), bukan URL
 * absolut. Di dev, next.config.ts me-rewrite /api/* ke backend Go lokal. Di
 * produksi, Caddy yang merutekan /api/* langsung ke backend dan sisanya ke
 * Next.js — satu origin, satu domain, tanpa CORS sama sekali.
 *
 * Autentikasi lewat cookie sesi HttpOnly; fetch same-origin sudah menyertakan
 * cookie itu sendiri tanpa perlu header tambahan.
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

type ErrorBody = {
  success: false;
  error: string;
  message: string;
  server_time?: number;
};

async function apiFetch<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(path, {
    ...init,
    headers: {
      "Content-Type": "application/json",
      ...init?.headers,
    },
  });

  if (!res.ok) {
    let body: ErrorBody | null = null;
    try {
      body = await res.json();
    } catch {
      // Body bukan JSON — bisa terjadi bila proxy di depan (mis. Caddy)
      // menjawab sendiri sebelum request sampai ke backend.
    }
    throw new ApiError(
      body?.error ?? "unknown_error",
      body?.message ?? `Permintaan gagal dengan status ${res.status}`,
      res.status,
    );
  }

  return res.json() as Promise<T>;
}

// --- Auth --------------------------------------------------------------

export function login(username: string, password: string): Promise<{ success: true }> {
  return apiFetch("/api/v1/admin/login", {
    method: "POST",
    body: JSON.stringify({ username, password }),
  });
}

export function logout(): Promise<{ success: true }> {
  return apiFetch("/api/v1/admin/logout", { method: "POST" });
}

// --- Overview ------------------------------------------------------------

export interface OverviewResponse {
  devices: {
    total: number;
    online: number;
    offline: number;
    disabled: number;
    pending: number;
  };
  events: {
    today: number;
    last_7_days: number;
    latest_at: string | null;
    /** 14 hari terakhir, hari tertua lebih dulu, hari sepi ikut disertakan dengan count 0. */
    daily: { date: string; count: number }[];
  };
  system: {
    backend: string;
    database: string;
  };
}

export function getOverview(): Promise<OverviewResponse> {
  return apiFetch("/api/v1/admin/overview");
}

// --- Devices ---------------------------------------------------------------

export type DeviceStatus = "PENDING" | "ONLINE" | "OFFLINE" | "DISABLED";

export interface AdminDevice {
  device_id: string;
  name: string;
  enabled: boolean;
  status: DeviceStatus;
  created_at: string;
  last_seen_at: string | null;
  heartbeat_at: string | null;
  app_version: string | null;
  android_version: string | null;
  listener_connected: boolean | null;
  pending_count: number | null;
  failed_count: number | null;
}

export async function getDevices(): Promise<AdminDevice[]> {
  const res = await apiFetch<{ devices: AdminDevice[] }>("/api/v1/admin/devices");
  return res.devices;
}

export function setDeviceEnabled(deviceId: string, enabled: boolean): Promise<{ success: true }> {
  return apiFetch(`/api/v1/admin/devices/${encodeURIComponent(deviceId)}`, {
    method: "PATCH",
    body: JSON.stringify({ enabled }),
  });
}

// --- Events ------------------------------------------------------------

export interface AdminEvent {
  event_id: string;
  device_id: string;
  source: string;
  package_name: string;
  title: string | null;
  text: string | null;
  big_text: string | null;
  amount_hint: number | null;
  posted_at: string;
  received_at: string;
  raw_payload: unknown;
}

export interface EventFilter {
  /** ID connector, mis. "gopay". Kosong/undefined berarti semua sumber. */
  source?: string;
  /** Cocok sebagian ke device_id atau title, tanpa peduli huruf besar/kecil. */
  q?: string;
  /** YYYY-MM-DD, inklusif. */
  from?: string;
  /** YYYY-MM-DD, inklusif. */
  to?: string;
}

export async function getEvents(
  limit = 50,
  offset = 0,
  filter: EventFilter = {},
): Promise<AdminEvent[]> {
  const params = new URLSearchParams({ limit: String(limit), offset: String(offset) });
  if (filter.source) params.set("source", filter.source);
  if (filter.q) params.set("q", filter.q);
  if (filter.from) params.set("from", filter.from);
  if (filter.to) params.set("to", filter.to);

  const res = await apiFetch<{ events: AdminEvent[] }>(`/api/v1/admin/events?${params}`);
  return res.events;
}

// --- Sources -------------------------------------------------------------

export interface SourceInfo {
  id: string;
  name: string;
  packages: string[];
}

export async function getSources(): Promise<SourceInfo[]> {
  const res = await apiFetch<{ sources: SourceInfo[] }>("/api/v1/sources");
  return res.sources;
}

// --- Invoices (Transactions) -----------------------------------------------

export type InvoiceStatus = "PENDING" | "PAID" | "EXPIRED";

export interface AdminInvoice {
  id: string;
  external_ref: string;
  requested_amount: number;
  unique_amount: number;
  status: InvoiceStatus;
  matched_event_id: string | null;
  created_at: string;
  expires_at: string;
  paid_at: string | null;
}

export interface InvoiceFilter {
  /** Satu status, atau beberapa sekaligus (mis. saat mencari invoice target di konsol pengecualian). */
  status?: InvoiceStatus | InvoiceStatus[];
  /** Cocok sebagian ke external_ref, tanpa peduli huruf besar/kecil. */
  q?: string;
  /** YYYY-MM-DD, inklusif. */
  from?: string;
  /** YYYY-MM-DD, inklusif. */
  to?: string;
}

export async function getInvoices(
  limit = 50,
  offset = 0,
  filter: InvoiceFilter = {},
): Promise<AdminInvoice[]> {
  const params = new URLSearchParams({ limit: String(limit), offset: String(offset) });
  if (filter.status) {
    params.set("status", Array.isArray(filter.status) ? filter.status.join(",") : filter.status);
  }
  if (filter.q) params.set("q", filter.q);
  if (filter.from) params.set("from", filter.from);
  if (filter.to) params.set("to", filter.to);

  const res = await apiFetch<{ invoices: AdminInvoice[] }>(`/api/v1/admin/invoices?${params}`);
  return res.invoices;
}

// --- API Keys ------------------------------------------------------------

export interface AdminAPIKey {
  id: string;
  name: string;
  created_at: string;
  revoked_at: string | null;
}

export async function getAPIKeys(): Promise<AdminAPIKey[]> {
  const res = await apiFetch<{ api_keys: AdminAPIKey[] }>("/api/v1/admin/api-keys");
  return res.api_keys;
}

/** `key` di response cuma ada sekali, tepat saat ini — tidak bisa diambil lagi setelahnya. */
export function createAPIKey(name: string): Promise<AdminAPIKey & { key: string }> {
  return apiFetch("/api/v1/admin/api-keys", {
    method: "POST",
    body: JSON.stringify({ name }),
  });
}

export function revokeAPIKey(id: string): Promise<{ success: true }> {
  return apiFetch(`/api/v1/admin/api-keys/${encodeURIComponent(id)}`, { method: "PATCH" });
}

// --- Webhooks --------------------------------------------------------------

export type WebhookEvent = "invoice.paid" | "invoice.expired";

export interface AdminWebhook {
  id: string;
  name: string;
  url: string;
  events: WebhookEvent[];
  enabled: boolean;
  created_at: string;
  last_delivery_at: string | null;
  last_delivery_status: string | null;
}

export async function getWebhooks(): Promise<AdminWebhook[]> {
  const res = await apiFetch<{ webhooks: AdminWebhook[] }>("/api/v1/admin/webhooks");
  return res.webhooks;
}

/** `secret` di response cuma ada sekali, tepat saat ini — tidak bisa diambil lagi setelahnya. */
export function createWebhook(
  name: string,
  url: string,
  events: WebhookEvent[],
): Promise<AdminWebhook & { secret: string }> {
  return apiFetch("/api/v1/admin/webhooks", {
    method: "POST",
    body: JSON.stringify({ name, url, events }),
  });
}

export function setWebhookEnabled(id: string, enabled: boolean): Promise<{ success: true }> {
  return apiFetch(`/api/v1/admin/webhooks/${encodeURIComponent(id)}`, {
    method: "PATCH",
    body: JSON.stringify({ enabled }),
  });
}

export function deleteWebhook(id: string): Promise<{ success: true }> {
  return apiFetch(`/api/v1/admin/webhooks/${encodeURIComponent(id)}`, { method: "DELETE" });
}

export interface WebhookTestResult {
  delivered: boolean;
  http_status: number;
  duration_ms: number;
}

export function testWebhook(id: string): Promise<WebhookTestResult> {
  return apiFetch(`/api/v1/admin/webhooks/${encodeURIComponent(id)}/test`, { method: "POST" });
}

export interface WebhookDelivery {
  id: string;
  event: string;
  invoice_id: string | null;
  status: "PENDING" | "RETRYING" | "DELIVERED" | "FAILED";
  attempt: number;
  http_status: number | null;
  duration_ms: number | null;
  created_at: string;
  delivered_at: string | null;
}

export async function getWebhookDeliveries(
  webhookId: string,
  limit = 50,
  offset = 0,
): Promise<WebhookDelivery[]> {
  const params = new URLSearchParams({ limit: String(limit), offset: String(offset) });
  const res = await apiFetch<{ deliveries: WebhookDelivery[] }>(
    `/api/v1/admin/webhooks/${encodeURIComponent(webhookId)}/deliveries?${params}`,
  );
  return res.deliveries;
}

// --- Exceptions ------------------------------------------------------------
//
// Event dengan amount_hint terisi yang tidak cocok invoice manapun — uang
// yang sudah masuk tapi belum jelas ini bayar untuk order yang mana.

export interface ExceptionFilter {
  q?: string;
  /** YYYY-MM-DD, inklusif. */
  from?: string;
  /** YYYY-MM-DD, inklusif. */
  to?: string;
}

export async function getExceptions(
  limit = 50,
  offset = 0,
  filter: ExceptionFilter = {},
): Promise<AdminEvent[]> {
  const params = new URLSearchParams({ limit: String(limit), offset: String(offset) });
  if (filter.q) params.set("q", filter.q);
  if (filter.from) params.set("from", filter.from);
  if (filter.to) params.set("to", filter.to);

  const res = await apiFetch<{ exceptions: AdminEvent[] }>(`/api/v1/admin/exceptions?${params}`);
  return res.exceptions;
}

export function matchException(eventId: string, invoiceId: string): Promise<{ success: true }> {
  return apiFetch(`/api/v1/admin/exceptions/${encodeURIComponent(eventId)}/match`, {
    method: "POST",
    body: JSON.stringify({ invoice_id: invoiceId }),
  });
}

export function dismissException(eventId: string, note?: string): Promise<{ success: true }> {
  return apiFetch(`/api/v1/admin/exceptions/${encodeURIComponent(eventId)}/dismiss`, {
    method: "POST",
    body: JSON.stringify({ note: note ?? "" }),
  });
}

// --- License -----------------------------------------------------------
//
// Lisensi offline (lihat docs/superpowers/specs/2026-09-13-license-system-design.md).
// Endpoint ini SENGAJA tetap bisa dipanggil walau lisensi tidak aktif —
// beda dari endpoint lain di file ini yang akan gagal dengan ApiError
// (code "license_expired"/"license_missing"/"license_invalid", status 402)
// bila lisensi tidak aktif.

export type LicenseStatus = "active" | "missing" | "invalid" | "expired";

export interface LicenseInfo {
  customer?: string;
  domain?: string;
  plan?: string;
  issued_at?: string;
  expires_at?: string;
  days_remaining?: number;
  status: LicenseStatus;
  reason?: string;
}

export async function getLicense(): Promise<LicenseInfo> {
  const res = await apiFetch<{ license: LicenseInfo }>("/api/v1/admin/license");
  return res.license;
}
