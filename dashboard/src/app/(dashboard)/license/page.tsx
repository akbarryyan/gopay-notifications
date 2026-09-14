"use client";

import Link from "next/link";
import { AlertTriangle, RotateCw, ShieldCheck, ShieldOff } from "lucide-react";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { getLicense, type LicenseInfo, type LicenseStatus } from "@/lib/api";
import { useApiData } from "@/lib/use-api-data";
import { formatDateOnly } from "@/lib/format";

const STATUS_LABEL: Record<LicenseStatus, string> = {
  active: "Aktif",
  expiring: "Akan berakhir",
  expired: "Kedaluwarsa",
  suspended: "Disuspend",
  revoked: "Dicabut",
};

// Status yang masih dianggap operasional (endpoint lain tetap jalan) --
// sinkron dengan store.Account.Operational() di backend. Type predicate
// (bukan cuma boolean) supaya TypeScript ikut menyempitkan tipe
// license.status di pemanggil setelah early-return.
function isOperational(status: LicenseStatus): status is "active" | "expiring" {
  return status === "active" || status === "expiring";
}

function StatusBadge({ status }: { status: LicenseStatus }) {
  if (status === "active") {
    return (
      <Badge className="border-transparent bg-emerald-500/15 text-emerald-700 dark:text-emerald-400">
        {STATUS_LABEL[status]}
      </Badge>
    );
  }
  if (status === "expiring") {
    return (
      <Badge className="border-transparent bg-amber-500/15 text-amber-700 dark:text-amber-400">
        {STATUS_LABEL[status]}
      </Badge>
    );
  }
  return (
    <Badge className="border-transparent bg-red-500/15 text-red-700 dark:text-red-400">
      {STATUS_LABEL[status]}
    </Badge>
  );
}

function StatusIcon({ status }: { status: LicenseStatus }) {
  if (isOperational(status)) {
    return <ShieldCheck className="size-8 text-emerald-600 dark:text-emerald-400" />;
  }
  return <ShieldOff className="size-8 text-red-600 dark:text-red-400" />;
}

const INACTIVE_EXPLANATION: Record<Exclude<LicenseStatus, "active" | "expiring">, string> = {
  expired:
    "Akun ini sudah kedaluwarsa. Pengiriman pembayaran, invoice, dan sebagian besar halaman lain di dashboard ini sedang tidak berfungsi sampai akun diperpanjang.",
  suspended:
    "Akun ini sedang disuspend. Pengiriman pembayaran, invoice, dan sebagian besar halaman lain di dashboard ini sedang tidak berfungsi.",
  revoked:
    "Akun ini sudah dicabut. Pengiriman pembayaran, invoice, dan sebagian besar halaman lain di dashboard ini tidak akan berfungsi lagi.",
};

function InactiveBanner({ license }: { license: LicenseInfo }) {
  if (isOperational(license.status)) return null;

  return (
    <Alert variant="destructive">
      <AlertTriangle className="size-4" />
      <AlertTitle>{STATUS_LABEL[license.status]}</AlertTitle>
      <AlertDescription>
        {INACTIVE_EXPLANATION[license.status]} Hubungi penyedia layanan kamu untuk memperbaiki akun
        ini.
      </AlertDescription>
    </Alert>
  );
}

function ExpiringWarning({ license }: { license: LicenseInfo }) {
  if (license.status !== "expiring") return null;

  return (
    <Alert>
      <AlertTriangle className="size-4" />
      <AlertTitle>Akun akan berakhir</AlertTitle>
      <AlertDescription>
        Akun akan berakhir dalam {Math.max(license.days_remaining, 0)} hari. Hubungi penyedia
        layanan kamu untuk memperpanjang sebelum tanggal itu.
      </AlertDescription>
    </Alert>
  );
}

function DetailRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-center justify-between border-b border-border/50 py-3 last:border-0">
      <span className="text-sm text-muted-foreground">{label}</span>
      <span className="text-sm font-medium">{value}</span>
    </div>
  );
}

export default function LicensePage() {
  const { data, loading, error, reload } = useApiData(getLicense);

  return (
    <div className="flex flex-col gap-6">
      <div>
        <h1 className="text-2xl font-semibold">License</h1>
        <p className="text-sm text-muted-foreground">Status akun instalasi ini.</p>
      </div>

      {error && (
        <Alert variant="destructive">
          <AlertTitle>Tidak dapat memuat status akun</AlertTitle>
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
        <Skeleton className="h-48 rounded-2xl" />
      ) : data ? (
        <>
          <InactiveBanner license={data} />
          <ExpiringWarning license={data} />

          <div className="rounded-2xl border border-border/60 p-6 shadow-sm">
            <div className="flex items-center gap-4">
              <StatusIcon status={data.status} />
              <div className="flex flex-col gap-1">
                <StatusBadge status={data.status} />
                <p className="text-lg font-semibold">{data.plan}</p>
              </div>
            </div>

            <div className="mt-6 border-t border-border/60 pt-2">
              <DetailRow label="Plan" value={data.plan} />
              <DetailRow
                label="Max devices"
                value={data.max_devices < 0 ? "Unlimited" : String(data.max_devices)}
              />
              <DetailRow label="Berakhir" value={formatDateOnly(data.expires_at)} />
              <DetailRow
                label="Sisa waktu"
                value={
                  data.days_remaining >= 0
                    ? `${data.days_remaining} hari`
                    : `Lewat ${Math.abs(data.days_remaining)} hari`
                }
              />
            </div>
          </div>

          <p className="text-sm text-muted-foreground">
            Pengingat masa aktif dikirim ke{" "}
            <span className="font-medium text-foreground">{data.email}</span>. Ubah email atau
            tambahkan Telegram di{" "}
            <Link
              href="/settings"
              className="font-medium text-foreground underline underline-offset-4"
            >
              Settings
            </Link>
            .
          </p>
        </>
      ) : null}
    </div>
  );
}
