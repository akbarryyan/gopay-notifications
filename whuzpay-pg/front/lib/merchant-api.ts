import {
  getMerchantEnvironment,
  getMerchantToken,
  MERCHANT_API_URL as API_URL,
} from "./merchant-auth";

type ApiErrorBody = { error?: string; message?: string };

function authHeaders(): HeadersInit {
  const token = getMerchantToken();
  if (!token) throw new Error("Session expired. Please sign in again.");
  return {
    "Content-Type": "application/json",
    Authorization: `Bearer ${token}`,
  };
}

function toQuery(params: Record<string, string | number | undefined | null>) {
  const q = new URLSearchParams();
  Object.entries(params).forEach(([k, v]) => {
    if (v === undefined || v === null || v === "") return;
    q.set(k, String(v));
  });
  const s = q.toString();
  return s ? `?${s}` : "";
}

async function merchantFetch<T>(path: string): Promise<T> {
  const res = await fetch(`${API_URL}${path}`, {
    method: "GET",
    headers: authHeaders(),
    cache: "no-store",
  });
  const body = (await res.json().catch(() => null)) as T | ApiErrorBody | null;
  if (!res.ok) {
    throw new Error(
      (body as ApiErrorBody | null)?.message ?? "Request failed.",
    );
  }
  return body as T;
}

export type MerchantDashboardSummary = {
  environment?: string;
  total_payments: number;
  paid_payments: number;
  pending_payments: number;
  failed_payments: number;
  expired_payments: number;
  paid_amount: number;
};

export type MerchantPayment = {
  id: string;
  reference: string;
  merchant_id: string;
  merchant_name?: string;
  amount: number;
  currency: string;
  payment_method: string;
  provider_name: string;
  provider_reference?: string | null;
  status: string;
  description: string;
  customer_name?: string | null;
  customer_email?: string | null;
  environment?: string;
  expires_at: string;
  paid_at?: string | null;
  created_at: string;
  updated_at: string;
  qris_data?: string | null;
  checkout_url?: string;
  callback_url?: string | null;
};

export type PaginatedPayments = {
  items: MerchantPayment[];
  total: number;
  limit: number;
  offset: number;
};

export function fetchMerchantDashboardSummary(environment?: string) {
  const env = environment ?? getMerchantEnvironment();
  return merchantFetch<MerchantDashboardSummary>(
    `/api/v1/merchant/dashboard/summary${toQuery({ environment: env })}`,
  );
}

export type MerchantChartDaily = {
  date: string;
  label: string;
  total: number;
  paid: number;
  pending: number;
  failed: number;
  expired: number;
  cancelled: number;
  paid_amount: number;
};

export type MerchantCharts = {
  days: number;
  daily: MerchantChartDaily[];
  status_breakdown: { status: string; count: number }[];
};

export function fetchMerchantDashboardCharts(days = 14, environment?: string) {
  const env = environment ?? getMerchantEnvironment();
  return merchantFetch<MerchantCharts>(
    `/api/v1/merchant/dashboard/charts${toQuery({ days, environment: env })}`,
  );
}

export type MerchantNotification = {
  id: string;
  kind: string;
  title: string;
  body: string;
  created_at: string;
  href?: string;
  attention: boolean;
};

export function fetchMerchantNotifications(environment?: string, limit = 30) {
  const env = environment ?? getMerchantEnvironment();
  return merchantFetch<{
    items: MerchantNotification[];
    attention_count: number;
    total: number;
  }>(`/api/v1/merchant/notifications${toQuery({ environment: env, limit })}`);
}

export type PublicPayment = {
  id: string;
  reference: string;
  amount: number;
  currency: string;
  status: string;
  description: string;
  payment_method: string;
  environment?: string;
  qris_data?: string | null;
  expires_at: string;
  paid_at?: string | null;
  created_at: string;
};

export async function fetchPublicPaymentByReference(
  reference: string,
): Promise<PublicPayment> {
  const res = await fetch(
    `${API_URL}/api/v1/public/payments/by-reference/${encodeURIComponent(reference)}`,
    { cache: "no-store" },
  );
  const body = await res.json().catch(() => null);
  if (!res.ok) {
    throw new Error(body?.message ?? "Payment not found.");
  }
  return body as PublicPayment;
}

export function fetchMerchantPayments(params?: {
  status?: string;
  search?: string;
  date_from?: string;
  date_to?: string;
  environment?: string;
  limit?: number;
  offset?: number;
}) {
  const environment = params?.environment ?? getMerchantEnvironment();
  return merchantFetch<PaginatedPayments>(
    `/api/v1/merchant/payments${toQuery({ ...params, environment })}`,
  );
}

export async function exportMerchantPayments(params?: {
  status?: string;
  search?: string;
  date_from?: string;
  date_to?: string;
  environment?: string;
}): Promise<void> {
  const environment = params?.environment ?? getMerchantEnvironment();
  const res = await fetch(
    `${API_URL}/api/v1/merchant/payments/export${toQuery({
      ...params,
      environment,
    })}`,
    {
      method: "GET",
      headers: authHeaders(),
    },
  );
  if (!res.ok) {
    const body = (await res.json().catch(() => null)) as ApiErrorBody | null;
    throw new Error(body?.message ?? "Failed to export payments.");
  }
  const blob = await res.blob();
  const disposition = res.headers.get("Content-Disposition") ?? "";
  const match = disposition.match(/filename="?([^"]+)"?/i);
  const filename = match?.[1] ?? `merchant-payments-${Date.now()}.csv`;
  const url = window.URL.createObjectURL(blob);
  const anchor = document.createElement("a");
  anchor.href = url;
  anchor.download = filename;
  document.body.appendChild(anchor);
  anchor.click();
  anchor.remove();
  window.URL.revokeObjectURL(url);
}

export function fetchMerchantPayment(id: string) {
  return merchantFetch<MerchantPayment>(`/api/v1/merchant/payments/${id}`);
}

export async function createMerchantPayment(payload: {
  amount: number;
  currency?: string;
  payment_method?: string;
  description: string;
  customer_name?: string;
  customer_email?: string;
  callback_url?: string;
  expires_in_minutes?: number;
  environment?: string;
}): Promise<MerchantPayment> {
  const environment = payload.environment ?? getMerchantEnvironment();
  const res = await fetch(`${API_URL}/api/v1/merchant/payments`, {
    method: "POST",
    headers: authHeaders(),
    body: JSON.stringify({
      ...payload,
      currency: payload.currency ?? "IDR",
      payment_method: payload.payment_method ?? "qris",
      environment,
    }),
  });
  const body = (await res.json().catch(() => null)) as
    | MerchantPayment
    | ApiErrorBody
    | null;
  if (!res.ok) {
    throw new Error(
      (body as ApiErrorBody | null)?.message ?? "Failed to create payment.",
    );
  }
  return body as MerchantPayment;
}

export type MerchantAPIKey = {
  id: string;
  merchant_id: string;
  name: string;
  key_prefix: string;
  is_active: boolean;
  last_used_at?: string | null;
  created_at: string;
  updated_at: string;
};

export function fetchMerchantAPIKeys() {
  return merchantFetch<{ items: MerchantAPIKey[] }>("/api/v1/merchant/api-keys");
}

export async function upsertMerchantAPIKey(payload: {
  environment: string;
  password: string;
}) {
  const res = await fetch(`${API_URL}/api/v1/merchant/api-keys`, {
    method: "PUT",
    headers: authHeaders(),
    body: JSON.stringify(payload),
  });
  const body = await res.json().catch(() => null);
  if (!res.ok) {
    throw new Error(body?.message ?? "Failed to update API key.");
  }
  return body as {
    key: MerchantAPIKey;
    secret: string;
    hint: string;
    rotated: boolean;
    environment: string;
  };
}

export async function deleteMerchantAPIKey(keyId: string, password: string) {
  const res = await fetch(`${API_URL}/api/v1/merchant/api-keys/${keyId}`, {
    method: "DELETE",
    headers: authHeaders(),
    body: JSON.stringify({ password }),
  });
  if (!res.ok) {
    const body = await res.json().catch(() => null);
    throw new Error(body?.message ?? "Failed to delete API key.");
  }
}

export type MerchantCallback = {
  id: string;
  payment_id: string;
  payment_reference?: string;
  merchant_id: string;
  event_type: string;
  target_url: string;
  attempt_number: number;
  status: string;
  http_status?: number | null;
  response_body?: string | null;
  error_message?: string | null;
  delivered_at?: string | null;
  next_retry_at?: string | null;
  created_at: string;
  updated_at: string;
};

export type PaginatedCallbacks = {
  items: MerchantCallback[];
  total: number;
  limit: number;
  offset: number;
};

export function fetchMerchantCallbacks(params?: {
  status?: string;
  limit?: number;
  offset?: number;
}) {
  return merchantFetch<PaginatedCallbacks>(
    `/api/v1/merchant/callbacks${toQuery(params ?? {})}`,
  );
}

export type MerchantBusiness = {
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

export function fetchMerchantBusiness() {
  return merchantFetch<MerchantBusiness>("/api/v1/merchant/business");
}

export async function updateMerchantBusiness(payload: {
  name?: string;
  phone?: string;
  business_name?: string;
  webhook_url?: string | null;
}): Promise<MerchantBusiness> {
  const res = await fetch(`${API_URL}/api/v1/merchant/business`, {
    method: "PUT",
    headers: authHeaders(),
    body: JSON.stringify(payload),
  });
  const body = (await res.json().catch(() => null)) as
    | MerchantBusiness
    | ApiErrorBody
    | null;
  if (!res.ok) {
    throw new Error(
      (body as ApiErrorBody | null)?.message ?? "Failed to update business.",
    );
  }
  return body as MerchantBusiness;
}

export function fetchMerchantWebhookSecret() {
  return merchantFetch<{ webhook_secret: string }>(
    "/api/v1/merchant/webhook-secret",
  );
}

export async function regenerateMerchantWebhookSecret(): Promise<{
  webhook_secret: string;
}> {
  const res = await fetch(`${API_URL}/api/v1/merchant/webhook-secret/regenerate`, {
    method: "POST",
    headers: authHeaders(),
  });
  const body = (await res.json().catch(() => null)) as
    | { webhook_secret: string }
    | ApiErrorBody
    | null;
  if (!res.ok) {
    throw new Error(
      (body as ApiErrorBody | null)?.message ??
        "Failed to regenerate webhook secret.",
    );
  }
  return body as { webhook_secret: string };
}

// ---- Payment Links --------------------------------------------------------
// A link is reusable: every checkout through it spawns a fresh one-time
// payment (see the backend's PaymentLinkService.InitiateCheckout). Creating
// a link never touches a provider — only the public "pay" call does.

export type PaymentLinkAmountType = "fixed" | "open";

export type MerchantPaymentLink = {
  id: string;
  merchant_id: string;
  slug: string;
  public_url: string;
  title: string;
  description?: string;
  amount_type: PaymentLinkAmountType;
  amount?: number | null;
  currency: string;
  min_amount?: number | null;
  max_amount?: number | null;
  is_active: boolean;
  expires_at?: string | null;
  environment: string;
  created_at: string;
  updated_at: string;
};

export type PaginatedPaymentLinks = {
  items: MerchantPaymentLink[];
  total: number;
  limit: number;
  offset: number;
};

export function fetchMerchantPaymentLinks(params?: {
  environment?: string;
  is_active?: boolean;
  limit?: number;
  offset?: number;
}) {
  const environment = params?.environment ?? getMerchantEnvironment();
  return merchantFetch<PaginatedPaymentLinks>(
    `/api/v1/merchant/payment-links${toQuery({
      environment,
      is_active:
        params?.is_active === undefined ? undefined : String(params.is_active),
      limit: params?.limit,
      offset: params?.offset,
    })}`,
  );
}

export function fetchMerchantPaymentLink(id: string) {
  return merchantFetch<MerchantPaymentLink>(`/api/v1/merchant/payment-links/${id}`);
}

export async function createMerchantPaymentLink(payload: {
  title: string;
  description?: string;
  amount_type: PaymentLinkAmountType;
  amount?: number;
  min_amount?: number;
  max_amount?: number;
  expires_at?: string | null;
  environment?: string;
}): Promise<MerchantPaymentLink> {
  const environment = payload.environment ?? getMerchantEnvironment();
  const res = await fetch(`${API_URL}/api/v1/merchant/payment-links`, {
    method: "POST",
    headers: authHeaders(),
    body: JSON.stringify({ ...payload, environment }),
  });
  const body = (await res.json().catch(() => null)) as
    | MerchantPaymentLink
    | ApiErrorBody
    | null;
  if (!res.ok) {
    throw new Error(
      (body as ApiErrorBody | null)?.message ?? "Failed to create payment link.",
    );
  }
  return body as MerchantPaymentLink;
}

export async function updateMerchantPaymentLink(
  id: string,
  payload: {
    title?: string;
    description?: string;
    is_active?: boolean;
    expires_at?: string | null;
    min_amount?: number | null;
    max_amount?: number | null;
  },
): Promise<MerchantPaymentLink> {
  const res = await fetch(`${API_URL}/api/v1/merchant/payment-links/${id}`, {
    method: "PUT",
    headers: authHeaders(),
    body: JSON.stringify(payload),
  });
  const body = (await res.json().catch(() => null)) as
    | MerchantPaymentLink
    | ApiErrorBody
    | null;
  if (!res.ok) {
    throw new Error(
      (body as ApiErrorBody | null)?.message ?? "Failed to update payment link.",
    );
  }
  return body as MerchantPaymentLink;
}

export async function setMerchantPaymentLinkActive(
  id: string,
  isActive: boolean,
): Promise<void> {
  const res = await fetch(`${API_URL}/api/v1/merchant/payment-links/${id}/status`, {
    method: "PUT",
    headers: authHeaders(),
    body: JSON.stringify({ is_active: isActive }),
  });
  if (!res.ok) {
    const body = (await res.json().catch(() => null)) as ApiErrorBody | null;
    throw new Error(body?.message ?? "Failed to update payment link status.");
  }
}

export function fetchMerchantPaymentLinkPayments(
  id: string,
  params?: { limit?: number; offset?: number },
) {
  return merchantFetch<PaginatedPayments>(
    `/api/v1/merchant/payment-links/${id}/payments${toQuery({
      limit: params?.limit,
      offset: params?.offset,
    })}`,
  );
}

export type PublicPaymentLink = {
  slug: string;
  title: string;
  description?: string;
  amount_type: PaymentLinkAmountType;
  amount?: number | null;
  currency: string;
  min_amount?: number;
  max_amount?: number;
  is_available: boolean;
  reason?: string;
  environment: string;
};

export async function fetchPublicPaymentLink(
  slug: string,
): Promise<PublicPaymentLink> {
  const res = await fetch(
    `${API_URL}/api/v1/public/payment-links/${encodeURIComponent(slug)}`,
    { cache: "no-store" },
  );
  const body = await res.json().catch(() => null);
  if (!res.ok) {
    throw new Error(body?.message ?? "Payment link not found.");
  }
  return body as PublicPaymentLink;
}

export async function initiatePaymentLinkCheckout(
  slug: string,
  payload: {
    amount?: number;
    customer_name?: string;
    customer_email?: string;
  },
): Promise<MerchantPayment> {
  const res = await fetch(
    `${API_URL}/api/v1/public/payment-links/${encodeURIComponent(slug)}/pay`,
    {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload),
    },
  );
  const body = (await res.json().catch(() => null)) as
    | MerchantPayment
    | ApiErrorBody
    | null;
  if (!res.ok) {
    throw new Error(
      (body as ApiErrorBody | null)?.message ?? "Failed to start checkout.",
    );
  }
  return body as MerchantPayment;
}
