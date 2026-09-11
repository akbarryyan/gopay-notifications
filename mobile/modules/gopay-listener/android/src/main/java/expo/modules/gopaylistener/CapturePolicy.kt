package expo.modules.gopaylistener

/**
 * Keputusan penyaringan, dipisah dari service agar dapat diuji tanpa Android.
 *
 * Di sinilah bug paling mahal bersembunyi: salah saring berarti tidak ada
 * event sama sekali, atau satu pembayaran terhitung dua kali.
 */
sealed interface CaptureDecision {
    /** Bukan dari package yang dipantau. Tidak dicatat sama sekali. */
    data object Skip : CaptureDecision

    /** Dari package yang dipantau tetapi cocok kata-diabaikan. Disimpan, tidak dikirim. */
    data object Ignore : CaptureDecision

    /** Disimpan dan diantre untuk dikirim. */
    data object Capture : CaptureDecision
}

object CapturePolicy {

    fun decide(
        packageName: String,
        title: String?,
        text: String?,
        bigText: String?,
        monitoredPackages: List<String>,
        ignoreKeywords: List<String>,
    ): CaptureDecision {
        if (packageName !in monitoredPackages) return CaptureDecision.Skip

        // Notifikasi tanpa teks sama sekali. Android memasang notifikasi
        // ringkasan grup berdampingan dengan yang asli, dan isinya kosong.
        // Ia tidak mungkin memuat pembayaran, jadi tidak perlu disimpan
        // maupun dikirim. Mode Discovery tetap menangkapnya bila suatu saat
        // perlu diperiksa.
        if (title.isNullOrBlank() && text.isNullOrBlank() && bigText.isNullOrBlank()) {
            return CaptureDecision.Skip
        }

        // Daftar kata-diabaikan hanya pengurang noise. Keamanan dijamin
        // allowlist judul di backend, jadi bila ragu lebih baik tetap kirim.
        val haystack = title.orEmpty() + " " + text.orEmpty()
        val ignored = ignoreKeywords.any { it.isNotEmpty() && haystack.contains(it, ignoreCase = true) }

        return if (ignored) CaptureDecision.Ignore else CaptureDecision.Capture
    }

    /**
     * Notification.when bertahan lintas repost; sbn.postTime berubah setiap
     * notifikasi di-posting ulang dan akan memecah satu transfer jadi beberapa
     * event. postTime hanya dipakai bila when bernilai 0.
     */
    fun effectiveWhen(notificationWhen: Long, postTime: Long): Long =
        if (notificationWhen > 0L) notificationWhen else postTime
}
