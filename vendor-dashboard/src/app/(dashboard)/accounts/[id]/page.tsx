"use client";

import { use, useCallback, useState } from "react";
import Link from "next/link";
import { RotateCw } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Skeleton } from "@/components/ui/skeleton";
import { Badge } from "@/components/ui/badge";
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
import { getAccount, renewAccount, revokeAccount, suspendAccount, type Account } from "@/lib/api";
import { useApiData } from "@/lib/use-api-data";
import { formatDateOnly } from "@/lib/format";

const STATUS_BADGE: Record<Account["status"], string> = {
  active: "border-transparent bg-emerald-500/15 text-emerald-700 dark:text-emerald-400",
  expiring: "border-transparent bg-amber-500/15 text-amber-700 dark:text-amber-400",
  expired: "border-transparent bg-red-500/15 text-red-700 dark:text-red-400",
  suspended: "border-transparent bg-red-500/15 text-red-700 dark:text-red-400",
  revoked: "border-transparent bg-red-500/15 text-red-700 dark:text-red-400",
};

export default function AccountDetailPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = use(params);
  const fetcher = useCallback(() => getAccount(id), [id]);
  const { data, loading, error, reload } = useApiData(fetcher);

  const [renewing, setRenewing] = useState(false);
  const [newExpiresAt, setNewExpiresAt] = useState("");
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
