/**
 * Format tampilan, subset dari dashboard/src/lib/format.ts (cuma yang
 * dipakai vendor console) — konsisten supaya Akbar tidak melihat dua
 * bahasa tanggal berbeda antar kedua dashboard.
 */

export function formatDateTime(iso: string | null | undefined): string {
  if (!iso) return "—";
  return new Date(iso).toLocaleString("id-ID", {
    day: "2-digit",
    month: "short",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

export function formatDateOnly(dateOnly: string | null | undefined): string {
  if (!dateOnly) return "—";
  return new Date(dateOnly + "T00:00:00Z").toLocaleDateString("id-ID", {
    day: "2-digit",
    month: "short",
    year: "numeric",
    timeZone: "UTC",
  });
}
