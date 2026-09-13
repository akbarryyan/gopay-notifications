"use client";

import { RotateCw } from "lucide-react";
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
import { getAuditLog } from "@/lib/api";
import { useApiData } from "@/lib/use-api-data";
import { formatDateTime } from "@/lib/format";

export default function AuditLogPage() {
  const { data, loading, error, reload } = useApiData(getAuditLog);

  return (
    <div className="flex flex-col gap-6">
      <div>
        <h1 className="text-2xl font-semibold">Audit Log</h1>
        <p className="text-sm text-muted-foreground">
          Seluruh aksi vendor terhadap customer/license/installation, terbaru dulu.
        </p>
      </div>

      {error && (
        <Alert variant="destructive">
          <AlertTitle>Tidak dapat memuat audit log</AlertTitle>
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
          {Array.from({ length: 5 }).map((_, i) => (
            <Skeleton key={i} className="h-10" />
          ))}
        </div>
      ) : data && data.length > 0 ? (
        <div className="overflow-x-auto rounded-xl border border-border/60 shadow-sm">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Waktu</TableHead>
                <TableHead>Actor</TableHead>
                <TableHead>Action</TableHead>
                <TableHead>Resource</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {data.map((e) => (
                <TableRow key={e.id}>
                  <TableCell>{formatDateTime(e.created_at)}</TableCell>
                  <TableCell className="font-mono text-xs">{e.actor}</TableCell>
                  <TableCell className="font-medium">{e.action}</TableCell>
                  <TableCell className="font-mono text-xs">{e.resource}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      ) : (
        <div className="rounded-2xl border border-dashed p-8 text-center text-sm text-muted-foreground">
          Belum ada aktivitas.
        </div>
      )}
    </div>
  );
}
