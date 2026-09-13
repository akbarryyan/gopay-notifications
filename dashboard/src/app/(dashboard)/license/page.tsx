"use client";

import { AlertTriangle, RotateCw, ShieldCheck, ShieldOff, ShieldQuestion } from "lucide-react";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { getLicense, type LicenseInfo, type LicenseStatus } from "@/lib/api";
import { useApiData } from "@/lib/use-api-data";
import { formatDateOnly } from "@/lib/format";

// Sinkron dengan licensecheck.WarningThresholdDays di backend — halaman ini
// satu-satunya tempat yang perlu tahu angka ini.
const WARNING_THRESHOLD_DAYS = 30;

const STATUS_LABEL: Record<LicenseStatus, string> = {
  active: "Aktif",
  expired: "Kedaluwarsa",
  invalid: "Tidak valid",
  missing: "Belum terpasang",
};

function StatusBadge({ status, daysRemaining }: { status: LicenseStatus; daysRemaining?: number }) {
  if (status === "active" && daysRemaining !== undefined && daysRemaining <= WARNING_THRESHOLD_DAYS) {
    return (
      <Badge className="border-transparent bg-amber-500/15 text-amber-700 dark:text-amber-400">
        Akan berakhir
      </Badge>
    );
  }
  if (status === "active") {
    return (
      <Badge className="border-transparent bg-emerald-500/15 text-emerald-700 dark:text-emerald-400">
        {STATUS_LABEL[status]}
      </Badge>
    );
  }
  if (status === "missing") {
    return (
      <Badge variant="outline" className="text-muted-foreground">
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
  switch (status) {
    case "active":
      return <ShieldCheck className="size-8 text-emerald-600 dark:text-emerald-400" />;
    case "missing":
      return <ShieldQuestion className="size-8 text-muted-foreground" />;
    default:
      return <ShieldOff className="size-8 text-red-600 dark:text-red-400" />;
  }
}

function InactiveBanner({ license }: { license: LicenseInfo }) {
  const explanation: Record<Exclude<LicenseStatus, "active">, string> = {
    missing:
      "Belum ada lisensi terpasang untuk instalasi ini. Pengiriman pembayaran, invoice, dan sebagian besar halaman lain di dashboard ini tidak akan berfungsi sampai lisensi terpasang.",
    expired:
      "Lisensi instalasi ini sudah kedaluwarsa. Pengiriman pembayaran, invoice, dan sebagian besar halaman lain di dashboard ini sedang tidak berfungsi sampai lisensi diperpanjang.",
    invalid:
      "Lisensi instalasi ini tidak valid. Pengiriman pembayaran, invoice, dan sebagian besar halaman lain di dashboard ini sedang tidak berfungsi sampai lisensi yang benar terpasang.",
  };

  if (license.status === "active") return null;

  return (
    <Alert variant="destructive">
      <AlertTriangle className="size-4" />
      <AlertTitle>{STATUS_LABEL[license.status]}</AlertTitle>
      <AlertDescription>
        {explanation[license.status]} Hubungi penyedia layanan kamu untuk memperpanjang atau
        memperbaiki lisensi ini.
      </AlertDescription>
    </Alert>
  );
}

function ExpiringWarning({ license }: { license: LicenseInfo }) {
  if (license.status !== "active" || license.days_remaining === undefined) return null;
  if (license.days_remaining > WARNING_THRESHOLD_DAYS) return null;

  return (
    <Alert>
      <AlertTriangle className="size-4" />
      <AlertTitle>Lisensi akan berakhir</AlertTitle>
      <AlertDescription>
        Lisensi akan berakhir dalam {Math.max(license.days_remaining, 0)} hari. Hubungi penyedia
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
        <p className="text-sm text-muted-foreground">Status lisensi instalasi ini.</p>
      </div>

      {error && (
        <Alert variant="destructive">
          <AlertTitle>Tidak dapat memuat status lisensi</AlertTitle>
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
                <StatusBadge status={data.status} daysRemaining={data.days_remaining} />
                {data.plan && <p className="text-lg font-semibold">{data.plan}</p>}
              </div>
            </div>

            {(data.status === "active" || data.status === "expired") && (
              <div className="mt-6 border-t border-border/60 pt-2">
                <DetailRow label="Customer" value={data.customer ?? "—"} />
                <DetailRow label="Domain" value={data.domain ?? "—"} />
                <DetailRow
                  label="Diaktifkan"
                  value={data.issued_at ? formatDateOnly(data.issued_at) : "—"}
                />
                <DetailRow
                  label="Berakhir"
                  value={data.expires_at ? formatDateOnly(data.expires_at) : "—"}
                />
                {data.days_remaining !== undefined && (
                  <DetailRow
                    label="Sisa waktu"
                    value={
                      data.days_remaining >= 0
                        ? `${data.days_remaining} hari`
                        : `Lewat ${Math.abs(data.days_remaining)} hari`
                    }
                  />
                )}
              </div>
            )}
          </div>
        </>
      ) : null}
    </div>
  );
}
