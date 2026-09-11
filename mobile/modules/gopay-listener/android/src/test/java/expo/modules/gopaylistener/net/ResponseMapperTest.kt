package expo.modules.gopaylistener.net

import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config

/**
 * Mengunci seluruh baris tabel kode → tindakan di api-contract.md §4.6.
 *
 * Robolectric dipakai hanya karena org.json adalah kelas Android;
 * tidak ada jaringan, database, maupun perangkat yang tersentuh.
 */
@RunWith(RobolectricTestRunner::class)
@Config(sdk = [34])
class ResponseMapperTest {

    @Test
    fun `200 accepted menjadi Sent`() {
        val out = ResponseMapper.map(200, """{"success":true,"status":"accepted"}""", null)
        assertTrue(out is UploadOutcome.Sent)
        assertEquals("accepted", (out as UploadOutcome.Sent).backendStatus)
    }

    @Test
    fun `200 duplicate juga menjadi Sent`() {
        val out = ResponseMapper.map(200, """{"success":true,"status":"duplicate"}""", null)
        assertTrue("duplicate adalah sukses, bukan error", out is UploadOutcome.Sent)
        assertEquals("duplicate", (out as UploadOutcome.Sent).backendStatus)
    }

    @Test
    fun `200 tanpa status yang dikenal menjadi Failed`() {
        assertTrue(ResponseMapper.map(200, """{"success":true}""", null) is UploadOutcome.Failed)
    }

    @Test
    fun `200 dengan body bukan JSON menjadi Failed`() {
        // Terjadi saat proxy menyisipkan halaman error dengan status 200.
        assertTrue(ResponseMapper.map(200, "<html>proxy error</html>", null) is UploadOutcome.Failed)
    }

    @Test
    fun `400 menjadi Failed tanpa retry`() {
        val out = ResponseMapper.map(400, """{"error":"invalid_payload"}""", null)
        assertTrue(out is UploadOutcome.Failed)
        assertEquals("invalid_payload", (out as UploadOutcome.Failed).error)
    }

    @Test
    fun `401 invalid_signature menjadi Failed tanpa retry`() {
        val out = ResponseMapper.map(401, """{"error":"invalid_signature"}""", null)
        assertTrue(out is UploadOutcome.Failed)
        assertEquals("invalid_signature", (out as UploadOutcome.Failed).error)
    }

    @Test
    fun `401 clock_skew dapat dibedakan dari invalid_signature`() {
        val out = ResponseMapper.map(401, """{"error":"clock_skew","server_time":1789036200}""", null)
        assertTrue(out is UploadOutcome.Failed)
        assertEquals(
            "keduanya 401 tapi menuntut tindakan berbeda: secret salah versus jam perlu disetel",
            "clock_skew",
            (out as UploadOutcome.Failed).error
        )
    }

    @Test
    fun `403 device_disabled menjadi Failed tanpa retry`() {
        val out = ResponseMapper.map(403, """{"error":"device_disabled"}""", null)
        assertTrue(out is UploadOutcome.Failed)
        assertEquals("device_disabled", (out as UploadOutcome.Failed).error)
    }

    @Test
    fun `429 menjadi Retry dan menghormati Retry-After`() {
        val out = ResponseMapper.map(429, null, "120")
        assertTrue(out is UploadOutcome.Retry)
        assertEquals(120L, (out as UploadOutcome.Retry).retryAfterSeconds)
    }

    @Test
    fun `429 tanpa Retry-After tetap Retry`() {
        val out = ResponseMapper.map(429, null, null)
        assertTrue(out is UploadOutcome.Retry)
        assertEquals(null, (out as UploadOutcome.Retry).retryAfterSeconds)
    }

    @Test
    fun `Retry-After berformat tanggal diabaikan, bukan bikin crash`() {
        val out = ResponseMapper.map(429, null, "Wed, 21 Oct 2026 07:28:00 GMT")
        assertTrue(out is UploadOutcome.Retry)
        assertEquals(null, (out as UploadOutcome.Retry).retryAfterSeconds)
    }

    @Test
    fun `5xx menjadi Retry`() {
        for (code in listOf(500, 502, 503, 504)) {
            assertTrue("kode $code seharusnya Retry", ResponseMapper.map(code, null, null) is UploadOutcome.Retry)
        }
    }

    @Test
    fun `kode 4xx tak terduga menjadi Failed`() {
        assertTrue(ResponseMapper.map(418, null, null) is UploadOutcome.Failed)
    }
}
