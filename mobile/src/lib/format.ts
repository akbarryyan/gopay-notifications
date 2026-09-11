export function formatRupiah(amount: number | null | undefined): string {
  if (amount === null || amount === undefined) return '—'
  return 'Rp' + amount.toLocaleString('id-ID')
}

export function formatWaktu(ms: number): string {
  return new Date(ms).toLocaleString('id-ID', {
    day: '2-digit',
    month: 'short',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}

/**
 * Dipakai Dashboard untuk memperlihatkan service yang mati diam-diam.
 *
 * Tanpa ini, ColorOS yang membunuh listener terlihat sama seperti "memang
 * belum ada pembayaran" — dan itu perbedaan yang mahal.
 */
export function sejakKapan(ms: number, nowMs = Date.now()): string {
  const detik = Math.max(0, Math.floor((nowMs - ms) / 1000))
  if (detik < 60) return 'baru saja'
  if (detik < 3600) return `${Math.floor(detik / 60)} menit lalu`
  if (detik < 86400) return `${Math.floor(detik / 3600)} jam lalu`
  return `${Math.floor(detik / 86400)} hari lalu`
}
