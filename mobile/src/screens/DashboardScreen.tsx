import { RefreshControl, ScrollView, StyleSheet, Text, TouchableOpacity, View } from 'react-native'
import GopayListener from '../../modules/gopay-listener'
import { useBridgeStatus } from '../lib/useBridge'
import { formatRupiah, formatWaktu, sejakKapan } from '../lib/format'

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

export default function DashboardScreen() {
  const { status, backend, refresh } = useBridgeStatus()
  const last = status.lastEvent

  return (
    <ScrollView
      contentContainerStyle={styles.root}
      refreshControl={<RefreshControl refreshing={false} onRefresh={refresh} />}
    >
      <View style={styles.kartu}>
        <Baris
          label="Notification Access"
          ok={status.notificationAccessGranted}
          detail={status.notificationAccessGranted ? 'aktif' : 'belum aktif'}
        />
        <Baris
          label="Listener"
          ok={status.listenerConnected}
          detail={status.listenerConnected ? 'terikat' : 'tidak terikat'}
        />
        <Baris
          label="Backend"
          ok={backend?.reachable ?? false}
          detail={
            !backend?.configured
              ? 'belum dikonfigurasi'
              : backend.reachable
                ? 'terhubung'
                : 'tidak dapat dihubungi'
          }
        />
      </View>

      {status.notificationAccessGranted && !status.listenerConnected && (
        <View style={styles.peringatan}>
          <Text style={styles.peringatanTeks}>
            Izin sudah diberikan tetapi listener tidak terikat. Ini biasanya berarti sistem membunuh
            service. Periksa pengaturan baterai dan mulai otomatis untuk aplikasi ini.
          </Text>
        </View>
      )}

      {backend?.clockOutOfSync && (
        <View style={styles.peringatan}>
          <Text style={styles.peringatanTeks}>
            Jam HP meleset {backend.skewSeconds} detik dari server. Selama selisihnya lebih dari 5
            menit, semua pengiriman akan ditolak. Setel jam ke otomatis.
          </Text>
        </View>
      )}

      <View style={styles.kartu}>
        <Text style={styles.subjudul}>Event terakhir</Text>
        {last ? (
          <>
            <Text style={styles.nominal}>{formatRupiah(last.amountHint)}</Text>
            <Text style={styles.teks}>{last.title ?? '—'}</Text>
            <Text style={styles.teksKecil}>
              {formatWaktu(last.receivedAt)} · {sejakKapan(last.receivedAt)} · {last.status}
            </Text>
          </>
        ) : (
          <Text style={styles.teksKecil}>Belum ada event.</Text>
        )}
      </View>

      <View style={styles.kartu}>
        <Text style={styles.subjudul}>Antrean</Text>
        <Text style={styles.teks}>
          {status.pendingCount} menunggu · {status.sendingCount} dikirim · {status.failedCount} gagal
        </Text>
        {status.failedCount > 0 && (
          <Text style={styles.teksKecil}>Buka Riwayat untuk mengirim ulang yang gagal.</Text>
        )}
      </View>

      {!status.notificationAccessGranted && (
        <TouchableOpacity
          style={styles.tombol}
          onPress={() => GopayListener.openNotificationAccessSettings()}
        >
          <Text style={styles.tombolTeks}>Aktifkan Notification Access</Text>
        </TouchableOpacity>
      )}
    </ScrollView>
  )
}

const styles = StyleSheet.create({
  root: { padding: 16, gap: 12 },
  subjudul: { fontSize: 13, color: '#6b7280', marginBottom: 6 },
  kartu: { backgroundColor: '#f9fafb', borderRadius: 12, padding: 16, gap: 4 },
  baris: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    paddingVertical: 6,
  },
  kanan: { flexDirection: 'row', alignItems: 'center', gap: 6 },
  label: { fontSize: 15 },
  titik: { fontSize: 14 },
  detail: { fontSize: 13, color: '#6b7280' },
  nominal: { fontSize: 24, fontWeight: '600' },
  teks: { fontSize: 15 },
  teksKecil: { fontSize: 13, color: '#6b7280' },
  peringatan: { backgroundColor: '#fef3c7', borderRadius: 12, padding: 14 },
  peringatanTeks: { fontSize: 13, color: '#92400e', lineHeight: 19 },
  tombol: { backgroundColor: '#111827', borderRadius: 12, padding: 16, alignItems: 'center' },
  tombolTeks: { color: '#fff', fontSize: 15, fontWeight: '600' },
})
