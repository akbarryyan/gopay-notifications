import GopayListener, { type AppVariant } from '../../modules/gopay-listener'

/**
 * Domain backend. Diedit sekali di sini saja — tidak tersebar ke berkas lain.
 */
const DOMAIN = 'GANTI-DOMAIN.com'

/**
 * Nilai awal Backend URL per varian.
 *
 * Ditanam per varian supaya URL produksi tidak pernah perlu diketik di
 * aplikasi UAT — itu persis kesalahan yang paling mahal di sistem pembayaran.
 * Tetap dapat diubah di layar Pengaturan.
 *
 * Development sengaja kosong: alamat LAN laptop berubah tiap ganti jaringan,
 * jadi menanamnya justru menyesatkan.
 */
const DEFAULT_BACKEND_URL: Record<AppVariant, string> = {
  development: '',
  uat: `https://uat.${DOMAIN}/api/v1`,
  production: `https://${DOMAIN}/api/v1`,
}

export const LABEL_VARIAN: Record<AppVariant, string> = {
  development: 'DEVELOPMENT',
  uat: 'UAT',
  production: 'PRODUCTION',
}

export const WARNA_VARIAN: Record<AppVariant, string> = {
  development: '#7c3aed',
  uat: '#ea580c',
  production: '#111827',
}

export function environment() {
  return GopayListener.getEnvironment()
}

/**
 * Mengisi Backend URL dengan nilai awal varian, HANYA bila masih kosong.
 *
 * Tidak pernah menimpa yang sudah diisi user — kalau ia sengaja mengarahkan
 * UAT ke backend lain, itu keputusannya.
 */
export function seedBackendUrl(): void {
  const { variant } = GopayListener.getEnvironment()
  const bawaan = DEFAULT_BACKEND_URL[variant]
  if (!bawaan) return

  if (GopayListener.getSettings().backendUrl === '') {
    GopayListener.saveSettings({ backendUrl: bawaan })
  }
}
