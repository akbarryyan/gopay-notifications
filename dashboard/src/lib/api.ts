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

export async function getEvents(limit = 50, offset = 0): Promise<AdminEvent[]> {
  const res = await apiFetch<{ events: AdminEvent[] }>(
    `/api/v1/admin/events?limit=${limit}&offset=${offset}`,
  );
  return res.events;
}
