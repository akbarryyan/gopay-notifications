"use client";

import { useCallback, useEffect, useState } from "react";
import { AlertTriangle, Check, RotateCw, Search } from "lucide-react";
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
import { DateRangeFilter } from "@/components/dashboard/date-range-filter";
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
  dismissException,
  getExceptions,
  getInvoices,
  matchException,
  type AdminEvent,
  type AdminInvoice,
} from "@/lib/api";
import { useApiData } from "@/lib/use-api-data";
import { formatDateTime, formatRupiah } from "@/lib/format";

function InvoiceStatusBadge({ status }: { status: AdminInvoice["status"] }) {
  if (status === "EXPIRED") {
    return (
      <Badge className="border-transparent bg-amber-500/15 text-amber-700 dark:text-amber-400">
        Expired
      </Badge>
    );
  }
  return (
    <Badge variant="outline" className="text-muted-foreground">
      Pending
    </Badge>
  );
}

function MatchDialog({
  event,
  onClose,
  onMatched,
}: {
  event: AdminEvent;
  onClose: () => void;
  onMatched: () => void;
}) {
  const [rawQuery, setRawQuery] = useState("");
  const [query, setQuery] = useState("");
  const [candidates, setCandidates] = useState<AdminInvoice[]>([]);
  const [loading, setLoading] = useState(true);
  const [selected, setSelected] = useState<AdminInvoice | null>(null);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    const t = setTimeout(() => setQuery(rawQuery), 300);
    return () => clearTimeout(t);
  }, [rawQuery]);

  useEffect(() => {
    let cancelled = false;
    // Pola fetch-saat-deps-berubah yang sama dengan use-api-data.ts —
    // tidak ada Suspense/RSC di sini, jadi ini cara paling sederhana yang
    // benar untuk memuat kandidat invoice tiap kali pencarian berubah.
    // eslint-disable-next-line react-hooks/set-state-in-effect
    setLoading(true);
    getInvoices(20, 0, { status: ["PENDING", "EXPIRED"], q: query || undefined })
      .then((list) => {
        if (!cancelled) setCandidates(list);
      })
      .catch(() => {
        if (!cancelled) toast.error("Gagal memuat daftar invoice.");
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [query]);

  async function onConfirm() {
    if (!selected) return;
    setBusy(true);
    try {
      await matchException(event.event_id, selected.id);
      toast.success(`Event dicocokkan ke invoice "${selected.external_ref}".`);
      onMatched();
    } catch {
      toast.error("Gagal mencocokkan — invoice mungkin sudah PAID atau event sudah dipakai.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <AlertDialog open onOpenChange={(open) => !open && onClose()}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Cocokkan ke invoice</AlertDialogTitle>
          <AlertDialogDescription>
            Cari invoice yang seharusnya dibayar oleh event ini ({formatRupiah(event.amount_hint)}
            ).
          </AlertDialogDescription>
        </AlertDialogHeader>

        <div className="flex items-center gap-2 rounded-lg border bg-background px-3 py-2">
          <Search className="size-4 shrink-0 text-muted-foreground" />
          <input
            value={rawQuery}
            onChange={(e) => setRawQuery(e.target.value)}
            placeholder="Cari referensi order..."
            autoFocus
            className="w-full bg-transparent text-sm outline-none placeholder:text-muted-foreground"
          />
        </div>

        <div className="max-h-64 overflow-y-auto rounded-lg border">
          {loading ? (
            <div className="flex flex-col gap-2 p-3">
              {Array.from({ length: 3 }).map((_, i) => (
                <Skeleton key={i} className="h-10" />
              ))}
            </div>
          ) : candidates.length > 0 ? (
            <div className="flex flex-col divide-y">
              {candidates.map((inv) => (
                <button
                  key={inv.id}
                  type="button"
                  onClick={() => setSelected(inv)}
                  className={`flex items-center justify-between gap-3 px-3 py-2 text-left text-sm transition-colors hover:bg-secondary ${
                    selected?.id === inv.id ? "bg-primary/10" : ""
                  }`}
                >
                  <div className="min-w-0">
                    <div className="truncate font-medium">{inv.external_ref}</div>
                    <div className="text-xs text-muted-foreground">
                      {formatRupiah(inv.unique_amount)}
                    </div>
                  </div>
                  <div className="flex shrink-0 items-center gap-2">
                    <InvoiceStatusBadge status={inv.status} />
                    {selected?.id === inv.id && <Check className="size-4 text-primary" />}
                  </div>
                </button>
              ))}
            </div>
          ) : (
            <p className="p-4 text-center text-sm text-muted-foreground">
              Tidak ada invoice PENDING/EXPIRED yang cocok pencarian.
            </p>
          )}
        </div>

        <AlertDialogFooter>
          <AlertDialogCancel disabled={busy}>Batal</AlertDialogCancel>
          <AlertDialogAction onClick={onConfirm} disabled={busy || !selected}>
            Cocokkan
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}

function DismissDialog({
  event,
  onClose,
  onDismissed,
}: {
  event: AdminEvent;
  onClose: () => void;
  onDismissed: () => void;
}) {
  const [note, setNote] = useState("");
  const [busy, setBusy] = useState(false);

  async function onConfirm() {
    setBusy(true);
    try {
      await dismissException(event.event_id, note.trim() || undefined);
      toast.success("Event diabaikan.");
      onDismissed();
    } catch {
      toast.error("Gagal mengabaikan event.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <AlertDialog open onOpenChange={(open) => !open && onClose()}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Abaikan event ini?</AlertDialogTitle>
          <AlertDialogDescription>
            Event ini tidak akan muncul lagi di daftar. Tidak bisa dibatalkan — pastikan memang
            bukan pembayaran untuk order manapun.
          </AlertDialogDescription>
        </AlertDialogHeader>
        <div className="flex flex-col gap-2 py-2">
          <Label htmlFor="dismiss-note">Catatan (opsional)</Label>
          <Input
            id="dismiss-note"
            value={note}
            onChange={(e) => setNote(e.target.value)}
            placeholder="mis. transfer pribadi, bukan order"
            autoFocus
          />
        </div>
        <AlertDialogFooter>
          <AlertDialogCancel disabled={busy}>Batal</AlertDialogCancel>
          <AlertDialogAction onClick={onConfirm} disabled={busy}>
            Abaikan
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}

export default function ExceptionsPage() {
  const [rawQuery, setRawQuery] = useState("");
  const [query, setQuery] = useState("");
  const [dateRange, setDateRange] = useState({ from: "", to: "" });
  const [matchingEvent, setMatchingEvent] = useState<AdminEvent | null>(null);
  const [dismissingEvent, setDismissingEvent] = useState<AdminEvent | null>(null);

  useEffect(() => {
    const t = setTimeout(() => setQuery(rawQuery), 350);
    return () => clearTimeout(t);
  }, [rawQuery]);

  const fetcher = useCallback(
    () =>
      getExceptions(50, 0, {
        q: query || undefined,
        from: dateRange.from || undefined,
        to: dateRange.to || undefined,
      }),
    [query, dateRange.from, dateRange.to],
  );
  const { data, loading, error, reload } = useApiData(fetcher);

  return (
    <div className="flex flex-col gap-6">
      <div>
        <h1 className="text-2xl font-semibold">Exceptions</h1>
        <p className="text-sm text-muted-foreground">
          Pembayaran yang masuk tapi belum jelas untuk order yang mana — cocokkan manual atau
          abaikan.
        </p>
      </div>

      <div className="flex flex-col gap-2 sm:flex-row sm:flex-wrap sm:items-center">
        <div className="flex h-9 min-w-0 flex-1 items-center gap-2 rounded-lg border bg-background px-3 sm:max-w-xs">
          <Search className="size-4 shrink-0 text-muted-foreground" />
          <input
            value={rawQuery}
            onChange={(e) => setRawQuery(e.target.value)}
            placeholder="Cari device atau judul..."
            className="w-full min-w-0 bg-transparent text-sm outline-none placeholder:text-muted-foreground"
          />
        </div>
        <DateRangeFilter from={dateRange.from} to={dateRange.to} onChange={setDateRange} />
      </div>

      {error && (
        <Alert variant="destructive">
          <AlertTitle>Tidak dapat memuat exceptions</AlertTitle>
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
          {Array.from({ length: 4 }).map((_, i) => (
            <Skeleton key={i} className="h-12" />
          ))}
        </div>
      ) : data && data.length > 0 ? (
        <div className="overflow-x-auto rounded-xl border border-border/60 shadow-sm">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Waktu</TableHead>
                <TableHead>Device</TableHead>
                <TableHead>Judul</TableHead>
                <TableHead className="text-right">Nominal</TableHead>
                <TableHead className="text-right">Aksi</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {data.map((e) => (
                <TableRow key={e.event_id}>
                  <TableCell className="whitespace-nowrap">
                    {formatDateTime(e.received_at)}
                  </TableCell>
                  <TableCell className="font-mono text-xs">{e.device_id}</TableCell>
                  <TableCell className="max-w-60 truncate">{e.title ?? "(tanpa judul)"}</TableCell>
                  <TableCell className="text-right font-medium">
                    {formatRupiah(e.amount_hint)}
                  </TableCell>
                  <TableCell className="text-right">
                    <div className="flex justify-end gap-1.5">
                      <Button size="sm" variant="outline" onClick={() => setMatchingEvent(e)}>
                        Cocokkan
                      </Button>
                      <Button size="sm" variant="outline" onClick={() => setDismissingEvent(e)}>
                        Abaikan
                      </Button>
                    </div>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      ) : (
        <div className="rounded-2xl border border-dashed p-8 text-center">
          <AlertTriangle className="mx-auto mb-2 size-8 text-muted-foreground" />
          <p className="font-medium">Tidak ada pembayaran yang perlu direkonsiliasi</p>
          <p className="mt-1 text-sm text-muted-foreground">
            Setiap event yang masuk sudah cocok dengan invoice-nya masing-masing.
          </p>
        </div>
      )}

      {matchingEvent && (
        <MatchDialog
          event={matchingEvent}
          onClose={() => setMatchingEvent(null)}
          onMatched={() => {
            setMatchingEvent(null);
            reload();
          }}
        />
      )}
      {dismissingEvent && (
        <DismissDialog
          event={dismissingEvent}
          onClose={() => setDismissingEvent(null)}
          onDismissed={() => {
            setDismissingEvent(null);
            reload();
          }}
        />
      )}
    </div>
  );
}
