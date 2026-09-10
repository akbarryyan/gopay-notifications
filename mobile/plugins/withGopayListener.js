const { AndroidConfig, withAndroidManifest } = require('expo/config-plugins')

const SERVICE_NAME = 'expo.modules.gopaylistener.GoPayListenerService'
const LISTENER_ACTION = 'android.service.notification.NotificationListenerService'
const BIND_PERMISSION = 'android.permission.BIND_NOTIFICATION_LISTENER_SERVICE'

/**
 * Mendaftarkan NotificationListenerService ke AndroidManifest.
 *
 * Ditulis sebagai config plugin, bukan edit manual, karena folder android/
 * dibuat ulang setiap prebuild dan suntingan manual akan hilang.
 */
module.exports = function withGopayListener(config) {
  return withAndroidManifest(config, (config) => {
    const app = AndroidConfig.Manifest.getMainApplicationOrThrow(config.modResults)

    app.service = app.service ?? []

    const already = app.service.some((s) => s.$['android:name'] === SERVICE_NAME)
    if (!already) {
      app.service.push({
        $: {
          'android:name': SERVICE_NAME,
          'android:label': 'GoPay Notification Bridge',
          'android:exported': 'false',
          'android:permission': BIND_PERMISSION,
        },
        'intent-filter': [{ action: [{ $: { 'android:name': LISTENER_ACTION } }] }],
      })
    }

    return config
  })
}
