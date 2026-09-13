/**
 * Klien API Vendor Console — memanggil License Server
 * (backend/internal/licenseserver), bukan backend customer manapun.
 *
 * Path relatif (/api/v1/admin/...), sama pola dengan dashboard customer:
 * next.config.ts me-rewrite /api/* ke LICENSE_SERVER_URL saat dev, Caddy
 * yang merutekan saat produksi. Autentikasi lewat cookie sesi HttpOnly
 * ("vendor_session").
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
  return apiFetch("/api/v1/admin/login", {
    method: "POST",
    body: JSON.stringify({ username, password }),
  });
}

export function logout(): Promise<{ success: true }> {
  return apiFetch("/api/v1/admin/logout", { method: "POST" });
}

// --- Customers -------------------------------------------------------------

export interface Customer {
  id: string;
  name: string;
  created_at: string;
}

export async function getCustomers(): Promise<Customer[]> {
  const res = await apiFetch<{ customers: Customer[] }>("/api/v1/admin/customers");
  return res.customers;
}

export function createCustomer(name: string): Promise<{ success: true; customer: Customer }> {
  return apiFetch("/api/v1/admin/customers", {
    method: "POST",
    body: JSON.stringify({ name }),
  });
}

// --- Licenses --------------------------------------------------------------

export type LicensePlan = "Starter" | "Business" | "Enterprise";
export type LicenseStatus = "active" | "expiring" | "expired" | "suspended" | "revoked";

export interface License {
  id: string;
  customer_id: string;
  plan: string;
  status: LicenseStatus;
  max_devices: number;
  production_installations: number;
  uat_installations: number;
  issued_at: string;
  expires_at: string;
  days_remaining: number;
}

export interface Installation {
  id: string;
  environment: "production" | "uat";
  product_version: string;
  activated_at: string;
  released_at: string | null;
}

export async function getCustomerDetail(
  customerId: string,
): Promise<{ customer: Customer; licenses: License[] }> {
  return apiFetch(`/api/v1/admin/customers/${encodeURIComponent(customerId)}`);
}

export async function createLicense(
  customerId: string,
  plan: LicensePlan,
  expiresAt: string,
): Promise<{ success: true; license: License & { key: string } }> {
  return apiFetch(`/api/v1/admin/customers/${encodeURIComponent(customerId)}/licenses`, {
    method: "POST",
    body: JSON.stringify({ plan, expires_at: expiresAt }),
  });
}

export async function getLicenseDetail(
  licenseId: string,
): Promise<{ license: License; installations: Installation[] }> {
  return apiFetch(`/api/v1/admin/licenses/${encodeURIComponent(licenseId)}`);
}

export function renewLicense(licenseId: string, expiresAt: string): Promise<{ success: true }> {
  return apiFetch(`/api/v1/admin/licenses/${encodeURIComponent(licenseId)}/renew`, {
    method: "POST",
    body: JSON.stringify({ expires_at: expiresAt }),
  });
}

export function suspendLicense(licenseId: string): Promise<{ success: true }> {
  return apiFetch(`/api/v1/admin/licenses/${encodeURIComponent(licenseId)}/suspend`, {
    method: "POST",
  });
}

export function revokeLicense(licenseId: string): Promise<{ success: true }> {
  return apiFetch(`/api/v1/admin/licenses/${encodeURIComponent(licenseId)}/revoke`, {
    method: "POST",
  });
}

export function resetInstallation(installationId: string): Promise<{ success: true }> {
  return apiFetch(`/api/v1/admin/installations/${encodeURIComponent(installationId)}/reset`, {
    method: "POST",
  });
}

// --- Audit log ---------------------------------------------------------------

export interface AuditEntry {
  id: number;
  actor: string;
  action: string;
  resource: string;
  metadata: unknown;
  created_at: string;
}

export async function getAuditLog(): Promise<AuditEntry[]> {
  const res = await apiFetch<{ entries: AuditEntry[] }>("/api/v1/admin/audit-log");
  return res.entries;
}
