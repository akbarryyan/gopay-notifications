package expo.modules.gopaylistener

import android.content.Context
import android.content.Intent
import android.provider.Settings as AndroidSettings
import expo.modules.gopaylistener.config.Settings
import android.os.Bundle
import expo.modules.gopaylistener.db.AppDatabase
import expo.modules.gopaylistener.db.EventEntity
import expo.modules.gopaylistener.db.EventStatus
import expo.modules.gopaylistener.discovery.DiscoveryLog
import expo.modules.gopaylistener.net.Uploader
import expo.modules.gopaylistener.work.PurgeWorker
import expo.modules.gopaylistener.work.UploadScheduler
import java.lang.ref.WeakReference
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

    private var captureObserver: ((Bundle) -> Unit)? = null

    override fun definition() = ModuleDefinition {
        Name("GopayListener")

        // Retensi 30 hari. Teks notifikasi merchant tidak memuat PII pihak
        // ketiga, tetapi menyimpannya selamanya tetap tidak ada gunanya.
        OnCreate {
            PurgeWorker.schedule(context)
        }

        Events(EVENT_CHANGED)

        // Dipancarkan saat event baru DITANGKAP maupun saat STATUSNYA BERUBAH.
        // Tanpa yang kedua, Dashboard membeku menampilkan PENDING sementara
        // event sebenarnya sudah SENT.
        //
        // WeakReference wajib: observer disimpan di objek statis CaptureBus,
        // dan tanpa ini instance module tertahan di memori setelah UI ditutup.
        OnStartObserving(EVENT_CHANGED) {
            val weakModule = WeakReference(this@GopayListenerModule)
            val observer: (Bundle) -> Unit = { payload ->
                weakModule.get()?.sendEvent(EVENT_CHANGED, payload)
            }
            captureObserver = observer
            CaptureBus.register(observer)
        }

        OnStopObserving(EVENT_CHANGED) {
            captureObserver?.let { CaptureBus.unregister(it) }
            captureObserver = null
        }

        /**
         * Varian diturunkan dari package name aplikasi yang BENAR-BENAR
         * terpasang, bukan dari konfigurasi build yang bisa menyimpang.
         * Tiga varian punya package berbeda, jadi ini tidak bisa berbohong.
         */
        Function("getEnvironment") {
            val pkg = context.packageName
            val variant = when {
                pkg.endsWith(".dev") -> "development"
                pkg.endsWith(".uat") -> "uat"
                else -> "production"
            }
            mapOf(
                "packageName" to pkg,
                "variant" to variant,
                "appVersion" to AppInfo.version(context),
            )
        }

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
            Uploader.deviceMe(cfg, AppInfo.version(context))
        }

        Function("getStatus") {
            val dao = AppDatabase.get(context).events()
            mapOf(
                "notificationAccessGranted" to isAccessGranted(),
                "listenerConnected" to GoPayListenerService.isConnected,
                "pendingCount" to dao.countByStatus(EventStatus.PENDING),
                "sendingCount" to dao.countByStatus(EventStatus.SENDING),
                "failedCount" to dao.countByStatus(EventStatus.FAILED),
                "lastEvent" to dao.latest()?.let { toMap(it) },
            )
        }

        Function("getEvents") { limit: Int ->
            AppDatabase.get(context).events().recent(limit.coerceIn(1, 500)).map { toMap(it) }
        }

        Function("retryEvent") { eventId: String ->
            AppDatabase.get(context).events().rescheduleForRetry(eventId, "dikirim ulang manual")
            UploadScheduler.enqueue(context)
        }

        Function("clearHistory") {
            AppDatabase.get(context).events().purgeOlderThan(System.currentTimeMillis())
        }
    }

    private fun toMap(e: EventEntity): Map<String, Any?> = mapOf(
        "eventId" to e.eventId,
        "packageName" to e.packageName,
        "title" to e.title,
        "text" to e.text,
        "amountHint" to e.amountHint,
        "postedAt" to e.postedAt,
        "receivedAt" to e.receivedAt,
        "status" to e.status.name,
        "attemptCount" to e.attemptCount,
        "lastError" to e.lastError,
        "backendStatus" to e.backendStatus,
        "sentAt" to e.sentAt,
    )

    private companion object {
        const val EVENT_CHANGED = "onBridgeChanged"
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
