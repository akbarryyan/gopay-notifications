"use client";

import Link from "next/link";
import { Smartphone, Bell, Server, RotateCw } from "lucide-react";
import { StatCard } from "@/components/dashboard/stat-card";
import { Button } from "@/components/ui/button";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Skeleton } from "@/components/ui/skeleton";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { getEvents, getOverview } from "@/lib/api";
import { useApiData } from "@/lib/use-api-data";
import { formatDateTime, formatRupiah, timeAgo } from "@/lib/format";

export default function OverviewPage() {
  const overview = useApiData(getOverview);
  const recentEvents = useApiData(() => getEvents(5, 0));

  if (overview.loading) {
    return (
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {Array.from({ length: 4 }).map((_, i) => (
          <Skeleton key={i} className="h-28" />
        ))}
      </div>
    );
  }

  if (overview.error || !overview.data) {
    return (
      <Alert variant="destructive">
        <AlertTitle>Tidak dapat memuat Overview</AlertTitle>
        <AlertDescription className="flex items-center justify-between gap-4">
          <span>{overview.error}</span>
          <Button size="sm" variant="outline" onClick={overview.reload}>
            <RotateCw className="mr-1.5 size-3.5" />
            Coba lagi
          </Button>
        </AlertDescription>
      </Alert>
    );
  }

  const { devices, events, system } = overview.data;

  return (
    <div className="flex flex-col gap-6">
      <div>
        <h1 className="text-2xl font-semibold">Overview</h1>
        <p className="text-sm text-muted-foreground">
          Kondisi sistem saat ini, diperbarui setiap halaman ini dimuat.
        </p>
      </div>

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <StatCard
          title="Devices"
          icon={<Smartphone className="size-4 text-muted-foreground" />}
          value={`${devices.online + devices.pending} / ${devices.total}`}
          subtitle={`${devices.online} online · ${devices.offline} offline · ${devices.disabled} nonaktif`}
        />
        <StatCard
          title="Events hari ini"
          icon={<Bell className="size-4 text-muted-foreground" />}
          value={events.today}
          subtitle={`${events.last_7_days} dalam 7 hari terakhir`}
        />
        <StatCard
          title="Event terakhir"
          value={events.latest_at ? timeAgo(events.latest_at) : "Belum ada"}
          subtitle={events.latest_at ? formatDateTime(events.latest_at) : undefined}
        />
        <StatCard
          title="Backend"
          icon={<Server className="size-4 text-muted-foreground" />}
          value={system.backend === "operational" ? "Operational" : system.backend}
          subtitle={`Database: ${system.database}`}
        />
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">Event terbaru</CardTitle>
        </CardHeader>
        <CardContent>
          {recentEvents.loading ? (
            <div className="flex flex-col gap-2">
              {Array.from({ length: 3 }).map((_, i) => (
                <Skeleton key={i} className="h-10" />
              ))}
            </div>
          ) : recentEvents.error ? (
            <p className="text-sm text-destructive">{recentEvents.error}</p>
          ) : recentEvents.data && recentEvents.data.length > 0 ? (
            <ul className="divide-y">
              {recentEvents.data.map((e) => (
                <li key={e.event_id} className="flex items-center justify-between gap-4 py-2.5">
                  <div className="min-w-0">
                    <p className="truncate text-sm font-medium">{e.title ?? "(tanpa judul)"}</p>
                    <p className="text-xs text-muted-foreground">
                      {e.device_id} · {formatDateTime(e.received_at)}
                    </p>
                  </div>
                  <span className="shrink-0 text-sm font-medium">
                    {formatRupiah(e.amount_hint)}
                  </span>
                </li>
              ))}
            </ul>
          ) : (
            <p className="text-sm text-muted-foreground">
              Belum ada event. Ia akan muncul di sini begitu perangkat menerima pembayaran.
            </p>
          )}
        </CardContent>
      </Card>

      <div className="text-right">
        <Link href="/events" className="text-sm text-primary underline-offset-4 hover:underline">
          Lihat semua event →
        </Link>
      </div>
    </div>
  );
}
