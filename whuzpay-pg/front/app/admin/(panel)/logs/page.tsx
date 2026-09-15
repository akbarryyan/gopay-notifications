"use client";

import Link from "next/link";
import { useEffect, useMemo, useState } from "react";
import toast from "react-hot-toast";
import {
  DownloadIcon,
  RefreshCwIcon,
} from "lucide-react";
import FilterDropdown from "@/app/components/admin/FilterDropdown";
import {
  exportAdminLogs,
  fetchAdminLogs,
  type AdminLog,
} from "@/lib/admin-api";
import { usePagination } from "@/lib/use-pagination";
import { Button } from "@/components/ui/button";
import {
  Pagination,
  PaginationContent,
  PaginationEllipsis,
  PaginationItem,
  PaginationLink,
  PaginationNext,
  PaginationPrevious,
} from "@/components/ui/pagination";
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
  LoadingBlock,
  PageHeader,
  StatusBadge,
  TableShell,
} from "../../../components/admin/ui";

const STATUS_OPTIONS = [
  { value: "", label: "All statuses" },
  { value: "success", label: "Success" },
  { value: "failed", label: "Failed" },
  { value: "pending", label: "Pending" },
  { value: "error", label: "Error" },
];

const PROCESSED_OPTIONS = [
  { value: "", label: "All processed" },
  { value: "true", label: "Processed" },
  { value: "false", label: "Not processed" },
];

const PROVIDER_OPTIONS = [
  { value: "", label: "All providers" },
  { value: "gopay", label: "GoPay" },
];

const PAGE_SIZE_OPTIONS = [
  { value: "10", label: "10 / page" },
  { value: "20", label: "20 / page" },
  { value: "50", label: "50 / page" },
  { value: "100", label: "100 / page" },
];

function buildPageItems(current: number, totalPages: number) {
  if (totalPages <= 7) {
    return Array.from({ length: totalPages }, (_, i) => i + 1);
  }

  const pages: Array<number | "ellipsis"> = [1];
  if (current > 3) pages.push("ellipsis");

  const start = Math.max(2, current - 1);
  const end = Math.min(totalPages - 1, current + 1);
  for (let p = start; p <= end; p += 1) pages.push(p);

  if (current < totalPages - 2) pages.push("ellipsis");
  pages.push(totalPages);
  return pages;
}

export default function AdminLogsPage() {
  const [items, setItems] = useState<AdminLog[]>([]);
  const [total, setTotal] = useState(0);
  const [status, setStatus] = useState("");
  const [provider, setProvider] = useState("");
  const [processed, setProcessed] = useState("");
  const [pageSize, setPageSize] = useState(20);
  const [loading, setLoading] = useState(true);
  const [refreshKey, setRefreshKey] = useState(0);
  const [exporting, setExporting] = useState(false);

  const totalPages = Math.max(1, Math.ceil(total / pageSize));
  const [page, setPage] = usePagination(
    [status, provider, processed, pageSize],
    totalPages,
  );
  const pageItems = useMemo(
    () => buildPageItems(page, totalPages),
    [page, totalPages],
  );

  useEffect(() => {
    let cancelled = false;

    async function load() {
      setLoading(true);
      try {
        const data = await fetchAdminLogs({
          status: status || undefined,
          provider: provider || undefined,
          processed: processed || undefined,
          limit: pageSize,
          offset: (page - 1) * pageSize,
        });
        if (cancelled) return;
        setItems(data.items ?? []);
        setTotal(data.total ?? 0);
      } catch (err) {
        if (cancelled) return;
        toast.error(err instanceof Error ? err.message : "Failed to load logs.");
      } finally {
        if (!cancelled) setLoading(false);
      }
    }

    void load();
    return () => {
      cancelled = true;
    };
  }, [status, provider, processed, page, pageSize, refreshKey]);

  const rangeStart = total === 0 ? 0 : (page - 1) * pageSize + 1;
  const rangeEnd = Math.min(page * pageSize, total);

  async function handleExport() {
    setExporting(true);
    try {
      await exportAdminLogs({
        status: status || undefined,
        provider: provider || undefined,
        processed: processed || undefined,
      });
      toast.success("Logs exported successfully.");
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to export logs.");
    } finally {
      setExporting(false);
    }
  }

  return (
    <div className="w-full space-y-5">
      <PageHeader
        title="Logs"
        description="Webhook and processing events for payment debugging. Secrets are never shown."
      />

      <Card className="p-4 sm:p-5">
        <div className="flex flex-col gap-3 xl:flex-row xl:items-end xl:justify-between">
          <div className="flex min-w-0 flex-1 flex-col gap-3 sm:flex-row sm:flex-wrap sm:items-end">
            <FilterDropdown
              label="Status"
              value={status}
              options={STATUS_OPTIONS}
              onChange={setStatus}
              className="w-full sm:w-40"
            />
            <FilterDropdown
              label="Provider"
              value={provider}
              options={PROVIDER_OPTIONS}
              onChange={setProvider}
              className="w-full sm:w-40"
            />
            <FilterDropdown
              label="Processed"
              value={processed}
              options={PROCESSED_OPTIONS}
              onChange={setProcessed}
              className="w-full sm:w-40"
            />
          </div>

          <div className="flex flex-wrap items-center gap-2">
            <FilterDropdown
              label="Rows"
              value={String(pageSize)}
              options={PAGE_SIZE_OPTIONS}
              onChange={(v) => setPageSize(Number(v))}
              className="w-[120px]"
            />

            <TooltipProvider delayDuration={200}>
              <div className="flex items-center gap-1.5 pt-5">
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
                    Refresh
                  </TooltipContent>
                </Tooltip>

                <Tooltip>
                  <TooltipTrigger asChild>
                    <Button
                      type="button"
                      variant="outline"
                      size="icon"
                      disabled={exporting}
                      className="size-9 rounded-full border-[#e8eef4] bg-white text-[#3d4b5c] shadow-none hover:bg-[#f4f7fb]"
                      aria-label="Export CSV"
                      onClick={() => void handleExport()}
                    >
                      <DownloadIcon />
                    </Button>
                  </TooltipTrigger>
                  <TooltipContent side="bottom" sideOffset={6}>
                    Export CSV
                  </TooltipContent>
                </Tooltip>
              </div>
            </TooltipProvider>
          </div>
        </div>
      </Card>

      <Card>
        <div className="border-b border-[#eef2f6] px-5 py-4 text-[12.5px] text-[#8a97a8]">
          {loading
            ? "Loading..."
            : total === 0
              ? "0 event(s)"
              : `Showing ${rangeStart}–${rangeEnd} of ${total} event(s)`}
        </div>

        {loading ? (
          <LoadingBlock />
        ) : items.length === 0 ? (
          <EmptyState message="No webhook events match these filters." />
        ) : (
          <TableShell>
            <table className="w-full min-w-[960px] text-left text-[12.5px]">
              <thead>
                <tr className="border-b border-[#eef2f6] text-[11px] font-semibold uppercase tracking-wide text-[#8a97a8]">
                  <th className="px-5 py-3">Time</th>
                  <th className="px-5 py-3">Provider</th>
                  <th className="px-5 py-3">Event</th>
                  <th className="px-5 py-3">Status</th>
                  <th className="px-5 py-3">Processed</th>
                  <th className="px-5 py-3">Payment</th>
                  <th className="px-5 py-3">Error</th>
                </tr>
              </thead>
              <tbody>
                {items.map((log) => (
                  <tr
                    key={log.id}
                    className="border-b border-[#f3f6f9] text-[#3d4b5c] last:border-0"
                  >
                    <td className="px-5 py-3.5 text-[#6b7c93]">
                      {formatDateTime(log.created_at)}
                    </td>
                    <td className="px-5 py-3.5 capitalize">
                      {log.provider_name}
                      <p className="mt-0.5 max-w-[180px] truncate text-[11px] text-[#8a97a8]">
                        {log.provider_reference}
                      </p>
                    </td>
                    <td className="px-5 py-3.5">
                      <Link
                        href={`/admin/logs/${log.id}`}
                        className="font-medium text-[#1f2a37] hover:text-[#ff5e16]"
                      >
                        {log.event_type}
                      </Link>
                    </td>
                    <td className="px-5 py-3.5">
                      <StatusBadge status={log.status} />
                    </td>
                    <td className="px-5 py-3.5">
                      <span
                        className={`inline-flex rounded-full px-2.5 py-0.5 text-[11px] font-semibold ${
                          log.is_processed
                            ? "bg-[#e8f8ee] text-[#2f9e5a]"
                            : "bg-[#fff7e6] text-[#c27a00]"
                        }`}
                      >
                        {log.is_processed ? "Yes" : "No"}
                      </span>
                    </td>
                    <td className="px-5 py-3.5">
                      {log.payment_id ? (
                        <Link
                          href={`/admin/payments/${log.payment_id}`}
                          className="font-medium text-[#ff5e16] hover:underline"
                        >
                          View
                        </Link>
                      ) : (
                        "—"
                      )}
                    </td>
                    <td className="max-w-[220px] truncate px-5 py-3.5 text-[#e85d3b]">
                      {log.processing_error || "—"}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </TableShell>
        )}

        {!loading && total > 0 && (
          <div className="flex flex-col gap-3 border-t border-[#eef2f6] px-4 py-3 sm:flex-row sm:items-center sm:justify-between">
            <p className="text-[12.5px] text-[#8a97a8]">
              Page {page} of {totalPages}
            </p>
            <Pagination className="mx-0 w-auto justify-end">
              <PaginationContent>
                <PaginationItem>
                  <PaginationPrevious
                    href="#"
                    onClick={(e) => {
                      e.preventDefault();
                      setPage((p) => Math.max(1, p - 1));
                    }}
                    className={
                      page <= 1 ? "pointer-events-none opacity-40" : undefined
                    }
                  />
                </PaginationItem>
                {pageItems.map((item, idx) =>
                  item === "ellipsis" ? (
                    <PaginationItem key={`e-${idx}`}>
                      <PaginationEllipsis />
                    </PaginationItem>
                  ) : (
                    <PaginationItem key={item}>
                      <PaginationLink
                        href="#"
                        isActive={item === page}
                        onClick={(e) => {
                          e.preventDefault();
                          setPage(item);
                        }}
                      >
                        {item}
                      </PaginationLink>
                    </PaginationItem>
                  ),
                )}
                <PaginationItem>
                  <PaginationNext
                    href="#"
                    onClick={(e) => {
                      e.preventDefault();
                      setPage((p) => Math.min(totalPages, p + 1));
                    }}
                    className={
                      page >= totalPages
                        ? "pointer-events-none opacity-40"
                        : undefined
                    }
                  />
                </PaginationItem>
              </PaginationContent>
            </Pagination>
          </div>
        )}
      </Card>
    </div>
  );
}
