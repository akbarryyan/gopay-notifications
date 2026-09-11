package expo.modules.gopaylistener.config

import android.content.Context
import androidx.test.core.app.ApplicationProvider
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config

/**
 * Menguji logika Settings dengan SharedPreferences biasa.
 *
 * Enkripsinya sendiri TIDAK diuji di sini: Robolectric tidak mengemulasi
 * Android Keystore, sehingga EncryptedSharedPreferences hanya dapat
 * diverifikasi di perangkat sungguhan.
 */
@RunWith(RobolectricTestRunner::class)
@Config(sdk = [34])
class SettingsTest {

    private lateinit var settings: Settings

    @Before
    fun setUp() {
        val ctx = ApplicationProvider.getApplicationContext<Context>()
        val prefs = ctx.getSharedPreferences("settings-test", Context.MODE_PRIVATE)
        prefs.edit().clear().commit()
        settings = Settings(prefs)
    }

    @Test
    fun `daftar package default berisi tepat satu entri merchant`() {
        // Tepat satu. com.gojek.gopay melaporkan pembayaran yang sama, sehingga
        // memantau keduanya membuat satu pembayaran terhitung dua kali.
        assertEquals(listOf("com.gojek.gopaymerchant"), settings.monitoredPackages)
    }

    @Test
    fun `daftar kata-diabaikan default kosong`() {
        assertTrue(
            "keamanan dijamin allowlist di backend, bukan filter di HP",
            settings.ignoreKeywords.isEmpty()
        )
    }

    @Test
    fun `config null selama salah satu nilai wajib kosong`() {
        assertNull(settings.config())

        settings.backendUrl = "https://contoh.test/api/v1"
        assertNull(settings.config())

        settings.deviceId = "dev_01ABC"
        assertNull(settings.config())

        settings.deviceSecret = "rahasia"
        assertNotNull(settings.config())
    }

    @Test
    fun `config mengembalikan nilai yang tersimpan`() {
        settings.backendUrl = "https://contoh.test/api/v1"
        settings.deviceId = "dev_01ABC"
        settings.deviceSecret = "rahasia"

        val cfg = settings.config()!!
        assertEquals("https://contoh.test/api/v1", cfg.backendUrl)
        assertEquals("dev_01ABC", cfg.deviceId)
        assertEquals("rahasia", cfg.deviceSecret)
    }

    @Test
    fun `backendUrl dinormalkan tanpa garis miring di belakang`() {
        settings.backendUrl = "  https://contoh.test/api/v1/  "
        assertEquals("https://contoh.test/api/v1", settings.backendUrl)
    }

    @Test
    fun `daftar package dapat diubah`() {
        settings.monitoredPackages = listOf("com.gojek.gopaymerchant", "com.example.sumberlain")
        assertEquals(
            listOf("com.gojek.gopaymerchant", "com.example.sumberlain"),
            settings.monitoredPackages
        )
    }

    @Test
    fun `daftar mengabaikan spasi dan entri kosong`() {
        settings.ignoreKeywords = listOf("promo", "", "  cashback  ")
        assertEquals(listOf("promo", "cashback"), settings.ignoreKeywords)
    }

    @Test
    fun `discovery mati secara default`() {
        assertFalse(settings.discoveryActive(nowMs = 1_000L))
    }

    @Test
    fun `discovery aktif hanya sampai batas waktunya`() {
        settings.discoveryUntilMs = 10_000L

        assertTrue(settings.discoveryActive(nowMs = 9_999L))
        assertFalse(
            "discovery harus mati sendiri setelah lewat batas",
            settings.discoveryActive(nowMs = 10_001L)
        )
    }

    @Test
    fun `clearAll mengembalikan seluruh nilai ke default`() {
        settings.backendUrl = "https://contoh.test/api/v1"
        settings.deviceSecret = "rahasia"
        settings.monitoredPackages = listOf("com.example.lain")

        settings.clearAll()

        assertNull(settings.config())
        assertEquals(listOf("com.gojek.gopaymerchant"), settings.monitoredPackages)
    }
}
