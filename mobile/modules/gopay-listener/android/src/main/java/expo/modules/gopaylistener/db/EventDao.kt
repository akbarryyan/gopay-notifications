package expo.modules.gopaylistener.db

import androidx.room.Dao
import androidx.room.Insert
import androidx.room.OnConflictStrategy
import androidx.room.Query
import androidx.room.Transaction

@Dao
interface EventDao {

    /**
     * Menyisipkan event. Mengembalikan -1 bila eventId sudah ada.
     *
     * Pencegahan duplikat bersandar pada primary key, bukan pada pemeriksaan
     * lebih dulu — Android memanggil onNotificationPosted berkali-kali untuk
     * notifikasi yang sama, kadang beriringan.
     */
    @Insert(onConflict = OnConflictStrategy.IGNORE)
    fun insertIgnoringDuplicate(event: EventEntity): Long

    /** Mengambil batch PENDING dan menandainya SENDING dalam satu transaksi. */
    @Transaction
    fun claimPending(limit: Int): List<EventEntity> {
        val batch = selectPending(limit)
        batch.forEach { setStatus(it.eventId, EventStatus.SENDING) }
        return batch
    }

    @Query("SELECT * FROM events WHERE status = 'PENDING' ORDER BY receivedAt ASC LIMIT :limit")
    fun selectPending(limit: Int): List<EventEntity>

    /**
     * Mengembalikan event yang tertinggal di SENDING menjadi PENDING.
     *
     * SENDING hanya boleh ada selama sebuah worker benar-benar berjalan. Bila
     * worker dibatalkan atau prosesnya dibunuh di tengah pengiriman, event itu
     * tertinggal di SENDING selamanya dan tidak pernah dicoba lagi — hilang
     * tanpa jejak kegagalan. Dipanggil di awal tiap worker.
     *
     * Aman terhadap pengiriman ganda: bila request sempat sampai ke backend,
     * percobaan berikutnya dijawab `duplicate` dan tetap dihitung berhasil.
     */
    @Query("UPDATE events SET status = 'PENDING' WHERE status = 'SENDING'")
    fun releaseStaleSending(): Int

    @Query("UPDATE events SET status = :status WHERE eventId = :eventId")
    fun setStatus(eventId: String, status: EventStatus)

    @Query(
        """UPDATE events
           SET status = 'SENT', backendStatus = :backendStatus, sentAt = :sentAt, lastError = NULL
           WHERE eventId = :eventId"""
    )
    fun markSent(eventId: String, backendStatus: String, sentAt: Long)

    @Query("UPDATE events SET status = 'FAILED', lastError = :error WHERE eventId = :eventId")
    fun markFailed(eventId: String, error: String)

    @Query(
        """UPDATE events
           SET status = 'PENDING', attemptCount = attemptCount + 1, lastError = :error
           WHERE eventId = :eventId"""
    )
    fun rescheduleForRetry(eventId: String, error: String)

    @Query("SELECT * FROM events ORDER BY receivedAt DESC LIMIT :limit")
    fun recent(limit: Int): List<EventEntity>

    @Query("SELECT * FROM events WHERE status != 'IGNORED' ORDER BY receivedAt DESC LIMIT 1")
    fun latest(): EventEntity?

    @Query("SELECT COUNT(*) FROM events WHERE status = :status")
    fun countByStatus(status: EventStatus): Int

    /**
     * Hanya SENT dan IGNORED yang didaur ulang.
     * FAILED tidak pernah dihapus otomatis — ia menandai sesuatu yang perlu
     * dilihat manusia.
     */
    @Query("DELETE FROM events WHERE receivedAt < :cutoffMs AND status IN ('SENT', 'IGNORED')")
    fun purgeOlderThan(cutoffMs: Long): Int
}
