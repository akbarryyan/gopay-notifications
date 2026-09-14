"use client";

import { use, useCallback, useState } from "react";
import Link from "next/link";
import { Check, Copy, KeyRound, Plus, RotateCw, Smartphone } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
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
import {
  ApiError,
  changeAccountPlan,
  createAccountDevice,
  deleteAccountDevice,
  getAccount,
  getAccountAPIKeys,
  getAccountDevices,
  getPlans,
  renewAccount,
  revokeAccount,
  revokeAccountAPIKey,
  sendPasswordReset,
  setAccountDeviceEnabled,
  suspendAccount,
  type AccountApiKey,
  type Device,
} from "@/lib/api";
import { useApiData } from "@/lib/use-api-data";
import { formatDateOnly, formatDateTime } from "@/lib/format";
import { STATUS_BADGE } from "@/lib/account-status";

export default function AccountDetailPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = use(params);
  const fetcher = useCallback(() => getAccount(id), [id]);
  const { data, loading, error, reload } = useApiData(fetcher);
  const devicesFetcher = useCallback(() => getAccountDevices(id), [id]);
  const devices = useApiData(devicesFetcher);
  const apiKeysFetcher = useCallback(() => getAccountAPIKeys(id), [id]);
  const apiKeys = useApiData(apiKeysFetcher);
  const plans = useApiData(getPlans);

  const [renewing, setRenewing] = useState(false);
  const [newExpiresAt, setNewExpiresAt] = useState("");
  const [changingPlan, setChangingPlan] = useState(false);
  const [newPlan, setNewPlan] = useState("");
  // Nama plan account ini sendiri SELALU jadi pilihan, walau sudah dihapus
  // dari halaman Plans -- supaya dialog tidak diam-diam melompat ke plan
  // lain begitu dibuka.
  const planOptions = data
    ? Array.from(new Set([data.plan, ...(plans.data ?? []).map((p) => p.name)]))
    : (plans.data ?? []).map((p) => p.name);
  const [pendingSuspend, setPendingSuspend] = useState(false);
  const [pendingRevoke, setPendingRevoke] = useState(false);
  const [pendingPasswordReset, setPendingPasswordReset] = useState(false);
  const [sendingReset, setSendingReset] = useState(false);
  const [busy, setBusy] = useState(false);

  const [creatingDevice, setCreatingDevice] = useState(false);
  const [newDeviceName, setNewDeviceName] = useState("");
  const [createdDevice, setCreatedDevice] = useState<{
    name: string;
    device_id: string;
    device_secret: string;
  } | null>(null);
  const [copiedField, setCopiedField] = useState<"id" | "secret" | null>(null);
  const [pendingToggleDevice, setPendingToggleDevice] = useState<Device | null>(null);
  const [pendingDeleteDevice, setPendingDeleteDevice] = useState<Device | null>(null);
  const [pendingRevokeKey, setPendingRevokeKey] = useState<AccountApiKey | null>(null);

  async function onRenew() {
    if (!newExpiresAt) return;
    setBusy(true);
    try {
      await renewAccount(id, newExpiresAt);
      toast.success("Akun diperpanjang.");
      setRenewing(false);
      setNewExpiresAt("");
      reload();
    } catch {
      toast.error("Gagal memperpanjang akun.");
    } finally {
      setBusy(false);
    }
  }

  async function onChangePlan() {
    setBusy(true);
    try {
      await changeAccountPlan(id, newPlan);
      toast.success(`Plan diubah ke ${newPlan}.`);
      setChangingPlan(false);
      reload();
    } catch {
      toast.error("Gagal mengubah plan.");
    } finally {
      setBusy(false);
    }
  }

  async function onSuspend() {
    setBusy(true);
    try {
      await suspendAccount(id);
      toast.success("Akun disuspend.");
      reload();
    } catch {
      toast.error("Gagal suspend akun.");
    } finally {
      setBusy(false);
      setPendingSuspend(false);
    }
  }

  async function onRevoke() {
    setBusy(true);
    try {
      await revokeAccount(id);
      toast.success("Akun dicabut.");
      reload();
    } catch {
      toast.error("Gagal mencabut akun.");
    } finally {
      setBusy(false);
      setPendingRevoke(false);
    }
  }

  async function onSendPasswordReset() {
    setSendingReset(true);
    try {
      await sendPasswordReset(id);
      toast.success("Link reset password dikirim ke email customer.");
      setPendingPasswordReset(false);
    } catch (err) {
      if (err instanceof ApiError && err.code === "not_available") {
        toast.error("Belum bisa mengirim -- isi SMTP di Settings dulu.");
      } else if (err instanceof ApiError && err.code === "too_many_attempts") {
        toast.error("Link baru saja dikirim, tunggu beberapa menit sebelum mengirim lagi.");
      } else if (err instanceof ApiError && err.code === "account_revoked") {
        toast.error("Account ini sudah dicabut.");
      } else {
        toast.error("Gagal mengirim link reset password.");
      }
    } finally {
      setSendingReset(false);
    }
  }

  async function onCreateDevice() {
    if (!newDeviceName.trim()) return;
    setBusy(true);
    try {
      const created = await createAccountDevice(id, newDeviceName.trim());
      toast.success(`Device "${newDeviceName.trim()}" dibuat.`);
      setCreatedDevice({
        name: newDeviceName.trim(),
        device_id: created.device_id,
        device_secret: created.device_secret,
      });
      setCreatingDevice(false);
      setNewDeviceName("");
      devices.reload();
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
      toast.error("Gagal menyalin -- salin manual dari kotak di atas.");
    }
  }

  async function confirmToggleDevice() {
    if (!pendingToggleDevice) return;
    setBusy(true);
    try {
      await setAccountDeviceEnabled(id, pendingToggleDevice.device_id, !pendingToggleDevice.enabled);
      toast.success(
        pendingToggleDevice.enabled
          ? `${pendingToggleDevice.name} dinonaktifkan.`
          : `${pendingToggleDevice.name} diaktifkan kembali.`,
      );
      devices.reload();
    } catch {
      toast.error("Gagal mengubah status device.");
    } finally {
      setBusy(false);
      setPendingToggleDevice(null);
    }
  }

  async function confirmDeleteDevice() {
    if (!pendingDeleteDevice) return;
    setBusy(true);
    try {
      await deleteAccountDevice(id, pendingDeleteDevice.device_id);
      toast.success(`${pendingDeleteDevice.name} dihapus.`);
      devices.reload();
    } catch (err) {
      if (err instanceof ApiError && err.code === "device_has_events") {
        toast.error("Device ini sudah punya riwayat event -- nonaktifkan saja, tidak bisa dihapus.");
      } else {
        toast.error("Gagal menghapus device.");
      }
    } finally {
      setBusy(false);
      setPendingDeleteDevice(null);
    }
  }

  async function confirmRevokeKey() {
    if (!pendingRevokeKey) return;
    setBusy(true);
    try {
      await revokeAccountAPIKey(id, pendingRevokeKey.id);
      toast.success(`API key "${pendingRevokeKey.name}" dicabut.`);
      apiKeys.reload();
    } catch {
      toast.error("Gagal mencabut API key.");
    } finally {
      setBusy(false);
      setPendingRevokeKey(null);
    }
  }

  return (
    <div className="flex flex-col gap-6">
      <div>
        <Link href="/accounts" className="text-sm text-muted-foreground hover:underline">
          ← Kembali ke Accounts
        </Link>
        <h1 className="mt-1 text-2xl font-semibold">{data?.business_name ?? "..."}</h1>
      </div>

      {error && (
        <Alert variant="destructive">
          <AlertTitle>Tidak dapat memuat akun</AlertTitle>
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
        <Skeleton className="h-40 rounded-2xl" />
      ) : data ? (
        <div className="rounded-2xl border border-border/60 p-6 shadow-sm">
          <div className="flex flex-wrap items-center justify-between gap-3">
            <div className="flex items-center gap-3">
              <Badge className={STATUS_BADGE[data.status]}>{data.status}</Badge>
              <span className="text-sm text-muted-foreground">
                Berakhir {formatDateOnly(data.expires_at)} · {data.days_remaining} hari tersisa
              </span>
            </div>
            <div className="flex gap-2">
              <Button size="sm" variant="outline" onClick={() => setRenewing(true)}>
                Renew
              </Button>
              <Button
                size="sm"
                variant="outline"
                onClick={() => {
                  setNewPlan(data.plan);
                  setChangingPlan(true);
                }}
              >
                Ubah Plan
              </Button>
              <Button size="sm" variant="outline" onClick={() => setPendingPasswordReset(true)}>
                <KeyRound className="mr-1.5 size-3.5" />
                Kirim link reset password
              </Button>
              <Button size="sm" variant="outline" onClick={() => setPendingSuspend(true)}>
                Suspend
              </Button>
              <Button size="sm" variant="destructive" onClick={() => setPendingRevoke(true)}>
                Revoke
              </Button>
            </div>
          </div>
          <div className="mt-4 grid grid-cols-2 gap-x-8 gap-y-2 text-sm sm:grid-cols-4">
            <div>
              <p className="text-muted-foreground">Username</p>
              <p className="font-mono font-medium">{data.username}</p>
            </div>
            <div>
              <p className="text-muted-foreground">Email</p>
              <p className="font-medium">{data.email}</p>
            </div>
            <div>
              <p className="text-muted-foreground">Plan</p>
              <p className="font-medium">{data.plan}</p>
            </div>
            <div>
              <p className="text-muted-foreground">Max devices</p>
              <p className="font-medium">{data.max_devices < 0 ? "Unlimited" : data.max_devices}</p>
            </div>
          </div>
        </div>
      ) : null}

      <div className="rounded-2xl border border-border/60 p-6 shadow-sm">
        <div className="mb-4 flex items-center justify-between gap-3">
          <div>
            <h2 className="text-base font-semibold">Devices</h2>
            <p className="text-xs text-muted-foreground">
              Bantu customer pasang/lepas HP lewat sini kalau mereka belum sempat lakukan sendiri
              di dashboard mereka.
            </p>
          </div>
          <Button size="sm" onClick={() => setCreatingDevice(true)}>
            <Plus className="mr-1.5 size-4" />
            Tambah Device
          </Button>
        </div>
        {devices.loading ? (
          <Skeleton className="h-14 rounded-xl" />
        ) : devices.error ? (
          <p className="text-sm text-destructive">{devices.error}</p>
        ) : devices.data && devices.data.length > 0 ? (
          <div className="overflow-x-auto rounded-xl border border-border/60">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Device</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Heartbeat terakhir</TableHead>
                  <TableHead>Versi Android</TableHead>
                  <TableHead className="text-right">Aksi</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {devices.data.map((d) => (
                  <TableRow key={d.device_id}>
                    <TableCell>
                      <div className="font-medium">{d.name}</div>
                      <div className="font-mono text-xs text-muted-foreground">{d.device_id}</div>
                    </TableCell>
                    <TableCell>
                      <DeviceStatusBadge status={d.status} />
                    </TableCell>
                    <TableCell>{formatDateTime(d.heartbeat_at)}</TableCell>
                    <TableCell>{d.android_version ?? "—"}</TableCell>
                    <TableCell className="text-right">
                      <div className="flex justify-end gap-2">
                        <Button
                          size="sm"
                          variant={d.enabled ? "outline" : "default"}
                          onClick={() => setPendingToggleDevice(d)}
                        >
                          {d.enabled ? "Nonaktifkan" : "Aktifkan"}
                        </Button>
                        <Button size="sm" variant="destructive" onClick={() => setPendingDeleteDevice(d)}>
                          Hapus
                        </Button>
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        ) : (
          <div className="rounded-xl border border-dashed p-6 text-center">
            <Smartphone className="mx-auto mb-2 size-6 text-muted-foreground" />
            <p className="text-sm text-muted-foreground">Belum ada device terhubung.</p>
          </div>
        )}
      </div>

      <div className="rounded-2xl border border-border/60 p-6 shadow-sm">
        <div className="mb-4 flex items-center justify-between gap-3">
          <div>
            <h2 className="text-base font-semibold">API Keys</h2>
            <p className="text-xs text-muted-foreground">
              Cabut kalau customer lapor key-nya bocor atau kepakai website lama. Key baru tetap
              dibuat customer sendiri lewat dashboard mereka.
            </p>
          </div>
        </div>
        {apiKeys.loading ? (
          <Skeleton className="h-14 rounded-xl" />
        ) : apiKeys.error ? (
          <p className="text-sm text-destructive">{apiKeys.error}</p>
        ) : apiKeys.data && apiKeys.data.length > 0 ? (
          <div className="overflow-x-auto rounded-xl border border-border/60">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Nama</TableHead>
                  <TableHead>Dibuat</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead className="text-right">Aksi</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {apiKeys.data.map((k) => (
                  <TableRow key={k.id}>
                    <TableCell>
                      <div className="font-medium">{k.name}</div>
                      <div className="font-mono text-xs text-muted-foreground">{k.id}</div>
                    </TableCell>
                    <TableCell>{formatDateTime(k.created_at)}</TableCell>
                    <TableCell>
                      {k.revoked_at ? (
                        <Badge variant="secondary" className="bg-muted text-muted-foreground">
                          Dicabut {formatDateTime(k.revoked_at)}
                        </Badge>
                      ) : (
                        <Badge className="bg-emerald-100 text-emerald-800 dark:bg-emerald-900/40 dark:text-emerald-300">
                          Aktif
                        </Badge>
                      )}
                    </TableCell>
                    <TableCell className="text-right">
                      <Button
                        size="sm"
                        variant="destructive"
                        disabled={!!k.revoked_at}
                        onClick={() => setPendingRevokeKey(k)}
                      >
                        Cabut
                      </Button>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        ) : (
          <div className="rounded-xl border border-dashed p-6 text-center">
            <KeyRound className="mx-auto mb-2 size-6 text-muted-foreground" />
            <p className="text-sm text-muted-foreground">Belum ada API key dibuat.</p>
          </div>
        )}
      </div>

      <AlertDialog open={creatingDevice} onOpenChange={(open) => !open && setCreatingDevice(false)}>
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
      <AlertDialog
        open={createdDevice !== null}
        onOpenChange={(open) => !open && setCreatedDevice(null)}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Device &ldquo;{createdDevice?.name}&rdquo; dibuat</AlertDialogTitle>
            <AlertDialogDescription>
              Salin lalu teruskan ke customer lewat kanal sendiri (WA/telepon). Secret{" "}
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
                  {copiedField === "secret" ? (
                    <Check className="size-4" />
                  ) : (
                    <Copy className="size-4" />
                  )}
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

      <AlertDialog
        open={pendingToggleDevice !== null}
        onOpenChange={(open) => !open && setPendingToggleDevice(null)}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>
              {pendingToggleDevice?.enabled ? "Nonaktifkan" : "Aktifkan"} device &ldquo;
              {pendingToggleDevice?.name}&rdquo;?
            </AlertDialogTitle>
            <AlertDialogDescription>
              {pendingToggleDevice?.enabled
                ? "Device berhenti bisa mengirim event sampai diaktifkan lagi. Tidak menghapus riwayatnya."
                : "Device bisa mengirim event lagi seperti biasa."}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={busy}>Batal</AlertDialogCancel>
            <AlertDialogAction onClick={confirmToggleDevice} disabled={busy}>
              {pendingToggleDevice?.enabled ? "Nonaktifkan" : "Aktifkan"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <AlertDialog
        open={pendingDeleteDevice !== null}
        onOpenChange={(open) => !open && setPendingDeleteDevice(null)}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Hapus device &ldquo;{pendingDeleteDevice?.name}&rdquo;?</AlertDialogTitle>
            <AlertDialogDescription>
              Tidak bisa dibatalkan. Device yang sudah punya riwayat event akan ditolak -- nonaktifkan
              saja kalau begitu.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={busy}>Batal</AlertDialogCancel>
            <AlertDialogAction onClick={confirmDeleteDevice} disabled={busy}>
              Hapus
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <AlertDialog
        open={pendingRevokeKey !== null}
        onOpenChange={(open) => !open && setPendingRevokeKey(null)}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Cabut API key &ldquo;{pendingRevokeKey?.name}&rdquo;?</AlertDialogTitle>
            <AlertDialogDescription>
              Tidak bisa dibatalkan -- website/integrasi customer yang masih memakai key ini langsung
              ditolak sejak permintaan berikutnya.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={busy}>Batal</AlertDialogCancel>
            <AlertDialogAction onClick={confirmRevokeKey} disabled={busy}>
              Cabut
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <AlertDialog open={renewing} onOpenChange={(open) => !open && setRenewing(false)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Perpanjang akun</AlertDialogTitle>
            <AlertDialogDescription>Tanggal berakhir baru.</AlertDialogDescription>
          </AlertDialogHeader>
          <div className="flex flex-col gap-2 py-2">
            <Label htmlFor="new-expires-at">Berakhir</Label>
            <Input
              id="new-expires-at"
              type="date"
              value={newExpiresAt}
              onChange={(e) => setNewExpiresAt(e.target.value)}
            />
          </div>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={busy}>Batal</AlertDialogCancel>
            <AlertDialogAction onClick={onRenew} disabled={busy || !newExpiresAt}>
              Perpanjang
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <AlertDialog open={changingPlan} onOpenChange={(open) => !open && setChangingPlan(false)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Ubah plan</AlertDialogTitle>
            <AlertDialogDescription>
              Kuota device menyesuaikan otomatis ke plan baru. Tidak mengubah tanggal kedaluwarsa
              atau status akun.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <div className="flex flex-col gap-2 py-2">
            <Label htmlFor="new-plan">Plan</Label>
            <select
              id="new-plan"
              value={newPlan}
              onChange={(e) => setNewPlan(e.target.value)}
              className="h-9 rounded-md border border-input bg-transparent px-3 text-sm"
            >
              {planOptions.map((p) => (
                <option key={p} value={p}>
                  {p}
                </option>
              ))}
            </select>
          </div>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={busy}>Batal</AlertDialogCancel>
            <AlertDialogAction onClick={onChangePlan} disabled={busy || newPlan === data?.plan}>
              Ubah
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <AlertDialog
        open={pendingPasswordReset}
        onOpenChange={(open) => !open && setPendingPasswordReset(false)}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Kirim link reset password?</AlertDialogTitle>
            <AlertDialogDescription>
              Link untuk membuat password baru dikirim ke{" "}
              <span className="font-medium text-foreground">{data?.email}</span>, berlaku 30 menit.
              Password saat ini tidak berubah sampai customer membukanya.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={sendingReset}>Batal</AlertDialogCancel>
            <AlertDialogAction onClick={onSendPasswordReset} disabled={sendingReset}>
              {sendingReset ? "Mengirim..." : "Kirim"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <AlertDialog open={pendingSuspend} onOpenChange={(open) => !open && setPendingSuspend(false)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Suspend akun?</AlertDialogTitle>
            <AlertDialogDescription>
              Seluruh endpoint device/admin/API key customer ini langsung ditolak 402 sejak
              permintaan berikutnya. Bisa diaktifkan lagi kapan saja.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={busy}>Batal</AlertDialogCancel>
            <AlertDialogAction onClick={onSuspend} disabled={busy}>
              Suspend
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <AlertDialog open={pendingRevoke} onOpenChange={(open) => !open && setPendingRevoke(false)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Cabut akun?</AlertDialogTitle>
            <AlertDialogDescription>
              Tidak bisa dibatalkan — buat akun baru kalau customer ini perlu diaktifkan lagi.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={busy}>Batal</AlertDialogCancel>
            <AlertDialogAction onClick={onRevoke} disabled={busy}>
              Cabut
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
