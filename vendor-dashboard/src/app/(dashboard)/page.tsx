"use client";

import Link from "next/link";
import { AlertTriangle, UserPlus, Users, Wallet } from "lucide-react";
import { StatCard } from "@/components/dashboard/stat-card";
import { OverviewTrendChart } from "@/components/dashboard/overview-trend-chart";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Skeleton } from "@/components/ui/skeleton";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { getOverview } from "@/lib/api";
import { useApiData } from "@/lib/use-api-data";
import { formatDateTime, formatRupiah } from "@/lib/format";
import { STATUS_BADGE } from "@/lib/account-status";

export default function VendorDashboardPage() {
  const overview = useApiData(getOverview);

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
        <AlertTitle>Tidak dapat memuat Dashboard</AlertTitle>
        <AlertDescription className="flex items-center justify-between gap-4">
          <span>{overview.error}</span>
          <Button size="sm" variant="outline" onClick={overview.reload}>
            Coba lagi
          </Button>
        </AlertDescription>
      </Alert>
    );
  }

  const { accounts, revenue } = overview.data;
  // Fallback ke [] -- kalau backend yang sedang jalan belum di-restart sejak
  // field ini ditambahkan, respons lama tidak punya recent_accounts sama
  // sekali (bukan array kosong, tapi undefined), dan .length akan melempar.
  const recentAccounts = overview.data.recent_accounts ?? [];

  return (
    <div className="flex flex-col gap-6">
      <div>
        <h1 className="text-2xl font-semibold">Dashboard</h1>
        <p className="text-sm text-muted-foreground">Ringkasan seluruh account, lintas platform.</p>
      </div>

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <StatCard
          title="Total Accounts"
          icon={<Users className="size-4" />}
          value={accounts.total}
          subtitle={`${accounts.active} aktif · ${accounts.suspended + accounts.revoked} nonaktif`}
        />
        <StatCard
          title="Akan Berakhir"
          icon={<AlertTriangle className="size-4" />}
          value={accounts.expiring}
          subtitle="dalam 30 hari, perlu ditindaklanjuti"
        />
        <StatCard
          title="Account Baru"
          icon={<UserPlus className="size-4" />}
          value={accounts.new_this_week}
          subtitle="7 hari terakhir"
        />
        <StatCard
          title="Total Pendapatan"
          icon={<Wallet className="size-4" />}
          value={formatRupiah(revenue.total_paid_rp)}
          subtitle="seluruh account, sepanjang waktu"
        />
      </div>

      <Card className="gap-0 rounded-2xl border-none py-0 shadow-sm ring-1 ring-border/60">
        <CardHeader className="border-b py-4">
          <CardTitle className="text-base">Tren platform</CardTitle>
          <p className="text-xs text-muted-foreground">14 hari terakhir, lintas semua account</p>
        </CardHeader>
        <CardContent className="pt-4 pb-2">
          <OverviewTrendChart data={overview.data.daily} />
        </CardContent>
      </Card>
go run ./cmd/server

      <Card className="gap-0 rounded-2xl border-none py-0 shadow-sm ring-1 ring-border/60">
        <CardHeader className="border-b py-4">
          <CardTitle className="text-base">Customer Terbaru</CardTitle>
          <p className="text-xs text-muted-foreground">5 account yang paling baru dibuat</p>
        </CardHeader>
        <CardContent className="p-2">
          {recentAccounts.length === 0 ? (
            <p className="p-3 text-sm text-muted-foreground">Belum ada account.</p>
          ) : (
            <ul className="divide-y">
              {recentAccounts.map((acc) => (
                <li key={acc.id}>
                  <Link
                    href={`/accounts/${acc.id}`}
                    className="flex items-center justify-between gap-4 rounded-xl px-3 py-2.5 transition-colors hover:bg-secondary/60"
                  >
                    <div className="min-w-0">
                      <p className="truncate text-sm font-medium">{acc.business_name}</p>
                      <p className="text-xs text-muted-foreground">
                        {acc.plan} · dibuat {formatDateTime(acc.created_at)}
                      </p>
                    </div>
                    <Badge className={STATUS_BADGE[acc.status]}>{acc.status}</Badge>
                  </Link>
                </li>
              ))}
            </ul>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
