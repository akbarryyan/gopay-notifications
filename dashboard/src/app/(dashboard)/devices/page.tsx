"use client";

import { useMemo, useState } from "react";
import { Check, Copy, Plus, RotateCw, Search } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
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
import toast from "react-hot-toast";
import { DeviceStatusBadge } from "@/components/dashboard/device-status-badge";
import { FilterDropdown } from "@/components/dashboard/filter-dropdown";
import {
  ApiError,
  createDevice,
  getDevices,
  setDeviceEnabled,
  type AdminDevice,
  type DeviceStatus,
} from "@/lib/api";
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
  const [creating, setCreating] = useState(false);
  const [newDeviceName, setNewDeviceName] = useState("");
  const [createdDevice, setCreatedDevice] = useState<{
    name: string;
    device_id: string;
    device_secret: string;
  } | null>(null);
  const [copiedField, setCopiedField] = useState<"id" | "secret" | null>(null);

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

  async function onCreateDevice() {
    if (!newDeviceName.trim()) return;
    setBusy(true);
    try {
      const created = await createDevice(newDeviceName.trim());
      toast.success(`Device "${newDeviceName.trim()}" dibuat.`);
      setCreatedDevice({
        name: newDeviceName.trim(),
        device_id: created.device_id,
        device_secret: created.device_secret,
      });
      setCreating(false);
      setNewDeviceName("");
      reload();
    } catch (err) {
      if (err instanceof ApiError && err.code === "device_limit_reached") {
        toast.error(err.message);
      } else {
        toast.error("Gagal membuat device.");
      }
    } finally {
      setBusy(false);
    }
  }

  async function copyDeviceField(field: "id" | "secret", value: string) {
    try {
      await navigator.clipboard.writeText(value);
      setCopiedField(field);
      setTimeout(() => setCopiedField(null), 2000);
    } catch {
      toast.error("Gagal menyalin — salin manual dari kotak di atas.");
    }
  }

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
      <div className="flex items-start justify-between gap-4">
        <div>
          <h1 className="text-2xl font-semibold">Devices</h1>
          <p className="text-sm text-muted-foreground">
            Perangkat Android yang terdaftar mengirim event ke instalasi ini.
          </p>
        </div>
        <Button size="sm" onClick={() => setCreating(true)}>
          <Plus className="mr-1.5 size-4" />
          Tambah Device
        </Button>
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
              : "Klik \"Tambah Device\" untuk membuat Device ID + Secret, lalu isikan ke layar Pengaturan aplikasi Android Bridge untuk mulai menerima event pembayaran."}
          </p>
        </div>
      )}

      <AlertDialog open={creating} onOpenChange={(open) => !open && setCreating(false)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Tambah device baru</AlertDialogTitle>
            <AlertDialogDescription>
              Beri nama supaya mudah dikenali, mis. &ldquo;HP Kasir Depan&rdquo;.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <div className="flex flex-col gap-2 py-2">
            <Label htmlFor="device-name">Nama</Label>
            <Input
              id="device-name"
              value={newDeviceName}
              onChange={(e) => setNewDeviceName(e.target.value)}
              placeholder="HP Kasir Depan"
              autoFocus
            />
          </div>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={busy}>Batal</AlertDialogCancel>
            <AlertDialogAction onClick={onCreateDevice} disabled={busy || !newDeviceName.trim()}>
              Buat
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* Device ID + Secret -- tampil satu kali saja, sama pola API key */}
      <AlertDialog open={createdDevice !== null} onOpenChange={(open) => !open && setCreatedDevice(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Device &ldquo;{createdDevice?.name}&rdquo; dibuat</AlertDialogTitle>
            <AlertDialogDescription>
              Salin dua nilai ini ke Settings aplikasi Android sekarang — secret{" "}
              <strong>tidak akan ditampilkan lagi</strong> setelah jendela ini ditutup.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <div className="flex flex-col gap-3 py-2">
            <div className="flex flex-col gap-1.5">
              <Label>Device ID</Label>
              <div className="flex items-center gap-2">
                <code className="min-w-0 flex-1 overflow-x-auto rounded-lg border bg-muted px-3 py-2 font-mono text-xs whitespace-nowrap">
                  {createdDevice?.device_id}
                </code>
                <Button
                  type="button"
                  variant="outline"
                  size="icon"
                  onClick={() => createdDevice && copyDeviceField("id", createdDevice.device_id)}
                >
                  {copiedField === "id" ? <Check className="size-4" /> : <Copy className="size-4" />}
                </Button>
              </div>
            </div>
            <div className="flex flex-col gap-1.5">
              <Label>Device Secret</Label>
              <div className="flex items-center gap-2">
                <code className="min-w-0 flex-1 overflow-x-auto rounded-lg border bg-muted px-3 py-2 font-mono text-xs whitespace-nowrap">
                  {createdDevice?.device_secret}
                </code>
                <Button
                  type="button"
                  variant="outline"
                  size="icon"
                  onClick={() =>
                    createdDevice && copyDeviceField("secret", createdDevice.device_secret)
                  }
                >
                  {copiedField === "secret" ? <Check className="size-4" /> : <Copy className="size-4" />}
                </Button>
              </div>
            </div>
          </div>
          <AlertDialogFooter>
            <AlertDialogAction onClick={() => setCreatedDevice(null)}>
              Sudah disalin, tutup
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

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
