"use client";

import { use, useState } from "react";
import Link from "next/link";
import { RotateCw } from "lucide-react";
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
import { toast } from "sonner";
import {
  getLicenseDetail,
  renewLicense,
  resetInstallation,
  revokeLicense,
  suspendLicense,
  type Installation,
} from "@/lib/api";
import { useApiData } from "@/lib/use-api-data";
import { formatDateOnly, formatDateTime } from "@/lib/format";

const STATUS_BADGE: Record<string, string> = {
  active: "border-transparent bg-emerald-500/15 text-emerald-700 dark:text-emerald-400",
  expiring: "border-transparent bg-amber-500/15 text-amber-700 dark:text-amber-400",
  expired: "border-transparent bg-red-500/15 text-red-700 dark:text-red-400",
  suspended: "border-transparent bg-red-500/15 text-red-700 dark:text-red-400",
  revoked: "border-transparent bg-red-500/15 text-red-700 dark:text-red-400",
};

export default function LicenseDetailPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = use(params);
  const { data, loading, error, reload } = useApiData(() => getLicenseDetail(id));

  const [renewing, setRenewing] = useState(false);
  const [newExpiresAt, setNewExpiresAt] = useState("");
  const [pendingSuspend, setPendingSuspend] = useState(false);
  const [pendingRevoke, setPendingRevoke] = useState(false);
  const [pendingReset, setPendingReset] = useState<Installation | null>(null);
  const [busy, setBusy] = useState(false);

  async function onRenew() {
    if (!newExpiresAt) return;
    setBusy(true);
    try {
      await renewLicense(id, newExpiresAt);
      toast.success("License diperpanjang.");
      setRenewing(false);
      setNewExpiresAt("");
      reload();
    } catch {
      toast.error("Gagal memperpanjang license.");
    } finally {
      setBusy(false);
    }
  }

  async function onSuspend() {
    setBusy(true);
    try {
      await suspendLicense(id);
      toast.success("License disuspend.");
      reload();
    } catch {
      toast.error("Gagal suspend license.");
    } finally {
      setBusy(false);
      setPendingSuspend(false);
    }
  }

  async function onRevoke() {
    setBusy(true);
    try {
      await revokeLicense(id);
      toast.success("License dicabut.");
      reload();
    } catch {
      toast.error("Gagal mencabut license.");
    } finally {
      setBusy(false);
      setPendingRevoke(false);
    }
  }

  async function onReset() {
    if (!pendingReset) return;
    setBusy(true);
    try {
      await resetInstallation(pendingReset.id);
      toast.success("Installation direset — kuota terbuka lagi.");
      reload();
    } catch {
      toast.error("Gagal reset installation.");
    } finally {
      setBusy(false);
      setPendingReset(null);
    }
  }

  return (
    <div className="flex flex-col gap-6">
      <div>
        {data && (
          <Link href={`/customers/${data.license.customer_id}`} className="text-sm text-muted-foreground hover:underline">
            ← Kembali ke customer
          </Link>
        )}
        <h1 className="mt-1 text-2xl font-semibold">{data?.license.plan ?? "..."} License</h1>
      </div>

      {error && (
        <Alert variant="destructive">
          <AlertTitle>Tidak dapat memuat license</AlertTitle>
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
        <>
          <div className="rounded-2xl border border-border/60 p-6 shadow-sm">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-3">
                <Badge className={STATUS_BADGE[data.license.status]}>{data.license.status}</Badge>
                <span className="text-sm text-muted-foreground">
                  Berakhir {formatDateOnly(data.license.expires_at)} · {data.license.days_remaining} hari
                  tersisa
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
                <p className="text-muted-foreground">Max devices</p>
                <p className="font-medium">{data.license.max_devices < 0 ? "Unlimited" : data.license.max_devices}</p>
              </div>
              <div>
                <p className="text-muted-foreground">Kuota production</p>
                <p className="font-medium">{data.license.production_installations}</p>
              </div>
              <div>
                <p className="text-muted-foreground">Kuota UAT</p>
                <p className="font-medium">{data.license.uat_installations}</p>
              </div>
              <div>
                <p className="text-muted-foreground">Diterbitkan</p>
                <p className="font-medium">{formatDateOnly(data.license.issued_at)}</p>
              </div>
            </div>
          </div>

          <div>
            <h2 className="mb-2 text-lg font-semibold">Installations</h2>
            {data.installations.length > 0 ? (
              <div className="overflow-x-auto rounded-xl border border-border/60 shadow-sm">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Environment</TableHead>
                      <TableHead>Versi</TableHead>
                      <TableHead>Aktivasi</TableHead>
                      <TableHead>Status</TableHead>
                      <TableHead className="text-right">Aksi</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {data.installations.map((inst) => (
                      <TableRow key={inst.id}>
                        <TableCell className="font-medium">{inst.environment}</TableCell>
                        <TableCell>{inst.product_version}</TableCell>
                        <TableCell>{formatDateTime(inst.activated_at)}</TableCell>
                        <TableCell>
                          {inst.released_at ? (
                            <Badge variant="outline" className="text-muted-foreground">
                              Direset {formatDateTime(inst.released_at)}
                            </Badge>
                          ) : (
                            <Badge className="border-transparent bg-emerald-500/15 text-emerald-700 dark:text-emerald-400">
                              Terikat
                            </Badge>
                          )}
                        </TableCell>
                        <TableCell className="text-right">
                          <Button
                            size="sm"
                            variant="outline"
                            disabled={!!inst.released_at}
                            onClick={() => setPendingReset(inst)}
                          >
                            Reset
                          </Button>
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
            ) : (
              <div className="rounded-2xl border border-dashed p-8 text-center text-sm text-muted-foreground">
                Belum ada installation — customer belum mengaktivasi.
              </div>
            )}
          </div>
        </>
      ) : null}

      <AlertDialog open={renewing} onOpenChange={(open) => !open && setRenewing(false)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Perpanjang license</AlertDialogTitle>
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
            <AlertDialogTitle>Suspend license?</AlertDialogTitle>
            <AlertDialogDescription>
              Instalasi customer akan langsung diblokir begitu validasi berikutnya berhasil
              (paling lambat 24 jam). Bisa diaktifkan lagi kapan saja.
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
            <AlertDialogTitle>Cabut license?</AlertDialogTitle>
            <AlertDialogDescription>
              Tidak bisa dibatalkan — buat license baru kalau customer ini perlu diaktifkan lagi.
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

      <AlertDialog open={pendingReset !== null} onOpenChange={(open) => !open && setPendingReset(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Reset installation?</AlertDialogTitle>
            <AlertDialogDescription>
              Dipakai saat customer migrasi VPS — installation lama dilepas, membuka kuota
              supaya installation baru bisa diaktivasi dengan key yang sama.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={busy}>Batal</AlertDialogCancel>
            <AlertDialogAction onClick={onReset} disabled={busy}>
              Reset
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
