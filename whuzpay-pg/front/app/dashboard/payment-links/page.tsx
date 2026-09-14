"use client";

import Link from "next/link";
import { useEffect, useMemo, useState } from "react";
import toast from "react-hot-toast";
import { CopyIcon, PlusIcon, RefreshCwIcon } from "lucide-react";
import FilterDropdown from "@/app/components/admin/FilterDropdown";
import {
  fetchMerchantPaymentLinks,
  type MerchantPaymentLink,
} from "@/lib/merchant-api";
import { useMerchantEnvironment } from "@/lib/use-merchant-environment";
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
  formatIDR,
  LoadingBlock,
  PageHeader,
  TableShell,
} from "@/app/components/admin/ui";

const STATUS_OPTIONS = [
  { value: "", label: "All statuses" },
  { value: "true", label: "Active" },
  { value: "false", label: "Inactive" },
];

const PAGE_SIZE_OPTIONS = [
  { value: "10", label: "10 / page" },
  { value: "20", label: "20 / page" },
  { value: "50", label: "50 / page" },
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

function AmountTypeBadge({ type }: { type: string }) {
  const style =
    type === "fixed"
      ? "bg-[#e6f4ff] text-[#1a73c9]"
      : "bg-[#f3f0ff] text-[#7c3aed]";
  return (
    <span
      className={`inline-flex rounded-full px-2.5 py-0.5 text-[11px] font-semibold capitalize ${style}`}
    >
      {type}
    </span>
  );
}

function ActiveBadge({ isActive }: { isActive: boolean }) {
  const style = isActive
    ? "bg-[#e8f8ee] text-[#2f9e5a]"
    : "bg-[#eef2f6] text-[#6b7c93]";
  return (
    <span
      className={`inline-flex rounded-full px-2.5 py-0.5 text-[11px] font-semibold ${style}`}
    >
      {isActive ? "Active" : "Inactive"}
    </span>
  );
}

export default function PaymentLinksPage() {
  const { environment } = useMerchantEnvironment();
  const [items, setItems] = useState<MerchantPaymentLink[]>([]);
  const [total, setTotal] = useState(0);
  const [status, setStatus] = useState("");
  const [pageSize, setPageSize] = useState(20);
  const [loading, setLoading] = useState(true);
  const [refreshKey, setRefreshKey] = useState(0);

  const totalPages = Math.max(1, Math.ceil(total / pageSize));
  const [page, setPage] = usePagination([status, pageSize, environment], totalPages);
  const pageItems = useMemo(
    () => buildPageItems(page, totalPages),
    [page, totalPages],
  );

  useEffect(() => {
    let cancelled = false;
    async function load() {
      setLoading(true);
      try {
        const data = await fetchMerchantPaymentLinks({
          environment,
          is_active: status === "" ? undefined : status === "true",
          limit: pageSize,
          offset: (page - 1) * pageSize,
        });
        if (cancelled) return;
        setItems(data.items ?? []);
        setTotal(data.total ?? 0);
      } catch (err) {
        if (!cancelled) {
          toast.error(
            err instanceof Error ? err.message : "Failed to load payment links.",
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
  }, [status, page, pageSize, refreshKey, environment]);

  async function copyLink(url: string) {
    try {
      await navigator.clipboard.writeText(url);
      toast.success("Link copied.");
    } catch {
      toast.error("Failed to copy.");
    }
  }

  const rangeStart = total === 0 ? 0 : (page - 1) * pageSize + 1;
  const rangeEnd = Math.min(page * pageSize, total);

  return (
    <div className="w-full space-y-5">
      <PageHeader
        title="Payment Links"
        description={`Reusable ${environment} links you can share — every checkout through one creates a fresh payment.`}
        actions={
          <Link
            href="/dashboard/payment-links/new"
            className="inline-flex items-center gap-1.5 rounded-full bg-[#ff5e16] px-4 py-2 text-[12.5px] font-semibold text-white shadow-[0_4px_14px_rgba(255,94,22,0.25)] transition hover:bg-[#ef5510]"
          >
            <PlusIcon className="size-4" />
            New link
          </Link>
        }
      />

      <Card className="p-4 sm:p-5">
        <div className="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
          <FilterDropdown
            label="Status"
            value={status}
            options={STATUS_OPTIONS}
            onChange={setStatus}
            className="w-full sm:w-44"
          />
          <div className="flex flex-wrap items-center gap-2">
            <FilterDropdown
              label="Rows"
              value={String(pageSize)}
              options={PAGE_SIZE_OPTIONS}
              onChange={(v) => setPageSize(Number(v))}
              className="w-30"
            />
            <TooltipProvider delayDuration={200}>
              <Tooltip>
                <TooltipTrigger asChild>
                  <Button
                    type="button"
                    variant="outline"
                    size="icon"
                    className="mt-5 size-9 rounded-full border-[#e8eef4] bg-white text-[#3d4b5c] shadow-none hover:bg-[#f4f7fb]"
                    aria-label="Refresh"
                    onClick={() => setRefreshKey((k) => k + 1)}
                  >
                    <RefreshCwIcon className={loading ? "animate-spin" : undefined} />
                  </Button>
                </TooltipTrigger>
                <TooltipContent side="bottom" sideOffset={6}>
                  Refresh
                </TooltipContent>
              </Tooltip>
            </TooltipProvider>
          </div>
        </div>
      </Card>

      <Card>
        <div className="border-b border-[#eef2f6] px-5 py-4 text-[12.5px] text-[#8a97a8]">
          {loading
            ? "Loading..."
            : total === 0
              ? "0 link(s)"
              : `Showing ${rangeStart}–${rangeEnd} of ${total} link(s)`}
        </div>

        {loading ? (
          <LoadingBlock />
        ) : items.length === 0 ? (
          <EmptyState message="No payment links yet. Create one to start sharing a reusable checkout URL." />
        ) : (
          <TableShell>
            <table className="w-full min-w-225 text-left text-[12.5px]">
              <thead>
                <tr className="border-b border-[#eef2f6] text-[11px] font-semibold uppercase tracking-wide text-[#8a97a8]">
                  <th className="px-5 py-3">Title</th>
                  <th className="px-5 py-3">Type</th>
                  <th className="px-5 py-3">Amount</th>
                  <th className="px-5 py-3">Status</th>
                  <th className="px-5 py-3">Created</th>
                  <th className="px-5 py-3 text-right">Actions</th>
                </tr>
              </thead>
              <tbody>
                {items.map((l) => (
                  <tr
                    key={l.id}
                    className="border-b border-[#f3f6f9] text-[#3d4b5c] last:border-0"
                  >
                    <td className="px-5 py-3.5">
                      <Link
                        href={`/dashboard/payment-links/${l.id}`}
                        className="font-medium text-[#1f2a37] hover:text-[#ff5e16]"
                      >
                        {l.title}
                      </Link>
                      {l.description && (
                        <p className="mt-0.5 max-w-55 truncate text-[11px] text-[#8a97a8]">
                          {l.description}
                        </p>
                      )}
                    </td>
                    <td className="px-5 py-3.5">
                      <AmountTypeBadge type={l.amount_type} />
                    </td>
                    <td className="px-5 py-3.5">
                      {l.amount_type === "fixed" && l.amount
                        ? formatIDR(l.amount)
                        : "Customer sets amount"}
                    </td>
                    <td className="px-5 py-3.5">
                      <ActiveBadge isActive={l.is_active} />
                    </td>
                    <td className="px-5 py-3.5 text-[#6b7c93]">
                      {formatDateTime(l.created_at)}
                    </td>
                    <td className="px-5 py-3.5">
                      <div className="flex justify-end">
                        <TooltipProvider delayDuration={200}>
                          <Tooltip>
                            <TooltipTrigger asChild>
                              <Button
                                type="button"
                                variant="outline"
                                size="icon"
                                className="size-8 rounded-full border-[#e8eef4] bg-white text-[#3d4b5c] shadow-none hover:bg-[#f4f7fb]"
                                aria-label="Copy link"
                                onClick={() => void copyLink(l.public_url)}
                              >
                                <CopyIcon className="size-3.5" />
                              </Button>
                            </TooltipTrigger>
                            <TooltipContent side="bottom" sideOffset={6}>
                              Copy link
                            </TooltipContent>
                          </Tooltip>
                        </TooltipProvider>
                      </div>
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
