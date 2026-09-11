package expo.modules.gopaylistener.work

import android.content.Context
import androidx.work.CoroutineWorker
import androidx.work.ExistingPeriodicWorkPolicy
import androidx.work.PeriodicWorkRequestBuilder
import androidx.work.WorkManager
import androidx.work.WorkerParameters
import expo.modules.gopaylistener.db.AppDatabase
import java.util.concurrent.TimeUnit

/**
 * Menghapus event lama.
 *
 * Hanya SENT dan IGNORED yang didaur ulang. FAILED tidak pernah dihapus
 * otomatis: ia satu-satunya jejak bahwa ada pembayaran yang tidak sampai ke
 * backend, dan menghapusnya berarti menghapus bukti masalah.
 */
class PurgeWorker(context: Context, params: WorkerParameters) : CoroutineWorker(context, params) {

    override suspend fun doWork(): Result {
        val cutoff = System.currentTimeMillis() - RETENTION_MS
        val deleted = AppDatabase.get(applicationContext).events().purgeOlderThan(cutoff)
        if (deleted > 0) {
            android.util.Log.i(TAG, "menghapus $deleted event lama")
        }
        return Result.success()
    }

    companion object {
        private const val TAG = "GoPayPurge"
        const val WORK_NAME = "event-purge"
        const val RETENTION_DAYS = 30L
        private const val RETENTION_MS = RETENTION_DAYS * 24 * 60 * 60 * 1000

        fun schedule(context: Context) {
            val request = PeriodicWorkRequestBuilder<PurgeWorker>(1, TimeUnit.DAYS).build()
            WorkManager.getInstance(context.applicationContext).enqueueUniquePeriodicWork(
                WORK_NAME,
                ExistingPeriodicWorkPolicy.KEEP,
                request,
            )
        }
    }
}
