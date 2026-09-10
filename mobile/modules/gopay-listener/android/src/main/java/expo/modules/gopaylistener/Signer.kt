package expo.modules.gopaylistener

import java.security.MessageDigest
import javax.crypto.Mac
import javax.crypto.spec.SecretKeySpec

/**
 * Menandatangani request ke backend, sesuai api-contract.md §3.2.
 *
 * Yang di-hash adalah byte mentah body yang benar-benar dikirim. Jangan pernah
 * menyusun ulang JSON setelah menandatanganinya — urutan field dan spasi akan
 * berbeda dan tanda tangan gagal secara acak.
 */
object Signer {

    fun signingString(deviceId: String, timestamp: Long, body: ByteArray): String {
        val sha = MessageDigest.getInstance("SHA-256").digest(body)
        return deviceId + "\n" + timestamp + "\n" + sha.toHex()
    }

    fun sign(secret: String, signingString: String): String {
        val mac = Mac.getInstance("HmacSHA256")
        mac.init(SecretKeySpec(secret.toByteArray(Charsets.UTF_8), "HmacSHA256"))
        return mac.doFinal(signingString.toByteArray(Charsets.UTF_8)).toHex()
    }

    private fun ByteArray.toHex(): String = joinToString("") { "%02x".format(it) }
}
