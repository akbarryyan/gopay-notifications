"use client";

import { Fragment, useState } from "react";
import { Check, ChevronDown, ChevronRight, Copy, Plus, RotateCw, Webhook as WebhookIcon } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Checkbox } from "@/components/ui/checkbox";
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
  createWebhook,
  deleteWebhook,
  getWebhookDeliveries,
  getWebhooks,
  setWebhookEnabled,
  testWebhook,
  type AdminWebhook,
  type WebhookDelivery,
  type WebhookEvent,
} from "@/lib/api";
import { useApiData } from "@/lib/use-api-data";
import { formatDateTime } from "@/lib/format";

const EVENT_OPTIONS: { value: WebhookEvent; label: string }[] = [
  { value: "invoice.paid", label: "invoice.paid" },
  { value: "invoice.expired", label: "invoice.expired" },
];

function DeliveryStatusBadge({ status }: { status: WebhookDelivery["status"] }) {
  if (status === "DELIVERED") {
    return (
      <Badge className="border-transparent bg-emerald-500/15 text-emerald-700 dark:text-emerald-400">
        Delivered
      </Badge>
    );
  }
  if (status === "FAILED") {
    return (
      <Badge className="border-transparent bg-red-500/15 text-red-700 dark:text-red-400">
        Failed
      </Badge>
    );
  }
  if (status === "RETRYING") {
    return (
      <Badge className="border-transparent bg-amber-500/15 text-amber-700 dark:text-amber-400">
        Retrying
      </Badge>
    );
  }
  return (
    <Badge variant="outline" className="text-muted-foreground">
      Pending
    </Badge>
  );
}

export default function WebhooksPage() {
  const { data, loading, error, reload } = useApiData(getWebhooks);

  const [creating, setCreating] = useState(false);
  const [newName, setNewName] = useState("");
  const [newURL, setNewURL] = useState("");
  const [newEvents, setNewEvents] = useState<WebhookEvent[]>(["invoice.paid", "invoice.expired"]);
  const [busy, setBusy] = useState(false);
  const [createdSecret, setCreatedSecret] = useState<{ name: string; secret: string } | null>(null);
  const [copied, setCopied] = useState(false);
  const [pendingDelete, setPendingDelete] = useState<AdminWebhook | null>(null);
  const [pendingToggle, setPendingToggle] = useState<AdminWebhook | null>(null);
  const [testingId, setTestingId] = useState<string | null>(null);

  const [expanded, setExpanded] = useState<string | null>(null);
  const [deliveries, setDeliveries] = useState<Record<string, WebhookDelivery[]>>({});
  const [deliveriesLoading, setDeliveriesLoading] = useState<string | null>(null);

  function toggleEvent(event: WebhookEvent) {
    setNewEvents((prev) =>
      prev.includes(event) ? prev.filter((e) => e !== event) : [...prev, event],
    );
  }

  async function onCreate() {
    if (!newName.trim() || !newURL.trim() || newEvents.length === 0) return;
    setBusy(true);
    try {
      const created = await createWebhook(newName.trim(), newURL.trim(), newEvents);
      setCreatedSecret({ name: created.name, secret: created.secret });
      setCreating(false);
      setNewName("");
      setNewURL("");
      setNewEvents(["invoice.paid", "invoice.expired"]);
      reload();
    } catch {
      toast.error("Gagal membuat webhook.");
    } finally {
      setBusy(false);
    }
  }

  async function onToggle() {
    if (!pendingToggle) return;
    setBusy(true);
    try {
      await setWebhookEnabled(pendingToggle.id, !pendingToggle.enabled);
      toast.success(
        pendingToggle.enabled ? `${pendingToggle.name} dinonaktifkan.` : `${pendingToggle.name} diaktifkan.`,
      );
      reload();
    } catch {
      toast.error("Gagal mengubah status webhook.");
    } finally {
      setBusy(false);
      setPendingToggle(null);
    }
  }

  async function onDelete() {
    if (!pendingDelete) return;
    setBusy(true);
    try {
      await deleteWebhook(pendingDelete.id);
      toast.success(`${pendingDelete.name} dihapus.`);
      reload();
    } catch {
      toast.error("Gagal menghapus webhook.");
    } finally {
      setBusy(false);
      setPendingDelete(null);
    }
  }

  async function onTest(w: AdminWebhook) {
    setTestingId(w.id);
    try {
      const result = await testWebhook(w.id);
      if (result.delivered) {
        toast.success(`Berhasil — HTTP ${result.http_status}, ${result.duration_ms}ms`);
      } else {
        toast.error(`Gagal — HTTP ${result.http_status || "tidak terhubung"}, ${result.duration_ms}ms`);
      }
    } catch {
      toast.error("Gagal mengirim test webhook.");
    } finally {
      setTestingId(null);
    }
  }

  async function onExpand(w: AdminWebhook) {
    if (expanded === w.id) {
      setExpanded(null);
      return;
    }
    setExpanded(w.id);
    if (!deliveries[w.id]) {
      setDeliveriesLoading(w.id);
      try {
        const list = await getWebhookDeliveries(w.id);
        setDeliveries((prev) => ({ ...prev, [w.id]: list }));
      } catch {
        toast.error("Gagal memuat riwayat pengiriman.");
      } finally {
        setDeliveriesLoading(null);
      }
    }
  }

  async function copySecret() {
    if (!createdSecret) return;
    try {
      await navigator.clipboard.writeText(createdSecret.secret);
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
          <h1 className="text-2xl font-semibold">Webhooks</h1>
          <p className="text-sm text-muted-foreground">
            Dikirim otomatis saat invoice berubah status (dibayar atau kedaluwarsa).
          </p>
        </div>
        <Button size="sm" onClick={() => setCreating(true)}>
          <Plus className="mr-1.5 size-4" />
          Buat webhook baru
        </Button>
      </div>

      {error && (
        <Alert variant="destructive">
          <AlertTitle>Tidak dapat memuat webhooks</AlertTitle>
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
        <div className="overflow-x-auto rounded-xl border border-border/60 shadow-sm">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-8" />
                <TableHead>Nama</TableHead>
                <TableHead>URL</TableHead>
                <TableHead>Events</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Percobaan terakhir</TableHead>
                <TableHead className="text-right">Aksi</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {data.map((w) => (
                <Fragment key={w.id}>
                  <TableRow className="cursor-pointer" onClick={() => onExpand(w)}>
                    <TableCell>
                      {expanded === w.id ? (
                        <ChevronDown className="size-4 text-muted-foreground" />
                      ) : (
                        <ChevronRight className="size-4 text-muted-foreground" />
                      )}
                    </TableCell>
                    <TableCell className="font-medium">{w.name}</TableCell>
                    <TableCell className="max-w-60 truncate font-mono text-xs">{w.url}</TableCell>
                    <TableCell>
                      <div className="flex flex-wrap gap-1">
                        {w.events.map((e) => (
                          <Badge key={e} variant="outline" className="font-mono text-[10px]">
                            {e}
                          </Badge>
                        ))}
                      </div>
                    </TableCell>
                    <TableCell>
                      {w.enabled ? (
                        <Badge className="border-transparent bg-emerald-500/15 text-emerald-700 dark:text-emerald-400">
                          Aktif
                        </Badge>
                      ) : (
                        <Badge variant="outline" className="text-muted-foreground">
                          Nonaktif
                        </Badge>
                      )}
                    </TableCell>
                    <TableCell className="whitespace-nowrap text-xs text-muted-foreground">
                      {w.last_delivery_at
                        ? `${formatDateTime(w.last_delivery_at)} · ${w.last_delivery_status}`
                        : "Belum pernah"}
                    </TableCell>
                    <TableCell className="text-right" onClick={(e) => e.stopPropagation()}>
                      <div className="flex justify-end gap-1.5">
                        <Button
                          size="sm"
                          variant="outline"
                          disabled={testingId === w.id}
                          onClick={() => onTest(w)}
                        >
                          Test
                        </Button>
                        <Button size="sm" variant="outline" onClick={() => setPendingToggle(w)}>
                          {w.enabled ? "Nonaktifkan" : "Aktifkan"}
                        </Button>
                        <Button
                          size="sm"
                          variant="outline"
                          className="text-destructive hover:text-destructive"
                          onClick={() => setPendingDelete(w)}
                        >
                          Hapus
                        </Button>
                      </div>
                    </TableCell>
                  </TableRow>
                  {expanded === w.id && (
                    <TableRow>
                      <TableCell colSpan={7} className="bg-muted/30">
                        {deliveriesLoading === w.id ? (
                          <div className="flex flex-col gap-2 py-2">
                            {Array.from({ length: 2 }).map((_, i) => (
                              <Skeleton key={i} className="h-8" />
                            ))}
                          </div>
                        ) : deliveries[w.id] && deliveries[w.id].length > 0 ? (
                          <div className="flex flex-col gap-1 py-2 text-sm">
                            {deliveries[w.id].map((d) => (
                              <div key={d.id} className="flex items-center justify-between gap-4">
                                <span className="font-mono text-xs">{d.event}</span>
                                <span className="text-xs text-muted-foreground">
                                  {formatDateTime(d.created_at)}
                                </span>
                                <DeliveryStatusBadge status={d.status} />
                                <span className="text-xs text-muted-foreground">
                                  {d.http_status ?? "—"} · {d.duration_ms ?? "—"}ms
                                </span>
                              </div>
                            ))}
                          </div>
                        ) : (
                          <p className="py-2 text-sm text-muted-foreground">
                            Belum ada percobaan pengiriman.
                          </p>
                        )}
                      </TableCell>
                    </TableRow>
                  )}
                </Fragment>
              ))}
            </TableBody>
          </Table>
        </div>
      ) : (
        <div className="rounded-2xl border border-dashed p-8 text-center">
          <WebhookIcon className="mx-auto mb-2 size-8 text-muted-foreground" />
          <p className="font-medium">Belum ada webhook</p>
          <p className="mt-1 text-sm text-muted-foreground">
            Daftarkan endpoint untuk menerima notifikasi otomatis saat invoice dibayar atau
            kedaluwarsa.
          </p>
        </div>
      )}

      {/* Buat webhook baru */}
      <AlertDialog open={creating} onOpenChange={(open) => !open && setCreating(false)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Buat webhook baru</AlertDialogTitle>
            <AlertDialogDescription>
              Kami akan mengirim POST ke URL ini saat event yang dipilih terjadi.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <div className="flex flex-col gap-4 py-2">
            <div className="flex flex-col gap-2">
              <Label htmlFor="webhook-name">Nama</Label>
              <Input
                id="webhook-name"
                value={newName}
                onChange={(e) => setNewName(e.target.value)}
                placeholder="Production"
                autoFocus
              />
            </div>
            <div className="flex flex-col gap-2">
              <Label htmlFor="webhook-url">Endpoint URL</Label>
              <Input
                id="webhook-url"
                value={newURL}
                onChange={(e) => setNewURL(e.target.value)}
                placeholder="https://situsmu.com/api/webhook"
              />
            </div>
            <div className="flex flex-col gap-2">
              <Label>Events</Label>
              {EVENT_OPTIONS.map((opt) => (
                <label key={opt.value} className="flex items-center gap-2 text-sm">
                  <Checkbox
                    checked={newEvents.includes(opt.value)}
                    onCheckedChange={() => toggleEvent(opt.value)}
                  />
                  <span className="font-mono">{opt.label}</span>
                </label>
              ))}
            </div>
          </div>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={busy}>Batal</AlertDialogCancel>
            <AlertDialogAction
              onClick={onCreate}
              disabled={busy || !newName.trim() || !newURL.trim() || newEvents.length === 0}
            >
              Buat
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* Secret — tampil satu kali saja */}
      <AlertDialog open={createdSecret !== null} onOpenChange={(open) => !open && setCreatedSecret(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Webhook &ldquo;{createdSecret?.name}&rdquo; dibuat</AlertDialogTitle>
            <AlertDialogDescription>
              Salin secret ini sekarang — <strong>tidak akan ditampilkan lagi</strong> setelah
              jendela ini ditutup. Dipakai untuk memverifikasi header{" "}
              <code>X-Webhook-Signature</code>.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <div className="flex items-center gap-2 py-2">
            <code className="min-w-0 flex-1 overflow-x-auto rounded-lg border bg-muted px-3 py-2 font-mono text-xs whitespace-nowrap">
              {createdSecret?.secret}
            </code>
            <Button type="button" variant="outline" size="icon" onClick={copySecret}>
              {copied ? <Check className="size-4" /> : <Copy className="size-4" />}
            </Button>
          </div>
          <AlertDialogFooter>
            <AlertDialogAction onClick={() => setCreatedSecret(null)}>
              Sudah disalin, tutup
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* Konfirmasi aktif/nonaktif */}
      <AlertDialog open={pendingToggle !== null} onOpenChange={(open) => !open && setPendingToggle(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>
              {pendingToggle?.enabled ? "Nonaktifkan webhook?" : "Aktifkan webhook?"}
            </AlertDialogTitle>
            <AlertDialogDescription>
              {pendingToggle?.enabled
                ? `"${pendingToggle?.name}" akan berhenti menerima event baru sampai diaktifkan kembali.`
                : `"${pendingToggle?.name}" akan mulai menerima event lagi.`}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={busy}>Batal</AlertDialogCancel>
            <AlertDialogAction onClick={onToggle} disabled={busy}>
              {pendingToggle?.enabled ? "Nonaktifkan" : "Aktifkan"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* Konfirmasi hapus */}
      <AlertDialog open={pendingDelete !== null} onOpenChange={(open) => !open && setPendingDelete(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Hapus webhook?</AlertDialogTitle>
            <AlertDialogDescription>
              {`"${pendingDelete?.name}" beserta seluruh riwayat pengirimannya akan dihapus permanen. Tidak bisa dibatalkan.`}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={busy}>Batal</AlertDialogCancel>
            <AlertDialogAction onClick={onDelete} disabled={busy}>
              Hapus
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
