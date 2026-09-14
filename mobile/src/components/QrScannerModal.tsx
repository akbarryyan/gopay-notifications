import { useEffect, useState } from 'react'
import { Modal, StyleSheet, Text, TouchableOpacity, View } from 'react-native'
import { CameraView, useCameraPermissions, type BarcodeScanningResult } from 'expo-camera'

/** Bentuk QR yang ditampilkan dashboard saat device baru dibuat. */
export interface PairingPayload {
  backendUrl: string
  deviceId: string
  deviceSecret: string
}

/**
 * QR-nya JSON: {"v":1,"backend_url":"...","device_id":"...","device_secret":"..."}.
 * Bukan token/kode kosong sama sekali -- QR ini cuma cara lain memindahkan
 * tiga nilai yang SAMA dengan yang sudah ditampilkan dashboard untuk disalin
 * manual (lihat SettingsScreen), jadi validasinya juga sama longgarnya:
 * ketiga field wajib ada dan berupa string tidak kosong.
 */
function parsePairingPayload(raw: string): PairingPayload | null {
  let parsed: unknown
  try {
    parsed = JSON.parse(raw)
  } catch {
    return null // bukan JSON -- QR lain, bukan salah pengguna, diamkan saja
  }
  if (typeof parsed !== 'object' || parsed === null) return null
  const p = parsed as Record<string, unknown>
  if (
    typeof p.backend_url !== 'string' ||
    typeof p.device_id !== 'string' ||
    typeof p.device_secret !== 'string' ||
    !p.backend_url ||
    !p.device_id ||
    !p.device_secret
  ) {
    return null
  }
  return { backendUrl: p.backend_url, deviceId: p.device_id, deviceSecret: p.device_secret }
}

export function QrScannerModal({
  visible,
  onClose,
  onScanned,
}: {
  visible: boolean
  onClose: () => void
  onScanned: (payload: PairingPayload) => void
}) {
  const [permission, requestPermission] = useCameraPermissions()
  // Kunci supaya satu QR yang sama tidak memicu onScanned berkali-kali
  // selama kameranya masih menghadap kode itu (onBarcodeScanned terus
  // menembak tiap frame sampai modal ditutup).
  const [handled, setHandled] = useState(false)

  useEffect(() => {
    if (visible) setHandled(false)
  }, [visible])

  function onBarcodeScanned(result: BarcodeScanningResult) {
    if (handled) return
    const payload = parsePairingPayload(result.data)
    if (!payload) return
    setHandled(true)
    onScanned(payload)
  }

  return (
    <Modal visible={visible} animationType="slide" onRequestClose={onClose}>
      <View style={styles.root}>
        {!permission ? (
          <View style={styles.center}>
            <Text style={styles.pesan}>Menyiapkan kamera...</Text>
          </View>
        ) : !permission.granted ? (
          <View style={styles.center}>
            <Text style={styles.pesan}>
              Izin kamera dibutuhkan untuk memindai QR pairing dari Dashboard.
            </Text>
            <TouchableOpacity style={styles.tombol} onPress={requestPermission}>
              <Text style={styles.tombolTeks}>Izinkan Kamera</Text>
            </TouchableOpacity>
          </View>
        ) : (
          <CameraView
            style={styles.kamera}
            facing="back"
            barcodeScannerSettings={{ barcodeTypes: ['qr'] }}
            onBarcodeScanned={onBarcodeScanned}
          >
            <View style={styles.overlay}>
              <View style={styles.bingkai} />
              <Text style={styles.petunjuk}>
                Arahkan ke QR di halaman Devices Dashboard
              </Text>
            </View>
          </CameraView>
        )}

        <TouchableOpacity style={styles.tombolBatal} onPress={onClose}>
          <Text style={styles.tombolBatalTeks}>Batal</Text>
        </TouchableOpacity>
      </View>
    </Modal>
  )
}

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: '#000' },
  center: { flex: 1, alignItems: 'center', justifyContent: 'center', padding: 24, gap: 16 },
  pesan: { color: '#fff', fontSize: 15, textAlign: 'center' },
  kamera: { flex: 1 },
  overlay: {
    flex: 1,
    alignItems: 'center',
    justifyContent: 'center',
    gap: 16,
    backgroundColor: 'rgba(0,0,0,0.15)',
  },
  bingkai: {
    width: 240,
    height: 240,
    borderWidth: 3,
    borderColor: '#fff',
    borderRadius: 16,
  },
  petunjuk: {
    color: '#fff',
    fontSize: 14,
    textAlign: 'center',
    paddingHorizontal: 32,
  },
  tombol: {
    backgroundColor: '#111827',
    borderRadius: 12,
    paddingVertical: 14,
    paddingHorizontal: 24,
  },
  tombolTeks: { color: '#fff', fontSize: 15, fontWeight: '600' },
  tombolBatal: { padding: 20, alignItems: 'center' },
  tombolBatalTeks: { color: '#fff', fontSize: 15 },
})
