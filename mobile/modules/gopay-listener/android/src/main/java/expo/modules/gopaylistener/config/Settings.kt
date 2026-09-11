package expo.modules.gopaylistener.config

import android.content.Context
import android.content.SharedPreferences
import androidx.security.crypto.EncryptedSharedPreferences
import androidx.security.crypto.MasterKey

data class BridgeConfig(
    val backendUrl: String,
    val deviceId: String,
    val deviceSecret: String,
)

/**
 * Konfigurasi bridge.
 *
 * Disimpan di sisi native, bukan lewat expo-secure-store, karena
 * EventUploadWorker membacanya saat runtime JavaScript sudah mati.
 * Mekanisme perlindungannya sama: Android Keystore.
 *
 * SharedPreferences dapat disuntik agar logika di kelas ini dapat diuji di JVM.
 * Robolectric tidak mengemulasi Android Keystore, sehingga membangun
 * EncryptedSharedPreferences di test akan gagal karena runtime-nya, bukan
 * karena kodenya. Enkripsinya sendiri hanya dapat diverifikasi di perangkat.
 */
class Settings internal constructor(private val prefs: SharedPreferences) {

    constructor(context: Context) : this(securePrefs(context))

    var backendUrl: String
        get() = prefs.getString(KEY_BACKEND_URL, "").orEmpty()
        set(value) = prefs.edit().putString(KEY_BACKEND_URL, value.trim().trimEnd('/')).apply()

    var deviceId: String
        get() = prefs.getString(KEY_DEVICE_ID, "").orEmpty()
        set(value) = prefs.edit().putString(KEY_DEVICE_ID, value.trim()).apply()

    var deviceSecret: String
        get() = prefs.getString(KEY_DEVICE_SECRET, "").orEmpty()
        set(value) = prefs.edit().putString(KEY_DEVICE_SECRET, value.trim()).apply()

    /**
     * Wajib tepat satu entri untuk sumber pembayaran.
     *
     * com.gojek.gopay melaporkan pembayaran yang SAMA dengan teks identik;
     * karena packageName ikut jadi bahan event_id, memantau keduanya membuat
     * satu pembayaran menghasilkan dua event dan terhitung dua kali.
     */
    var monitoredPackages: List<String>
        get() = splitList(prefs.getString(KEY_PACKAGES, DEFAULT_PACKAGES))
        set(value) = prefs.edit().putString(KEY_PACKAGES, value.joinToString(",")).apply()

    /**
     * Default sengaja kosong. Keamanan dijamin allowlist judul di backend;
     * daftar ini hanya pengurang noise. Bila ragu, lebih baik tetap kirim.
     */
    var ignoreKeywords: List<String>
        get() = splitList(prefs.getString(KEY_IGNORE, ""))
        set(value) = prefs.edit().putString(KEY_IGNORE, value.joinToString(",")).apply()

    var discoveryUntilMs: Long
        get() = prefs.getLong(KEY_DISCOVERY_UNTIL, 0L)
        set(value) = prefs.edit().putLong(KEY_DISCOVERY_UNTIL, value).apply()

    fun discoveryActive(nowMs: Long = System.currentTimeMillis()): Boolean =
        nowMs < discoveryUntilMs

    /** null selama salah satu dari tiga nilai wajib masih kosong. */
    fun config(): BridgeConfig? {
        val url = backendUrl
        val id = deviceId
        val secret = deviceSecret
        if (url.isEmpty() || id.isEmpty() || secret.isEmpty()) return null
        return BridgeConfig(url, id, secret)
    }

    fun clearAll() = prefs.edit().clear().apply()

    private fun splitList(raw: String?): List<String> =
        raw.orEmpty().split(",").map { it.trim() }.filter { it.isNotEmpty() }

    companion object {
        const val PREFS_NAME = "gopay-bridge-settings"
        const val DEFAULT_PACKAGES = "com.gojek.gopaymerchant"

        private const val KEY_BACKEND_URL = "backend_url"
        private const val KEY_DEVICE_ID = "device_id"
        private const val KEY_DEVICE_SECRET = "device_secret"
        private const val KEY_PACKAGES = "monitored_packages"
        private const val KEY_IGNORE = "ignore_keywords"
        private const val KEY_DISCOVERY_UNTIL = "discovery_until"

        private fun securePrefs(context: Context): SharedPreferences {
            val app = context.applicationContext
            val masterKey = MasterKey.Builder(app)
                .setKeyScheme(MasterKey.KeyScheme.AES256_GCM)
                .build()

            return EncryptedSharedPreferences.create(
                app,
                PREFS_NAME,
                masterKey,
                EncryptedSharedPreferences.PrefKeyEncryptionScheme.AES256_SIV,
                EncryptedSharedPreferences.PrefValueEncryptionScheme.AES256_GCM,
            )
        }
    }
}
