"use client";

import { useCallback, useEffect, useState } from "react";
import { Globe, KeyRound, LogIn, RotateCw, Search, Smartphone } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
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
import { getActivityLog, type ActivityAction, type ActivityEntry } from "@/lib/api";
import { useApiData } from "@/lib/use-api-data";
import { formatDateTime } from "@/lib/format";

const PAGE_SIZE = 50;

const ACTION_LABEL: Record<ActivityAction, string> = {
  login_success: "Login berhasil",
  login_failed: "Login gagal",
  password_changed: "Password diganti",
  password_reset: "Password direset lewat email",
  api_key_created: "API key dibuat",
  api_key_revoked: "API key dicabut",
  device_added: "Device ditambahkan",
  device_deleted: "Device dihapus",
};

const ACTION_BADGE: Record<ActivityAction, string> = {
  login_success: "border-transparent bg-emerald-500/15 text-emerald-700 dark:text-emerald-400",
  login_failed: "border-transparent bg-red-500/15 text-red-700 dark:text-red-400",
  password_changed: "border-transparent bg-sky-500/15 text-sky-700 dark:text-sky-400",
  password_reset: "border-transparent bg-sky-500/15 text-sky-700 dark:text-sky-400",
  api_key_created: "border-transparent bg-slate-500/15 text-slate-700 dark:text-slate-300",
  api_key_revoked: "border-transparent bg-amber-500/15 text-amber-700 dark:text-amber-400",
  device_added: "border-transparent bg-slate-500/15 text-slate-700 dark:text-slate-300",
  device_deleted: "border-transparent bg-amber-500/15 text-amber-700 dark:text-amber-400",
};

const ACTION_OPTIONS = (Object.keys(ACTION_LABEL) as ActivityAction[]).map((value) => ({
  value,
  label: ACTION_LABEL[value],
}));

function ActionIcon({ action }: { action: ActivityAction }) {
  switch (action) {
    case "login_success":
    case "login_failed":
      return <LogIn className="size-3.5" />;
    case "api_key_created":
    case "api_key_revoked":
      return <KeyRound className="size-3.5" />;
    case "device_added":
    case "device_deleted":
      return <Smartphone className="size-3.5" />;
    default:
      return null;
  }
}

function metadataText(entry: ActivityEntry): string | null {
  const m = entry.metadata;
  if (!m) return null;
  if (typeof m.name === "string") return m.name;
  if (typeof m.device_id === "string") return m.device_id;
  if (typeof m.id === "string") return m.id;
  return null;
}

export default function LogsPage() {
  const [offset, setOffset] = useState(0);
  const [action, setAction] = useState("");
  const [dateRange, setDateRange] = useState({ from: "", to: "" });

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect
    setOffset(0);
  }, [action, dateRange.from, dateRange.to]);

  const fetcher = useCallback(
    () =>
      getActivityLog(PAGE_SIZE, offset, {
        action: action ? [action as ActivityAction] : undefined,
        from: dateRange.from || undefined,
        to: dateRange.to || undefined,
      }),
    [offset, action, dateRange.from, dateRange.to],
  );
  const { data, loading, error, reload } = useApiData(fetcher);
  const hasFilter = action !== "" || dateRange.from !== "" || dateRange.to !== "";

  return (
    <div className="flex flex-col gap-6">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <h1 className="text-2xl font-semibold">Logs</h1>
          <p className="text-sm text-muted-foreground">
            Riwayat aktivitas akun kamu — login, ganti password, API key, dan device. Berguna untuk
            memastikan tidak ada akses yang bukan dari kamu.
          </p>
        </div>
        <Button variant="outline" size="sm" onClick={reload} disabled={loading}>
          <RotateCw className="mr-1.5 size-3.5" />
          Muat ulang
        </Button>
      </div>

      <div className="flex flex-col gap-2 sm:flex-row sm:flex-wrap sm:items-center">
        <FilterDropdown
          className="w-full sm:w-56"
          allLabel="Semua Aktivitas"
          value={action}
          options={ACTION_OPTIONS}
          onChange={setAction}
          searchPlaceholder="Cari aktivitas..."
        />
        <DateRangeFilter from={dateRange.from} to={dateRange.to} onChange={setDateRange} />
      </div>

      {error && (
        <Alert variant="destructive">
          <AlertTitle>Tidak dapat memuat riwayat aktivitas</AlertTitle>
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
                  <TableHead>Aktivitas</TableHead>
                  <TableHead>Detail</TableHead>
                  <TableHead>Alamat IP</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {data.map((entry) => (
                  <TableRow key={entry.id}>
                    <TableCell className="whitespace-nowrap">
                      {formatDateTime(entry.created_at)}
                    </TableCell>
                    <TableCell>
                      <Badge className={ACTION_BADGE[entry.action]}>
                        <ActionIcon action={entry.action} />
                        {ACTION_LABEL[entry.action]}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-sm text-muted-foreground">
                      {metadataText(entry) ?? "—"}
                    </TableCell>
                    <TableCell>
                      <div className="flex items-center gap-1.5 font-mono text-xs text-muted-foreground">
                        <Globe className="size-3.5 shrink-0" />
                        {entry.ip_address ?? "—"}
                      </div>
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
          <Search className="mx-auto mb-2 size-6 text-muted-foreground" />
          <p className="font-medium">
            {hasFilter
              ? "Tidak ada aktivitas yang cocok dengan filter ini"
              : "Belum ada aktivitas tercatat"}
          </p>
          <p className="mt-1 text-sm text-muted-foreground">
            {hasFilter
              ? "Coba ubah atau bersihkan filter di atas."
              : "Riwayat login, ganti password, API key, dan device akan muncul di sini."}
          </p>
        </div>
      )}
    </div>
  );
}
