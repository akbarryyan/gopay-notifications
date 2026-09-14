import { API_URL } from "./config";

const TOKEN_KEY = "merchant_token";
const USER_KEY = "merchant_profile";
const REDIRECT_KEY = "merchant_redirect_after_login";
const AUTH_NOTICE_KEY = "merchant_auth_notice";
const ENV_KEY = "merchant_environment";

export type MerchantEnvironment = "sandbox" | "production";

export type MerchantAuthNotice = "required" | "expired";

export type MerchantProfile = {
  id: string;
  merchant_id: string;
  name: string;
  email: string;
  role: string;
  is_active: boolean;
  last_login_at?: string | null;
  created_at: string;
  updated_at: string;
  business_name?: string;
  webhook_url?: string | null;
};

export type MerchantLoginResponse = {
  token: string;
  token_type: string;
  expires_in: number;
  user: MerchantProfile;
};

type ApiErrorBody = {
  error?: string;
  message?: string;
};

export function getMerchantToken(): string | null {
  if (typeof window === "undefined") return null;
  return localStorage.getItem(TOKEN_KEY) ?? sessionStorage.getItem(TOKEN_KEY);
}

export function getMerchantProfile(): MerchantProfile | null {
  if (typeof window === "undefined") return null;
  const raw = localStorage.getItem(USER_KEY) ?? sessionStorage.getItem(USER_KEY);
  if (!raw) return null;
  try {
    return JSON.parse(raw) as MerchantProfile;
  } catch {
    return null;
  }
}

export function saveMerchantSession(
  data: MerchantLoginResponse,
  remember = true,
): void {
  const storage = remember ? localStorage : sessionStorage;
  const other = remember ? sessionStorage : localStorage;
  storage.setItem(TOKEN_KEY, data.token);
  storage.setItem(USER_KEY, JSON.stringify(data.user));
  other.removeItem(TOKEN_KEY);
  other.removeItem(USER_KEY);
}

export function updateStoredMerchantProfile(profile: MerchantProfile): void {
  if (typeof window === "undefined") return;
  if (localStorage.getItem(TOKEN_KEY)) {
    localStorage.setItem(USER_KEY, JSON.stringify(profile));
    sessionStorage.removeItem(USER_KEY);
    return;
  }
  if (sessionStorage.getItem(TOKEN_KEY)) {
    sessionStorage.setItem(USER_KEY, JSON.stringify(profile));
    localStorage.removeItem(USER_KEY);
  }
}

export function clearMerchantSession(): void {
  localStorage.removeItem(TOKEN_KEY);
  localStorage.removeItem(USER_KEY);
  sessionStorage.removeItem(TOKEN_KEY);
  sessionStorage.removeItem(USER_KEY);
  // Keep environment preference across logins
}

export function getMerchantEnvironment(): MerchantEnvironment {
  if (typeof window === "undefined") return "sandbox";
  const raw =
    localStorage.getItem(ENV_KEY) ?? sessionStorage.getItem(ENV_KEY) ?? "sandbox";
  return raw === "production" ? "production" : "sandbox";
}

export function setMerchantEnvironment(env: MerchantEnvironment): void {
  if (typeof window === "undefined") return;
  localStorage.setItem(ENV_KEY, env);
  sessionStorage.setItem(ENV_KEY, env);
  // Notify same-tab listeners (storage event only fires cross-tab)
  window.dispatchEvent(
    new CustomEvent("merchant-environment-change", { detail: env }),
  );
}

export function setMerchantRedirectPath(path: string): void {
  if (typeof window === "undefined") return;
  const safe =
    path.startsWith("/dashboard") || path.startsWith("/payments")
      ? path
      : "/dashboard";
  sessionStorage.setItem(REDIRECT_KEY, safe);
}

export function consumeMerchantRedirectPath(fallback = "/dashboard"): string {
  if (typeof window === "undefined") return fallback;
  const path = sessionStorage.getItem(REDIRECT_KEY);
  sessionStorage.removeItem(REDIRECT_KEY);
  if (!path || path === "/login") return fallback;
  return path;
}

export function setMerchantAuthNotice(notice: MerchantAuthNotice): void {
  if (typeof window === "undefined") return;
  sessionStorage.setItem(AUTH_NOTICE_KEY, notice);
}

export function consumeMerchantAuthNotice(): MerchantAuthNotice | null {
  if (typeof window === "undefined") return null;
  const value = sessionStorage.getItem(AUTH_NOTICE_KEY);
  sessionStorage.removeItem(AUTH_NOTICE_KEY);
  if (value === "required" || value === "expired") return value;
  return null;
}

export async function loginMerchant(
  email: string,
  password: string,
): Promise<MerchantLoginResponse> {
  const res = await fetch(`${API_URL}/api/v1/auth/login`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email, password }),
  });
  const body = (await res.json().catch(() => null)) as
    | MerchantLoginResponse
    | ApiErrorBody
    | null;
  if (!res.ok) {
    throw new Error(
      (body as ApiErrorBody | null)?.message ??
        "Gagal masuk. Periksa email dan password.",
    );
  }
  return body as MerchantLoginResponse;
}

export type RegisteredMerchant = {
  id: string;
  name: string;
  email: string;
  phone: string;
  business_name: string;
  webhook_url?: string | null;
  is_active: boolean;
  created_at: string;
  updated_at: string;
};

export async function registerMerchant(payload: {
  name: string;
  business_name: string;
  email: string;
  phone?: string;
  password: string;
}): Promise<RegisteredMerchant> {
  const res = await fetch(`${API_URL}/api/v1/auth/register`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
  const body = (await res.json().catch(() => null)) as
    | RegisteredMerchant
    | ApiErrorBody
    | null;
  if (res.status === 409) {
    throw new Error("Email ini sudah terdaftar. Silakan masuk atau gunakan email lain.");
  }
  if (res.status === 429) {
    throw new Error("Terlalu banyak percobaan. Coba lagi beberapa saat lagi.");
  }
  if (!res.ok) {
    throw new Error("Gagal mendaftar. Periksa kembali data Anda.");
  }
  return body as RegisteredMerchant;
}

function authHeaders(): HeadersInit {
  const token = getMerchantToken();
  if (!token) throw new Error("Session expired. Please sign in again.");
  return {
    "Content-Type": "application/json",
    Authorization: `Bearer ${token}`,
  };
}

export async function fetchMerchantMe(): Promise<MerchantProfile> {
  const res = await fetch(`${API_URL}/api/v1/auth/me`, {
    method: "GET",
    headers: authHeaders(),
  });
  const body = (await res.json().catch(() => null)) as
    | MerchantProfile
    | ApiErrorBody
    | null;
  if (!res.ok) {
    throw new Error(
      (body as ApiErrorBody | null)?.message ?? "Failed to load profile.",
    );
  }
  return body as MerchantProfile;
}

export async function updateMerchantProfile(
  name: string,
  email: string,
): Promise<MerchantProfile> {
  const res = await fetch(`${API_URL}/api/v1/auth/profile`, {
    method: "PUT",
    headers: authHeaders(),
    body: JSON.stringify({ name, email }),
  });
  const body = (await res.json().catch(() => null)) as
    | MerchantProfile
    | ApiErrorBody
    | null;
  if (!res.ok) {
    throw new Error(
      (body as ApiErrorBody | null)?.message ?? "Failed to update profile.",
    );
  }
  return body as MerchantProfile;
}

export async function changeMerchantPassword(
  currentPassword: string,
  newPassword: string,
): Promise<void> {
  const res = await fetch(`${API_URL}/api/v1/auth/change-password`, {
    method: "POST",
    headers: authHeaders(),
    body: JSON.stringify({
      current_password: currentPassword,
      new_password: newPassword,
    }),
  });
  if (!res.ok) {
    const body = (await res.json().catch(() => null)) as ApiErrorBody | null;
    throw new Error(body?.message ?? "Failed to change password.");
  }
}

export { authHeaders as merchantAuthHeaders, API_URL as MERCHANT_API_URL };
