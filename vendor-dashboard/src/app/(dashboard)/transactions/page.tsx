"use client";

import { useCallback, useEffect, useState } from "react";
import Link from "next/link";
import { Download, RotateCw, Search } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
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
import toast from "react-hot-toast";
import { FilterDropdown } from "@/components/dashboard/filter-dropdown";
import { DateRangeFilter } from "@/components/dashboard/date-range-filter";
import { getTransactions, type InvoiceStatus, type VendorInvoice } from "@/lib/api";
import { useApiData } from "@/lib/use-api-data";
import { formatDateTime, formatRupiah } from "@/lib/format";
import { downloadCsv, toCsv, type CsvColumn } from "@/lib/csv";

const PAGE_SIZE = 50;
// Ekspor CSV mengikuti filter yang aktif, bukan cuma satu halaman yang
// sedang tampil -- diambil ulang khusus saat tombol diklik, terpisah dari
// data paginasi di layar.
//
// Backend membatasi `limit` maksimal 1000 per permintaan (sama seperti
// endpoint invoice customer), jadi ekspor mengambil beberapa halaman
// berturut-turut, bukan satu permintaan besar -- menaikkan batas di server
// cuma memindahkan masalahnya ke query yang lebih berat.
const EXPORT_PAGE_SIZE = 1000;
// Pagar pengaman supaya tombol ekspor tidak pernah menggantung browser
// kalau datanya sudah sangat banyak: berhenti di 10 halaman (10.000 baris)
// dan beri tahu bahwa hasilnya terpotong.
const EXPORT_MAX_PAGES = 10;

const STATUS_OPTIONS: { value: InvoiceStatus; label: string }[] = [
  { value: "PENDING", label: "Pending" },
  { value: "PAID", label: "Paid" },
  { value: "EXPIRED", label: "Expired" },
];

const TRANSACTION_CSV_COLUMNS: CsvColumn<VendorInvoice>[] = [
  { label: "Business Name", value: (i) => i.business_name },
  { label: "External Ref", value: (i) => i.external_ref },
  { label: "Requested Amount", value: (i) => i.requested_amount },
  { label: "Unique Amount", value: (i) => i.unique_amount },
  { label: "Status", value: (i) => i.status },
  { label: "Created At", value: (i) => i.created_at },
  { label: "Paid At", value: (i) => i.paid_at ?? "" },
  { label: "Expires At", value: (i) => i.expires_at },
];

function StatusBadge({ status }: { status: InvoiceStatus }) {
  if (status === "PAID") {
    return (
      <Badge className="border-transparent bg-emerald-500/15 text-emerald-700 dark:text-emerald-400">
        Paid
      </Badge>
    );
  }
  if (status === "EXPIRED") {
    return (
      <Badge variant="outline" className="text-muted-foreground">
        Expired
      </Badge>
    );
  }
  return (
    <Badge className="border-transparent bg-amber-500/15 text-amber-700 dark:text-amber-400">
      Pending
    </Badge>
  );
}

export default function TransactionsPage() {
  const [offset, setOffset] = useState(0);
  const [rawQuery, setRawQuery] = useState("");
  const [query, setQuery] = useState("");
  const [status, setStatus] = useState("");
  const [dateRange, setDateRange] = useState({ from: "", to: "" });
  const [exporting, setExporting] = useState(false);

  useEffect(() => {
    const t = setTimeout(() => setQuery(rawQuery), 350);
    return () => clearTimeout(t);
  }, [rawQuery]);

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect
    setOffset(0);
  }, [query, status, dateRange.from, dateRange.to]);

  const fetcher = useCallback(
    () =>
      getTransactions(PAGE_SIZE, offset, {
        q: query || undefined,
        status: (status || undefined) as InvoiceStatus | undefined,
        from: dateRange.from || undefined,
        to: dateRange.to || undefined,
      }),
    [offset, query, status, dateRange.from, dateRange.to],
  );
  const { data, loading, error, reload } = useApiData(fetcher);
  const hasFilter = query.trim() !== "" || status !== "" || dateRange.from !== "" || dateRange.to !== "";

  async function onExport() {
    setExporting(true);
    try {
      const activeFilter = {
        q: query || undefined,
        status: (status || undefined) as InvoiceStatus | undefined,
        from: dateRange.from || undefined,
        to: dateRange.to || undefined,
      };

      const rows: VendorInvoice[] = [];
      let truncated = false;
      for (let page = 0; page < EXPORT_MAX_PAGES; page++) {
        const batch = await getTransactions(EXPORT_PAGE_SIZE, page * EXPORT_PAGE_SIZE, activeFilter);
        rows.push(...batch);
        if (batch.length < EXPORT_PAGE_SIZE) break;
        if (page === EXPORT_MAX_PAGES - 1) truncated = true;
      }

      if (rows.length === 0) {
        toast.error("Tidak ada transaksi yang cocok dengan filter ini.");
        return;
      }

      downloadCsv(
        `transactions-${new Date().toISOString().slice(0, 10)}.csv`,
        toCsv(rows, TRANSACTION_CSV_COLUMNS),
      );
      toast.success(
        truncated
          ? `${rows.length} transaksi diekspor (dipotong di batas ${rows.length} baris — persempit filternya untuk sisanya).`
          : `${rows.length} transaksi diekspor.`,
      );
    } catch {
      toast.error("Gagal mengekspor transaksi.");
    } finally {
      setExporting(false);
    }
  }

  return (
    <div className="flex flex-col gap-6">
      <div className="flex items-start justify-between gap-4">
        <div>
          <h1 className="text-2xl font-semibold">Transactions</h1>
          <p className="text-sm text-muted-foreground">Invoice lintas semua account, terbaru dulu.</p>
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
            placeholder="Cari nama bisnis atau referensi order..."
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
        <DateRangeFilter from={dateRange.from} to={dateRange.to} onChange={setDateRange} />
      </div>

      {error && (
        <Alert variant="destructive">
          <AlertTitle>Tidak dapat memuat transactions</AlertTitle>
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
                  <TableHead>Business</TableHead>
                  <TableHead>Referensi</TableHead>
                  <TableHead className="text-right">Nominal diminta</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Dibuat</TableHead>
                  <TableHead>Dibayar / Kedaluwarsa</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {data.map((inv) => (
                  <TableRow key={inv.id}>
                    <TableCell>
                      <Link href={`/accounts/${inv.account_id}`} className="font-medium hover:underline">
                        {inv.business_name}
                      </Link>
                    </TableCell>
                    <TableCell className="font-mono text-xs">{inv.external_ref}</TableCell>
                    <TableCell className="text-right">{formatRupiah(inv.requested_amount)}</TableCell>
                    <TableCell>
                      <StatusBadge status={inv.status} />
                    </TableCell>
                    <TableCell className="whitespace-nowrap">{formatDateTime(inv.created_at)}</TableCell>
                    <TableCell className="whitespace-nowrap">
                      {inv.paid_at ? formatDateTime(inv.paid_at) : formatDateTime(inv.expires_at)}
                    </TableCell>
                  </TableRow>
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
            {hasFilter ? "Tidak ada transaksi yang cocok dengan filter ini" : "Belum ada transaksi"}
          </p>
          <p className="mt-1 text-sm text-muted-foreground">
            {hasFilter
              ? "Coba ubah atau bersihkan filter di atas."
              : "Transaksi akan muncul di sini begitu ada customer yang membuat invoice."}
          </p>
        </div>
      )}
    </div>
  );
}
