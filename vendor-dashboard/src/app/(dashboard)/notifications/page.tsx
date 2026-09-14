"use client";

import { useCallback, useEffect, useState } from "react";
import Link from "next/link";
import { Mail, RotateCw, Search, Send } from "lucide-react";
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
import {
  getNotificationLog,
  type NotificationChannel,
  type NotificationKind,
  type NotificationStatus,
} from "@/lib/api";
import { useApiData } from "@/lib/use-api-data";
import { formatDateTime } from "@/lib/format";

const PAGE_SIZE = 50;

const KIND_LABEL: Record<NotificationKind, string> = {
  expiry_reminder: "Pengingat kedaluwarsa",
  device_offline: "HP offline",
  device_online: "HP kembali online",
  test: "Pesan uji",
};

const KIND_BADGE: Record<NotificationKind, string> = {
  expiry_reminder: "border-transparent bg-amber-500/15 text-amber-700 dark:text-amber-400",
  device_offline: "border-transparent bg-red-500/15 text-red-700 dark:text-red-400",
  device_online: "border-transparent bg-emerald-500/15 text-emerald-700 dark:text-emerald-400",
  test: "border-transparent bg-slate-500/15 text-slate-700 dark:text-slate-300",
};

const KIND_OPTIONS = (Object.keys(KIND_LABEL) as NotificationKind[]).map((value) => ({
  value,
  label: KIND_LABEL[value],
}));

const CHANNEL_OPTIONS: { value: NotificationChannel; label: string }[] = [
  { value: "email", label: "Email" },
  { value: "telegram", label: "Telegram" },
];

const STATUS_OPTIONS: { value: NotificationStatus; label: string }[] = [
  { value: "sent", label: "Terkirim" },
  { value: "failed", label: "Gagal" },
];

export default function NotificationsPage() {
  const [offset, setOffset] = useState(0);
  const [rawQuery, setRawQuery] = useState("");
  const [query, setQuery] = useState("");
  const [kind, setKind] = useState("");
  const [channel, setChannel] = useState("");
  const [status, setStatus] = useState("");
  const [dateRange, setDateRange] = useState({ from: "", to: "" });

  useEffect(() => {
    const t = setTimeout(() => setQuery(rawQuery), 350);
    return () => clearTimeout(t);
  }, [rawQuery]);

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect
    setOffset(0);
  }, [query, kind, channel, status, dateRange.from, dateRange.to]);

  const fetcher = useCallback(
    () =>
      getNotificationLog(PAGE_SIZE, offset, {
        q: query || undefined,
        kind: (kind || undefined) as NotificationKind | undefined,
        channel: (channel || undefined) as NotificationChannel | undefined,
        status: (status || undefined) as NotificationStatus | undefined,
        from: dateRange.from || undefined,
        to: dateRange.to || undefined,
      }),
    [offset, query, kind, channel, status, dateRange.from, dateRange.to],
  );
  const { data, loading, error, reload } = useApiData(fetcher);
  const hasFilter =
    query.trim() !== "" ||
    kind !== "" ||
    channel !== "" ||
    status !== "" ||
    dateRange.from !== "" ||
    dateRange.to !== "";

  return (
    <div className="flex flex-col gap-6">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <h1 className="text-2xl font-semibold">Notifications</h1>
          <p className="text-sm text-muted-foreground">
            Riwayat email dan Telegram yang dikirim ke customer — pengingat kedaluwarsa, HP
            offline/online, dan pesan uji. Satu baris per channel.
          </p>
        </div>
        <Button variant="outline" size="sm" onClick={reload} disabled={loading}>
          <RotateCw className="mr-1.5 size-3.5" />
          Muat ulang
        </Button>
      </div>

      <div className="flex flex-col gap-2 sm:flex-row sm:flex-wrap sm:items-center">
        <div className="flex h-9 min-w-0 flex-1 items-center gap-2 rounded-lg border bg-background px-3 sm:max-w-xs">
          <Search className="size-4 shrink-0 text-muted-foreground" />
          <input
            value={rawQuery}
            onChange={(e) => setRawQuery(e.target.value)}
            placeholder="Cari nama bisnis, tujuan, atau nama HP..."
            className="w-full min-w-0 bg-transparent text-sm outline-none placeholder:text-muted-foreground"
          />
        </div>
        <FilterDropdown
          className="w-full sm:w-52"
          allLabel="Semua Jenis"
          value={kind}
          options={KIND_OPTIONS}
          onChange={setKind}
          searchPlaceholder="Cari jenis..."
        />
        <FilterDropdown
          className="w-full sm:w-40"
          allLabel="Semua Channel"
          value={channel}
          options={CHANNEL_OPTIONS}
          onChange={setChannel}
          searchPlaceholder="Cari channel..."
        />
        <FilterDropdown
          className="w-full sm:w-40"
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
          <AlertTitle>Tidak dapat memuat riwayat notifikasi</AlertTitle>
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
                  <TableHead>Waktu</TableHead>
                  <TableHead>Business</TableHead>
                  <TableHead>Jenis</TableHead>
                  <TableHead>Tujuan</TableHead>
                  <TableHead>Status</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {data.map((n) => (
                  <TableRow key={n.id} className="align-top">
                    <TableCell className="whitespace-nowrap">{formatDateTime(n.created_at)}</TableCell>
                    <TableCell>
                      {n.account_id ? (
                        <Link
                          href={`/accounts/${n.account_id}`}
                          className="font-medium hover:underline"
                        >
                          {n.business_name ?? n.account_id}
                        </Link>
                      ) : (
                        <span className="text-muted-foreground">—</span>
                      )}
                      {n.device_name && (
                        <div className="text-xs text-muted-foreground">{n.device_name}</div>
                      )}
                    </TableCell>
                    <TableCell>
                      <Badge className={KIND_BADGE[n.kind]}>{KIND_LABEL[n.kind]}</Badge>
                      <div className="mt-1 max-w-xs truncate text-xs text-muted-foreground" title={n.subject}>
                        {n.subject}
                      </div>
                    </TableCell>
                    <TableCell>
                      <div className="flex items-center gap-1.5">
                        {n.channel === "email" ? (
                          <Mail className="size-3.5 shrink-0 text-muted-foreground" />
                        ) : (
                          <Send className="size-3.5 shrink-0 text-muted-foreground" />
                        )}
                        <span className="font-mono text-xs">{n.recipient}</span>
                      </div>
                    </TableCell>
                    <TableCell>
                      {n.status === "sent" ? (
                        <Badge className="border-transparent bg-emerald-500/15 text-emerald-700 dark:text-emerald-400">
                          Terkirim
                        </Badge>
                      ) : (
                        <Badge className="border-transparent bg-red-500/15 text-red-700 dark:text-red-400">
                          Gagal
                        </Badge>
                      )}
                      {n.error && (
                        <p className="mt-1 max-w-sm text-xs break-words text-destructive">{n.error}</p>
                      )}
                    </TableCell>
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
            {hasFilter ? "Tidak ada notifikasi yang cocok dengan filter ini" : "Belum ada notifikasi terkirim"}
          </p>
          <p className="mt-1 text-sm text-muted-foreground">
            {hasFilter ? (
              "Coba ubah atau bersihkan filter di atas."
            ) : (
              <>
                Riwayat muncul setelah SMTP diisi di{" "}
                <Link href="/settings" className="underline underline-offset-4">
                  Settings
                </Link>{" "}
                dan ada customer yang perlu dikabari, atau setelah kamu mengirim pesan uji.
              </>
            )}
          </p>
        </div>
      )}
    </div>
  );
}
