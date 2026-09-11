package expo.modules.gopaylistener

import android.app.Notification
import android.content.ComponentName
import android.os.Bundle
import android.service.notification.NotificationListenerService
import android.service.notification.StatusBarNotification
import android.util.Log
import expo.modules.gopaylistener.config.Settings
import expo.modules.gopaylistener.db.AppDatabase
import expo.modules.gopaylistener.db.EventEntity
import expo.modules.gopaylistener.db.EventStatus
import expo.modules.gopaylistener.discovery.DiscoveryLog
import expo.modules.gopaylistener.work.UploadScheduler
import java.util.concurrent.Executors

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

        // onNotificationPosted dipanggil di main thread; Room tidak boleh
        // disentuh dari sana.
        private val io = Executors.newSingleThreadExecutor()
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
        val app = applicationContext
        val pkg = sbn.packageName

        val extras = sbn.notification.extras
        val title = extras.getCharSequence(Notification.EXTRA_TITLE)?.toString()
        val text = extras.getCharSequence(Notification.EXTRA_TEXT)?.toString()
        val bigText = extras.getCharSequence(Notification.EXTRA_BIG_TEXT)?.toString()
        val whenMs = CapturePolicy.effectiveWhen(sbn.notification.`when`, sbn.postTime)

        io.execute {
            try {
                val settings = Settings(app)

                if (settings.discoveryActive()) {
                    DiscoveryLog.record(app, pkg, title.orEmpty())
                }

                val decision = CapturePolicy.decide(
                    packageName = pkg,
                    title = title,
                    text = text,
                    monitoredPackages = settings.monitoredPackages,
                    ignoreKeywords = settings.ignoreKeywords,
                )
                if (decision is CaptureDecision.Skip) return@execute

                val entity = EventEntity(
                    eventId = EventIdBuilder.build(pkg, title, text, whenMs),
                    packageName = pkg,
                    title = title,
                    text = text,
                    bigText = bigText,
                    amountHint = AmountParser.parse(text ?: bigText),
                    postedAt = whenMs,
                    receivedAt = System.currentTimeMillis(),
                    status = if (decision is CaptureDecision.Ignore) {
                        EventStatus.IGNORED
                    } else {
                        EventStatus.PENDING
                    },
                    attemptCount = 0,
                    lastError = null,
                    backendStatus = null,
                    sentAt = null,
                )

                val dao = AppDatabase.get(app).events()
                if (dao.insertIgnoringDuplicate(entity) == -1L) {
                    // Android memang memanggil ulang untuk notifikasi yang sama.
                    Log.i(TAG, "event sudah ada, dilewati")
                    return@execute
                }

                Log.i(TAG, "event baru ditangkap, status=${entity.status}")

                if (decision is CaptureDecision.Capture) {
                    UploadScheduler.enqueue(app)
                }

                CaptureBus.emit(
                    Bundle().apply {
                        putString("eventId", entity.eventId)
                        putString("title", entity.title)
                        putString("text", entity.text)
                        entity.amountHint?.let { putLong("amountHint", it) }
                        putLong("receivedAt", entity.receivedAt)
                        putString("status", entity.status.name)
                    }
                )
            } catch (t: Throwable) {
                // Satu notifikasi yang gagal diproses tidak boleh mematikan service.
                Log.e(TAG, "gagal memproses notifikasi", t)
            }
        }
    }
}
