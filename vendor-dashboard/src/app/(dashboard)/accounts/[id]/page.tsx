"use client";

import { use, useCallback, useState } from "react";
import Link from "next/link";
import { RotateCw, Smartphone } from "lucide-react";
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
  changeAccountPlan,
  getAccount,
  getAccountDevices,
  renewAccount,
  revokeAccount,
  suspendAccount,
  type AccountPlan,
} from "@/lib/api";
import { useApiData } from "@/lib/use-api-data";
import { formatDateOnly, formatDateTime } from "@/lib/format";
import { STATUS_BADGE } from "@/lib/account-status";

const PLANS: AccountPlan[] = ["Starter", "Business", "Enterprise"];

export default function AccountDetailPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = use(params);
  const fetcher = useCallback(() => getAccount(id), [id]);
  const { data, loading, error, reload } = useApiData(fetcher);
  const devicesFetcher = useCallback(() => getAccountDevices(id), [id]);
  const devices = useApiData(devicesFetcher);

  const [renewing, setRenewing] = useState(false);
  const [newExpiresAt, setNewExpiresAt] = useState("");
  const [changingPlan, setChangingPlan] = useState(false);
  const [newPlan, setNewPlan] = useState<AccountPlan>("Starter");
  const [pendingSuspend, setPendingSuspend] = useState(false);
  const [pendingRevoke, setPendingRevoke] = useState(false);
  const [busy, setBusy] = useState(false);

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
          <h2 className="text-base font-semibold">Devices</h2>
          <p className="text-xs text-muted-foreground">
            Read-only -- customer menambah/menonaktifkan device sendiri lewat dashboard mereka.
          </p>
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
              Kuota device menyesuaikan otomatis ke plan baru. Tidak mengubah tanggal
              kedaluwarsa atau status akun.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <div className="flex flex-col gap-2 py-2">
            <Label htmlFor="new-plan">Plan</Label>
            <select
              id="new-plan"
              value={newPlan}
              onChange={(e) => setNewPlan(e.target.value as AccountPlan)}
              className="h-9 rounded-md border border-input bg-transparent px-3 text-sm"
            >
              {PLANS.map((p) => (
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

      <AlertDialog open={pendingSuspend} onOpenChange={(open) => !open && setPendingSuspend(false)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Suspend akun?</AlertDialogTitle>
            <AlertDialogDescription>
              Seluruh endpoint device/admin/API key customer ini langsung ditolak
              402 sejak permintaan berikutnya. Bisa diaktifkan lagi kapan saja.
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
              Tidak bisa dibatalkan — buat akun baru kalau customer ini perlu
              diaktifkan lagi.
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
