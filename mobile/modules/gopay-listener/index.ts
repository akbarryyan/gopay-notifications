import { requireNativeModule } from 'expo-modules-core'

export interface BridgeSettings {
  backendUrl: string
  deviceId: string
  /** Secret tidak pernah dibaca balik ke JS — hanya penandanya. */
  hasDeviceSecret: boolean
  monitoredPackages: string[]
  ignoreKeywords: string[]
  discoveryUntilMs: number
}

/** Field yang tidak disertakan berarti "jangan ubah", bukan "kosongkan". */
export interface SettingsPatch {
  backendUrl?: string
  deviceId?: string
  deviceSecret?: string
  monitoredPackages?: string[]
  ignoreKeywords?: string[]
}

export interface DiscoveryEntry {
  packageName: string
  title: string
  seenAt: number
}

export interface BackendHealth {
  configured: boolean
  reachable: boolean
  serverTime?: number | null
  /** Selisih jam server dikurangi jam HP, dalam detik. */
  skewSeconds?: number | null
  /** true bila selisihnya melebihi toleransi backend (±300 detik). */
  clockOutOfSync?: boolean
}

export interface ConnectionTest {
  ok: boolean
  deviceName?: string
  error?: string
}

interface GopayListenerModule {
  /** Apakah user sudah memberi Notification Access lewat Android Settings. */
  isNotificationAccessGranted(): boolean
  /** Membuka halaman Notification Access; izin ini tidak bisa diminta lewat dialog. */
  openNotificationAccessSettings(): void
  /**
   * Apakah sistem sedang benar-benar terikat ke service.
   * Bisa `false` walau izin aktif — itu gejala service dibunuh OEM.
   */
  isListenerConnected(): boolean

  getSettings(): BridgeSettings
  saveSettings(patch: SettingsPatch): void

  /** Dibatasi 1–10 menit; mode Discovery mati sendiri setelahnya. */
  startDiscovery(minutes: number): void
  stopDiscovery(): void
  getDiscoveryEntries(): DiscoveryEntry[]
  clearDiscovery(): void

  /** Status backend untuk Dashboard, sekaligus deteksi jam HP yang meleset. */
  checkBackend(): Promise<BackendHealth>
  /** Tombol Test Connection di Settings. */
  testConnection(): Promise<ConnectionTest>
}

export default requireNativeModule<GopayListenerModule>('GopayListener')
