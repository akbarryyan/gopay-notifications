package expo.modules.gopaylistener

import java.security.MessageDigest

/**
 * Membentuk event_id yang deterministik, sesuai api-contract.md §4.2.
 *
 * Bahan sengaja TIDAK memuat:
 *
 *  - notificationKey — GoPay memakai id -1 tanpa tag untuk seluruh
 *    notifikasinya, sehingga key-nya identik dan tidak menyumbang apa pun.
 *  - postTime — berubah setiap notifikasi di-posting ulang, sehingga satu
 *    transfer yang di-repost akan terkirim sebagai dua event berbeda.
 *
 * whenMs berasal dari Notification.when, yang bertahan lintas repost.
 */
object EventIdBuilder {

    fun build(packageName: String, title: String?, text: String?, whenMs: Long): String {
        val raw = listOf(
            packageName,
            title.orEmpty(),
            text.orEmpty(),
            whenMs.toString()
        ).joinToString("|")

        val digest = MessageDigest.getInstance("SHA-256")
            .digest(raw.toByteArray(Charsets.UTF_8))

        return "evt_" + digest.joinToString("") { "%02x".format(it) }.take(32)
    }
}
