"use client";

import { useState } from "react";
import { RotateCw } from "lucide-react";
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
import { getDevices, setDeviceEnabled, type AdminDevice } from "@/lib/api";
import { useApiData } from "@/lib/use-api-data";
import { formatDateTime, timeAgo } from "@/lib/format";

export default function DevicesPage() {
  const { data, loading, error, reload } = useApiData(getDevices);
  const [pendingToggle, setPendingToggle] = useState<AdminDevice | null>(null);
  const [busy, setBusy] = useState(false);

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
      ) : data && data.length > 0 ? (
        <div className="overflow-x-auto rounded-md border">
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
              {data.map((d) => (
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
        <div className="rounded-md border border-dashed p-8 text-center">
          <p className="font-medium">Belum ada device terhubung</p>
          <p className="mt-1 text-sm text-muted-foreground">
            Pasang aplikasi Android Bridge dan isi Device ID / Secret di layar Pengaturannya
            untuk mulai menerima event pembayaran.
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
