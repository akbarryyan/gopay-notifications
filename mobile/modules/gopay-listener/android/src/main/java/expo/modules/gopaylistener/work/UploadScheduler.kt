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
     * Menjadwalkan pengiriman, dan menjalankannya SEGERA.
     *
     * Constraint NetworkType.CONNECTED berarti Android sendiri yang
     * membangunkan kita saat jaringan pulih — tidak ada polling sama sekali,
     * dan itu yang memenuhi tuntutan hemat baterai di detail-project.md §32.
     *
     * REPLACE, bukan APPEND_OR_REPLACE. Dengan APPEND, pembayaran baru antre
     * di belakang pekerjaan yang sedang dalam backoff panjang — satu kegagalan
     * lama bisa menunda pembayaran berikutnya sampai setengah jam, padahal
     * penyebabnya mungkin sudah lama beres. REPLACE membatalkan backoff itu
     * dan mencoba sekarang.
     *
     * Yang membuatnya aman: worker memulihkan event yang tertinggal di SENDING
     * saat mulai, dan backend menjawab `duplicate` untuk event yang sudah
     * diterima. Jadi pembatalan di tengah jalan tidak pernah menghilangkan
     * event maupun menghasilkan pembayaran ganda.
     *
     * Batas menyerah tidak ikut tereset: hitungannya per event di database,
     * bukan runAttemptCount milik WorkManager.
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
            .enqueueUniqueWork(WORK_NAME, ExistingWorkPolicy.REPLACE, request)
    }
}
