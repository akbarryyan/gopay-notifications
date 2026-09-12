package expo.modules.gopaylistener.work

import android.content.Context
import android.os.Build
import android.util.Log
import androidx.work.Constraints
import androidx.work.CoroutineWorker
import androidx.work.ExistingPeriodicWorkPolicy
import androidx.work.NetworkType
import androidx.work.PeriodicWorkRequestBuilder
import androidx.work.WorkManager
import androidx.work.WorkerParameters
import expo.modules.gopaylistener.AppInfo
import expo.modules.gopaylistener.GoPayListenerService
import expo.modules.gopaylistener.config.Settings
import expo.modules.gopaylistener.db.AppDatabase
import expo.modules.gopaylistener.db.EventStatus
import expo.modules.gopaylistener.net.Uploader
import java.util.concurrent.TimeUnit

/**
 * Melaporkan kondisi perangkat ke backend secara berkala.
 *
 * Sebelum ini, backend hanya mendengar dari perangkat ketika ada pembayaran.
 * Toko dengan tiga transaksi sehari tampak mati hampir sepanjang waktu, dan HP
 * yang benar-benar dibunuh ColorOS baru diketahui saat sebuah pembayaran
 * terlewat — untuk produk berbayar itu tidak dapat diterima.
 *
 * Ini menyimpang dari detail-project.md §32 yang melarang network request
 * berkala. Biayanya 96 request sehari, dan yang dibeli dengan itu adalah
 * kemampuan mengetahui HP mati sebelum pembayaran hilang.
 */
class HeartbeatWorker(
    context: Context,
    params: WorkerParameters,
) : CoroutineWorker(context, params) {

    override suspend fun doWork(): Result {
        val app = applicationContext

        val cfg = Settings(app).config() ?: return Result.success()
        val dao = AppDatabase.get(app).events()

        val terkirim = Uploader.heartbeat(
            cfg = cfg,
            appVersion = AppInfo.version(app),
            report = Uploader.HeartbeatReport(
                androidVersion = Build.VERSION.RELEASE ?: "",
                listenerConnected = GoPayListenerService.isConnected,
                pendingCount = dao.countByStatus(EventStatus.PENDING),
                failedCount = dao.countByStatus(EventStatus.FAILED),
            ),
        )

        if (!terkirim) {
            // Tidak di-retry: denyut berikutnya sudah dijadwalkan, dan
            // memaksa retry hanya menumpuk pekerjaan tanpa menambah informasi.
            Log.i(TAG, "heartbeat gagal terkirim, menunggu denyut berikutnya")
        }

        return Result.success()
    }

    companion object {
        private const val TAG = "GoPayHeartbeat"
        const val WORK_NAME = "device-heartbeat"

        /** 15 menit adalah interval minimum WorkManager untuk periodic work. */
        const val INTERVAL_MINUTES = 15L

        fun schedule(context: Context) {
            val request = PeriodicWorkRequestBuilder<HeartbeatWorker>(
                INTERVAL_MINUTES, TimeUnit.MINUTES,
            ).setConstraints(
                Constraints.Builder()
                    .setRequiredNetworkType(NetworkType.CONNECTED)
                    .build()
            ).build()

            WorkManager.getInstance(context.applicationContext).enqueueUniquePeriodicWork(
                WORK_NAME,
                // KEEP, bukan UPDATE: mengganti pekerjaan periodik akan
                // mengulang jadwalnya dari nol setiap aplikasi dibuka.
                ExistingPeriodicWorkPolicy.KEEP,
                request,
            )
        }
    }
}
