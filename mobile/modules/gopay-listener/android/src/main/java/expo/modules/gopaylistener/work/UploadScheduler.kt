package expo.modules.gopaylistener.work

import android.content.Context
import androidx.work.BackoffPolicy
import androidx.work.Constraints
import androidx.work.ExistingWorkPolicy
import androidx.work.NetworkType
import androidx.work.OneTimeWorkRequestBuilder
import androidx.work.WorkManager
import java.util.concurrent.TimeUnit

object UploadScheduler {

    const val WORK_NAME = "event-upload"

    /**
     * Menjadwalkan pengiriman.
     *
     * Constraint NetworkType.CONNECTED berarti Android sendiri yang
     * membangunkan kita saat jaringan pulih — tidak ada polling sama sekali,
     * dan itu yang memenuhi tuntutan hemat baterai di detail-project.md §32.
     */
    fun enqueue(context: Context) {
        val request = OneTimeWorkRequestBuilder<EventUploadWorker>()
            .setConstraints(
                Constraints.Builder()
                    .setRequiredNetworkType(NetworkType.CONNECTED)
                    .build()
            )
            .setBackoffCriteria(BackoffPolicy.EXPONENTIAL, 30, TimeUnit.SECONDS)
            .build()

        WorkManager.getInstance(context.applicationContext)
            .enqueueUniqueWork(WORK_NAME, ExistingWorkPolicy.APPEND_OR_REPLACE, request)
    }
}
