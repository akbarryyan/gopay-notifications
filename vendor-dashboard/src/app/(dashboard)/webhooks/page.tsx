"use client";

import { useCallback, useEffect, useState } from "react";
import Link from "next/link";
import { RotateCw, Search } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Skeleton } from "@/components/ui/skeleton";
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
import { getWebhookDeliveries, type DeliveryStatus } from "@/lib/api";
import { useApiData } from "@/lib/use-api-data";
import { formatDateTime } from "@/lib/format";

const PAGE_SIZE = 50;

const STATUS_OPTIONS: { value: DeliveryStatus; label: string }[] = [
  { value: "DELIVERED", label: "Delivered" },
  { value: "RETRYING", label: "Retrying" },
  { value: "FAILED", label: "Failed" },
  { value: "PENDING", label: "Pending" },
];

const STATUS_BADGE: Record<DeliveryStatus, string> = {
  DELIVERED: "border-transparent bg-emerald-500/15 text-emerald-700 dark:text-emerald-400",
  RETRYING: "border-transparent bg-amber-500/15 text-amber-700 dark:text-amber-400",
  FAILED: "border-transparent bg-red-500/15 text-red-700 dark:text-red-400",
  PENDING: "border-transparent bg-slate-500/15 text-slate-700 dark:text-slate-300",
};

export default function WebhooksPage() {
  const [offset, setOffset] = useState(0);
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
      getWebhookDeliveries(PAGE_SIZE, offset, {
        q: query || undefined,
        status: (status || undefined) as DeliveryStatus | undefined,
        from: dateRange.from || undefined,
        to: dateRange.to || undefined,
      }),
    [offset, query, status, dateRange.from, dateRange.to],
  );
  const { data, loading, error, reload } = useApiData(fetcher);
  const hasFilter =
    query.trim() !== "" || status !== "" || dateRange.from !== "" || dateRange.to !== "";

  return (
    <div className="flex flex-col gap-6">
      <div>
        <h1 className="text-2xl font-semibold">Webhooks</h1>
        <p className="text-sm text-muted-foreground">
          Riwayat pengiriman webhook lintas semua account — read-only, untuk menelusuri
          kegagalan tanpa membuka database.
        </p>
      </div>

      <div className="flex flex-col gap-2 sm:flex-row sm:flex-wrap sm:items-center">
        <div className="flex h-9 min-w-0 flex-1 items-center gap-2 rounded-lg border bg-background px-3 sm:max-w-xs">
          <Search className="size-4 shrink-0 text-muted-foreground" />
          <input
            value={rawQuery}
            onChange={(e) => setRawQuery(e.target.value)}
            placeholder="Cari nama bisnis, nama endpoint, atau URL..."
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
          <AlertTitle>Tidak dapat memuat riwayat webhook</AlertTitle>
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
                  <TableHead>Business</TableHead>
                  <TableHead>Endpoint</TableHead>
                  <TableHead>Event</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead className="text-right">Percobaan</TableHead>
                  <TableHead className="text-right">HTTP</TableHead>
                  <TableHead className="text-right">Durasi</TableHead>
                  <TableHead>Dibuat</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {data.map((d) => (
                  <TableRow key={d.id}>
                    <TableCell>
                      <Link
                        href={`/accounts/${d.account_id}`}
                        className="font-medium hover:underline"
                      >
                        {d.business_name}
                      </Link>
                    </TableCell>
                    <TableCell>
                      <div className="font-medium">{d.endpoint_name}</div>
                      <div className="max-w-xs truncate font-mono text-xs text-muted-foreground">
                        {d.endpoint_url}
                      </div>
                    </TableCell>
                    <TableCell className="font-mono text-xs">{d.event}</TableCell>
                    <TableCell>
                      <Badge className={STATUS_BADGE[d.status]}>{d.status}</Badge>
                      {d.status === "RETRYING" && d.next_attempt_at && (
                        <p className="mt-1 text-xs text-muted-foreground">
                          Coba lagi {formatDateTime(d.next_attempt_at)}
                        </p>
                      )}
                    </TableCell>
                    <TableCell className="text-right">{d.attempt}</TableCell>
                    <TableCell className="text-right">
                      {d.http_status ?? <span className="text-muted-foreground">tidak terhubung</span>}
                    </TableCell>
                    <TableCell className="text-right">
                      {d.duration_ms !== null ? `${d.duration_ms}ms` : "—"}
                    </TableCell>
                    <TableCell className="whitespace-nowrap">{formatDateTime(d.created_at)}</TableCell>
                  </TableRow>
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
            {hasFilter
              ? "Tidak ada pengiriman yang cocok dengan filter ini"
              : "Belum ada pengiriman webhook"}
          </p>
          <p className="mt-1 text-sm text-muted-foreground">
            {hasFilter
              ? "Coba ubah atau bersihkan filter di atas."
              : "Riwayat akan muncul begitu ada customer yang mendaftarkan endpoint webhook dan invoice-nya lunas/kedaluwarsa."}
          </p>
        </div>
      )}
    </div>
  );
}
