package expo.modules.gopaylistener.db

import androidx.room.Room
import androidx.test.core.app.ApplicationProvider
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotEquals
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config

/**
 * SDK dipatok eksplisit, tidak dibiarkan mengikuti targetSdk proyek.
 * targetSdk di sini 36, sementara Robolectric 4.14 belum mengenal API 36 —
 * membiarkannya berarti test gagal karena runtime, bukan karena kode.
 */
@RunWith(RobolectricTestRunner::class)
@Config(sdk = [34])
class EventDaoTest {

    private lateinit var db: AppDatabase
    private lateinit var dao: EventDao

    @Before
    fun setUp() {
        db = Room.inMemoryDatabaseBuilder(
            ApplicationProvider.getApplicationContext(),
            AppDatabase::class.java
        ).allowMainThreadQueries().build()
        dao = db.events()
    }

    @After
    fun tearDown() = db.close()

    private fun event(
        id: String,
        status: EventStatus = EventStatus.PENDING,
        receivedAt: Long = 1000L,
    ) = EventEntity(
        eventId = id,
        packageName = "com.gojek.gopaymerchant",
        title = "Pembayaran QRIS statis diterima",
        text = "Rp 1 di AKBAR RAYYAN AL GHIFARI, Digital & Kreatif.",
        bigText = null,
        amountHint = 1L,
        postedAt = 1_789_051_832_829L,
        receivedAt = receivedAt,
        status = status,
        attemptCount = 0,
        lastError = null,
        backendStatus = null,
        sentAt = null,
    )

    @Test
    fun `menyisipkan event baru`() {
        assertNotEquals(-1L, dao.insertIgnoringDuplicate(event("evt_a")))
        assertEquals(1, dao.countByStatus(EventStatus.PENDING))
    }

    @Test
    fun `menolak event_id yang sama tanpa melempar exception`() {
        dao.insertIgnoringDuplicate(event("evt_a"))
        val second = dao.insertIgnoringDuplicate(event("evt_a"))

        assertEquals("penyisipan kedua harus dilaporkan sebagai duplikat", -1L, second)
        assertEquals(1, dao.countByStatus(EventStatus.PENDING))
    }

    @Test
    fun `claimPending memindahkan status ke SENDING`() {
        dao.insertIgnoringDuplicate(event("evt_a"))
        dao.insertIgnoringDuplicate(event("evt_b"))

        val claimed = dao.claimPending(10)

        assertEquals(2, claimed.size)
        assertEquals(0, dao.countByStatus(EventStatus.PENDING))
        assertEquals(2, dao.countByStatus(EventStatus.SENDING))
    }

    @Test
    fun `claimPending tidak mengambil event IGNORED atau SENT`() {
        dao.insertIgnoringDuplicate(event("evt_ignored", EventStatus.IGNORED))
        dao.insertIgnoringDuplicate(event("evt_sent", EventStatus.SENT))
        dao.insertIgnoringDuplicate(event("evt_pending"))

        val claimed = dao.claimPending(10)

        assertEquals(1, claimed.size)
        assertEquals("evt_pending", claimed[0].eventId)
    }

    @Test
    fun `claimPending menghormati batas dan mendahulukan yang terlama`() {
        dao.insertIgnoringDuplicate(event("evt_baru", receivedAt = 3000L))
        dao.insertIgnoringDuplicate(event("evt_lama", receivedAt = 1000L))

        val claimed = dao.claimPending(1)

        assertEquals(1, claimed.size)
        assertEquals("antrean dikirim urut kedatangan", "evt_lama", claimed[0].eventId)
    }

    @Test
    fun `markSent mengisi backendStatus dan sentAt`() {
        dao.insertIgnoringDuplicate(event("evt_a"))
        dao.claimPending(10)

        dao.markSent("evt_a", "accepted", 5000L)

        val got = dao.recent(10).single()
        assertEquals(EventStatus.SENT, got.status)
        assertEquals("accepted", got.backendStatus)
        assertEquals(5000L, got.sentAt)
        assertNull("lastError dibersihkan saat berhasil", got.lastError)
    }

    @Test
    fun `rescheduleForRetry mengembalikan ke PENDING dan menaikkan attemptCount`() {
        dao.insertIgnoringDuplicate(event("evt_a"))
        dao.claimPending(10)

        dao.rescheduleForRetry("evt_a", "timeout")
        dao.claimPending(10)
        dao.rescheduleForRetry("evt_a", "timeout")

        val got = dao.recent(10).single()
        assertEquals(EventStatus.PENDING, got.status)
        assertEquals(2, got.attemptCount)
        assertEquals("timeout", got.lastError)
    }

    @Test
    fun `markFailed menghentikan event dari antrean`() {
        dao.insertIgnoringDuplicate(event("evt_a"))
        dao.claimPending(10)

        dao.markFailed("evt_a", "invalid_signature")

        assertEquals(1, dao.countByStatus(EventStatus.FAILED))
        assertTrue(dao.claimPending(10).isEmpty())
    }

    @Test
    fun `recent mengembalikan yang terbaru lebih dulu`() {
        dao.insertIgnoringDuplicate(event("evt_lama", receivedAt = 1000L))
        dao.insertIgnoringDuplicate(event("evt_baru", receivedAt = 2000L))

        assertEquals("evt_baru", dao.recent(10)[0].eventId)
    }

    @Test
    fun `latest melewati event IGNORED`() {
        dao.insertIgnoringDuplicate(event("evt_nyata", receivedAt = 1000L))
        dao.insertIgnoringDuplicate(event("evt_promo", EventStatus.IGNORED, receivedAt = 9000L))

        assertEquals(
            "Dashboard tidak boleh menampilkan notifikasi promo sebagai event terakhir",
            "evt_nyata",
            dao.latest()?.eventId
        )
    }

    @Test
    fun `purgeOlderThan hanya menghapus SENT dan IGNORED`() {
        dao.insertIgnoringDuplicate(event("evt_sent", EventStatus.SENT, receivedAt = 100L))
        dao.insertIgnoringDuplicate(event("evt_ignored", EventStatus.IGNORED, receivedAt = 100L))
        dao.insertIgnoringDuplicate(event("evt_failed", EventStatus.FAILED, receivedAt = 100L))
        dao.insertIgnoringDuplicate(event("evt_baru", EventStatus.SENT, receivedAt = 9999L))

        val deleted = dao.purgeOlderThan(1000L)

        assertEquals(2, deleted)
        assertEquals(
            setOf("evt_failed", "evt_baru"),
            dao.recent(10).map { it.eventId }.toSet()
        )
    }

    @Test
    fun `status tersimpan sebagai nama enum dalam TEXT`() {
        // Beberapa query memakai literal SQL seperti status = 'PENDING',
        // jadi representasi tersimpan wajib persis nama enum-nya.
        dao.insertIgnoringDuplicate(event("evt_a"))

        db.query("SELECT status FROM events WHERE eventId = 'evt_a'", null).use { c ->
            assertTrue(c.moveToFirst())
            assertEquals("PENDING", c.getString(0))
        }
    }
}
