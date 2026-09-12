"use client";

import { Fragment, useCallback, useEffect, useState } from "react";
import { RotateCw, ChevronDown, ChevronRight, Search } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Skeleton } from "@/components/ui/skeleton";
import { Badge } from "@/components/ui/badge";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { FilterDropdown } from "@/components/dashboard/filter-dropdown";
import { DateRangeFilter } from "@/components/dashboard/date-range-filter";
import { getInvoices, type InvoiceStatus } from "@/lib/api";
import { useApiData } from "@/lib/use-api-data";
import { formatDateTime, formatRupiah } from "@/lib/format";

const PAGE_SIZE = 50;

const STATUS_OPTIONS: { value: InvoiceStatus; label: string }[] = [
  { value: "PENDING", label: "Pending" },
  { value: "PAID", label: "Paid" },
  { value: "EXPIRED", label: "Expired" },
];

function StatusBadge({ status }: { status: InvoiceStatus }) {
  if (status === "PAID") {
    return (
      <Badge className="border-transparent bg-emerald-500/15 text-emerald-700 dark:text-emerald-400">
        Paid
      </Badge>
    );
  }
  if (status === "EXPIRED") {
    return (
      <Badge variant="outline" className="text-muted-foreground">
        Expired
      </Badge>
    );
  }
  return (
    <Badge className="border-transparent bg-amber-500/15 text-amber-700 dark:text-amber-400">
      Pending
    </Badge>
  );
}

export default function TransactionsPage() {
  const [offset, setOffset] = useState(0);
  const [expanded, setExpanded] = useState<string | null>(null);

  const [rawQuery, setRawQuery] = useState("");
  const [query, setQuery] = useState("");
  const [status, setStatus] = useState("");
  const [dateRange, setDateRange] = useState({ from: "", to: "" });

  useEffect(() => {
    const t = setTimeout(() => setQuery(rawQuery), 350);
    return () => clearTimeout(t);
  }, [rawQuery]);

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect
    setOffset(0);
  }, [query, status, dateRange.from, dateRange.to]);

  const fetcher = useCallback(
    () =>
      getInvoices(PAGE_SIZE, offset, {
        q: query || undefined,
        status: (status || undefined) as InvoiceStatus | undefined,
        from: dateRange.from || undefined,
        to: dateRange.to || undefined,
      }),
    [offset, query, status, dateRange.from, dateRange.to],
  );
  const { data, loading, error, reload } = useApiData(fetcher);

  return (
    <div className="flex flex-col gap-6">
      <div>
        <h1 className="text-2xl font-semibold">Transactions</h1>
        <p className="text-sm text-muted-foreground">
          Invoice yang dibuat lewat API dan status pencocokannya ke pembayaran yang masuk.
        </p>
      </div>

      <div className="flex flex-col gap-2 sm:flex-row sm:flex-wrap sm:items-center">
        <div className="flex h-9 min-w-0 flex-1 items-center gap-2 rounded-lg border bg-background px-3 sm:max-w-xs">
          <Search className="size-4 shrink-0 text-muted-foreground" />
          <input
            value={rawQuery}
            onChange={(e) => setRawQuery(e.target.value)}
            placeholder="Cari referensi order..."
            className="w-full min-w-0 bg-transparent text-sm outline-none placeholder:text-muted-foreground"
          />
        </div>
        <FilterDropdown
          className="w-full sm:w-44"
          allLabel="Semua Status"
          value={status}
          options={STATUS_OPTIONS}
          onChange={setStatus}
          searchPlaceholder="Cari status..."
        />
        <DateRangeFilter from={dateRange.from} to={dateRange.to} onChange={setDateRange} />
      </div>

      {error && (
        <Alert variant="destructive">
          <AlertTitle>Tidak dapat memuat transactions</AlertTitle>
          <AlertDescription className="flex items-center justify-between gap-4">
            <span>{error}</span>
            <Button size="sm" variant="outline" onClick={reload}>
              <RotateCw className="mr-1.5 size-3.5" />
              Coba lagi
            </Button>
          </AlertDescription>
        </Alert>
      )}

      {loading ? (
        <div className="flex flex-col gap-2">
          {Array.from({ length: 6 }).map((_, i) => (
            <Skeleton key={i} className="h-12" />
          ))}
        </div>
      ) : data && data.length > 0 ? (
        <>
          <div className="overflow-x-auto rounded-xl border border-border/60 shadow-sm">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="w-8" />
                  <TableHead>Referensi</TableHead>
                  <TableHead className="text-right">Nominal diminta</TableHead>
                  <TableHead className="text-right">Nominal unik</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Dibuat</TableHead>
                  <TableHead>Dibayar / Kedaluwarsa</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {data.map((inv) => (
                  <Fragment key={inv.id}>
                    <TableRow
                      className="cursor-pointer"
                      onClick={() => setExpanded(expanded === inv.id ? null : inv.id)}
                    >
                      <TableCell>
                        {expanded === inv.id ? (
                          <ChevronDown className="size-4 text-muted-foreground" />
                        ) : (
                          <ChevronRight className="size-4 text-muted-foreground" />
                        )}
                      </TableCell>
                      <TableCell className="font-medium">{inv.external_ref}</TableCell>
                      <TableCell className="text-right">
                        {formatRupiah(inv.requested_amount)}
                      </TableCell>
                      <TableCell className="text-right font-medium">
                        {formatRupiah(inv.unique_amount)}
                      </TableCell>
                      <TableCell>
                        <StatusBadge status={inv.status} />
                      </TableCell>
                      <TableCell className="whitespace-nowrap">
                        {formatDateTime(inv.created_at)}
                      </TableCell>
                      <TableCell className="whitespace-nowrap">
                        {inv.paid_at ? formatDateTime(inv.paid_at) : formatDateTime(inv.expires_at)}
                      </TableCell>
                    </TableRow>
                    {expanded === inv.id && (
                      <TableRow>
                        <TableCell colSpan={7} className="bg-muted/30">
                          <dl className="grid grid-cols-1 gap-x-6 gap-y-2 py-2 text-sm sm:grid-cols-2">
                            <div>
                              <dt className="text-xs text-muted-foreground">Invoice ID</dt>
                              <dd className="font-mono text-xs">{inv.id}</dd>
                            </div>
                            <div>
                              <dt className="text-xs text-muted-foreground">Event yang cocok</dt>
                              <dd className="font-mono text-xs">{inv.matched_event_id ?? "—"}</dd>
                            </div>
                          </dl>
                        </TableCell>
                      </TableRow>
                    )}
                  </Fragment>
                ))}
              </TableBody>
            </Table>
          </div>

          <div className="flex items-center justify-between">
            <Button
              variant="outline"
              size="sm"
              disabled={offset === 0}
              onClick={() => setOffset(Math.max(0, offset - PAGE_SIZE))}
            >
              Sebelumnya
            </Button>
            <span className="text-xs text-muted-foreground">
              Menampilkan {offset + 1}–{offset + data.length}
            </span>
            <Button
              variant="outline"
              size="sm"
              disabled={data.length < PAGE_SIZE}
              onClick={() => setOffset(offset + PAGE_SIZE)}
            >
              Berikutnya
            </Button>
          </div>
        </>
      ) : (
        <div className="rounded-2xl border border-dashed p-8 text-center">
          <p className="font-medium">
            {query || status || dateRange.from || dateRange.to
              ? "Tidak ada transaksi yang cocok dengan filter ini"
              : "Belum ada transaksi"}
          </p>
          <p className="mt-1 text-sm text-muted-foreground">
            {query || status || dateRange.from || dateRange.to
              ? "Coba ubah atau bersihkan filter di atas."
              : "Invoice akan muncul di sini begitu dibuat lewat POST /api/v1/invoices."}
          </p>
        </div>
      )}
    </div>
  );
}
