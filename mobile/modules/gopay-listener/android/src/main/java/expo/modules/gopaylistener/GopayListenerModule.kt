package expo.modules.gopaylistener

import android.content.Context
import android.content.Intent
import android.provider.Settings as AndroidSettings
import expo.modules.gopaylistener.config.Settings
import expo.modules.gopaylistener.discovery.DiscoveryLog
import expo.modules.gopaylistener.net.Uploader
import expo.modules.kotlin.modules.Module
import expo.modules.kotlin.modules.ModuleDefinition
import expo.modules.kotlin.records.Field
import expo.modules.kotlin.records.Record

/**
 * Perubahan konfigurasi dari sisi UI.
 *
 * Field null berarti "jangan ubah", bukan "kosongkan" — sehingga layar
 * Settings dapat menyimpan sebagian nilai tanpa menghapus sisanya.
 */
class SettingsPatch : Record {
    @Field var backendUrl: String? = null
    @Field var deviceId: String? = null
    @Field var deviceSecret: String? = null
    @Field var monitoredPackages: List<String>? = null
    @Field var ignoreKeywords: List<String>? = null
}

class GopayListenerModule : Module() {

    private val context: Context
        get() = requireNotNull(appContext.reactContext) { "reactContext tidak tersedia" }

    override fun definition() = ModuleDefinition {
        Name("GopayListener")

        Function("isNotificationAccessGranted") {
            isAccessGranted()
        }

        Function("openNotificationAccessSettings") {
            val intent = Intent(AndroidSettings.ACTION_NOTIFICATION_LISTENER_SETTINGS)
            intent.addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
            context.startActivity(intent)
        }

        /**
         * Status ikatan yang sebenarnya, bukan sekadar status izin.
         * Keduanya bisa berbeda, dan perbedaannya adalah informasi penting.
         */
        Function("isListenerConnected") {
            GoPayListenerService.isConnected
        }

        Function("getSettings") {
            val s = Settings(context)
            mapOf(
                "backendUrl" to s.backendUrl,
                "deviceId" to s.deviceId,
                // Secret tidak pernah dikembalikan ke JS, hanya penandanya.
                "hasDeviceSecret" to s.deviceSecret.isNotEmpty(),
                "monitoredPackages" to s.monitoredPackages,
                "ignoreKeywords" to s.ignoreKeywords,
                "discoveryUntilMs" to s.discoveryUntilMs,
            )
        }

        Function("saveSettings") { patch: SettingsPatch ->
            val s = Settings(context)
            patch.backendUrl?.let { s.backendUrl = it }
            patch.deviceId?.let { s.deviceId = it }
            // String kosong berarti "biarkan seperti semula", bukan "hapus".
            patch.deviceSecret?.let { if (it.isNotEmpty()) s.deviceSecret = it }
            patch.monitoredPackages?.let { s.monitoredPackages = it }
            patch.ignoreKeywords?.let { s.ignoreKeywords = it }
        }

        /** Dibatasi 1–10 menit agar mode Discovery tidak tertinggal menyala. */
        Function("startDiscovery") { minutes: Int ->
            val capped = minutes.coerceIn(1, 10)
            Settings(context).discoveryUntilMs = System.currentTimeMillis() + capped * 60_000L
        }

        Function("stopDiscovery") {
            Settings(context).discoveryUntilMs = 0L
        }

        Function("getDiscoveryEntries") {
            DiscoveryLog.entries(context).map {
                mapOf(
                    "packageName" to it.packageName,
                    "title" to it.title,
                    "seenAt" to it.seenAt,
                )
            }
        }

        Function("clearDiscovery") {
            DiscoveryLog.clear(context)
        }

        /**
         * Status backend untuk Dashboard. Sekaligus mendeteksi jam HP yang
         * meleset SEBELUM ada event yang dikirim — tanpa ini, gejalanya
         * muncul belakangan sebagai kegagalan autentikasi yang membingungkan.
         */
        AsyncFunction("checkBackend") {
            val url = Settings(context).backendUrl
            if (url.isEmpty()) {
                return@AsyncFunction mapOf("configured" to false, "reachable" to false)
            }

            val (reachable, serverTime) = Uploader.health(url)
            val skewSeconds = serverTime?.let { it - System.currentTimeMillis() / 1000 }

            mapOf(
                "configured" to true,
                "reachable" to reachable,
                "serverTime" to serverTime,
                "skewSeconds" to skewSeconds,
                // Batas yang sama dengan backend, agar peringatan muncul
                // sebelum request sungguhan ditolak.
                "clockOutOfSync" to (skewSeconds != null && kotlin.math.abs(skewSeconds) > 300),
            )
        }

        AsyncFunction("testConnection") {
            val cfg = Settings(context).config()
                ?: return@AsyncFunction mapOf("ok" to false, "error" to "belum_dikonfigurasi")
            Uploader.deviceMe(cfg)
        }
    }

    private fun isAccessGranted(): Boolean {
        val enabled = AndroidSettings.Secure.getString(
            context.contentResolver,
            "enabled_notification_listeners"
        ) ?: return false

        // Nilainya berupa daftar ComponentName dipisah ":".
        return enabled.split(":").any { it.startsWith(context.packageName + "/") }
    }
}
