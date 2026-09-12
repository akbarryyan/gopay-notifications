"use client";

import { useCallback } from "react";
import Link from "next/link";
import { Smartphone, Bell, Server, Clock, RotateCw } from "lucide-react";
import { StatCard } from "@/components/dashboard/stat-card";
import { Button } from "@/components/ui/button";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Skeleton } from "@/components/ui/skeleton";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { EventsTrendChart } from "@/components/dashboard/events-trend-chart";
import { getEvents, getOverview } from "@/lib/api";
import { useApiData } from "@/lib/use-api-data";
import { formatDateTime, formatRupiah, timeAgo } from "@/lib/format";

export default function OverviewPage() {
  const overview = useApiData(getOverview);
  const getRecentEvents = useCallback(() => getEvents(5, 0), []);
  const recentEvents = useApiData(getRecentEvents);

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
  // Judul dan copy di kartu ini sengaja tidak menyebut "backend"/"database" —
  // dashboard ini dipakai customer pemilik instalasi, bukan vendor, dan
  // istilah arsitektur internal bukan urusan mereka. Field system.backend /
  // system.database dari API tetap dipakai sebagai sumber data, hanya
  // labelnya yang diringkas jadi satu status layanan.
  const systemHealthy = system.backend === "operational" && system.database === "operational";

  return (
    <div className="flex flex-col gap-6">
      <div>
        <h1 className="text-2xl font-valley font-semibold">Dashboard</h1>
        <p className="text-sm font-sans text-muted-foreground">
          Kondisi sistem saat ini, diperbarui setiap halaman ini dimuat.
        </p>
      </div>

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <StatCard
          title="Devices"
          icon={<Smartphone className="size-4" />}
          value={`${devices.online + devices.pending} / ${devices.total}`}
          subtitle={`${devices.online} online · ${devices.offline} offline · ${devices.disabled} nonaktif`}
        />
        <StatCard
          title="Events hari ini"
          icon={<Bell className="size-4" />}
          value={events.today}
          subtitle={`${events.last_7_days} dalam 7 hari terakhir`}
        />
        <StatCard
          title="Event terakhir"
          icon={<Clock className="size-4" />}
          value={events.latest_at ? timeAgo(events.latest_at) : "Belum ada"}
          subtitle={events.latest_at ? formatDateTime(events.latest_at) : undefined}
        />
        <StatCard
          title="Status Layanan"
          icon={<Server className="size-4" />}
          value={systemHealthy ? "Aktif" : "Gangguan"}
          subtitle={
            systemHealthy ? "Semua layanan berjalan normal" : "Sebagian layanan mengalami gangguan"
          }
        />
      </div>

      <Card className="gap-0 rounded-2xl border-none py-0 shadow-sm ring-1 ring-border/60">
        <CardHeader className="border-b py-4">
          <CardTitle className="text-base">Tren event</CardTitle>
          <p className="text-xs text-muted-foreground">14 hari terakhir</p>
        </CardHeader>
        <CardContent className="pt-4 pb-2">
          <EventsTrendChart data={events.daily} />
        </CardContent>
      </Card>

      <Card className="gap-0 rounded-2xl border-none py-0 shadow-sm ring-1 ring-border/60">
        <CardHeader className="border-b py-4">
          <CardTitle className="text-base">Event terbaru</CardTitle>
        </CardHeader>
        <CardContent className="p-2">
          {recentEvents.loading ? (
            <div className="flex flex-col gap-2 p-3">
              {Array.from({ length: 3 }).map((_, i) => (
                <Skeleton key={i} className="h-10" />
              ))}
            </div>
          ) : recentEvents.error ? (
            <p className="p-3 text-sm text-destructive">{recentEvents.error}</p>
          ) : recentEvents.data && recentEvents.data.length > 0 ? (
            <ul className="divide-y">
              {recentEvents.data.map((e) => (
                <li
                  key={e.event_id}
                  className="flex items-center justify-between gap-4 rounded-xl px-3 py-2.5 transition-colors hover:bg-secondary/60"
                >
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
            <p className="p-3 text-sm text-muted-foreground">
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
