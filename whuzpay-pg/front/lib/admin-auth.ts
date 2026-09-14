import { API_URL } from "./config";

const TOKEN_KEY = "admin_token";
const ADMIN_KEY = "admin_profile";
const REDIRECT_KEY = "admin_redirect_after_login";
const AUTH_NOTICE_KEY = "admin_auth_notice";

export type AdminAuthNotice = "required" | "expired";

export type AdminProfile = {
  id: string;
  name: string;
  email: string;
  is_active: boolean;
  last_login_at?: string | null;
  created_at: string;
  updated_at: string;
};

export type AdminLoginResponse = {
  token: string;
  token_type: string;
  expires_in: number;
  admin: AdminProfile;
};

export type ApiErrorBody = {
  error?: string;
  message?: string;
};

export function getAdminToken(): string | null {
  if (typeof window === "undefined") return null;
  return localStorage.getItem(TOKEN_KEY) ?? sessionStorage.getItem(TOKEN_KEY);
}

export function getAdminProfile(): AdminProfile | null {
  if (typeof window === "undefined") return null;
  const raw = localStorage.getItem(ADMIN_KEY) ?? sessionStorage.getItem(ADMIN_KEY);
  if (!raw) return null;
  try {
    return JSON.parse(raw) as AdminProfile;
  } catch {
    return null;
  }
}

export function saveAdminSession(
  data: AdminLoginResponse,
  remember: boolean,
): void {
  const storage = remember ? localStorage : sessionStorage;
  const other = remember ? sessionStorage : localStorage;

  storage.setItem(TOKEN_KEY, data.token);
  storage.setItem(ADMIN_KEY, JSON.stringify(data.admin));
  other.removeItem(TOKEN_KEY);
  other.removeItem(ADMIN_KEY);
}

/** Update cached profile in whichever storage currently holds the session. */
export function updateStoredAdminProfile(profile: AdminProfile): void {
  if (typeof window === "undefined") return;

  if (localStorage.getItem(TOKEN_KEY)) {
    localStorage.setItem(ADMIN_KEY, JSON.stringify(profile));
    sessionStorage.removeItem(ADMIN_KEY);
    return;
  }

  if (sessionStorage.getItem(TOKEN_KEY)) {
    sessionStorage.setItem(ADMIN_KEY, JSON.stringify(profile));
    localStorage.removeItem(ADMIN_KEY);
  }
}

export function clearAdminSession(): void {
  localStorage.removeItem(TOKEN_KEY);
  localStorage.removeItem(ADMIN_KEY);
  sessionStorage.removeItem(TOKEN_KEY);
  sessionStorage.removeItem(ADMIN_KEY);
}

function authHeaders(): HeadersInit {
  const token = getAdminToken();
  if (!token) {
    throw new Error("Session expired. Please sign in again.");
  }
  return {
    "Content-Type": "application/json",
    Authorization: `Bearer ${token}`,
  };
}

/** Remember where to go after login without polluting the login URL. */
export function setAdminRedirectPath(path: string): void {
  if (typeof window === "undefined") return;
  const safePath =
    path.startsWith("/admin") && !path.startsWith("/admin/login")
      ? path
      : "/admin";
  sessionStorage.setItem(REDIRECT_KEY, safePath);
}

export function consumeAdminRedirectPath(fallback = "/admin"): string {
  if (typeof window === "undefined") return fallback;
  const path = sessionStorage.getItem(REDIRECT_KEY);
  sessionStorage.removeItem(REDIRECT_KEY);
  if (!path || !path.startsWith("/admin") || path.startsWith("/admin/login")) {
    return fallback;
  }
  return path;
}

/** One-shot notice for login page (toast), kept out of the URL. */
export function setAdminAuthNotice(notice: AdminAuthNotice): void {
  if (typeof window === "undefined") return;
  sessionStorage.setItem(AUTH_NOTICE_KEY, notice);
}

export function consumeAdminAuthNotice(): AdminAuthNotice | null {
  if (typeof window === "undefined") return null;
  const value = sessionStorage.getItem(AUTH_NOTICE_KEY);
  sessionStorage.removeItem(AUTH_NOTICE_KEY);
  if (value === "required" || value === "expired") return value;
  return null;
}

export async function loginAdmin(
  email: string,
  password: string,
): Promise<AdminLoginResponse> {
  const res = await fetch(`${API_URL}/api/v1/auth/admin/login`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email, password }),
  });

  const body = (await res.json().catch(() => null)) as
    | AdminLoginResponse
    | ApiErrorBody
    | null;

  if (!res.ok) {
    const message =
      (body as ApiErrorBody | null)?.message ??
      "Gagal masuk. Periksa email dan password.";
    throw new Error(message);
  }

  return body as AdminLoginResponse;
}

export async function fetchAdminMe(): Promise<AdminProfile> {
  const res = await fetch(`${API_URL}/api/v1/auth/admin/me`, {
    method: "GET",
    headers: authHeaders(),
  });

  const body = (await res.json().catch(() => null)) as
    | AdminProfile
    | ApiErrorBody
    | null;

  if (!res.ok) {
    throw new Error(
      (body as ApiErrorBody | null)?.message ?? "Failed to load profile.",
    );
  }

  return body as AdminProfile;
}

export async function updateAdminProfile(
  name: string,
  email: string,
): Promise<AdminProfile> {
  const res = await fetch(`${API_URL}/api/v1/auth/admin/profile`, {
    method: "PUT",
    headers: authHeaders(),
    body: JSON.stringify({ name, email }),
  });

  const body = (await res.json().catch(() => null)) as
    | AdminProfile
    | ApiErrorBody
    | null;

  if (!res.ok) {
    throw new Error(
      (body as ApiErrorBody | null)?.message ?? "Failed to update profile.",
    );
  }

  return body as AdminProfile;
}

export async function changeAdminPassword(
  currentPassword: string,
  newPassword: string,
): Promise<void> {
  const res = await fetch(`${API_URL}/api/v1/auth/admin/change-password`, {
    method: "POST",
    headers: authHeaders(),
    body: JSON.stringify({
      current_password: currentPassword,
      new_password: newPassword,
    }),
  });

  const body = (await res.json().catch(() => null)) as ApiErrorBody | null;

  if (!res.ok) {
    throw new Error(body?.message ?? "Failed to change password.");
  }
}
