"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import toast from "react-hot-toast";
import { RefreshCwIcon, SearchCheckIcon } from "lucide-react";
import {
  checkAdminReconciliationBatch,
  checkAdminReconciliationPayment,
  fetchAdminReconciliation,
  type AdminReconciliationItem,
  type AdminReconcileResult,
} from "@/lib/admin-api";
import { Button } from "@/components/ui/button";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import {
  Card,
  EmptyState,
  formatDateTime,
  formatIDR,
  LoadingBlock,
  PageHeader,
  StatusBadge,
  TableShell,
} from "../../../components/admin/ui";

function checkStatusStyle(status: string) {
  switch (status) {
    case "possibly_expired":
      return "bg-[#fff1ed] text-[#e85d3b]";
    case "missing_provider_ref":
      return "bg-[#fff7e6] text-[#c27a00]";
    case "pending_review":
    default:
      return "bg-[#eef2f6] text-[#6b7c93]";
  }
}

function checkStatusLabel(status: string) {
  switch (status) {
    case "possibly_expired":
      return "Possibly expired";
    case "missing_provider_ref":
      return "Missing provider ref";
    case "pending_review":
      return "Pending review";
    default:
      return status;
  }
}

function actionStyle(action: string) {
  switch (action) {
    case "updated":
      return "bg-[#e8f8ee] text-[#2f9e5a]";
    case "expired_local":
      return "bg-[#fff1ed] text-[#e85d3b]";
    case "error":
      return "bg-[#fff1ed] text-[#e85d3b]";
    case "skipped":
      return "bg-[#eef2f6] text-[#6b7c93]";
    default:
      return "bg-[#eef5ff] text-[#3b6fb6]";
  }
}

export default function AdminReconciliationPage() {
  const [items, setItems] = useState<AdminReconciliationItem[]>([]);
  const [total, setTotal] = useState(0);
  const [message, setMessage] = useState("");
  const [loading, setLoading] = useState(true);
  const [refreshKey, setRefreshKey] = useState(0);
  const [checkingId, setCheckingId] = useState<string | null>(null);
  const [batchRunning, setBatchRunning] = useState(false);
  const [lastResults, setLastResults] = useState<AdminReconcileResult[]>([]);
  const [lastSummary, setLastSummary] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;

    async function load() {
      setLoading(true);
      try {
        const data = await fetchAdminReconciliation(50);
        if (cancelled) return;
        setItems(data.items ?? []);
        setTotal(data.total ?? 0);
        setMessage(data.message ?? "");
      } catch (err) {
        if (cancelled) return;
        toast.error(
          err instanceof Error
            ? err.message
            : "Failed to load reconciliation candidates.",
        );
      } finally {
        if (!cancelled) setLoading(false);
      }
    }

    void load();
    return () => {
      cancelled = true;
    };
  }, [refreshKey]);

  async function handleCheckOne(paymentId: string) {
    setCheckingId(paymentId);
    try {
      const result = await checkAdminReconciliationPayment(paymentId);
      setLastResults([result]);
      setLastSummary(
        `${result.action}: ${result.reference} → ${result.current_status}`,
      );
      if (result.action === "updated" || result.action === "expired_local") {
        toast.success(result.message);
        setRefreshKey((k) => k + 1);
      } else if (result.action === "error") {
        toast.error(result.message);
      } else {
        toast.success(result.message);
      }
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Failed to check payment.",
      );
    } finally {
      setCheckingId(null);
    }
  }

  async function handleCheckAll() {
    setBatchRunning(true);
    try {
      const data = await checkAdminReconciliationBatch(20);
      setLastResults(data.items ?? []);
      const s = data.summary;
      setLastSummary(
        `Batch: ${s.total} checked · ${s.updated} updated · ${s.expired} expired · ${s.unchanged} unchanged · ${s.skipped} skipped · ${s.errors} errors`,
      );
      toast.success(
        `Checked ${s.total}: ${s.updated} updated, ${s.expired} expired.`,
      );
      setRefreshKey((k) => k + 1);
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Failed to run batch check.",
      );
    } finally {
      setBatchRunning(false);
    }
  }

  return (
    <div className="w-full space-y-5">
      <PageHeader
        title="Reconciliation"
        description="Check pending payments against the provider and sync local status when needed."
        actions={
          <div className="flex flex-wrap items-center gap-2">
            <Button
              type="button"
              disabled={batchRunning || loading || items.length === 0}
              onClick={() => void handleCheckAll()}
              className="h-9 rounded-full bg-[#06163a] px-4 text-[12.5px] font-semibold text-white shadow-none hover:bg-[#0b2048] disabled:opacity-50"
            >
              <SearchCheckIcon className="size-4" />
              {batchRunning ? "Checking..." : "Check all (max 20)"}
            </Button>
            <TooltipProvider delayDuration={200}>
              <Tooltip>
                <TooltipTrigger asChild>
                  <Button
                    type="button"
                    variant="outline"
                    size="icon"
                    className="size-9 rounded-full border-[#e8eef4] bg-white text-[#3d4b5c] shadow-none hover:bg-[#f4f7fb]"
                    aria-label="Refresh"
                    onClick={() => setRefreshKey((k) => k + 1)}
                  >
                    <RefreshCwIcon
                      className={loading ? "animate-spin" : undefined}
                    />
                  </Button>
                </TooltipTrigger>
                <TooltipContent side="bottom" sideOffset={6}>
                  Refresh list
                </TooltipContent>
              </Tooltip>
            </TooltipProvider>
          </div>
        }
      />

      {message && (
        <div className="rounded-xl border border-[#d6e8ff] bg-[#f3f8ff] px-4 py-3 text-[13px] text-[#2f5f9e]">
          {message}
        </div>
      )}

      {lastSummary && (
        <div className="rounded-xl border border-[#e8eef4] bg-white px-4 py-3 text-[13px] text-[#3d4b5c]">
          <span className="font-semibold text-[#1f2a37]">Last run · </span>
          {lastSummary}
        </div>
      )}

      <Card>
        <div className="border-b border-[#eef2f6] px-5 py-4 text-[12.5px] text-[#8a97a8]">
          {loading ? "Loading..." : `${total} candidate(s)`}
        </div>

        {loading ? (
          <LoadingBlock />
        ) : items.length === 0 ? (
          <EmptyState message="No pending payments needing reconciliation." />
        ) : (
          <TableShell>
            <table className="w-full min-w-[1180px] text-left text-[12.5px]">
              <thead>
                <tr className="border-b border-[#eef2f6] text-[11px] font-semibold uppercase tracking-wide text-[#8a97a8]">
                  <th className="px-5 py-3">Reference</th>
                  <th className="px-5 py-3">Merchant</th>
                  <th className="px-5 py-3">Provider</th>
                  <th className="px-5 py-3">Amount</th>
                  <th className="px-5 py-3">Status</th>
                  <th className="px-5 py-3">Check</th>
                  <th className="px-5 py-3">Expires</th>
                  <th className="px-5 py-3">Note</th>
                  <th className="px-5 py-3 text-right">Actions</th>
                </tr>
              </thead>
              <tbody>
                {items.map((row) => (
                  <tr
                    key={row.payment_id}
                    className="border-b border-[#f3f6f9] text-[#3d4b5c] last:border-0"
                  >
                    <td className="px-5 py-3.5">
                      <Link
                        href={`/admin/payments/${row.payment_id}`}
                        className="font-medium text-[#1f2a37] hover:text-[#ff5e16]"
                      >
                        {row.reference}
                      </Link>
                      {row.provider_reference && (
                        <p className="mt-0.5 max-w-[160px] truncate text-[11px] text-[#8a97a8]">
                          {row.provider_reference}
                        </p>
                      )}
                    </td>
                    <td className="px-5 py-3.5">{row.merchant_name || "—"}</td>
                    <td className="px-5 py-3.5 capitalize">
                      {row.provider_name}
                    </td>
                    <td className="px-5 py-3.5">
                      {formatIDR(row.amount, row.currency)}
                    </td>
                    <td className="px-5 py-3.5">
                      <StatusBadge status={row.status} />
                    </td>
                    <td className="px-5 py-3.5">
                      <span
                        className={`inline-flex rounded-full px-2.5 py-0.5 text-[11px] font-semibold ${checkStatusStyle(
                          row.check_status,
                        )}`}
                      >
                        {checkStatusLabel(row.check_status)}
                      </span>
                    </td>
                    <td className="px-5 py-3.5 text-[#6b7c93]">
                      {formatDateTime(row.expires_at)}
                    </td>
                    <td className="max-w-[220px] px-5 py-3.5 text-[12px] text-[#6b7c93]">
                      {row.note}
                    </td>
                    <td className="px-5 py-3.5">
                      <div className="flex justify-end">
                        <Button
                          type="button"
                          variant="outline"
                          size="sm"
                          disabled={
                            checkingId === row.payment_id || batchRunning
                          }
                          className="h-8 rounded-full border-[#e8eef4] bg-white px-3 text-[12px] font-semibold text-[#3d4b5c] shadow-none hover:bg-[#f4f7fb] disabled:opacity-40"
                          onClick={() => void handleCheckOne(row.payment_id)}
                        >
                          {checkingId === row.payment_id
                            ? "Checking..."
                            : "Check now"}
                        </Button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </TableShell>
        )}
      </Card>

      {lastResults.length > 0 && (
        <Card>
          <div className="border-b border-[#eef2f6] px-5 py-4">
            <h3 className="text-[15px] font-semibold text-[#1f2a37]">
              Last check results
            </h3>
            <p className="mt-0.5 text-[12.5px] text-[#8a97a8]">
              Outcome of the most recent check or batch run.
            </p>
          </div>
          <TableShell>
            <table className="w-full min-w-[900px] text-left text-[12.5px]">
              <thead>
                <tr className="border-b border-[#eef2f6] text-[11px] font-semibold uppercase tracking-wide text-[#8a97a8]">
                  <th className="px-5 py-3">Reference</th>
                  <th className="px-5 py-3">Action</th>
                  <th className="px-5 py-3">Before</th>
                  <th className="px-5 py-3">After</th>
                  <th className="px-5 py-3">Provider</th>
                  <th className="px-5 py-3">Message</th>
                </tr>
              </thead>
              <tbody>
                {lastResults.map((r) => (
                  <tr
                    key={`${r.payment_id}-${r.action}-${r.message}`}
                    className="border-b border-[#f3f6f9] text-[#3d4b5c] last:border-0"
                  >
                    <td className="px-5 py-3.5">
                      <Link
                        href={`/admin/payments/${r.payment_id}`}
                        className="font-medium text-[#1f2a37] hover:text-[#ff5e16]"
                      >
                        {r.reference}
                      </Link>
                    </td>
                    <td className="px-5 py-3.5">
                      <span
                        className={`inline-flex rounded-full px-2.5 py-0.5 text-[11px] font-semibold capitalize ${actionStyle(
                          r.action,
                        )}`}
                      >
                        {r.action.replace("_", " ")}
                      </span>
                    </td>
                    <td className="px-5 py-3.5 capitalize">
                      {r.previous_status}
                    </td>
                    <td className="px-5 py-3.5 capitalize">
                      {r.current_status}
                    </td>
                    <td className="px-5 py-3.5 capitalize">
                      {r.provider_status || "—"}
                    </td>
                    <td className="max-w-[320px] px-5 py-3.5 text-[12px] text-[#6b7c93]">
                      {r.message}
                      {r.merchant_notified ? " · merchant notified" : ""}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </TableShell>
        </Card>
      )}
    </div>
  );
}
