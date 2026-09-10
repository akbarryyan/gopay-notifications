import { useCallback, useEffect, useState } from 'react'
import { AppState, SafeAreaView, ScrollView, StyleSheet, Text, TouchableOpacity, View } from 'react-native'
import GopayListener from './modules/gopay-listener'

type Status = {
  granted: boolean
  connected: boolean
}

export default function App() {
  const [status, setStatus] = useState<Status>({ granted: false, connected: false })

  const refresh = useCallback(() => {
    setStatus({
      granted: GopayListener.isNotificationAccessGranted(),
      connected: GopayListener.isListenerConnected(),
    })
  }, [])

  useEffect(() => {
    refresh()
    // Status berubah di luar aplikasi (Android Settings), jadi dimuat ulang
    // setiap aplikasi kembali ke depan.
    const sub = AppState.addEventListener('change', (s) => {
      if (s === 'active') refresh()
    })
    return () => sub.remove()
  }, [refresh])

  return (
    <SafeAreaView style={styles.root}>
      <ScrollView contentContainerStyle={styles.isi}>
        <Text style={styles.judul}>GoPay Notification Bridge</Text>
        <Text style={styles.subjudul}>M2 — verifikasi jembatan native</Text>

        <View style={styles.kartu}>
          <Baris
            label="Notification Access"
            ok={status.granted}
            detail={status.granted ? 'aktif' : 'belum aktif'}
          />
          <Baris
            label="Listener terikat"
            ok={status.connected}
            detail={status.connected ? 'ya' : 'tidak'}
          />
        </View>

        {status.granted && !status.connected && (
          <View style={styles.peringatan}>
            <Text style={styles.peringatanTeks}>
              Izin sudah diberikan tetapi listener tidak terikat. Ini biasanya berarti sistem
              membunuh service. Periksa pengaturan baterai dan mulai otomatis untuk aplikasi ini.
            </Text>
          </View>
        )}

        <TouchableOpacity
          style={styles.tombol}
          onPress={() => GopayListener.openNotificationAccessSettings()}
        >
          <Text style={styles.tombolTeks}>Buka Notification Access</Text>
        </TouchableOpacity>

        <TouchableOpacity style={styles.tombolSekunder} onPress={refresh}>
          <Text style={styles.tombolSekunderTeks}>Muat ulang status</Text>
        </TouchableOpacity>
      </ScrollView>
    </SafeAreaView>
  )
}

function Baris({ label, ok, detail }: { label: string; ok: boolean; detail: string }) {
  return (
    <View style={styles.baris}>
      <Text style={styles.label}>{label}</Text>
      <View style={styles.kanan}>
        <Text style={[styles.titik, { color: ok ? '#16a34a' : '#dc2626' }]}>●</Text>
        <Text style={styles.detail}>{detail}</Text>
      </View>
    </View>
  )
}

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: '#fff' },
  isi: { padding: 20, gap: 12, paddingTop: 40 },
  judul: { fontSize: 20, fontWeight: '600' },
  subjudul: { fontSize: 13, color: '#6b7280', marginBottom: 8 },
  kartu: { backgroundColor: '#f9fafb', borderRadius: 12, padding: 16 },
  baris: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center', paddingVertical: 8 },
  kanan: { flexDirection: 'row', alignItems: 'center', gap: 6 },
  label: { fontSize: 15 },
  titik: { fontSize: 14 },
  detail: { fontSize: 13, color: '#6b7280' },
  peringatan: { backgroundColor: '#fef3c7', borderRadius: 12, padding: 14 },
  peringatanTeks: { fontSize: 13, color: '#92400e', lineHeight: 19 },
  tombol: { backgroundColor: '#111827', borderRadius: 12, padding: 16, alignItems: 'center', marginTop: 8 },
  tombolTeks: { color: '#fff', fontSize: 15, fontWeight: '600' },
  tombolSekunder: { borderWidth: 1, borderColor: '#d1d5db', borderRadius: 12, padding: 14, alignItems: 'center' },
  tombolSekunderTeks: { fontSize: 15 },
})
