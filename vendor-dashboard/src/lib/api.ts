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
