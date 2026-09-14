"use client";

import Link from "next/link";
import { useEffect, useMemo, useState } from "react";
import toast from "react-hot-toast";
import {
  DownloadIcon,
  PlusIcon,
  RefreshCwIcon,
  SearchIcon,
} from "lucide-react";
import CreateMerchantDrawer from "@/app/components/admin/CreateMerchantDrawer";
import FilterDropdown from "@/app/components/admin/FilterDropdown";
import {
  exportAdminMerchants,
  fetchAdminMerchants,
  type AdminMerchant,
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
  TableShell,
} from "../../../components/admin/ui";

const STATUS_OPTIONS = [
  { value: "", label: "All statuses" },
  { value: "active", label: "Active" },
  { value: "inactive", label: "Inactive" },
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

  for (let p = start; p <= end; p += 1) {
    pages.push(p);
  }

  if (current < totalPages - 2) pages.push("ellipsis");

  pages.push(totalPages);
  return pages;
}

export default function AdminMerchantsPage() {
  const [items, setItems] = useState<AdminMerchant[]>([]);
  const [total, setTotal] = useState(0);
  const [status, setStatus] = useState("");
  const [search, setSearch] = useState("");
  const [searchInput, setSearchInput] = useState("");
  const [pageSize, setPageSize] = useState(10);
  const [loading, setLoading] = useState(true);
  const [refreshKey, setRefreshKey] = useState(0);
  const [createOpen, setCreateOpen] = useState(false);
  const [exporting, setExporting] = useState(false);

  const totalPages = Math.max(1, Math.ceil(total / pageSize));
  const [page, setPage] = usePagination([status, search, pageSize], totalPages);
  const pageItems = useMemo(
    () => buildPageItems(page, totalPages),
    [page, totalPages],
  );

  useEffect(() => {
    let cancelled = false;

    async function load() {
      setLoading(true);
      try {
        const data = await fetchAdminMerchants({
          status: status || undefined,
          search: search || undefined,
          limit: pageSize,
          offset: (page - 1) * pageSize,
        });
        if (cancelled) return;
        setItems(data.items ?? []);
        setTotal(data.total ?? 0);
      } catch (err) {
        if (cancelled) return;
        toast.error(
          err instanceof Error ? err.message : "Failed to load merchants.",
        );
      } finally {
        if (!cancelled) setLoading(false);
      }
    }

    void load();
    return () => {
      cancelled = true;
    };
  }, [status, search, page, pageSize, refreshKey]);

  const rangeStart = total === 0 ? 0 : (page - 1) * pageSize + 1;
  const rangeEnd = Math.min(page * pageSize, total);

  function handleRefresh() {
    setRefreshKey((k) => k + 1);
  }

  async function handleExport() {
    setExporting(true);
    try {
      await exportAdminMerchants({
        status: status || undefined,
        search: search || undefined,
      });
      toast.success("Merchants exported successfully.");
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Failed to export merchants.",
      );
    } finally {
      setExporting(false);
    }
  }

  return (
    <div className="w-full space-y-5">
      <PageHeader
        title="Merchants"
        description="Businesses receiving payments through the platform."
      />

      <Card className="p-4 sm:p-5">
        <div className="flex flex-col gap-3 xl:flex-row xl:items-end xl:justify-between">
          {/* Left: status + search */}
          <div className="flex min-w-0 flex-1 flex-col gap-3 sm:flex-row sm:items-end">
            <FilterDropdown
              label="Status"
              value={status}
              options={STATUS_OPTIONS}
              onChange={setStatus}
              className="w-full sm:w-42.5"
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
                  placeholder="Search business, owner, email..."
                  className="h-9 w-full rounded-full border border-[#e8eef4] bg-[#f8fafc] pr-3 pl-9 text-[13px] text-[#1f2a37] outline-none focus:border-[#3b9eff] focus:bg-white focus:ring-2 focus:ring-[#3b9eff]/15"
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

          {/* Right: action icons bersebelahan langsung dengan rows filter */}
          <TooltipProvider delayDuration={200}>
            <div className="flex shrink-0 flex-nowrap items-center justify-end gap-1.5">
              <Tooltip>
                <TooltipTrigger asChild>
                  <Button
                    type="button"
                    variant="outline"
                    size="icon"
                    className="size-9 rounded-full border-[#e8eef4] bg-white text-[#3d4b5c] shadow-none hover:bg-[#f4f7fb]"
                    aria-label="Add merchant"
                    onClick={() => setCreateOpen(true)}
                  >
                    <PlusIcon />
                  </Button>
                </TooltipTrigger>
                <TooltipContent side="bottom" sideOffset={6}>
                  Add merchant
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
                    onClick={handleRefresh}
                  >
                    <RefreshCwIcon className={loading ? "animate-spin" : undefined} />
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
                    className="size-9 rounded-full border-[#e8eef4] bg-white text-[#3d4b5c] shadow-none hover:bg-[#f4f7fb]"
                    aria-label="Export"
                    disabled={exporting}
                    onClick={() => void handleExport()}
                  >
                    <DownloadIcon className={exporting ? "animate-pulse" : undefined} />
                  </Button>
                </TooltipTrigger>
                <TooltipContent side="bottom" sideOffset={6}>
                  Export CSV
                </TooltipContent>
              </Tooltip>

              <FilterDropdown
                value={String(pageSize)}
                options={PAGE_SIZE_OPTIONS}
                onChange={(value) => setPageSize(Number(value))}
                className="w-29.5 min-w-29.5"
                triggerClassName="h-9"
              />
            </div>
          </TooltipProvider>
        </div>
      </Card>

      <Card>
        <div className="flex items-center justify-between border-b border-[#eef2f6] px-5 py-3 text-[12.5px] text-[#8a97a8]">
          <span>
            {loading
              ? "Loading..."
              : total === 0
                ? "0 merchants"
                : `Showing ${rangeStart}–${rangeEnd} of ${total}`}
          </span>
        </div>

        {loading ? (
          <LoadingBlock />
        ) : items.length === 0 ? (
          <EmptyState message="No merchants found." />
        ) : (
          <TableShell>
            <table className="w-full min-w-200 text-left text-[12.5px]">
              <thead>
                <tr className="border-b border-[#eef2f6] text-[11px] font-semibold uppercase tracking-wide text-[#8a97a8]">
                  <th className="px-5 py-3">Business</th>
                  <th className="px-5 py-3">Owner</th>
                  <th className="px-5 py-3">Contact</th>
                  <th className="px-5 py-3">Status</th>
                  <th className="px-5 py-3">Created</th>
                </tr>
              </thead>
              <tbody>
                {items.map((m) => (
                  <tr
                    key={m.id}
                    className="border-b border-[#f3f6f9] text-[#3d4b5c] last:border-0"
                  >
                    <td className="px-5 py-3.5">
                      <Link
                        href={`/admin/merchants/${m.id}`}
                        className="font-medium text-[#1f2a37] hover:text-[#ff5e16]"
                      >
                        {m.business_name}
                      </Link>
                    </td>
                    <td className="px-5 py-3.5">{m.name}</td>
                    <td className="px-5 py-3.5">
                      {m.email}
                      {m.phone && (
                        <p className="mt-0.5 text-[11px] text-[#8a97a8]">
                          {m.phone}
                        </p>
                      )}
                    </td>
                    <td className="px-5 py-3.5">
                      <span
                        className={`inline-flex rounded-full px-2.5 py-0.5 text-[11px] font-semibold ${
                          m.is_active
                            ? "bg-[#e8f8ee] text-[#2f9e5a]"
                            : "bg-[#fff1ed] text-[#e85d3b]"
                        }`}
                      >
                        {m.is_active ? "Active" : "Inactive"}
                      </span>
                    </td>
                    <td className="px-5 py-3.5 text-[#6b7c93]">
                      {formatDateTime(m.created_at)}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </TableShell>
        )}

        {!loading && total > 0 && (
          <div className="flex flex-col gap-3 border-t border-[#eef2f6] px-4 py-4 sm:flex-row sm:items-center sm:justify-between">
            <p className="text-[12.5px] text-[#8a97a8]">
              Page {page} of {totalPages}
            </p>

            <Pagination className="mx-0 w-auto justify-end">
              <PaginationContent>
                <PaginationItem>
                  <PaginationPrevious
                    href="#"
                    text="Prev"
                    className={
                      page <= 1
                        ? "pointer-events-none opacity-40"
                        : "cursor-pointer"
                    }
                    onClick={(e) => {
                      e.preventDefault();
                      if (page > 1) setPage((p) => p - 1);
                    }}
                  />
                </PaginationItem>

                {pageItems.map((item, index) =>
                  item === "ellipsis" ? (
                    <PaginationItem key={`ellipsis-${index}`}>
                      <PaginationEllipsis />
                    </PaginationItem>
                  ) : (
                    <PaginationItem key={item}>
                      <PaginationLink
                        href="#"
                        isActive={item === page}
                        className="cursor-pointer rounded-full"
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
                    text="Next"
                    className={
                      page >= totalPages
                        ? "pointer-events-none opacity-40"
                        : "cursor-pointer"
                    }
                    onClick={(e) => {
                      e.preventDefault();
                      if (page < totalPages) setPage((p) => p + 1);
                    }}
                  />
                </PaginationItem>
              </PaginationContent>
            </Pagination>
          </div>
        )}
      </Card>

      <CreateMerchantDrawer
        open={createOpen}
        onOpenChange={setCreateOpen}
        onCreated={() => {
          setPage(1);
          handleRefresh();
        }}
      />
    </div>
  );
}
