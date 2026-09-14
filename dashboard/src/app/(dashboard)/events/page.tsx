"use client";

import { Fragment, useCallback, useEffect, useState } from "react";
import { Download, RotateCw, ChevronDown, ChevronRight, Search } from "lucide-react";
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
import { FilterDropdown } from "@/components/dashboard/filter-dropdown";
import { DateRangeFilter } from "@/components/dashboard/date-range-filter";
import toast from "react-hot-toast";
import { getEvents, getSources, type AdminEvent } from "@/lib/api";
import { useApiData } from "@/lib/use-api-data";
import { formatDateTime, formatRupiah } from "@/lib/format";
import { downloadCsv, fetchAllPages, toCsv, type CsvColumn } from "@/lib/csv";

const PAGE_SIZE = 50;

// raw_payload sengaja TIDAK ikut diekspor: isinya JSON utuh per baris,
// yang membuat CSV-nya nyaris tidak bisa dibaca di spreadsheet. Payload
// mentah tetap bisa dilihat per event lewat baris yang diperluas.
const CSV_COLUMNS: CsvColumn<AdminEvent>[] = [
  { label: "Event ID", value: (e) => e.event_id },
  { label: "Device ID", value: (e) => e.device_id },
  { label: "Source", value: (e) => e.source },
  { label: "Title", value: (e) => e.title ?? "" },
  { label: "Text", value: (e) => e.text ?? "" },
  { label: "Amount Hint", value: (e) => e.amount_hint ?? "" },
  { label: "Posted At", value: (e) => e.posted_at },
  { label: "Received At", value: (e) => e.received_at },
];

export default function EventsPage() {
  const [offset, setOffset] = useState(0);
  const [expanded, setExpanded] = useState<string | null>(null);

  const [rawQuery, setRawQuery] = useState("");
  const [query, setQuery] = useState("");
  const [source, setSource] = useState("");
  const [dateRange, setDateRange] = useState({ from: "", to: "" });

  // Jeda 350ms sebelum mengetikan pencarian benar-benar memicu request baru
  // — tiap ketukan tombol tidak perlu langsung memanggil backend.
  useEffect(() => {
    const t = setTimeout(() => setQuery(rawQuery), 350);
    return () => clearTimeout(t);
  }, [rawQuery]);

  // Filter berubah → halaman kembali ke awal, supaya tidak nyasar di
  // offset yang sudah tidak relevan dengan hasil filter yang baru. Tidak ada
  // cara menyinkronkan ini selain di efek: offset harus ikut nilai filter
  // sebelumnya, bukan nilai yang baru saja berubah.
  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect
    setOffset(0);
  }, [query, source, dateRange.from, dateRange.to]);

  const sources = useApiData(getSources);
  const fetcher = useCallback(
    () =>
      getEvents(PAGE_SIZE, offset, {
        q: query || undefined,
        source: source || undefined,
        from: dateRange.from || undefined,
        to: dateRange.to || undefined,
      }),
    [offset, query, source, dateRange.from, dateRange.to],
  );
  const { data, loading, error, reload } = useApiData(fetcher);

  const sourceOptions = (sources.data ?? []).map((s) => ({ value: s.id, label: s.name }));
  const [exporting, setExporting] = useState(false);

  async function onExport() {
    setExporting(true);
    try {
      const activeFilter = {
        q: query || undefined,
        source: source || undefined,
        from: dateRange.from || undefined,
        to: dateRange.to || undefined,
      };
      const { rows, truncated } = await fetchAllPages((limit, offset) =>
        getEvents(limit, offset, activeFilter),
      );
      if (rows.length === 0) {
        toast.error("Tidak ada event yang cocok dengan filter ini.");
        return;
      }
      downloadCsv(`events-${new Date().toISOString().slice(0, 10)}.csv`, toCsv(rows, CSV_COLUMNS));
      toast.success(
        truncated
          ? `${rows.length} event diekspor (dipotong di batas — persempit filternya untuk sisanya).`
          : `${rows.length} event diekspor.`,
      );
    } catch {
      toast.error("Gagal mengekspor event.");
    } finally {
      setExporting(false);
    }
  }

  return (
    <div className="flex flex-col gap-6">
      <div className="flex items-start justify-between gap-4">
        <div>
          <h1 className="text-2xl font-semibold">Events</h1>
          <p className="text-sm text-muted-foreground">
            Notifikasi pembayaran mentah yang diterima dari perangkat Android, apa adanya.
          </p>
        </div>
        <Button size="sm" variant="outline" disabled={exporting} onClick={onExport}>
          <Download className="mr-1.5 size-4" />
          {exporting ? "Mengekspor..." : "Export CSV"}
        </Button>
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
        <FilterDropdown
          className="w-full sm:w-44"
          allLabel="Semua Sumber"
          value={source}
          options={sourceOptions}
          onChange={setSource}
          searchPlaceholder="Cari sumber..."
        />
        <DateRangeFilter
          from={dateRange.from}
          to={dateRange.to}
          onChange={setDateRange}
        />
      </div>

      {error && (
        <Alert variant="destructive">
          <AlertTitle>Tidak dapat memuat events</AlertTitle>
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
          {Array.from({ length: 6 }).map((_, i) => (
            <Skeleton key={i} className="h-12" />
          ))}
        </div>
      ) : data && data.length > 0 ? (
        <>
          <div className="overflow-x-auto rounded-xl border border-border/60 shadow-sm">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="w-8" />
                  <TableHead>Waktu</TableHead>
                  <TableHead>Sumber</TableHead>
                  <TableHead>Device</TableHead>
                  <TableHead>Judul</TableHead>
                  <TableHead className="text-right">Nominal</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {data.map((e) => (
                  <Fragment key={e.event_id}>
                    <TableRow
                      className="cursor-pointer"
                      onClick={() => setExpanded(expanded === e.event_id ? null : e.event_id)}
                    >
                      <TableCell>
                        {expanded === e.event_id ? (
                          <ChevronDown className="size-4 text-muted-foreground" />
                        ) : (
                          <ChevronRight className="size-4 text-muted-foreground" />
                        )}
                      </TableCell>
                      <TableCell className="whitespace-nowrap">
                        {formatDateTime(e.received_at)}
                      </TableCell>
                      <TableCell className="capitalize">{e.source}</TableCell>
                      <TableCell className="font-mono text-xs">{e.device_id}</TableCell>
                      <TableCell className="max-w-55 truncate">
                        {e.title ?? "(tanpa judul)"}
                      </TableCell>
                      <TableCell className="text-right font-medium">
                        {formatRupiah(e.amount_hint)}
                      </TableCell>
                    </TableRow>
                    {expanded === e.event_id && (
                      <TableRow>
                        <TableCell colSpan={6} className="bg-muted/30">
                          <dl className="grid grid-cols-1 gap-x-6 gap-y-2 py-2 text-sm sm:grid-cols-2">
                            <div>
                              <dt className="text-xs text-muted-foreground">Event ID</dt>
                              <dd className="font-mono text-xs">{e.event_id}</dd>
                            </div>
                            <div>
                              <dt className="text-xs text-muted-foreground">Package</dt>
                              <dd className="font-mono text-xs">{e.package_name}</dd>
                            </div>
                            <div className="sm:col-span-2">
                              <dt className="text-xs text-muted-foreground">Teks notifikasi</dt>
                              <dd>{e.text ?? "—"}</dd>
                            </div>
                          </dl>
                        </TableCell>
                      </TableRow>
                    )}
                  </Fragment>
                ))}
              </TableBody>
            </Table>
          </div>

          <div className="flex items-center justify-between">
            <Button
              variant="outline"
              size="sm"
              disabled={offset === 0}
              onClick={() => setOffset(Math.max(0, offset - PAGE_SIZE))}
            >
              Sebelumnya
            </Button>
            <span className="text-xs text-muted-foreground">
              Menampilkan {offset + 1}–{offset + data.length}
            </span>
            <Button
              variant="outline"
              size="sm"
              disabled={data.length < PAGE_SIZE}
              onClick={() => setOffset(offset + PAGE_SIZE)}
            >
              Berikutnya
            </Button>
          </div>
        </>
      ) : (
        <div className="rounded-2xl border border-dashed p-8 text-center">
          <p className="font-medium">
            {query || source || dateRange.from || dateRange.to
              ? "Tidak ada event yang cocok dengan filter ini"
              : "Belum ada event"}
          </p>
          <p className="mt-1 text-sm text-muted-foreground">
            {query || source || dateRange.from || dateRange.to
              ? "Coba ubah atau bersihkan filter di atas."
              : "Event akan muncul di sini begitu perangkat menerima notifikasi pembayaran."}
          </p>
        </div>
      )}
    </div>
  );
}
