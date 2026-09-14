import { useState } from 'react'
import { Alert, ScrollView, StyleSheet, Text, TextInput, TouchableOpacity, View } from 'react-native'
import GopayListener from '../../modules/gopay-listener'
import { QrScannerModal, type PairingPayload } from '../components/QrScannerModal'

const PESAN_ERROR: Record<string, string> = {
  belum_dikonfigurasi: 'Backend URL, Device ID, dan Device Secret harus diisi lebih dulu.',
  invalid_signature: 'Device ID atau Device Secret salah.',
  clock_skew: 'Jam HP meleset lebih dari 5 menit dari server. Setel jam ke otomatis.',
  device_disabled: 'Device ini dinonaktifkan di backend.',
  network: 'Backend tidak dapat dihubungi.',
}

export default function SettingsScreen() {
  const awal = GopayListener.getSettings()

  const [backendUrl, setBackendUrl] = useState(awal.backendUrl)
  const [deviceId, setDeviceId] = useState(awal.deviceId)
  const [deviceSecret, setDeviceSecret] = useState('')
  const [packages, setPackages] = useState(awal.monitoredPackages.join(', '))
  const [keywords, setKeywords] = useState(awal.ignoreKeywords.join(', '))
  const [punyaSecret, setPunyaSecret] = useState(awal.hasDeviceSecret)
  const [scannerVisible, setScannerVisible] = useState(false)

  const daftarPackage = packages.split(',').map((s) => s.trim()).filter(Boolean)
  const duaSumberGoPay =
    daftarPackage.includes('com.gojek.gopay') && daftarPackage.includes('com.gojek.gopaymerchant')

  function simpan() {
    GopayListener.saveSettings({
      backendUrl,
      deviceId,
      // Kosong berarti "jangan ubah" — UI tidak pernah bisa membaca secret
      // yang tersimpan untuk ditampilkan ulang.
      deviceSecret: deviceSecret || undefined,
      monitoredPackages: daftarPackage,
      ignoreKeywords: keywords.split(',').map((s) => s.trim()).filter(Boolean),
    })
    if (deviceSecret) setPunyaSecret(true)
    setDeviceSecret('')
    Alert.alert('Tersimpan')
  }

  async function uji() {
    const hasil = await GopayListener.testConnection()
    if (hasil.ok) {
      Alert.alert('Berhasil', `Terhubung sebagai ${hasil.deviceName}`)
      return
    }
    Alert.alert('Gagal', PESAN_ERROR[hasil.error ?? ''] ?? `Kesalahan: ${hasil.error}`)
  }

  // Menyimpan lalu langsung menguji koneksi -- pindai QR berarti pengguna
  // mengharapkan hasilnya "sudah tersambung", bukan sekadar "sudah tersalin".
  async function onScanned(payload: PairingPayload) {
    setScannerVisible(false)
    GopayListener.saveSettings({
      backendUrl: payload.backendUrl,
      deviceId: payload.deviceId,
      deviceSecret: payload.deviceSecret,
    })
    setBackendUrl(payload.backendUrl)
    setDeviceId(payload.deviceId)
    setDeviceSecret('')
    setPunyaSecret(true)

    const hasil = await GopayListener.testConnection()
    if (hasil.ok) {
      Alert.alert('Terhubung', `Pairing berhasil sebagai ${hasil.deviceName}.`)
    } else {
      Alert.alert(
        'Tersimpan, tapi belum terhubung',
        PESAN_ERROR[hasil.error ?? ''] ?? `Kesalahan: ${hasil.error}`,
      )
    }
  }

  function hapusRiwayat() {
    Alert.alert('Hapus semua riwayat?', 'Event yang belum terkirim ikut terhapus.', [
      { text: 'Batal', style: 'cancel' },
      {
        text: 'Hapus',
        style: 'destructive',
        onPress: () => {
          GopayListener.clearHistory()
          Alert.alert('Riwayat dihapus')
        },
      },
    ])
  }

  return (
    <ScrollView contentContainerStyle={styles.root}>
      <TouchableOpacity style={styles.tombolQr} onPress={() => setScannerVisible(true)}>
        <Text style={styles.tombolQrTeks}>📷 Scan QR dari Dashboard</Text>
      </TouchableOpacity>
      <Text style={styles.bantuan}>
        Cara tercepat dan paling akurat -- Backend URL, Device ID, dan Device Secret terisi
        otomatis dari QR yang tampil saat kamu membuat device baru di Dashboard.
      </Text>

      <View style={styles.pemisahAtau}>
        <View style={styles.garisAtau} />
        <Text style={styles.teksAtau}>atau isi manual</Text>
        <View style={styles.garisAtau} />
      </View>

      <QrScannerModal
        visible={scannerVisible}
        onClose={() => setScannerVisible(false)}
        onScanned={onScanned}
      />

      <Text style={styles.label}>Backend URL</Text>
      <TextInput
        style={styles.input}
        value={backendUrl}
        onChangeText={setBackendUrl}
        autoCapitalize="none"
        autoCorrect={false}
        placeholder="https://contoh.com/api/v1"
      />

      <Text style={styles.label}>Device ID</Text>
      <TextInput
        style={styles.input}
        value={deviceId}
        onChangeText={setDeviceId}
        autoCapitalize="none"
        autoCorrect={false}
      />

      <Text style={styles.label}>Device Secret</Text>
      <TextInput
        style={styles.input}
        value={deviceSecret}
        onChangeText={setDeviceSecret}
        secureTextEntry
        autoCapitalize="none"
        autoCorrect={false}
        placeholder={punyaSecret ? 'tersimpan — isi untuk mengganti' : 'belum diisi'}
      />
      <Text style={styles.bantuan}>
        Secret tidak pernah ditampilkan kembali. Kosongkan bila tidak ingin mengubahnya.
      </Text>

      <Text style={styles.label}>Package yang dipantau</Text>
      <TextInput
        style={styles.input}
        value={packages}
        onChangeText={setPackages}
        autoCapitalize="none"
        autoCorrect={false}
      />

      {duaSumberGoPay && (
        <View style={styles.peringatan}>
          <Text style={styles.peringatanTeks}>
            com.gojek.gopay dan com.gojek.gopaymerchant melaporkan pembayaran yang sama. Memantau
            keduanya membuat satu pembayaran terhitung dua kali. Pilih salah satu.
          </Text>
        </View>
      )}

      <Text style={styles.label}>Kata yang diabaikan</Text>
      <TextInput
        style={styles.input}
        value={keywords}
        onChangeText={setKeywords}
        autoCapitalize="none"
        autoCorrect={false}
        placeholder="kosongkan bila ragu"
      />
      <Text style={styles.bantuan}>
        Hanya pengurang noise. Keamanan dijamin backend — bila ragu, biarkan kosong.
      </Text>

      <TouchableOpacity style={styles.tombol} onPress={simpan}>
        <Text style={styles.tombolTeks}>Simpan</Text>
      </TouchableOpacity>

      <TouchableOpacity style={styles.tombolSekunder} onPress={uji}>
        <Text style={styles.tombolSekunderTeks}>Test Connection</Text>
      </TouchableOpacity>

      <TouchableOpacity
        style={styles.tombolSekunder}
        onPress={() => GopayListener.openNotificationAccessSettings()}
      >
        <Text style={styles.tombolSekunderTeks}>Buka Notification Access</Text>
      </TouchableOpacity>

      <View style={styles.pemisah} />

      <TouchableOpacity style={styles.tombolBahaya} onPress={hapusRiwayat}>
        <Text style={styles.tombolBahayaTeks}>Hapus riwayat event</Text>
      </TouchableOpacity>
    </ScrollView>
  )
}

const styles = StyleSheet.create({
  root: { padding: 16, gap: 8, paddingBottom: 40 },
  label: { fontSize: 13, color: '#6b7280', marginTop: 10 },
  input: { borderWidth: 1, borderColor: '#d1d5db', borderRadius: 10, padding: 12, fontSize: 15 },
  bantuan: { fontSize: 12, color: '#9ca3af', lineHeight: 17 },
  peringatan: { backgroundColor: '#fef3c7', borderRadius: 10, padding: 12, marginTop: 4 },
  peringatanTeks: { fontSize: 12, color: '#92400e', lineHeight: 18 },
  tombol: {
    backgroundColor: '#111827',
    borderRadius: 12,
    padding: 16,
    alignItems: 'center',
    marginTop: 16,
  },
  tombolTeks: { color: '#fff', fontSize: 15, fontWeight: '600' },
  tombolSekunder: {
    borderWidth: 1,
    borderColor: '#d1d5db',
    borderRadius: 12,
    padding: 14,
    alignItems: 'center',
  },
  tombolSekunderTeks: { fontSize: 15 },
  pemisah: { height: 24 },
  tombolQr: {
    backgroundColor: '#0f766e',
    borderRadius: 12,
    padding: 16,
    alignItems: 'center',
  },
  tombolQrTeks: { color: '#fff', fontSize: 15, fontWeight: '600' },
  pemisahAtau: { flexDirection: 'row', alignItems: 'center', gap: 10, marginVertical: 4 },
  garisAtau: { flex: 1, height: 1, backgroundColor: '#e5e7eb' },
  teksAtau: { fontSize: 12, color: '#9ca3af' },
  tombolBahaya: {
    borderWidth: 1,
    borderColor: '#fecaca',
    borderRadius: 12,
    padding: 14,
    alignItems: 'center',
  },
  tombolBahayaTeks: { color: '#dc2626', fontSize: 15 },
})
