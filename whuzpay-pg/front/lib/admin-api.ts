import { getAdminToken } from "./admin-auth";
import { API_URL } from "./config";

export type DashboardSummary = {
  total_payments: number;
  pending_payments: number;
  paid_payments: number;
  expired_payments: number;
  failed_payments: number;
  total_merchants: number;
  paid_amount: number;
  webhook_events: number;
};

export type AdminPayment = {
  id: string;
  reference: string;
  merchant_id: string;
  merchant_name: string;
  amount: number;
  currency: string;
  payment_method: string;
  provider_name: string;
  provider_reference?: string | null;
  status: string;
  description: string;
  customer_name?: string | null;
  customer_email?: string | null;
  expires_at: string;
  paid_at?: string | null;
  created_at: string;
  updated_at: string;
  qris_data?: string | null;
  callback_url?: string | null;
};

export type AdminMerchant = {
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

export type AdminLog = {
  id: string;
  payment_id?: string | null;
  provider_name: string;
  provider_reference: string;
  event_type: string;
  status: string;
  is_processed: boolean;
  processing_error?: string | null;
  processed_at?: string | null;
  created_at: string;
};

export type AdminLogDetail = AdminLog & {
  raw_payload?: Record<string, unknown>;
};

export type AdminProvider = {
  name: string;
  payment_methods: string[];
  is_registered: boolean;
  health: {
    provider_name: string;
    status: string;
    reason?: string;
    updated_at: string;
  };
};

export type Paginated<T> = {
  items: T[];
  total: number;
  limit: number;
  offset: number;
};

type ApiErrorBody = {
  error?: string;
  message?: string;
};

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

async function adminFetch<T>(path: string): Promise<T> {
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

function toQuery(params: Record<string, string | number | undefined | null>) {
  const q = new URLSearchParams();
  Object.entries(params).forEach(([key, value]) => {
    if (value === undefined || value === null || value === "") return;
    q.set(key, String(value));
  });
  const s = q.toString();
  return s ? `?${s}` : "";
}

export function fetchDashboardSummary() {
  return adminFetch<DashboardSummary>("/api/v1/admin/dashboard/summary");
}

export type DashboardDailyPoint = {
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

export type DashboardStatusPoint = {
  status: string;
  count: number;
};

export type DashboardCharts = {
  days: number;
  daily: DashboardDailyPoint[];
  status_breakdown: DashboardStatusPoint[];
};

export function fetchDashboardCharts(days = 14) {
  return adminFetch<DashboardCharts>(
    `/api/v1/admin/dashboard/charts${toQuery({ days })}`,
  );
}

export function fetchAdminPayments(params: {
  status?: string;
  search?: string;
  merchant_id?: string;
  limit?: number;
  offset?: number;
}) {
  return adminFetch<Paginated<AdminPayment>>(
    `/api/v1/admin/payments${toQuery(params)}`,
  );
}

export function fetchAdminPayment(id: string) {
  return adminFetch<AdminPayment>(`/api/v1/admin/payments/${id}`);
}

export type CreatePaymentPayload = {
  merchant_id: string;
  amount: number;
  currency?: string;
  payment_method?: string;
  description: string;
  customer_name?: string;
  customer_email?: string;
  callback_url?: string;
  expires_in_minutes?: number;
};

export async function createAdminPayment(
  payload: CreatePaymentPayload,
): Promise<AdminPayment> {
  const res = await fetch(`${API_URL}/api/v1/admin/payments`, {
    method: "POST",
    headers: authHeaders(),
    body: JSON.stringify({
      ...payload,
      currency: payload.currency ?? "IDR",
      payment_method: payload.payment_method ?? "qris",
    }),
  });

  const body = (await res.json().catch(() => null)) as
    | AdminPayment
    | ApiErrorBody
    | null;

  if (!res.ok) {
    throw new Error(
      (body as ApiErrorBody | null)?.message ?? "Failed to create payment.",
    );
  }

  // Map payment response fields if shape differs slightly
  const payment = body as AdminPayment & { merchant_id?: string };
  return payment;
}

export async function exportAdminPayments(params?: {
  status?: string;
  search?: string;
  merchant_id?: string;
}): Promise<void> {
  const res = await fetch(
    `${API_URL}/api/v1/admin/payments/export${toQuery(params ?? {})}`,
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
  const filename = match?.[1] ?? `payments-${Date.now()}.csv`;

  const url = window.URL.createObjectURL(blob);
  const anchor = document.createElement("a");
  anchor.href = url;
  anchor.download = filename;
  document.body.appendChild(anchor);
  anchor.click();
  anchor.remove();
  window.URL.revokeObjectURL(url);
}

export function fetchAdminMerchants(params?: {
  search?: string;
  status?: string;
  limit?: number;
  offset?: number;
}) {
  return adminFetch<Paginated<AdminMerchant>>(
    `/api/v1/admin/merchants${toQuery(params ?? {})}`,
  );
}

export type CreateMerchantPayload = {
  name: string;
  email: string;
  phone?: string;
  business_name: string;
  webhook_url?: string;
};

export async function createAdminMerchant(
  payload: CreateMerchantPayload,
): Promise<AdminMerchant> {
  const res = await fetch(`${API_URL}/api/v1/admin/merchants`, {
    method: "POST",
    headers: authHeaders(),
    body: JSON.stringify(payload),
  });

  const body = (await res.json().catch(() => null)) as
    | AdminMerchant
    | ApiErrorBody
    | null;

  if (!res.ok) {
    throw new Error(
      (body as ApiErrorBody | null)?.message ?? "Failed to create merchant.",
    );
  }

  return body as AdminMerchant;
}

export async function exportAdminMerchants(params?: {
  search?: string;
  status?: string;
}): Promise<void> {
  const res = await fetch(
    `${API_URL}/api/v1/admin/merchants/export${toQuery(params ?? {})}`,
    {
      method: "GET",
      headers: authHeaders(),
    },
  );

  if (!res.ok) {
    const body = (await res.json().catch(() => null)) as ApiErrorBody | null;
    throw new Error(body?.message ?? "Failed to export merchants.");
  }

  const blob = await res.blob();
  const disposition = res.headers.get("Content-Disposition") ?? "";
  const match = disposition.match(/filename="?([^"]+)"?/i);
  const filename = match?.[1] ?? `merchants-${Date.now()}.csv`;

  const url = window.URL.createObjectURL(blob);
  const anchor = document.createElement("a");
  anchor.href = url;
  anchor.download = filename;
  document.body.appendChild(anchor);
  anchor.click();
  anchor.remove();
  window.URL.revokeObjectURL(url);
}

export function fetchAdminMerchant(id: string) {
  return adminFetch<AdminMerchant>(`/api/v1/admin/merchants/${id}`);
}

export function fetchAdminMerchantPayments(
  id: string,
  params?: { limit?: number; offset?: number },
) {
  return adminFetch<Paginated<AdminPayment>>(
    `/api/v1/admin/merchants/${id}/payments${toQuery(params ?? {})}`,
  );
}

export function fetchAdminLogs(params?: {
  status?: string;
  provider?: string;
  processed?: string;
  limit?: number;
  offset?: number;
}) {
  return adminFetch<Paginated<AdminLog>>(
    `/api/v1/admin/logs${toQuery(params ?? {})}`,
  );
}

export async function exportAdminLogs(params?: {
  status?: string;
  provider?: string;
  processed?: string;
}): Promise<void> {
  const res = await fetch(
    `${API_URL}/api/v1/admin/logs/export${toQuery(params ?? {})}`,
    {
      method: "GET",
      headers: authHeaders(),
    },
  );

  if (!res.ok) {
    const body = (await res.json().catch(() => null)) as ApiErrorBody | null;
    throw new Error(body?.message ?? "Failed to export logs.");
  }

  const blob = await res.blob();
  const disposition = res.headers.get("Content-Disposition") ?? "";
  const match = disposition.match(/filename="?([^"]+)"?/i);
  const filename = match?.[1] ?? `logs-${Date.now()}.csv`;

  const url = window.URL.createObjectURL(blob);
  const anchor = document.createElement("a");
  anchor.href = url;
  anchor.download = filename;
  document.body.appendChild(anchor);
  anchor.click();
  anchor.remove();
  window.URL.revokeObjectURL(url);
}

export type AdminNotification = {
  id: string;
  kind:
    | "webhook_error"
    | "webhook_pending"
    | "payment_failed"
    | "payment_expired"
    | "info";
  title: string;
  body: string;
  created_at: string;
  href?: string;
  attention: boolean;
};

export type AdminNotificationsResponse = {
  items: AdminNotification[];
  attention_count: number;
  total: number;
};

export function fetchAdminNotifications(limit = 30) {
  return adminFetch<AdminNotificationsResponse>(
    `/api/v1/admin/notifications${toQuery({ limit })}`,
  );
}

export function fetchAdminLog(id: string) {
  return adminFetch<AdminLogDetail>(`/api/v1/admin/logs/${id}`);
}

export function fetchAdminProviders() {
  return adminFetch<{ items: AdminProvider[] }>("/api/v1/admin/providers");
}

export type AdminRoutingItem = {
  id: string;
  merchant_id: string;
  merchant_name: string;
  merchant_email: string;
  provider_name: string;
  payment_method: string;
  priority: number;
  weight: number;
  failover_enabled: boolean;
  is_enabled: boolean;
  created_at: string;
  updated_at: string;
};

export type AdminProviderDetail = {
  provider: AdminProvider;
  merchant_count: number;
  merchant_routes: AdminRoutingItem[];
};

export function fetchAdminProvider(name: string) {
  return adminFetch<AdminProviderDetail>(
    `/api/v1/admin/providers/${encodeURIComponent(name)}`,
  );
}

export type UpdateProviderHealthPayload = {
  status: "healthy" | "degraded" | "unhealthy";
  reason?: string;
};

export async function updateAdminProviderHealth(
  name: string,
  payload: UpdateProviderHealthPayload,
): Promise<AdminProviderDetail> {
  const res = await fetch(
    `${API_URL}/api/v1/admin/providers/${encodeURIComponent(name)}/health`,
    {
      method: "PUT",
      headers: authHeaders(),
      body: JSON.stringify({
        status: payload.status,
        reason: payload.reason ?? "",
      }),
    },
  );
  const body = (await res.json().catch(() => null)) as
    | AdminProviderDetail
    | ApiErrorBody
    | null;
  if (!res.ok) {
    throw new Error(
      (body as ApiErrorBody | null)?.message ??
        "Failed to update provider health.",
    );
  }
  return body as AdminProviderDetail;
}

export type AdminMerchantProviderConfig = {
  id: string;
  merchant_id: string;
  provider_name: string;
  payment_method: string;
  priority: number;
  weight: number;
  failover_enabled: boolean;
  is_enabled: boolean;
  created_at: string;
  updated_at: string;
};

export type UpsertMerchantProviderConfigPayload = {
  provider_name: string;
  payment_method: string;
  priority?: number;
  weight?: number;
  failover_enabled?: boolean;
  is_enabled?: boolean;
};

export function fetchAdminMerchantProviderConfigs(merchantId: string) {
  return adminFetch<{ items: AdminMerchantProviderConfig[] }>(
    `/api/v1/admin/merchants/${merchantId}/provider-configs`,
  );
}

export async function upsertAdminMerchantProviderConfig(
  merchantId: string,
  payload: UpsertMerchantProviderConfigPayload,
): Promise<AdminMerchantProviderConfig[]> {
  const res = await fetch(
    `${API_URL}/api/v1/admin/merchants/${merchantId}/provider-configs`,
    {
      method: "PUT",
      headers: authHeaders(),
      body: JSON.stringify({
        provider_name: payload.provider_name,
        payment_method: payload.payment_method,
        priority: payload.priority ?? 1,
        weight: payload.weight ?? 100,
        failover_enabled: payload.failover_enabled ?? true,
        is_enabled: payload.is_enabled ?? true,
      }),
    },
  );
  const body = (await res.json().catch(() => null)) as
    | { items: AdminMerchantProviderConfig[] }
    | ApiErrorBody
    | null;
  if (!res.ok) {
    throw new Error(
      (body as ApiErrorBody | null)?.message ??
        "Failed to save provider config.",
    );
  }
  return (body as { items: AdminMerchantProviderConfig[] }).items ?? [];
}

export async function deleteAdminMerchantProviderConfig(
  merchantId: string,
  providerName: string,
  paymentMethod: string,
): Promise<void> {
  const res = await fetch(
    `${API_URL}/api/v1/admin/merchants/${merchantId}/provider-configs${toQuery({
      provider_name: providerName,
      payment_method: paymentMethod,
    })}`,
    {
      method: "DELETE",
      headers: authHeaders(),
    },
  );
  if (!res.ok) {
    const body = (await res.json().catch(() => null)) as ApiErrorBody | null;
    throw new Error(body?.message ?? "Failed to delete provider config.");
  }
}

export function fetchAdminRouting(params?: {
  limit?: number;
  offset?: number;
}) {
  return adminFetch<Paginated<AdminRoutingItem>>(
    `/api/v1/admin/routing${toQuery(params ?? {})}`,
  );
}

export type AdminReconciliationItem = {
  payment_id: string;
  reference: string;
  merchant_name: string;
  provider_name: string;
  provider_reference?: string | null;
  status: string;
  amount: number;
  currency: string;
  created_at: string;
  expires_at: string;
  check_status: string;
  note: string;
};

export type AdminReconciliationResponse = {
  items: AdminReconciliationItem[];
  total: number;
  message: string;
};

export function fetchAdminReconciliation(limit = 50) {
  return adminFetch<AdminReconciliationResponse>(
    `/api/v1/admin/reconciliation${toQuery({ limit })}`,
  );
}

export type AdminReconcileResult = {
  payment_id: string;
  reference: string;
  previous_status: string;
  current_status: string;
  provider_status?: string | null;
  action: "updated" | "unchanged" | "expired_local" | "skipped" | "error" | string;
  message: string;
  merchant_notified?: boolean;
};

export type AdminReconcileBatchResponse = {
  items: AdminReconcileResult[];
  summary: {
    total: number;
    updated: number;
    expired: number;
    unchanged: number;
    skipped: number;
    errors: number;
  };
};

export async function checkAdminReconciliationPayment(
  paymentId: string,
): Promise<AdminReconcileResult> {
  const res = await fetch(
    `${API_URL}/api/v1/admin/reconciliation/${paymentId}/check`,
    {
      method: "POST",
      headers: authHeaders(),
    },
  );
  const body = (await res.json().catch(() => null)) as
    | AdminReconcileResult
    | ApiErrorBody
    | null;
  if (!res.ok) {
    throw new Error(
      (body as ApiErrorBody | null)?.message ?? "Failed to check payment.",
    );
  }
  return body as AdminReconcileResult;
}

export async function checkAdminReconciliationBatch(
  limit = 20,
): Promise<AdminReconcileBatchResponse> {
  const res = await fetch(
    `${API_URL}/api/v1/admin/reconciliation/check${toQuery({ limit })}`,
    {
      method: "POST",
      headers: authHeaders(),
    },
  );
  const body = (await res.json().catch(() => null)) as
    | AdminReconcileBatchResponse
    | ApiErrorBody
    | null;
  if (!res.ok) {
    throw new Error(
      (body as ApiErrorBody | null)?.message ?? "Failed to run batch check.",
    );
  }
  return body as AdminReconcileBatchResponse;
}

export function fetchAdminPaymentEvents(paymentId: string) {
  return adminFetch<{ items: AdminLog[] }>(
    `/api/v1/admin/payments/${paymentId}/events`,
  );
}

export type AdminCallback = {
  id: string;
  payment_id: string;
  payment_reference?: string;
  merchant_id: string;
  merchant_name?: string;
  event_type: string;
  target_url: string;
  request_payload?: Record<string, unknown>;
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

export function fetchAdminCallbacks(params?: {
  status?: string;
  merchant_id?: string;
  limit?: number;
  offset?: number;
}) {
  return adminFetch<Paginated<AdminCallback>>(
    `/api/v1/admin/callbacks${toQuery(params ?? {})}`,
  );
}

export function fetchAdminPaymentCallbacks(paymentId: string) {
  return adminFetch<{ items: AdminCallback[] }>(
    `/api/v1/admin/payments/${paymentId}/callbacks`,
  );
}

export async function retryAdminCallback(
  id: string,
): Promise<AdminCallback> {
  const res = await fetch(`${API_URL}/api/v1/admin/callbacks/${id}/retry`, {
    method: "POST",
    headers: authHeaders(),
  });
  const body = (await res.json().catch(() => null)) as
    | AdminCallback
    | ApiErrorBody
    | null;
  if (!res.ok) {
    throw new Error(
      (body as ApiErrorBody | null)?.message ?? "Failed to retry callback.",
    );
  }
  return body as AdminCallback;
}

export type UpdateMerchantPayload = {
  name?: string;
  phone?: string;
  business_name?: string;
  webhook_url?: string | null;
};

export async function updateAdminMerchant(
  id: string,
  payload: UpdateMerchantPayload,
): Promise<AdminMerchant> {
  const res = await fetch(`${API_URL}/api/v1/admin/merchants/${id}`, {
    method: "PUT",
    headers: authHeaders(),
    body: JSON.stringify(payload),
  });
  const body = (await res.json().catch(() => null)) as
    | AdminMerchant
    | ApiErrorBody
    | null;
  if (!res.ok) {
    throw new Error(
      (body as ApiErrorBody | null)?.message ?? "Failed to update merchant.",
    );
  }
  return body as AdminMerchant;
}

export async function setAdminMerchantActive(
  id: string,
  isActive: boolean,
): Promise<AdminMerchant> {
  const res = await fetch(`${API_URL}/api/v1/admin/merchants/${id}/status`, {
    method: "PUT",
    headers: authHeaders(),
    body: JSON.stringify({ is_active: isActive }),
  });
  const body = (await res.json().catch(() => null)) as
    | AdminMerchant
    | ApiErrorBody
    | null;
  if (!res.ok) {
    throw new Error(
      (body as ApiErrorBody | null)?.message ??
        "Failed to update merchant status.",
    );
  }
  return body as AdminMerchant;
}

export type AdminMerchantAPIKey = {
  id: string;
  merchant_id: string;
  name: string;
  key_prefix: string;
  is_active: boolean;
  last_used_at?: string | null;
  revoked_at?: string | null;
  created_at: string;
  updated_at: string;
};

export type UpsertMerchantAPIKeyResponse = {
  key: AdminMerchantAPIKey;
  secret: string;
  hint: string;
  rotated: boolean;
  environment: string;
};

export function fetchAdminMerchantAPIKeys(merchantId: string) {
  return adminFetch<{ items: AdminMerchantAPIKey[] }>(
    `/api/v1/admin/merchants/${merchantId}/api-keys`,
  );
}

export async function upsertAdminMerchantAPIKey(
  merchantId: string,
  payload: { environment: "sandbox" | "production" | string; password: string },
): Promise<UpsertMerchantAPIKeyResponse> {
  const res = await fetch(
    `${API_URL}/api/v1/admin/merchants/${merchantId}/api-keys`,
    {
      method: "PUT",
      headers: authHeaders(),
      body: JSON.stringify({
        environment: payload.environment || "sandbox",
        password: payload.password,
      }),
    },
  );
  const body = (await res.json().catch(() => null)) as
    | UpsertMerchantAPIKeyResponse
    | ApiErrorBody
    | null;
  if (!res.ok) {
    throw new Error(
      (body as ApiErrorBody | null)?.message ?? "Failed to update API key.",
    );
  }
  return body as UpsertMerchantAPIKeyResponse;
}

export async function deleteAdminMerchantAPIKey(
  merchantId: string,
  keyId: string,
  password: string,
): Promise<void> {
  const res = await fetch(
    `${API_URL}/api/v1/admin/merchants/${merchantId}/api-keys/${keyId}`,
    {
      method: "DELETE",
      headers: authHeaders(),
      body: JSON.stringify({ password }),
    },
  );
  if (!res.ok) {
    const body = (await res.json().catch(() => null)) as ApiErrorBody | null;
    throw new Error(body?.message ?? "Failed to delete API key.");
  }
}
