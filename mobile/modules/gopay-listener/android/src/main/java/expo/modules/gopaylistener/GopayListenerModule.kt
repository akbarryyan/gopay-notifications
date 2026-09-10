package expo.modules.gopaylistener

import android.content.Context
import android.content.Intent
import android.provider.Settings
import expo.modules.kotlin.modules.Module
import expo.modules.kotlin.modules.ModuleDefinition

class GopayListenerModule : Module() {

    private val context: Context
        get() = requireNotNull(appContext.reactContext) { "reactContext tidak tersedia" }

    override fun definition() = ModuleDefinition {
        Name("GopayListener")

        Function("isNotificationAccessGranted") {
            isAccessGranted()
        }

        Function("openNotificationAccessSettings") {
            val intent = Intent(Settings.ACTION_NOTIFICATION_LISTENER_SETTINGS)
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
    }

    private fun isAccessGranted(): Boolean {
        val enabled = Settings.Secure.getString(
            context.contentResolver,
            "enabled_notification_listeners"
        ) ?: return false

        // Nilainya berupa daftar ComponentName dipisah ":".
        return enabled.split(":").any { it.startsWith(context.packageName + "/") }
    }
}
