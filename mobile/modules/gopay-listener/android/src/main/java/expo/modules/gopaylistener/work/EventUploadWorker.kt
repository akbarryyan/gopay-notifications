package expo.modules.gopaylistener.work

import android.content.Context
import android.os.Bundle
import android.util.Log
import expo.modules.gopaylistener.CaptureBus
import androidx.work.CoroutineWorker
import androidx.work.WorkerParameters
import expo.modules.gopaylistener.config.Settings
import expo.modules.gopaylistener.db.AppDatabase
import expo.modules.gopaylistener.net.UploadOutcome
import expo.modules.gopaylistener.net.Uploader

class EventUploadWorker(
    context: Context,
    params: WorkerParameters,
) : CoroutineWorker(context, params) {

    override suspend fun doWork(): Result {
        val app = applicationContext

        val cfg = Settings(app).config()
        if (cfg == null) {
            Log.i(TAG, "backend belum dikonfigurasi, pengiriman ditunda")
            return Result.success()
        }

        val dao = AppDatabase.get(app).events()
        val batch = dao.claimPending(BATCH_SIZE)
        if (batch.isEmpty()) return Result.success()

        var needRetry = false

        for (event in batch) {
            when (val outcome = Uploader.send(cfg, event)) {
                is UploadOutcome.Sent -> {
                    dao.markSent(event.eventId, outcome.backendStatus, System.currentTimeMillis())
                    Log.i(TAG, "event terkirim, status=${outcome.backendStatus}")
                }

                is UploadOutcome.Failed -> {
                    dao.markFailed(event.eventId, outcome.error)
                    Log.w(TAG, "event gagal permanen: ${outcome.error}")
                }

                is UploadOutcome.Retry -> {
                    val attempts = event.attemptCount + 1
                    if (attempts >= MAX_ATTEMPTS) {
                        dao.markFailed(
                            event.eventId,
                            "menyerah setelah $MAX_ATTEMPTS percobaan: ${outcome.error}"
                        )
                        Log.w(TAG, "event menyerah setelah $MAX_ATTEMPTS percobaan")
                    } else {
                        dao.rescheduleForRetry(event.eventId, outcome.error)
                        needRetry = true
                    }
                }
            }
        }

        // Memberi tahu UI yang sedang terbuka bahwa status berubah. Bila UI
        // mati, tidak ada observer dan panggilan ini tidak berbiaya.
        CaptureBus.emit(Bundle().apply { putString("reason", "upload") })

        return if (needRetry) Result.retry() else Result.success()
    }

    companion object {
        private const val TAG = "GoPayUpload"
        const val BATCH_SIZE = 50

        /**
         * WorkManager memakai backoff eksponensial dari 30 detik, dibatasi
         * 5 jam per jeda. Tiga belas percobaan menghasilkan total sekitar
         * 24 jam sebelum menyerah:
         *
         *   30d, 1m, 2m, 4m, 8m, 16m, 32m, 64m, 2,1j, lalu 5j x 4
         *
         * Backend mati lebih dari sehari adalah masalah yang perlu dilihat
         * sendiri, bukan dicoba diam-diam selamanya sambil menghabiskan baterai.
         */
        const val MAX_ATTEMPTS = 13
    }
}
