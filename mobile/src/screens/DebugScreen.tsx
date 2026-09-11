import { useCallback, useState } from 'react'
import { useFocusEffect } from '@react-navigation/native'
import { Alert, FlatList, StyleSheet, Text, TouchableOpacity, View } from 'react-native'
import GopayListener, {
  type BridgeEvent,
  type DiscoveryEntry,
} from '../../modules/gopay-listener'
import { formatRupiah, formatWaktu } from '../lib/format'

export default function DebugScreen() {
  const [entries, setEntries] = useState<DiscoveryEntry[]>([])
  const [aktifSampai, setAktifSampai] = useState(0)
  const [terakhir, setTerakhir] = useState<BridgeEvent | null>(null)

  const muat = useCallback(() => {
    setEntries(GopayListener.getDiscoveryEntries())
    setAktifSampai(GopayListener.getSettings().discoveryUntilMs)
    setTerakhir(GopayListener.getEvents(1)[0] ?? null)
  }, [])
  useFocusEffect(muat)

  /**
   * Mengantre ulang event yang sudah ada, apa pun statusnya.
   *
   * Ini satu-satunya cara menguji seluruh rantai pengiriman tanpa menunggu
   * pembayaran sungguhan. Pada event yang sudah SENT, backend akan menjawab
   * `duplicate` — dan itu justru yang ingin dibuktikan: kiriman ulang tidak
   * pernah menghasilkan pembayaran ganda.
   */
  function kirimUlangTerakhir() {
    if (!terakhir) return
    Alert.alert(
      'Kirim ulang event terakhir?',
      `${formatRupiah(terakhir.amountHint)} · status ${terakhir.status}\n\n` +
        'Bila event ini sudah pernah diterima backend, jawabannya akan ' +
        '`duplicate` dan statusnya tetap SENT.',
      [
        { text: 'Batal', style: 'cancel' },
        {
          text: 'Kirim',
          onPress: () => {
            GopayListener.retryEvent(terakhir.eventId)
            muat()
          },
        },
      ],
    )
  }

  const aktif = aktifSampai > Date.now()

  function mulai() {
    Alert.alert(
      'Nyalakan mode Discovery?',
      'Selama aktif, aplikasi mencatat nama package dan judul dari SEMUA notifikasi, ' +
        'termasuk aplikasi lain. Isi notifikasi tidak disimpan dan tidak ada yang dikirim ' +
        'keluar HP. Mati sendiri setelah 10 menit.',
      [
        { text: 'Batal', style: 'cancel' },
        {
          text: 'Nyalakan',
          onPress: () => {
            GopayListener.startDiscovery(10)
            muat()
          },
        },
      ],
    )
  }

  return (
    <FlatList
      contentContainerStyle={styles.root}
      data={entries}
      keyExtractor={(e, i) => `${e.packageName}-${e.seenAt}-${i}`}
      ListHeaderComponent={
        <View style={styles.header}>
          <Text style={styles.status}>
            Mode Discovery: {aktif ? `aktif sampai ${formatWaktu(aktifSampai)}` : 'mati'}
          </Text>

          {aktif ? (
            <TouchableOpacity
              style={styles.tombol}
              onPress={() => {
                GopayListener.stopDiscovery()
                muat()
              }}
            >
              <Text style={styles.tombolTeks}>Matikan sekarang</Text>
            </TouchableOpacity>
          ) : (
            <TouchableOpacity style={styles.tombol} onPress={mulai}>
              <Text style={styles.tombolTeks}>Nyalakan 10 menit</Text>
            </TouchableOpacity>
          )}

          <TouchableOpacity style={styles.tombolSekunder} onPress={muat}>
            <Text style={styles.tombolSekunderTeks}>Muat ulang</Text>
          </TouchableOpacity>

          <View style={styles.pemisah} />

          <Text style={styles.status}>
            {terakhir
              ? `Event terakhir: ${formatRupiah(terakhir.amountHint)} · ${terakhir.status}`
              : 'Belum ada event untuk dikirim ulang.'}
          </Text>

          <TouchableOpacity
            style={[styles.tombolSekunder, !terakhir && styles.tombolMati]}
            disabled={!terakhir}
            onPress={kirimUlangTerakhir}
          >
            <Text style={styles.tombolSekunderTeks}>Kirim ulang event terakhir</Text>
          </TouchableOpacity>

          <TouchableOpacity
            style={styles.tombolSekunder}
            onPress={() => {
              GopayListener.clearDiscovery()
              muat()
            }}
          >
            <Text style={styles.tombolSekunderTeks}>Bersihkan catatan</Text>
          </TouchableOpacity>
        </View>
      }
      ListEmptyComponent={<Text style={styles.kosong}>Belum ada catatan.</Text>}
      renderItem={({ item }) => (
        <View style={styles.kartu}>
          <Text style={styles.pkg}>{item.packageName}</Text>
          <Text style={styles.judul}>{item.title || '(tanpa judul)'}</Text>
          <Text style={styles.waktu}>{formatWaktu(item.seenAt)}</Text>
        </View>
      )}
    />
  )
}

const styles = StyleSheet.create({
  root: { padding: 16, gap: 8 },
  header: { gap: 8, marginBottom: 8 },
  status: { fontSize: 14, marginBottom: 4 },
  kosong: { textAlign: 'center', color: '#6b7280', marginTop: 24 },
  kartu: { backgroundColor: '#f9fafb', borderRadius: 10, padding: 12, gap: 2 },
  pkg: { fontSize: 14, fontWeight: '600' },
  judul: { fontSize: 13 },
  waktu: { fontSize: 12, color: '#6b7280' },
  tombol: { backgroundColor: '#111827', borderRadius: 12, padding: 14, alignItems: 'center' },
  tombolTeks: { color: '#fff', fontSize: 15, fontWeight: '600' },
  tombolSekunder: {
    borderWidth: 1,
    borderColor: '#d1d5db',
    borderRadius: 12,
    padding: 12,
    alignItems: 'center',
  },
  tombolSekunderTeks: { fontSize: 14 },
  tombolMati: { opacity: 0.4 },
  pemisah: { height: 12 },
})
