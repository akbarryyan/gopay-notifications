"use client";

import { useMemo, useState } from "react";
import { RotateCw, Search } from "lucide-react";
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
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { toast } from "sonner";
import { DeviceStatusBadge } from "@/components/dashboard/device-status-badge";
import { FilterDropdown } from "@/components/dashboard/filter-dropdown";
import { getDevices, setDeviceEnabled, type AdminDevice, type DeviceStatus } from "@/lib/api";
import { useApiData } from "@/lib/use-api-data";
import { formatDateTime, timeAgo } from "@/lib/format";

const STATUS_OPTIONS: { value: DeviceStatus; label: string }[] = [
  { value: "ONLINE", label: "Online" },
  { value: "OFFLINE", label: "Offline" },
  { value: "PENDING", label: "Pending" },
  { value: "DISABLED", label: "Nonaktif" },
];

export default function DevicesPage() {
  const { data, loading, error, reload } = useApiData(getDevices);
  const [pendingToggle, setPendingToggle] = useState<AdminDevice | null>(null);
  const [busy, setBusy] = useState(false);
  const [query, setQuery] = useState("");
  const [status, setStatus] = useState("");

  // Devices sudah diambil seutuhnya (tidak berpaginasi di backend), jadi
  // pencarian dan filter status cukup dilakukan di sisi klien atas data yang
  // sudah ada — tidak perlu request baru ke server.
  const filtered = useMemo(() => {
    if (!data) return data;
    return data.filter((d) => {
      if (status && d.status !== status) return false;
      if (query.trim() !== "") {
        const q = query.trim().toLowerCase();
        if (!d.name.toLowerCase().includes(q) && !d.device_id.toLowerCase().includes(q)) {
          return false;
        }
      }
      return true;
    });
  }, [data, query, status]);

  async function confirmToggle() {
    if (!pendingToggle) return;
    setBusy(true);
    try {
      await setDeviceEnabled(pendingToggle.device_id, !pendingToggle.enabled);
      toast.success(
        pendingToggle.enabled
          ? `${pendingToggle.name} dinonaktifkan.`
          : `${pendingToggle.name} diaktifkan kembali.`,
      );
      reload();
    } catch {
      toast.error("Gagal mengubah status device.");
    } finally {
      setBusy(false);
      setPendingToggle(null);
    }
  }

  return (
    <div className="flex flex-col gap-6">
      <div>
        <h1 className="text-2xl font-semibold">Devices</h1>
        <p className="text-sm text-muted-foreground">
          Perangkat Android yang terdaftar mengirim event ke instalasi ini.
        </p>
      </div>

      <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
        <div className="flex h-9 min-w-0 flex-1 items-center gap-2 rounded-lg border bg-background px-3 sm:max-w-xs">
          <Search className="size-4 shrink-0 text-muted-foreground" />
          <input
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Cari nama atau device ID..."
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
      </div>

      {error && (
        <Alert variant="destructive">
          <AlertTitle>Tidak dapat memuat devices</AlertTitle>
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
          {Array.from({ length: 3 }).map((_, i) => (
            <Skeleton key={i} className="h-14" />
          ))}
        </div>
      ) : filtered && filtered.length > 0 ? (
        <div className="overflow-x-auto rounded-xl border border-border/60 shadow-sm">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Device</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Heartbeat terakhir</TableHead>
                <TableHead>Versi Android</TableHead>
                <TableHead>Versi aplikasi</TableHead>
                <TableHead className="text-right">Antrean</TableHead>
                <TableHead className="text-right">Aksi</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {filtered.map((d) => (
                <TableRow key={d.device_id}>
                  <TableCell>
                    <div className="font-medium">{d.name}</div>
                    <div className="text-xs text-muted-foreground">{d.device_id}</div>
                  </TableCell>
                  <TableCell>
                    <DeviceStatusBadge status={d.status} />
                    {d.listener_connected === false && d.status !== "DISABLED" && (
                      <p className="mt-1 text-xs text-amber-600 dark:text-amber-400">
                        Listener tidak terikat
                      </p>
                    )}
                  </TableCell>
                  <TableCell title={formatDateTime(d.heartbeat_at)}>
                    {timeAgo(d.heartbeat_at)}
                  </TableCell>
                  <TableCell>{d.android_version ?? "—"}</TableCell>
                  <TableCell>{d.app_version ?? "—"}</TableCell>
                  <TableCell className="text-right text-xs text-muted-foreground">
                    {d.pending_count ?? 0} pending · {d.failed_count ?? 0} gagal
                  </TableCell>
                  <TableCell className="text-right">
                    <Button
                      size="sm"
                      variant={d.enabled ? "outline" : "default"}
                      onClick={() => setPendingToggle(d)}
                    >
                      {d.enabled ? "Nonaktifkan" : "Aktifkan"}
                    </Button>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      ) : (
        <div className="rounded-2xl border border-dashed p-8 text-center">
          <p className="font-medium">
            {query || status ? "Tidak ada device yang cocok dengan filter ini" : "Belum ada device terhubung"}
          </p>
          <p className="mt-1 text-sm text-muted-foreground">
            {query || status
              ? "Coba ubah atau bersihkan filter di atas."
              : "Pasang aplikasi Android Bridge dan isi Device ID / Secret di layar Pengaturannya untuk mulai menerima event pembayaran."}
          </p>
        </div>
      )}

      <AlertDialog open={pendingToggle !== null} onOpenChange={(open) => !open && setPendingToggle(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>
              {pendingToggle?.enabled ? "Nonaktifkan device?" : "Aktifkan device?"}
            </AlertDialogTitle>
            <AlertDialogDescription>
              {pendingToggle?.enabled
                ? `"${pendingToggle?.name}" akan berhenti dapat mengirim event sampai diaktifkan kembali.`
                : `"${pendingToggle?.name}" akan dapat mengirim event lagi.`}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={busy}>Batal</AlertDialogCancel>
            <AlertDialogAction onClick={confirmToggle} disabled={busy}>
              {pendingToggle?.enabled ? "Nonaktifkan" : "Aktifkan"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
