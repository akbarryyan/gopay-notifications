"use client";

import Link from "next/link";
import { useSearchParams } from "next/navigation";
import { useEffect, useMemo, useState } from "react";
import toast from "react-hot-toast";
import {
  DownloadIcon,
  PlusIcon,
  RefreshCwIcon,
  SearchIcon,
} from "lucide-react";
import FilterDropdown from "@/app/components/admin/FilterDropdown";
import {
  createMerchantPayment,
  exportMerchantPayments,
  fetchMerchantPayments,
  type MerchantPayment,
} from "@/lib/merchant-api";
import { useMerchantEnvironment } from "@/lib/use-merchant-environment";
import { usePagination } from "@/lib/use-pagination";
import { Button } from "@/components/ui/button";
import { DatePicker } from "@/components/ui/date-picker";
import {
  Pagination,
  PaginationContent,
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
  formatIDR,
  LoadingBlock,
  PageHeader,
  StatusBadge,
  TableShell,
} from "../../components/admin/ui";

const STATUS_OPTIONS = [
  { value: "", label: "All statuses" },
  { value: "pending", label: "Pending" },
  { value: "paid", label: "Paid" },
  { value: "expired", label: "Expired" },
  { value: "failed", label: "Failed" },
];

const PAGE_SIZE_OPTIONS = [
  { value: "10", label: "10 / page" },
  { value: "20", label: "20 / page" },
  { value: "50", label: "50 / page" },
];

const VALID = new Set(["pending", "paid", "expired", "failed", "cancelled"]);

export default function MerchantPaymentsPage() {
  const searchParams = useSearchParams();
  const statusFromUrl = searchParams.get("status") ?? "";
  const { environment } = useMerchantEnvironment();
  const [items, setItems] = useState<MerchantPayment[]>([]);
  const [total, setTotal] = useState(0);
  const [status, setStatus] = useState(() =>
    VALID.has(statusFromUrl) ? statusFromUrl : "",
  );
  const [search, setSearch] = useState("");
  const [searchInput, setSearchInput] = useState("");
  const [dateFrom, setDateFrom] = useState("");
  const [dateTo, setDateTo] = useState("");
  const [pageSize, setPageSize] = useState(10);
  const [loading, setLoading] = useState(true);
  const [refreshKey, setRefreshKey] = useState(0);
  const [creating, setCreating] = useState(false);
  const [exporting, setExporting] = useState(false);

  const totalPages = Math.max(1, Math.ceil(total / pageSize));
  const [page, setPage] = usePagination(
    [status, search, dateFrom, dateTo, pageSize, environment],
    totalPages,
  );

  // Forward a valid ?status= from the URL into local filter state whenever
  // it changes — adjusted during render (not an effect) so this doesn't
  // cost an extra render/fetch cycle for a value we already have.
  const [prevStatusFromUrl, setPrevStatusFromUrl] = useState(statusFromUrl);
  if (prevStatusFromUrl !== statusFromUrl) {
    setPrevStatusFromUrl(statusFromUrl);
    if (VALID.has(statusFromUrl)) setStatus(statusFromUrl);
  }

  useEffect(() => {
    let cancelled = false;
    async function load() {
      setLoading(true);
      try {
        const data = await fetchMerchantPayments({
          status: status || undefined,
          search: search || undefined,
          date_from: dateFrom || undefined,
          date_to: dateTo || undefined,
          environment,
          limit: pageSize,
          offset: (page - 1) * pageSize,
        });
        if (cancelled) return;
        setItems(data.items ?? []);
        setTotal(data.total ?? 0);
      } catch (err) {
        if (!cancelled) {
          toast.error(
            err instanceof Error ? err.message : "Failed to load payments.",
          );
        }
      } finally {
        if (!cancelled) setLoading(false);
      }
    }
    void load();
    return () => {
      cancelled = true;
    };
  }, [status, search, dateFrom, dateTo, page, pageSize, refreshKey, environment]);

  async function handleQuickCreate() {
    setCreating(true);
    try {
      // Production goes through the real gopay-notifications provider, which
      // adds a random 1-999 "unique nominal" on top of this base amount to
      // tell payments apart — so 1 here caps the actual payable amount at
      // Rp 1.000, safe to pay with pocket change while testing.
      // Sandbox is a mock adapter that enforces its own Rp 2.000 floor.
      const amount = environment === "production" ? 1 : 2000;
      const p = await createMerchantPayment({
        amount,
        description: "Quick test payment from dashboard",
        payment_method: "qris",
        environment,
      });
      toast.success(`Payment ${p.reference} created (${environment}).`);
      setRefreshKey((k) => k + 1);
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Failed to create payment.",
      );
    } finally {
      setCreating(false);
    }
  }

  async function handleExport() {
    setExporting(true);
    try {
      await exportMerchantPayments({
        status: status || undefined,
        search: search || undefined,
        date_from: dateFrom || undefined,
        date_to: dateTo || undefined,
        environment,
      });
      toast.success("Payments exported successfully.");
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Failed to export payments.",
      );
    } finally {
      setExporting(false);
    }
  }

  const rangeStart = total === 0 ? 0 : (page - 1) * pageSize + 1;
  const rangeEnd = Math.min(page * pageSize, total);
  const pageItems = useMemo(() => {
    if (totalPages <= 7)
      return Array.from({ length: totalPages }, (_, i) => i + 1);
    const pages: Array<number | "e"> = [1];
    if (page > 3) pages.push("e");
    for (
      let p = Math.max(2, page - 1);
      p <= Math.min(totalPages - 1, page + 1);
      p++
    )
      pages.push(p);
    if (page < totalPages - 2) pages.push("e");
    pages.push(totalPages);
    return pages;
  }, [page, totalPages]);

  return (
    <div className="w-full space-y-5">
      <PageHeader
        title="Payments"
        description={`Showing ${environment} transactions. Switch environment from the header.`}
      />

      <Card className="p-4 sm:p-5">
        <div className="flex flex-col gap-3 xl:flex-row xl:items-end xl:justify-between">
          <div className="flex min-w-0 flex-1 flex-col gap-3 lg:flex-row lg:flex-wrap lg:items-end">
            <FilterDropdown
              label="Status"
              value={status}
              options={STATUS_OPTIONS}
              onChange={setStatus}
              className="w-full sm:w-40"
            />

            <DatePicker
              label="From"
              value={dateFrom}
              onChange={(v) => {
                setDateFrom(v);
                // Keep range valid if To is before new From
                if (v && dateTo && dateTo < v) setDateTo(v);
              }}
              placeholder="Start date"
              max={dateTo || undefined}
              className="w-full sm:w-42"
            />

            <DatePicker
              label="To"
              value={dateTo}
              onChange={setDateTo}
              placeholder="End date"
              min={dateFrom || undefined}
              className="w-full sm:w-42"
            />

            <form
              className="flex w-full flex-1 gap-2 sm:max-w-md"
              onSubmit={(e) => {
                e.preventDefault();
                setSearch(searchInput.trim());
              }}
            >
              <div className="relative w-full">
                <SearchIcon className="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-[#8a97a8]" />
                <input
                  value={searchInput}
                  onChange={(e) => setSearchInput(e.target.value)}
                  placeholder="Search reference, customer..."
                  className="h-9 w-full rounded-full border border-[#e8eef4] bg-[#f8fafc] pr-3 pl-9 text-[13px] outline-none focus:border-[#3b9eff] focus:bg-white focus:ring-2 focus:ring-[#3b9eff]/15"
                />
              </div>
              <button
                type="submit"
                className="h-9 shrink-0 rounded-full bg-[#ff5e16] px-5 text-[12.5px] font-semibold text-white"
              >
                Search
              </button>
            </form>
          </div>

          <TooltipProvider delayDuration={200}>
            <div className="flex shrink-0 flex-wrap items-end justify-end gap-1.5">
              <FilterDropdown
                label="Rows"
                value={String(pageSize)}
                options={PAGE_SIZE_OPTIONS}
                onChange={(v) => setPageSize(Number(v))}
                className="w-30"
              />

              <div className="flex items-center gap-1.5 pt-5">
                <Tooltip>
                  <TooltipTrigger asChild>
                    <Button
                      type="button"
                      variant="outline"
                      size="icon"
                      disabled={creating}
                      className="size-9 rounded-full border-[#e8eef4] bg-white text-[#3d4b5c] shadow-none hover:bg-[#f4f7fb]"
                      aria-label="New test payment"
                      onClick={() => void handleQuickCreate()}
                    >
                      <PlusIcon />
                    </Button>
                  </TooltipTrigger>
                  <TooltipContent side="bottom" sideOffset={6}>
                    New test payment
                  </TooltipContent>
                </Tooltip>

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
            </div>
          </TooltipProvider>
        </div>

        {(dateFrom || dateTo) && (
          <div className="mt-3 flex flex-wrap items-center gap-2 text-[12.5px] text-[#6b7c93]">
            <span>
              Date filter: {dateFrom || "…"} → {dateTo || "…"}
            </span>
            <button
              type="button"
              className="font-semibold text-[#ff5e16] hover:underline"
              onClick={() => {
                setDateFrom("");
                setDateTo("");
              }}
            >
              Clear dates
            </button>
          </div>
        )}
      </Card>

      <Card>
        <div className="border-b border-[#eef2f6] px-5 py-4 text-[12.5px] text-[#8a97a8]">
          {loading
            ? "Loading..."
            : total === 0
              ? "0 payment(s)"
              : `Showing ${rangeStart}–${rangeEnd} of ${total}`}
        </div>
        {loading ? (
          <LoadingBlock />
        ) : items.length === 0 ? (
          <EmptyState message="No payments match these filters." />
        ) : (
          <TableShell>
            <table className="w-full min-w-200 text-left text-[12.5px]">
              <thead>
                <tr className="border-b border-[#eef2f6] text-[11px] font-semibold uppercase tracking-wide text-[#8a97a8]">
                  <th className="px-5 py-3">Reference</th>
                  <th className="px-5 py-3">Amount</th>
                  <th className="px-5 py-3">Status</th>
                  <th className="px-5 py-3">Provider</th>
                  <th className="px-5 py-3">Created</th>
                </tr>
              </thead>
              <tbody>
                {items.map((p) => (
                  <tr
                    key={p.id}
                    className="border-b border-[#f3f6f9] text-[#3d4b5c] last:border-0"
                  >
                    <td className="px-5 py-3.5">
                      <Link
                        href={`/dashboard/payments/${p.id}`}
                        className="font-medium text-[#1f2a37] hover:text-[#ff5e16]"
                      >
                        {p.reference}
                      </Link>
                      <p className="mt-0.5 max-w-55 truncate text-[11px] text-[#8a97a8]">
                        {p.description}
                      </p>
                    </td>
                    <td className="px-5 py-3.5">
                      {formatIDR(p.amount, p.currency)}
                    </td>
                    <td className="px-5 py-3.5">
                      <StatusBadge status={p.status} />
                    </td>
                    <td className="px-5 py-3.5 capitalize">{p.provider_name}</td>
                    <td className="px-5 py-3.5 text-[#6b7c93]">
                      {formatDateTime(p.created_at)}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </TableShell>
        )}

        {!loading && total > 0 && (
          <div className="flex items-center justify-between border-t border-[#eef2f6] px-4 py-3">
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
                  item === "e" ? (
                    <PaginationItem key={`e-${idx}`}>
                      <span className="px-2 text-[#8a97a8] rounded-full">…</span>
                    </PaginationItem>
                  ) : (
                    <PaginationItem key={item}>
                      <PaginationLink
                        href="#"
                        className="rounded-full"
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
