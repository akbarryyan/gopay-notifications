"use client";

import { use, useCallback, useState } from "react";
import Link from "next/link";
import { Check, Copy, Plus, RotateCw } from "lucide-react";
import { Button, buttonVariants } from "@/components/ui/button";
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
import { createLicense, getCustomerDetail, type LicensePlan } from "@/lib/api";
import { useApiData } from "@/lib/use-api-data";
import { formatDateOnly } from "@/lib/format";

const STATUS_BADGE: Record<string, string> = {
  active: "border-transparent bg-emerald-500/15 text-emerald-700 dark:text-emerald-400",
  expiring: "border-transparent bg-amber-500/15 text-amber-700 dark:text-amber-400",
  expired: "border-transparent bg-red-500/15 text-red-700 dark:text-red-400",
  suspended: "border-transparent bg-red-500/15 text-red-700 dark:text-red-400",
  revoked: "border-transparent bg-red-500/15 text-red-700 dark:text-red-400",
};

export default function CustomerDetailPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = use(params);
  const fetcher = useCallback(() => getCustomerDetail(id), [id]);
  const { data, loading, error, reload } = useApiData(fetcher);

  const [creating, setCreating] = useState(false);
  const [plan, setPlan] = useState<LicensePlan>("Business");
  const [expiresAt, setExpiresAt] = useState("");
  const [busy, setBusy] = useState(false);
  const [createdKey, setCreatedKey] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);

  async function onCreate() {
    if (!expiresAt) return;
    setBusy(true);
    try {
      const res = await createLicense(id, plan, expiresAt);
      setCreatedKey(res.license.key);
      setCreating(false);
      setExpiresAt("");
      reload();
    } catch {
      toast.error("Gagal membuat license.");
    } finally {
      setBusy(false);
    }
  }

  async function copyKey() {
    if (!createdKey) return;
    try {
      await navigator.clipboard.writeText(createdKey);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      toast.error("Gagal menyalin — salin manual dari kotak di atas.");
    }
  }

  return (
    <div className="flex flex-col gap-6">
      <div className="flex items-start justify-between gap-4">
        <div>
          <Link href="/" className="text-sm text-muted-foreground hover:underline">
            ← Customers
          </Link>
          <h1 className="mt-1 text-2xl font-semibold">{data?.customer.name ?? "..."}</h1>
        </div>
        <Button size="sm" onClick={() => setCreating(true)}>
          <Plus className="mr-1.5 size-4" />
          Buat license
        </Button>
      </div>

      {error && (
        <Alert variant="destructive">
          <AlertTitle>Tidak dapat memuat customer</AlertTitle>
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
      ) : data && data.licenses.length > 0 ? (
        <div className="overflow-x-auto rounded-xl border border-border/60 shadow-sm">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Plan</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Berakhir</TableHead>
                <TableHead>Sisa hari</TableHead>
                <TableHead className="text-right">Detail</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {data.licenses.map((l) => (
                <TableRow key={l.id}>
                  <TableCell className="font-medium">{l.plan}</TableCell>
                  <TableCell>
                    <Badge className={STATUS_BADGE[l.status]}>{l.status}</Badge>
                  </TableCell>
                  <TableCell>{formatDateOnly(l.expires_at)}</TableCell>
                  <TableCell>{l.days_remaining}</TableCell>
                  <TableCell className="text-right">
                    <Link
                      href={`/licenses/${l.id}`}
                      className={buttonVariants({ size: "sm", variant: "outline" })}
                    >
                      Kelola
                    </Link>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      ) : (
        <div className="rounded-2xl border border-dashed p-8 text-center">
          <p className="font-medium">Belum ada license untuk customer ini</p>
        </div>
      )}

      <AlertDialog open={creating} onOpenChange={(open) => !open && setCreating(false)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Buat license baru</AlertDialogTitle>
            <AlertDialogDescription>
              Key mentah cuma ditampilkan sekali setelah ini — salin dan kirim ke customer
              di luar sistem.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <div className="flex flex-col gap-4 py-2">
            <div className="flex flex-col gap-2">
              <Label htmlFor="plan">Plan</Label>
              <select
                id="plan"
                value={plan}
                onChange={(e) => setPlan(e.target.value as LicensePlan)}
                className="h-8 rounded-lg border border-border bg-background px-2.5 text-sm"
              >
                <option value="Starter">Starter (3 device)</option>
                <option value="Business">Business (10 device)</option>
                <option value="Enterprise">Enterprise (unlimited)</option>
              </select>
            </div>
            <div className="flex flex-col gap-2">
              <Label htmlFor="expires-at">Berakhir</Label>
              <Input
                id="expires-at"
                type="date"
                value={expiresAt}
                onChange={(e) => setExpiresAt(e.target.value)}
              />
            </div>
          </div>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={busy}>Batal</AlertDialogCancel>
            <AlertDialogAction onClick={onCreate} disabled={busy || !expiresAt}>
              Buat
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <AlertDialog open={createdKey !== null} onOpenChange={(open) => !open && setCreatedKey(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>License dibuat</AlertDialogTitle>
            <AlertDialogDescription>
              Salin sekarang — key ini <strong>tidak akan ditampilkan lagi</strong> setelah
              jendela ini ditutup.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <div className="flex items-center gap-2 py-2">
            <code className="min-w-0 flex-1 overflow-x-auto rounded-lg border bg-muted px-3 py-2 font-mono text-xs whitespace-nowrap">
              {createdKey}
            </code>
            <Button type="button" variant="outline" size="icon" onClick={copyKey}>
              {copied ? <Check className="size-4" /> : <Copy className="size-4" />}
            </Button>
          </div>
          <AlertDialogFooter>
            <AlertDialogAction onClick={() => setCreatedKey(null)}>
              Sudah disalin, tutup
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
