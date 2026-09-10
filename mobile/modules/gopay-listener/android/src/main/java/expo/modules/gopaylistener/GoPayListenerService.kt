package expo.modules.gopaylistener

import android.content.ComponentName
import android.service.notification.NotificationListenerService
import android.service.notification.StatusBarNotification
import android.util.Log

/**
 * Menerima notifikasi dari sistem Android.
 *
 * Berjalan di proses aplikasi tetapi lepas dari lifecycle React Native —
 * tetap dipanggil saat UI tertutup. Pipeline penangkapan ditambahkan di Task 8.
 */
class GoPayListenerService : NotificationListenerService() {

    companion object {
        private const val TAG = "GoPayListener"

        /**
         * Menandai apakah sistem sedang benar-benar terikat ke service ini.
         *
         * Berbeda dari "izin sudah diberikan": ColorOS dapat mencabut ikatan
         * tanpa mencabut izin, dan justru selisih antara keduanya yang menjadi
         * gejala service dibunuh diam-diam.
         */
        @Volatile
        var isConnected: Boolean = false
            private set

        private fun setConnected(value: Boolean) {
            isConnected = value
        }
    }

    override fun onListenerConnected() {
        super.onListenerConnected()
        setConnected(true)
        Log.i(TAG, "listener terikat")
    }

    override fun onListenerDisconnected() {
        super.onListenerDisconnected()
        setConnected(false)
        Log.w(TAG, "listener terlepas, meminta rebind")
        requestRebind(ComponentName(this, GoPayListenerService::class.java))
    }

    override fun onNotificationPosted(sbn: StatusBarNotification) {
        // Task 8: filter package, bentuk event, simpan ke Room, antre WorkManager.
    }
}
