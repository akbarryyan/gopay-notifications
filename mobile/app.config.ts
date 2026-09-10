import type { ExpoConfig } from 'expo/config'

// Varian development memakai package Android berbeda, sehingga build dev dan
// build production dapat terpasang berdampingan di HP yang sama.
const IS_DEV = process.env.APP_VARIANT === 'development'

const config: ExpoConfig = {
  name: IS_DEV ? 'GoPay Bridge (Dev)' : 'GoPay Bridge',
  slug: 'gopay-bridge',
  version: '1.0.0',
  orientation: 'portrait',
  icon: './assets/icon.png',
  android: {
    package: IS_DEV ? 'id.manjo.gopaybridge.dev' : 'id.manjo.gopaybridge',
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
        },
      },
    ],
  ],
}

export default config
