"use client";

import { Fragment, useCallback, useState } from "react";
import { RotateCw, ChevronDown, ChevronRight } from "lucide-react";
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
import { getEvents } from "@/lib/api";
import { useApiData } from "@/lib/use-api-data";
import { formatDateTime, formatRupiah } from "@/lib/format";

const PAGE_SIZE = 50;

export default function EventsPage() {
  const [offset, setOffset] = useState(0);
  const [expanded, setExpanded] = useState<string | null>(null);
  const fetcher = useCallback(() => getEvents(PAGE_SIZE, offset), [offset]);
  const { data, loading, error, reload } = useApiData(fetcher);

  return (
    <div className="flex flex-col gap-6">
      <div>
        <h1 className="text-2xl font-semibold">Events</h1>
        <p className="text-sm text-muted-foreground">
          Notifikasi pembayaran mentah yang diterima dari perangkat Android, apa adanya.
        </p>
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
          <div className="overflow-x-auto rounded-md border">
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
        <div className="rounded-md border border-dashed p-8 text-center">
          <p className="font-medium">Belum ada event</p>
          <p className="mt-1 text-sm text-muted-foreground">
            Event akan muncul di sini begitu perangkat menerima notifikasi pembayaran.
          </p>
        </div>
      )}
    </div>
  );
}
