package expo.modules.gopaylistener.net

import expo.modules.gopaylistener.Signer
import expo.modules.gopaylistener.config.BridgeConfig
import expo.modules.gopaylistener.db.EventEntity
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import org.json.JSONObject
import java.io.IOException
import java.time.Instant
import java.time.OffsetDateTime
import java.time.ZoneId
import java.time.format.DateTimeFormatter
import java.util.concurrent.TimeUnit

object Uploader {

    private val JSON = "application/json".toMediaType()

    private val client = OkHttpClient.Builder()
        .connectTimeout(15, TimeUnit.SECONDS)
        .readTimeout(30, TimeUnit.SECONDS)
        .writeTimeout(30, TimeUnit.SECONDS)
        .build()

    fun send(
        cfg: BridgeConfig,
        e: EventEntity,
        appVersion: String = "",
        nowSeconds: Long = System.currentTimeMillis() / 1000,
    ): UploadOutcome {
        // Byte ini yang ditandatangani DAN yang dikirim, tanpa disusun ulang
        // di antaranya. Membangun ulang JSON setelah menandatangani akan
        // mengubah urutan field dan spasi, dan tanda tangan gagal secara acak.
        val payload = buildPayload(cfg.deviceId, e).toByteArray(Charsets.UTF_8)

        val signature = Signer.sign(
            cfg.deviceSecret,
            Signer.signingString(cfg.deviceId, nowSeconds, payload)
        )

        // Nama connector TIDAK ada di URL: payload identik untuk setiap
        // sumber dan pembedanya hanya field "source".
        val request = Request.Builder()
            .url(cfg.backendUrl + "/events")
            .post(payload.toRequestBody(JSON))
            .header("X-Device-Id", cfg.deviceId)
            .header("X-Timestamp", nowSeconds.toString())
            .header("X-Signature", signature)
            .header("X-App-Version", appVersion)
            .build()

        return try {
            client.newCall(request).execute().use { response ->
                ResponseMapper.map(
                    response.code,
                    response.body?.string(),
                    response.header("Retry-After")
                )
            }
        } catch (_: IOException) {
            // Jaringan mati atau timeout — selalu dapat dipulihkan.
            UploadOutcome.Retry("network", null)
        }
    }

    /**
     * Laporan kondisi berkala.
     *
     * listenerConnected adalah informasi paling berharga di sini: izin aktif
     * tetapi listener tidak terikat adalah gejala service dibunuh OEM, dan
     * tanpa heartbeat hal itu baru diketahui saat sebuah pembayaran terlewat.
     */
    data class HeartbeatReport(
        val androidVersion: String,
        val listenerConnected: Boolean,
        val pendingCount: Int,
        val failedCount: Int,
    )

    /** Mengembalikan true bila backend menerima laporan. */
    fun heartbeat(
        cfg: BridgeConfig,
        appVersion: String,
        report: HeartbeatReport,
        nowSeconds: Long = System.currentTimeMillis() / 1000,
    ): Boolean {
        val payload = JSONObject().apply {
            put("android_version", report.androidVersion)
            put("listener_connected", report.listenerConnected)
            put("pending_count", report.pendingCount)
            put("failed_count", report.failedCount)
        }.toString().toByteArray(Charsets.UTF_8)

        val signature = Signer.sign(
            cfg.deviceSecret,
            Signer.signingString(cfg.deviceId, nowSeconds, payload)
        )

        val request = Request.Builder()
            .url(cfg.backendUrl + "/devices/heartbeat")
            .post(payload.toRequestBody(JSON))
            .header("X-Device-Id", cfg.deviceId)
            .header("X-Timestamp", nowSeconds.toString())
            .header("X-Signature", signature)
            .header("X-App-Version", appVersion)
            .build()

        return try {
            client.newCall(request).execute().use { it.isSuccessful }
        } catch (_: IOException) {
            false
        }
    }

    /** Mengembalikan (backend hidup, jam server dalam detik). */
    fun health(backendUrl: String): Pair<Boolean, Long?> {
        val request = Request.Builder().url("$backendUrl/health").get().build()
        return try {
            client.newCall(request).execute().use { response ->
                if (!response.isSuccessful) return false to null
                val json = JSONObject(response.body?.string() ?: "")
                true to json.optLong("server_time").takeIf { it > 0 }
            }
        } catch (_: Throwable) {
            false to null
        }
    }

    /** Dipakai tombol Test Connection di layar Settings. */
    fun deviceMe(
        cfg: BridgeConfig,
        appVersion: String = "",
        nowSeconds: Long = System.currentTimeMillis() / 1000,
    ): Map<String, Any?> {
        val signature = Signer.sign(
            cfg.deviceSecret,
            Signer.signingString(cfg.deviceId, nowSeconds, ByteArray(0))
        )

        val request = Request.Builder()
            .url(cfg.backendUrl + "/device/me")
            .get()
            .header("X-Device-Id", cfg.deviceId)
            .header("X-Timestamp", nowSeconds.toString())
            .header("X-Signature", signature)
            .header("X-App-Version", appVersion)
            .build()

        return try {
            client.newCall(request).execute().use { response ->
                val body = response.body?.string()
                if (response.isSuccessful) {
                    mapOf("ok" to true, "deviceName" to JSONObject(body ?: "{}").optString("name"))
                } else {
                    val error = try {
                        JSONObject(body ?: "{}").optString("error")
                    } catch (_: Throwable) {
                        ""
                    }
                    mapOf("ok" to false, "error" to error.ifEmpty { "http_${response.code}" })
                }
            }
        } catch (_: IOException) {
            mapOf("ok" to false, "error" to "network")
        }
    }

    internal fun buildPayload(deviceId: String, e: EventEntity): String {
        val notification = JSONObject().apply {
            put("package_name", e.packageName)
            put("title", e.title ?: JSONObject.NULL)
            put("text", e.text ?: JSONObject.NULL)
            put("big_text", e.bigText ?: JSONObject.NULL)
            put("posted_at", e.postedAt)
        }

        val receivedAt = OffsetDateTime
            .ofInstant(Instant.ofEpochMilli(e.receivedAt), ZoneId.systemDefault())
            .format(DateTimeFormatter.ISO_OFFSET_DATE_TIME)

        return JSONObject().apply {
            put("event_id", e.eventId)
            put("device_id", deviceId)
            put("source", "gopay")
            put("notification", notification)
            put("amount_hint", e.amountHint ?: JSONObject.NULL)
            put("received_at", receivedAt)
        }.toString()
    }
}
