"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { useEffect, useState } from "react";
import toast from "react-hot-toast";
import {
  fetchAdminProvider,
  updateAdminProviderHealth,
  type AdminProviderDetail,
} from "@/lib/admin-api";
import { Button } from "@/components/ui/button";
import {
  Card,
  EmptyState,
  formatDateTime,
  LoadingBlock,
  PageHeader,
  TableShell,
} from "../../../../components/admin/ui";

const HEALTH_OPTIONS = [
  {
    value: "healthy" as const,
    label: "Healthy",
    hint: "Used normally for new payments.",
    className:
      "border-[#c8efd4] bg-[#f0faf3] text-[#2f9e5a] data-[active=true]:ring-2 data-[active=true]:ring-[#2f9e5a]/35",
  },
  {
    value: "degraded" as const,
    label: "Degraded",
    hint: "Still eligible; mark when slow or flaky.",
    className:
      "border-[#ffe0a8] bg-[#fff8eb] text-[#c27a00] data-[active=true]:ring-2 data-[active=true]:ring-[#c27a00]/35",
  },
  {
    value: "unhealthy" as const,
    label: "Unhealthy",
    hint: "Skipped by routing (failover may apply).",
    className:
      "border-[#ffd4c8] bg-[#fff5f2] text-[#e85d3b] data-[active=true]:ring-2 data-[active=true]:ring-[#e85d3b]/35",
  },
];

function healthStyle(status: string) {
  switch (status) {
    case "healthy":
      return "bg-[#e8f8ee] text-[#2f9e5a]";
    case "degraded":
      return "bg-[#fff7e6] text-[#c27a00]";
    case "unhealthy":
      return "bg-[#fff1ed] text-[#e85d3b]";
    default:
      return "bg-[#eef2f6] text-[#6b7c93]";
  }
}

export default function AdminProviderDetailPage() {
  const params = useParams<{ name: string }>();
  const name = params?.name ? decodeURIComponent(params.name) : "";
  const [detail, setDetail] = useState<AdminProviderDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [status, setStatus] = useState<"healthy" | "degraded" | "unhealthy">(
    "healthy",
  );
  const [reason, setReason] = useState("");
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    if (!name) return;
    let cancelled = false;

    async function load() {
      setLoading(true);
      try {
        const data = await fetchAdminProvider(name);
        if (cancelled) return;
        setDetail(data);
        const hs = data.provider?.health?.status;
        if (hs === "healthy" || hs === "degraded" || hs === "unhealthy") {
          setStatus(hs);
        }
        setReason(data.provider?.health?.reason ?? "");
      } catch (err) {
        if (!cancelled) {
          toast.error(
            err instanceof Error ? err.message : "Failed to load provider.",
          );
          setDetail(null);
        }
      } finally {
        if (!cancelled) setLoading(false);
      }
    }

    void load();
    return () => {
      cancelled = true;
    };
  }, [name]);

  async function handleSaveHealth() {
    if (!name) return;
    setSaving(true);
    try {
      const updated = await updateAdminProviderHealth(name, {
        status,
        reason: reason.trim(),
      });
      setDetail(updated);
      const hs = updated.provider?.health?.status;
      if (hs === "healthy" || hs === "degraded" || hs === "unhealthy") {
        setStatus(hs);
      }
      setReason(updated.provider?.health?.reason ?? "");
      toast.success(`Provider marked as ${status}.`);
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Failed to update health.",
      );
    } finally {
      setSaving(false);
    }
  }

  const provider = detail?.provider;
  const current = provider?.health?.status ?? "unknown";
  const dirty =
    status !== current ||
    (reason.trim() || "") !== (provider?.health?.reason?.trim() || "");

  return (
    <div className="w-full space-y-5">
      <PageHeader
        title="Provider detail"
        description="Health, supported methods, and merchant routing that uses this provider."
        actions={
          <Link
            href="/admin/providers"
            className="text-[13px] font-semibold text-[#6b7c93] hover:text-[#06163a]"
          >
            ← Back to providers
          </Link>
        }
      />

      {loading ? (
        <Card>
          <LoadingBlock />
        </Card>
      ) : !provider ? (
        <Card className="p-6 text-[13px] text-[#8a97a8]">
          Provider not found.
        </Card>
      ) : (
        <>
          <Card className="p-5 sm:p-6">
            <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
              <div>
                <h2 className="text-[18px] font-semibold capitalize text-[#1f2a37]">
                  {provider.name}
                </h2>
                <p className="mt-1 text-[13px] text-[#6b7c93]">
                  {provider.is_registered
                    ? "Registered adapter"
                    : "Not registered"}
                </p>
              </div>
              <span
                className={`inline-flex w-fit rounded-full px-2.5 py-0.5 text-[11px] font-semibold capitalize ${healthStyle(
                  provider.health?.status ?? "unknown",
                )}`}
              >
                {provider.health?.status ?? "unknown"}
              </span>
            </div>

            <div className="mt-5 grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-4">
              <div className="rounded-lg bg-[#f8fafc] px-4 py-3">
                <p className="text-[11px] font-semibold uppercase tracking-wide text-[#8a97a8]">
                  Payment methods
                </p>
                <p className="mt-1 text-[13.5px] font-medium capitalize text-[#1f2a37]">
                  {(provider.payment_methods ?? []).length > 0
                    ? provider.payment_methods.join(", ")
                    : "—"}
                </p>
              </div>
              <div className="rounded-lg bg-[#f8fafc] px-4 py-3">
                <p className="text-[11px] font-semibold uppercase tracking-wide text-[#8a97a8]">
                  Merchant routes
                </p>
                <p className="mt-1 text-[13.5px] font-medium text-[#1f2a37]">
                  {detail?.merchant_count ?? 0}
                </p>
              </div>
              <div className="rounded-lg bg-[#f8fafc] px-4 py-3">
                <p className="text-[11px] font-semibold uppercase tracking-wide text-[#8a97a8]">
                  Health reason
                </p>
                <p className="mt-1 text-[13.5px] font-medium text-[#1f2a37]">
                  {provider.health?.reason || "—"}
                </p>
              </div>
              <div className="rounded-lg bg-[#f8fafc] px-4 py-3">
                <p className="text-[11px] font-semibold uppercase tracking-wide text-[#8a97a8]">
                  Health updated
                </p>
                <p className="mt-1 text-[13.5px] font-medium text-[#1f2a37]">
                  {formatDateTime(provider.health?.updated_at)}
                </p>
              </div>
            </div>
          </Card>

          <Card className="p-5 sm:p-6">
            <div className="flex flex-col gap-1 sm:flex-row sm:items-start sm:justify-between">
              <div>
                <h3 className="text-[15px] font-semibold text-[#1f2a37]">
                  Health operations
                </h3>
                <p className="mt-1 text-[12.5px] leading-relaxed text-[#8a97a8]">
                  Manually set provider status for routing.{" "}
                  <span className="font-medium text-[#6b7c93]">
                    Unhealthy
                  </span>{" "}
                  providers are skipped when creating payments (failover can
                  take over). Health is in-memory for this process and resets
                  on API restart.
                </p>
              </div>
            </div>

            <div className="mt-5 grid grid-cols-1 gap-3 sm:grid-cols-3">
              {HEALTH_OPTIONS.map((opt) => (
                <button
                  key={opt.value}
                  type="button"
                  data-active={status === opt.value}
                  onClick={() => setStatus(opt.value)}
                  className={`rounded-xl border px-4 py-3 text-left transition ${opt.className}`}
                >
                  <p className="text-[13.5px] font-semibold">{opt.label}</p>
                  <p className="mt-1 text-[12px] opacity-80">{opt.hint}</p>
                </button>
              ))}
            </div>

            <label className="mt-5 flex flex-col gap-1.5">
              <span className="text-[11px] font-semibold uppercase tracking-wide text-[#8a97a8]">
                Reason (optional)
              </span>
              <textarea
                value={reason}
                onChange={(e) => setReason(e.target.value)}
                rows={3}
                placeholder="e.g. Cashi API timeouts since 14:00 — fail over to next provider"
                className="w-full resize-y rounded-lg border border-[#e8eef4] bg-white px-3 py-2.5 text-[13px] text-[#1f2a37] outline-none placeholder:text-[#a8b3c2] focus:border-[#3b9eff] focus:ring-2 focus:ring-[#3b9eff]/15"
              />
            </label>

            <div className="mt-4 flex flex-wrap items-center gap-3">
              <Button
                type="button"
                disabled={saving || !dirty}
                onClick={() => void handleSaveHealth()}
                className="h-10 rounded-full bg-[#06163a] px-5 text-[13px] font-semibold text-white shadow-none hover:bg-[#0b2048] disabled:opacity-50"
              >
                {saving ? "Saving..." : "Update health"}
              </Button>
              {!dirty && (
                <span className="text-[12.5px] text-[#8a97a8]">
                  No changes to save
                </span>
              )}
              {status === "unhealthy" && dirty && (
                <span className="text-[12.5px] font-medium text-[#e85d3b]">
                  New payments will skip this provider until restored.
                </span>
              )}
            </div>
          </Card>

          <Card>
            <div className="border-b border-[#eef2f6] px-5 py-4">
              <h3 className="text-[15px] font-semibold text-[#1f2a37]">
                Merchant routing
              </h3>
              <p className="mt-0.5 text-[12.5px] text-[#8a97a8]">
                Merchants configured to use this provider (priority / weight /
                failover).
              </p>
            </div>
            {(detail?.merchant_routes ?? []).length === 0 ? (
              <EmptyState message="No merchant routes use this provider yet." />
            ) : (
              <TableShell>
                <table className="w-full min-w-[880px] text-left text-[12.5px]">
                  <thead>
                    <tr className="border-b border-[#eef2f6] text-[11px] font-semibold uppercase tracking-wide text-[#8a97a8]">
                      <th className="px-5 py-3">Merchant</th>
                      <th className="px-5 py-3">Method</th>
                      <th className="px-5 py-3">Priority</th>
                      <th className="px-5 py-3">Weight</th>
                      <th className="px-5 py-3">Failover</th>
                      <th className="px-5 py-3">Enabled</th>
                      <th className="px-5 py-3">Updated</th>
                    </tr>
                  </thead>
                  <tbody>
                    {detail!.merchant_routes.map((row) => (
                      <tr
                        key={row.id}
                        className="border-b border-[#f3f6f9] text-[#3d4b5c] last:border-0"
                      >
                        <td className="px-5 py-3.5">
                          <Link
                            href={`/admin/merchants/${row.merchant_id}`}
                            className="font-medium text-[#1f2a37] hover:text-[#ff5e16]"
                          >
                            {row.merchant_name || row.merchant_email || "—"}
                          </Link>
                          <p className="mt-0.5 text-[11px] text-[#8a97a8]">
                            {row.merchant_email}
                          </p>
                        </td>
                        <td className="px-5 py-3.5 capitalize">
                          {row.payment_method}
                        </td>
                        <td className="px-5 py-3.5">{row.priority}</td>
                        <td className="px-5 py-3.5">{row.weight}</td>
                        <td className="px-5 py-3.5">
                          {row.failover_enabled ? "Yes" : "No"}
                        </td>
                        <td className="px-5 py-3.5">
                          <span
                            className={`inline-flex rounded-full px-2.5 py-0.5 text-[11px] font-semibold ${
                              row.is_enabled
                                ? "bg-[#e8f8ee] text-[#2f9e5a]"
                                : "bg-[#fff1ed] text-[#e85d3b]"
                            }`}
                          >
                            {row.is_enabled ? "Enabled" : "Disabled"}
                          </span>
                        </td>
                        <td className="px-5 py-3.5 text-[#6b7c93]">
                          {formatDateTime(row.updated_at)}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </TableShell>
            )}
          </Card>
        </>
      )}
    </div>
  );
}
