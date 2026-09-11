package expo.modules.gopaylistener.db

import androidx.room.Entity
import androidx.room.PrimaryKey
import androidx.room.TypeConverter

/**
 * Lima status, bukan sembilan seperti detail-project.md §27.
 *
 * RECEIVED, PARSED, PERSISTED, dan QUEUED terjadi dalam satu transaksi yang
 * tak terpisahkan — menyimpannya sebagai status terpisah hanya menambah
 * kemungkinan salah tanpa memberi informasi. PARSE_FAILED gugur karena teks
 * mentah selalu diteruskan: gagal parsing nominal bukan kegagalan event,
 * hanya amountHint bernilai null. RETRY_PENDING diwakili PENDING dengan
 * attemptCount > 0.
 */
enum class EventStatus {
    PENDING,
    SENDING,
    SENT,
    FAILED,
    IGNORED,
}

/**
 * Konversi eksplisit ke nama enum sebagai TEXT.
 *
 * Bukan sekadar formalitas: beberapa query di EventDao memakai literal SQL
 * seperti `status = 'PENDING'`, sehingga representasi tersimpan wajib persis
 * nama enum-nya.
 */
class EventStatusConverter {
    @TypeConverter
    fun toStatus(value: String): EventStatus = EventStatus.valueOf(value)

    @TypeConverter
    fun fromStatus(status: EventStatus): String = status.name
}

@Entity(tableName = "events")
data class EventEntity(
    @PrimaryKey val eventId: String,
    val packageName: String,
    val title: String?,
    val text: String?,
    val bigText: String?,
    /** Display-only. Backend melakukan ekstraksi otoritatifnya sendiri. */
    val amountHint: Long?,
    /** Notification.when, atau sbn.postTime bila when bernilai 0. */
    val postedAt: Long,
    val receivedAt: Long,
    val status: EventStatus,
    val attemptCount: Int,
    /** Pesan singkat. Tidak boleh memuat secret atau tanda tangan. */
    val lastError: String?,
    val backendStatus: String?,
    val sentAt: Long?,
)
