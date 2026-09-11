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

export type AppVariant = 'development' | 'uat' | 'production'

export interface AppEnvironment {
  packageName: string
  /** Diturunkan dari package name yang benar-benar terpasang. */
  variant: AppVariant
}

export type EventStatus = 'PENDING' | 'SENDING' | 'SENT' | 'FAILED' | 'IGNORED'

export interface BridgeEvent {
  eventId: string
  packageName: string
  title: string | null
  text: string | null
  /** Display-only. Backend melakukan ekstraksi otoritatifnya sendiri. */
  amountHint: number | null
  postedAt: number
  receivedAt: number
  status: EventStatus
  attemptCount: number
  lastError: string | null
  backendStatus: string | null
  sentAt: number | null
}

export interface BridgeStatus {
  notificationAccessGranted: boolean
  /** Bisa false walau izin aktif — gejala service dibunuh OEM. */
  listenerConnected: boolean
  pendingCount: number
  sendingCount: number
  failedCount: number
  lastEvent: BridgeEvent | null
}

/**
 * Sinyal "ada yang berubah" untuk menyegarkan UI yang sedang terbuka.
 *
 * Dipancarkan saat event baru ditangkap maupun saat statusnya berubah setelah
 * pengiriman. Isinya sengaja tidak dijamin — penerima harus membaca ulang
 * lewat `getStatus()`, bukan bersandar pada payload ini.
 */
export interface BridgeChange {
  reason?: string
  eventId?: string
  status?: EventStatus
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
  getEnvironment(): AppEnvironment

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

  getStatus(): BridgeStatus
  getEvents(limit: number): BridgeEvent[]
  /** Mengantre ulang event FAILED secara manual. */
  retryEvent(eventId: string): void
  clearHistory(): void

  addListener(
    event: 'onBridgeChanged',
    handler: (e: BridgeChange) => void,
  ): { remove(): void }
}

export default requireNativeModule<GopayListenerModule>('GopayListener')
