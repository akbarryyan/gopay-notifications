package expo.modules.gopaylistener

import android.content.Context
import android.content.pm.PackageManager

/**
 * Versi aplikasi yang benar-benar terpasang.
 *
 * Dibaca dari PackageManager, bukan dari konstanta di kode — konstanta bisa
 * menyimpang dari APK yang sungguhan ada di perangkat, dan backend memakai
 * nilai ini untuk memutuskan kompatibilitas.
 */
object AppInfo {

    fun version(context: Context): String =
        try {
            context.packageManager
                .getPackageInfo(context.packageName, 0)
                .versionName
                ?: ""
        } catch (_: PackageManager.NameNotFoundException) {
            ""
        }
}
