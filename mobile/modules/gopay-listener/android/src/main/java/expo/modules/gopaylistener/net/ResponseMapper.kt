package expo.modules.gopaylistener.net

import org.json.JSONObject

sealed interface UploadOutcome {
    data class Sent(val backendStatus: String) : UploadOutcome
    data class Failed(val error: String) : UploadOutcome
    data class Retry(val error: String, val retryAfterSeconds: Long?) : UploadOutcome
}

/**
 * Menerjemahkan response backend menjadi tindakan, sesuai tabel di
 * api-contract.md §4.6.
 *
 * Dipisah sebagai fungsi murni agar seluruh baris tabel itu dapat diuji tanpa
 * jaringan, emulator, maupun backend yang menyala.
 */
object ResponseMapper {

    fun map(httpCode: Int, body: String?, retryAfterHeader: String?): UploadOutcome = when {
        httpCode == 200 -> mapOk(body)

        httpCode == 429 -> UploadOutcome.Retry("rate_limited", retryAfterHeader?.toLongOrNull())

        httpCode >= 500 -> UploadOutcome.Retry("server_error_$httpCode", null)

        httpCode in 400..499 -> UploadOutcome.Failed(errorCode(body) ?: "http_$httpCode")

        else -> UploadOutcome.Failed("http_$httpCode")
    }

    private fun mapOk(body: String?): UploadOutcome {
        val status = try {
            JSONObject(body ?: "").optString("status")
        } catch (_: Throwable) {
            return UploadOutcome.Failed("response_tidak_terbaca")
        }

        // duplicate diperlakukan sebagai sukses. Bila response hilang di tengah
        // jalan padahal backend sudah menerima, percobaan berikutnya membalas
        // duplicate dan event beres — tidak ada pembayaran terhitung dua kali.
        return if (status == "accepted" || status == "duplicate") {
            UploadOutcome.Sent(status)
        } else {
            UploadOutcome.Failed("status_tidak_dikenal")
        }
    }

    private fun errorCode(body: String?): String? = try {
        JSONObject(body ?: "").optString("error").takeIf { it.isNotEmpty() }
    } catch (_: Throwable) {
        null
    }
}
