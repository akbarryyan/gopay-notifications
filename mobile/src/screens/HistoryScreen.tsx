import { useCallback, useState } from 'react'
import { useFocusEffect } from '@react-navigation/native'
import { FlatList, StyleSheet, Text, TouchableOpacity, View } from 'react-native'
import GopayListener, { type BridgeEvent, type EventStatus } from '../../modules/gopay-listener'
import { formatRupiah, formatWaktu } from '../lib/format'

const WARNA: Record<EventStatus, string> = {
  SENT: '#16a34a',
  PENDING: '#ca8a04',
  SENDING: '#2563eb',
  FAILED: '#dc2626',
  IGNORED: '#9ca3af',
}

export default function HistoryScreen() {
  const [events, setEvents] = useState<BridgeEvent[]>([])

  const muat = useCallback(() => setEvents(GopayListener.getEvents(200)), [])
  useFocusEffect(muat)

  function kirimUlang(eventId: string) {
    GopayListener.retryEvent(eventId)
    muat()
  }

  return (
    <FlatList
      contentContainerStyle={styles.root}
      data={events}
      keyExtractor={(e) => e.eventId}
      ListEmptyComponent={<Text style={styles.kosong}>Belum ada event.</Text>}
      renderItem={({ item }) => (
        <View style={styles.kartu}>
          <View style={styles.baris}>
            <Text style={styles.nominal}>{formatRupiah(item.amountHint)}</Text>
            <Text style={[styles.status, { color: WARNA[item.status] }]}>{item.status}</Text>
          </View>
          <Text style={styles.teks}>{item.title ?? '—'}</Text>
          <Text style={styles.teksKecil}>{formatWaktu(item.receivedAt)}</Text>

          {item.attemptCount > 0 && (
            <Text style={styles.teksKecil}>Percobaan: {item.attemptCount}</Text>
          )}
          {item.lastError && <Text style={styles.error}>{item.lastError}</Text>}

          {item.status === 'FAILED' && (
            <TouchableOpacity style={styles.tombolKecil} onPress={() => kirimUlang(item.eventId)}>
              <Text style={styles.tombolKecilTeks}>Kirim ulang</Text>
            </TouchableOpacity>
          )}
        </View>
      )}
    />
  )
}

const styles = StyleSheet.create({
  root: { padding: 16, gap: 10 },
  kosong: { textAlign: 'center', color: '#6b7280', marginTop: 40 },
  kartu: { backgroundColor: '#f9fafb', borderRadius: 12, padding: 14, gap: 3 },
  baris: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center' },
  nominal: { fontSize: 18, fontWeight: '600' },
  status: { fontSize: 12, fontWeight: '600' },
  teks: { fontSize: 14 },
  teksKecil: { fontSize: 12, color: '#6b7280' },
  error: { fontSize: 12, color: '#dc2626' },
  tombolKecil: {
    marginTop: 8,
    backgroundColor: '#111827',
    borderRadius: 8,
    padding: 10,
    alignItems: 'center',
  },
  tombolKecilTeks: { color: '#fff', fontSize: 13, fontWeight: '600' },
})
