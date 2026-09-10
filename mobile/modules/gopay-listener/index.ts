import { requireNativeModule } from 'expo-modules-core'

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
}

export default requireNativeModule<GopayListenerModule>('GopayListener')
