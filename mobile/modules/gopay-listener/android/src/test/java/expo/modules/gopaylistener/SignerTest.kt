package expo.modules.gopaylistener

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotEquals
import org.junit.Test

/**
 * Vektor uji di berkas ini DIHASILKAN dari implementasi Go di
 * backend/internal/auth, bukan dikarang.
 *
 * Dua implementasi HMAC di dua bahasa yang sama-sama terlihat benar tetapi
 * tidak sepakat hanya akan gagal saat request sungguhan, dengan pesan
 * invalid_signature yang tidak menjelaskan apa pun.
 */
class SignerTest {

    private val secret = "secret-untuk-test"

    @Test
    fun `signing string untuk body kosong cocok dengan backend`() {
        // sha256("") = e3b0c442...b855
        assertEquals(
            "dev_01ABC\n1789036200\n" +
                "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
            Signer.signingString("dev_01ABC", 1789036200L, ByteArray(0))
        )
    }

    @Test
    fun `tanda tangan body kosong cocok dengan vektor Go`() {
        val s = Signer.signingString("dev_01ABC", 1789036200L, ByteArray(0))
        assertEquals("a86ca9056866b18aba88c5719d6692777e926601e9d4c856c385026d66d90dc3", Signer.sign(secret, s))
    }

    @Test
    fun `signing string dengan body cocok dengan vektor Go`() {
        val body = """{"event_id":"evt_abc"}""".toByteArray(Charsets.UTF_8)
        assertEquals(
            "dev_01ABC\n1789036200\n" +
                "fa3bb500e65a28245d2f49ae4def090e112f967b9392673a96bc9a58a1da4add",
            Signer.signingString("dev_01ABC", 1789036200L, body)
        )
    }

    @Test
    fun `tanda tangan dengan body cocok dengan vektor Go`() {
        val body = """{"event_id":"evt_abc"}""".toByteArray(Charsets.UTF_8)
        val s = Signer.signingString("dev_01ABC", 1789036200L, body)
        assertEquals("a2876b7e01eceb6d99819ed2183ed6e7655ff03950e5d98553e3b7b3c5aaec86", Signer.sign(secret, s))
    }

    @Test
    fun `body yang berubah satu byte menghasilkan tanda tangan berbeda`() {
        val a = Signer.sign(secret, Signer.signingString("dev_01ABC", 1789036200L, """{"a":1}""".toByteArray()))
        val b = Signer.sign(secret, Signer.signingString("dev_01ABC", 1789036200L, """{"a":2}""".toByteArray()))
        assertNotEquals(a, b)
    }

    @Test
    fun `secret yang berbeda menghasilkan tanda tangan berbeda`() {
        val s = Signer.signingString("dev_01ABC", 1789036200L, ByteArray(0))
        assertNotEquals(Signer.sign(secret, s), Signer.sign("secret-lain", s))
    }

    @Test
    fun `tanda tangan berupa 64 karakter heksadesimal huruf kecil`() {
        val sig = Signer.sign(secret, Signer.signingString("dev_01ABC", 1789036200L, ByteArray(0)))
        assertEquals(64, sig.length)
        assertEquals(sig, sig.lowercase())
    }
}
