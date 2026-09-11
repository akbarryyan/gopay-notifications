import type { ExpoConfig } from 'expo/config'

type Variant = 'development' | 'uat' | 'production'

// Default production supaya build tanpa APP_VARIANT tidak diam-diam
// menghasilkan aplikasi development.
const VARIANT = (process.env.APP_VARIANT ?? 'production') as Variant

/**
 * Package berbeda per varian sehingga ketiganya dapat terpasang bersamaan.
 *
 * Lebih penting dari itu: Android menyandera data tiap package di sandbox-nya
 * sendiri, jadi database event, Device Secret, dan Settings UAT tidak akan
 * pernah bercampur dengan produksi.
 */
const VARIANTS: Record<Variant, { name: string; package: string; cleartext: boolean }> = {
  development: {
    name: 'GoPay Bridge (Dev)',
    package: 'id.manjo.gopaybridge.dev',
    // HTTP polos hanya di sini, untuk Metro dan backend yang jalan di laptop.
    cleartext: true,
  },
  uat: {
    name: 'GoPay Bridge (UAT)',
    package: 'id.manjo.gopaybridge.uat',
    cleartext: false,
  },
  production: {
    name: 'GoPay Bridge',
    package: 'id.manjo.gopaybridge',
    cleartext: false,
  },
}

const v = VARIANTS[VARIANT]

if (!v) {
  throw new Error(
    `APP_VARIANT tidak dikenal: "${VARIANT}". Pilih development, uat, atau production.`,
  )
}

const config: ExpoConfig = {
  name: v.name,
  slug: 'gopay-bridge',
  version: '1.0.0',
  orientation: 'portrait',
  icon: './assets/icon.png',
  android: {
    package: v.package,
    adaptiveIcon: {
      backgroundColor: '#E6F4FE',
      foregroundImage: './assets/android-icon-foreground.png',
      backgroundImage: './assets/android-icon-background.png',
      monochromeImage: './assets/android-icon-monochrome.png',
    },
    predictiveBackGestureEnabled: false,
    permissions: ['android.permission.INTERNET', 'android.permission.ACCESS_NETWORK_STATE'],
  },
  plugins: [
    'expo-dev-client',
    // Menyuntikkan NotificationListenerService ke AndroidManifest.
    './plugins/withGopayListener',
    [
      'expo-build-properties',
      {
        android: {
          // Hanya minSdkVersion yang dipatok. compileSdk dan targetSdk
          // dibiarkan mengikuti bawaan Expo SDK 57 — memaksanya ke versi
          // lebih rendah justru merusak pustaka yang menuntut versi baru.
          //
          // 26 dipilih karena EncryptedSharedPreferences menuntut API 23,
          // requestRebind menuntut 24, dan java.time menuntut 26.
          minSdkVersion: 26,

          // UAT dan production tidak menyetelnya sama sekali, dan Android
          // sejak targetSdk 28 memblokir cleartext secara default — jadi
          // keduanya HTTPS-only tanpa konfigurasi apa pun.
          usesCleartextTraffic: v.cleartext,
        },
      },
    ],
  ],
}

export default config
