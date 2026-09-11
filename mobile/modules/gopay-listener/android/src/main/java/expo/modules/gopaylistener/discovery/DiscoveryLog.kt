package expo.modules.gopaylistener.discovery

import android.content.Context
import org.json.JSONArray
import org.json.JSONObject

data class DiscoveryEntry(
    val packageName: String,
    val title: String,
    val seenAt: Long,
)

/**
 * Menyimpan jejak mode Discovery secara lokal.
 *
 * HANYA nama package dan judul yang dicatat, dan TIDAK PERNAH dikirim keluar
 * perangkat. Isi notifikasi tidak disimpan.
 *
 * Memakai SharedPreferences biasa, bukan Room, karena datanya sementara dan
 * tidak layak ikut mengubah skema database. Tidak dienkripsi karena tidak ada
 * kredensial di dalamnya.
 */
object DiscoveryLog {

    const val PREFS = "gopay-bridge-discovery"
    private const val KEY = "entries"
    private const val MAX_ENTRIES = 100

    fun record(context: Context, packageName: String, title: String, nowMs: Long = System.currentTimeMillis()) {
        val prefs = context.applicationContext.getSharedPreferences(PREFS, Context.MODE_PRIVATE)
        val existing = JSONArray(prefs.getString(KEY, "[]"))

        // Terbaru di depan, dipotong pada MAX_ENTRIES.
        val trimmed = JSONArray()
        trimmed.put(
            JSONObject().apply {
                put("package_name", packageName)
                put("title", title)
                put("seen_at", nowMs)
            }
        )
        for (i in 0 until minOf(existing.length(), MAX_ENTRIES - 1)) {
            trimmed.put(existing.get(i))
        }

        prefs.edit().putString(KEY, trimmed.toString()).apply()
    }

    fun entries(context: Context): List<DiscoveryEntry> {
        val prefs = context.applicationContext.getSharedPreferences(PREFS, Context.MODE_PRIVATE)
        val arr = JSONArray(prefs.getString(KEY, "[]"))
        return (0 until arr.length()).map { i ->
            val o = arr.getJSONObject(i)
            DiscoveryEntry(
                packageName = o.getString("package_name"),
                title = o.optString("title"),
                seenAt = o.getLong("seen_at"),
            )
        }
    }

    fun clear(context: Context) {
        context.applicationContext.getSharedPreferences(PREFS, Context.MODE_PRIVATE)
            .edit().clear().apply()
    }
}
