import { Badge } from "@/components/ui/badge";
import type { DeviceStatus } from "@/lib/api";

/**
 * Status tidak hanya dibedakan lewat warna — teksnya sendiri sudah cukup
 * bagi pembaca yang buta warna atau memakai pembaca layar.
 */
const STYLES: Record<DeviceStatus, string> = {
  ONLINE: "bg-emerald-100 text-emerald-800 dark:bg-emerald-900/40 dark:text-emerald-300",
  OFFLINE: "bg-red-100 text-red-800 dark:bg-red-900/40 dark:text-red-300",
  PENDING: "bg-amber-100 text-amber-800 dark:bg-amber-900/40 dark:text-amber-300",
  DISABLED: "bg-muted text-muted-foreground",
};

const LABELS: Record<DeviceStatus, string> = {
  ONLINE: "Online",
  OFFLINE: "Offline",
  PENDING: "Menunggu heartbeat pertama",
  DISABLED: "Dinonaktifkan",
};

export function DeviceStatusBadge({ status }: { status: DeviceStatus }) {
  return (
    <Badge variant="secondary" className={STYLES[status]} title={LABELS[status]}>
      <span
        className={
          "mr-1.5 inline-block size-1.5 rounded-full " +
          (status === "ONLINE"
            ? "bg-emerald-500"
            : status === "OFFLINE"
              ? "bg-red-500"
              : status === "PENDING"
                ? "bg-amber-500"
                : "bg-muted-foreground")
        }
      />
      {status}
    </Badge>
  );
}
