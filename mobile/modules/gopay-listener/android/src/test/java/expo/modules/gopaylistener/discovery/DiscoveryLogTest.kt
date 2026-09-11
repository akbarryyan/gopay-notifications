package expo.modules.gopaylistener.discovery

import android.content.Context
import androidx.test.core.app.ApplicationProvider
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config

@RunWith(RobolectricTestRunner::class)
@Config(sdk = [34])
class DiscoveryLogTest {

    private lateinit var ctx: Context

    @Before
    fun setUp() {
        ctx = ApplicationProvider.getApplicationContext()
        DiscoveryLog.clear(ctx)
    }

    @Test
    fun `mencatat entri dan mengembalikannya`() {
        DiscoveryLog.record(ctx, "com.gojek.gopaymerchant", "Pembayaran QRIS statis diterima", 1000L)

        val got = DiscoveryLog.entries(ctx).single()
        assertEquals("com.gojek.gopaymerchant", got.packageName)
        assertEquals("Pembayaran QRIS statis diterima", got.title)
        assertEquals(1000L, got.seenAt)
    }

    @Test
    fun `terbaru muncul lebih dulu`() {
        DiscoveryLog.record(ctx, "com.satu", "lama", 1000L)
        DiscoveryLog.record(ctx, "com.dua", "baru", 2000L)

        assertEquals("com.dua", DiscoveryLog.entries(ctx)[0].packageName)
    }

    @Test
    fun `dipotong pada 100 entri`() {
        for (i in 1..120) {
            DiscoveryLog.record(ctx, "com.contoh.$i", "judul $i", i.toLong())
        }

        val got = DiscoveryLog.entries(ctx)
        assertEquals(100, got.size)
        assertEquals("entri terbaru harus dipertahankan", "com.contoh.120", got[0].packageName)
    }

    @Test
    fun `clear mengosongkan catatan`() {
        DiscoveryLog.record(ctx, "com.satu", "judul", 1000L)
        DiscoveryLog.clear(ctx)

        assertTrue(DiscoveryLog.entries(ctx).isEmpty())
    }

    @Test
    fun `menangani judul kosong`() {
        DiscoveryLog.record(ctx, "com.satu", "", 1000L)

        assertEquals("", DiscoveryLog.entries(ctx).single().title)
    }
}
