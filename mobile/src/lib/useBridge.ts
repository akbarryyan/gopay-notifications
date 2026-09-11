import { useCallback, useEffect, useState } from 'react'
import { AppState } from 'react-native'
import { useFocusEffect } from '@react-navigation/native'
import GopayListener, {
  type BackendHealth,
  type BridgeStatus,
} from '../../modules/gopay-listener'

/**
 * Menyegarkan status pada empat kejadian, dan tidak pernah polling:
 *
 *  1. layar pertama dibuat
 *  2. tab ini kembali mendapat fokus
 *  3. aplikasi kembali dari latar belakang
 *  4. native melaporkan ada yang berubah — event ditangkap atau status berubah
 *
 * Nomor 2 dan 4 keduanya diperlukan. Tanpa 4, Dashboard yang sedang terbuka
 * membeku menampilkan PENDING sementara event sudah SENT. Tanpa 2, berpindah
 * tab tidak memuat ulang, karena bottom tab menjaga layar tetap hidup.
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

  // Membaca ulang dari database itu murah; status backend tidak ikut diminta
  // di sini supaya berpindah tab tidak memicu request jaringan tiap kali.
  useFocusEffect(
    useCallback(() => {
      setStatus(GopayListener.getStatus())
    }, []),
  )

  useEffect(() => {
    refresh()

    const changed = GopayListener.addListener('onBridgeChanged', () => {
      setStatus(GopayListener.getStatus())
    })
    const appState = AppState.addEventListener('change', (s) => {
      if (s === 'active') refresh()
    })

    return () => {
      changed.remove()
      appState.remove()
    }
  }, [refresh])

  return { status, backend, refresh }
}
