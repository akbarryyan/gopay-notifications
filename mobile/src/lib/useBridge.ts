import { useCallback, useEffect, useState } from 'react'
import { AppState } from 'react-native'
import GopayListener, {
  type BackendHealth,
  type BridgeStatus,
} from '../../modules/gopay-listener'

/**
 * Menyegarkan status saat layar dibuka, saat aplikasi kembali ke depan, dan
 * saat native melaporkan notifikasi baru.
 *
 * Tidak ada polling: Notification Access dan status ikatan hanya berubah di
 * luar aplikasi, dan event baru dilaporkan native lewat CaptureBus.
 */
export function useBridgeStatus() {
  const [status, setStatus] = useState<BridgeStatus>(() => GopayListener.getStatus())
  const [backend, setBackend] = useState<BackendHealth | null>(null)

  const refresh = useCallback(() => {
    setStatus(GopayListener.getStatus())
    GopayListener.checkBackend()
      .then(setBackend)
      .catch(() => setBackend(null))
  }, [])

  useEffect(() => {
    refresh()

    const captured = GopayListener.addListener('onNotificationCaptured', () => {
      setStatus(GopayListener.getStatus())
    })
    const appState = AppState.addEventListener('change', (s) => {
      if (s === 'active') refresh()
    })

    return () => {
      captured.remove()
      appState.remove()
    }
  }, [refresh])

  return { status, backend, refresh }
}
