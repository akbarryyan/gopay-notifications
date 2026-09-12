/**
 * Format tampilan, sengaja konsisten dengan mobile/src/lib/format.ts —
 * dashboard dan aplikasi Android menampilkan nominal dan waktu dengan cara
 * yang sama, supaya operator tidak melihat dua bahasa berbeda untuk hal
 * yang sama.
 */

export function formatRupiah(amount: number | null | undefined): string {
  if (amount === null || amount === undefined) return "—";
  return "Rp" + amount.toLocaleString("id-ID");
}

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

/**
 * "12 Sep" dari tanggal murni "YYYY-MM-DD" (tanpa jam/zona) — dipakai di
 * label grafik tren. timeZone: "UTC" sengaja dipaksa: tanpa itu, Date
 * mem-parse "YYYY-MM-DD" sebagai tengah malam UTC lalu toLocaleDateString
 * menampilkannya di zona waktu lokal pembaca, yang bisa mundur satu hari
 * bagi siapa pun di sebelah barat UTC.
 */
export function formatShortDate(dateOnly: string): string {
  return new Date(dateOnly + "T00:00:00Z").toLocaleDateString("id-ID", {
    day: "2-digit",
    month: "short",
    timeZone: "UTC",
  });
}

/**
 * "Baru saja", "5 menit lalu", dst.
 *
 * Dipakai di kartu Devices untuk memperlihatkan device yang diam-diam
 * berhenti mengirim heartbeat — angka relatif jauh lebih cepat terbaca
 * daripada membandingkan dua stempel waktu absolut.
 */
export function timeAgo(iso: string | null | undefined, nowMs = Date.now()): string {
  if (!iso) return "belum pernah";
  const then = new Date(iso).getTime();
  const detik = Math.max(0, Math.floor((nowMs - then) / 1000));

  if (detik < 60) return "baru saja";
  if (detik < 3600) return `${Math.floor(detik / 60)} menit lalu`;
  if (detik < 86400) return `${Math.floor(detik / 3600)} jam lalu`;
  return `${Math.floor(detik / 86400)} hari lalu`;
}
