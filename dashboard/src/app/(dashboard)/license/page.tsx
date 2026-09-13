"use client";

import { AlertTriangle, RotateCw, ShieldAlert, ShieldCheck, ShieldOff, ShieldQuestion } from "lucide-react";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { getLicense, type LicenseInfo, type LicenseStatus } from "@/lib/api";
import { useApiData } from "@/lib/use-api-data";
import { formatDateOnly, formatDateTime } from "@/lib/format";

const STATUS_LABEL: Record<LicenseStatus, string> = {
  active: "Aktif",
  expiring: "Akan berakhir",
  expired: "Kedaluwarsa",
  suspended: "Disuspend",
  revoked: "Dicabut",
  missing: "Belum diaktivasi",
  invalid: "Tidak valid",
  unreachable: "Tidak terjangkau",
};

// Status yang masih dianggap operasional (endpoint lain tetap jalan) —
// sinkron dengan licensecheck.Status.Operational() di backend. Type
// predicate (bukan cuma boolean) supaya TypeScript ikut menyempitkan tipe
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
    case "expiring":
      return <ShieldCheck className="size-8 text-emerald-600 dark:text-emerald-400" />;
    case "missing":
      return <ShieldQuestion className="size-8 text-muted-foreground" />;
    case "unreachable":
      return <ShieldAlert className="size-8 text-red-600 dark:text-red-400" />;
    default:
      return <ShieldOff className="size-8 text-red-600 dark:text-red-400" />;
  }
}

const INACTIVE_EXPLANATION: Record<Exclude<LicenseStatus, "active" | "expiring">, string> = {
  missing:
    "Lisensi belum diaktivasi di instalasi ini. Isi LICENSE_KEY di berkas .env, lalu restart layanan untuk mengaktivasi.",
  expired:
    "Lisensi instalasi ini sudah kedaluwarsa. Pengiriman pembayaran, invoice, dan sebagian besar halaman lain di dashboard ini sedang tidak berfungsi sampai lisensi diperpanjang.",
  suspended:
    "Lisensi instalasi ini sedang disuspend. Pengiriman pembayaran, invoice, dan sebagian besar halaman lain di dashboard ini sedang tidak berfungsi.",
  revoked:
    "Lisensi instalasi ini sudah dicabut. Pengiriman pembayaran, invoice, dan sebagian besar halaman lain di dashboard ini tidak akan berfungsi lagi.",
  unreachable:
    "Instalasi ini sudah lama tidak berhasil menghubungi server lisensi. Periksa koneksi keluar (outbound HTTPS) dari server ini.",
  invalid:
    "Lisensi instalasi ini tidak valid. Pengiriman pembayaran, invoice, dan sebagian besar halaman lain di dashboard ini sedang tidak berfungsi.",
};

function InactiveBanner({ license }: { license: LicenseInfo }) {
  if (isOperational(license.status)) return null;

  return (
    <Alert variant="destructive">
      <AlertTriangle className="size-4" />
      <AlertTitle>{STATUS_LABEL[license.status]}</AlertTitle>
      <AlertDescription>
        {INACTIVE_EXPLANATION[license.status]}
        {license.status !== "missing" && " Hubungi penyedia layanan kamu untuk memperbaiki lisensi ini."}
      </AlertDescription>
    </Alert>
  );
}

function ExpiringWarning({ license }: { license: LicenseInfo }) {
  if (license.status !== "expiring" || license.days_remaining === undefined) return null;

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

// Detail cuma ditampilkan untuk status yang punya data sungguhan untuk
// ditunjukkan (signature & installation sudah lolos verifikasi) — sinkron
// dengan handleAdminLicense di backend.
function hasDetail(status: LicenseStatus): boolean {
  return status !== "missing" && status !== "invalid";
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
                <StatusBadge status={data.status} />
                {data.plan && <p className="text-lg font-semibold">{data.plan}</p>}
              </div>
            </div>

            {hasDetail(data.status) && (
              <div className="mt-6 border-t border-border/60 pt-2">
                <DetailRow label="Customer" value={data.customer ?? "—"} />
                <DetailRow label="Installation ID" value={data.installation_id ?? "—"} />
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
                <DetailRow
                  label="Validasi terakhir"
                  value={data.validated_at ? formatDateTime(data.validated_at) : "—"}
                />
              </div>
            )}
          </div>
        </>
      ) : null}
    </div>
  );
}
