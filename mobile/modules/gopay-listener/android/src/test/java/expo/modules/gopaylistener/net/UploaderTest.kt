package expo.modules.gopaylistener.net

import expo.modules.gopaylistener.Signer
import expo.modules.gopaylistener.config.BridgeConfig
import expo.modules.gopaylistener.db.EventEntity
import expo.modules.gopaylistener.db.EventStatus
import okhttp3.mockwebserver.MockResponse
import okhttp3.mockwebserver.MockWebServer
import org.json.JSONObject
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config

/**
 * Menguji bentuk request yang BENAR-BENAR dikirim ke jaringan.
 *
 * Yang paling penting di sini: tanda tangan dihitung atas byte body yang sama
 * persis dengan yang terkirim. Bug di titik itu hanya muncul saat request
 * sungguhan, dengan pesan invalid_signature yang tidak menjelaskan apa pun.
 */
@RunWith(RobolectricTestRunner::class)
@Config(sdk = [34])
class UploaderTest {

    private lateinit var server: MockWebServer
    private lateinit var cfg: BridgeConfig

    private val event = EventEntity(
        eventId = "evt_3f9a2c8b1d4e5f6a7b8c9d0e1f2a3b4c",
        packageName = "com.gojek.gopaymerchant",
        title = "Pembayaran QRIS statis diterima",
        text = "Rp 1 di AKBAR RAYYAN AL GHIFARI, Digital & Kreatif.",
        bigText = null,
        amountHint = 1L,
        postedAt = 1_789_051_832_829L,
        receivedAt = 1_789_051_833_000L,
        status = EventStatus.SENDING,
        attemptCount = 0,
        lastError = null,
        backendStatus = null,
        sentAt = null,
    )

    @Before
    fun setUp() {
        server = MockWebServer()
        server.start()
        cfg = BridgeConfig(
            backendUrl = server.url("/api/v1").toString().trimEnd('/'),
            deviceId = "dev_01ABC",
            deviceSecret = "secret-untuk-test",
        )
    }

    @After
    fun tearDown() = server.shutdown()

    @Test
    fun `mengirim POST ke callback dengan ketiga header autentikasi`() {
        server.enqueue(MockResponse().setResponseCode(200).setBody("""{"success":true,"status":"accepted"}"""))

        Uploader.send(cfg, event, appVersion = "1.4.2", nowSeconds = 1789036200L)

        val req = server.takeRequest()
        assertEquals("POST", req.method)
        assertEquals("/api/v1/events", req.path)
        assertEquals("dev_01ABC", req.getHeader("X-Device-Id"))
        assertEquals("1789036200", req.getHeader("X-Timestamp"))
        assertTrue(req.getHeader("X-Signature")!!.matches(Regex("^[0-9a-f]{64}$")))
    }

    @Test
    fun `mengirim header versi aplikasi`() {
        server.enqueue(MockResponse().setResponseCode(200).setBody("""{"success":true,"status":"accepted"}"""))

        Uploader.send(cfg, event, appVersion = "1.4.2", nowSeconds = 1789036200L)

        assertEquals("1.4.2", server.takeRequest().getHeader("X-App-Version"))
    }

    @Test
    fun `versi aplikasi tidak ikut ditandatangani`() {
        // Header ini bersifat informatif dan tidak masuk signing string.
        // Kalau ia ikut ditandatangani, backend harus mengetahui versinya
        // lebih dulu untuk memverifikasi — dan itu mustahil.
        server.enqueue(MockResponse().setResponseCode(200).setBody("""{"success":true,"status":"accepted"}"""))

        Uploader.send(cfg, event, appVersion = "9.9.9", nowSeconds = 1789036200L)

        val req = server.takeRequest()
        val expected = Signer.sign(
            cfg.deviceSecret,
            Signer.signingString(cfg.deviceId, 1789036200L, req.body.readByteArray())
        )
        assertEquals(expected, req.getHeader("X-Signature"))
    }

    @Test
    fun `tanda tangan dihitung atas byte body yang persis terkirim`() {
        server.enqueue(MockResponse().setResponseCode(200).setBody("""{"success":true,"status":"accepted"}"""))

        Uploader.send(cfg, event, appVersion = "1.4.2", nowSeconds = 1789036200L)

        val req = server.takeRequest()
        val sentBytes = req.body.readByteArray()

        val expected = Signer.sign(
            cfg.deviceSecret,
            Signer.signingString(cfg.deviceId, 1789036200L, sentBytes)
        )
        assertEquals(
            "tanda tangan harus cocok dengan byte yang benar-benar dikirim, bukan hasil susun ulang",
            expected,
            req.getHeader("X-Signature")
        )
    }

    @Test
    fun `payload memuat teks mentah dan amount_hint`() {
        server.enqueue(MockResponse().setResponseCode(200).setBody("""{"success":true,"status":"accepted"}"""))

        Uploader.send(cfg, event, appVersion = "1.4.2", nowSeconds = 1789036200L)

        val body = JSONObject(server.takeRequest().body.readUtf8())
        assertEquals("evt_3f9a2c8b1d4e5f6a7b8c9d0e1f2a3b4c", body.getString("event_id"))
        assertEquals("dev_01ABC", body.getString("device_id"))
        assertEquals("gopay", body.getString("source"))
        assertEquals(1L, body.getLong("amount_hint"))

        val n = body.getJSONObject("notification")
        assertEquals("com.gojek.gopaymerchant", n.getString("package_name"))
        assertEquals("Pembayaran QRIS statis diterima", n.getString("title"))
        assertEquals("Rp 1 di AKBAR RAYYAN AL GHIFARI, Digital & Kreatif.", n.getString("text"))
        assertEquals(1_789_051_832_829L, n.getLong("posted_at"))
        assertTrue("big_text null harus hadir sebagai key", n.isNull("big_text"))
    }

    @Test
    fun `received_at berformat ISO 8601 dengan offset zona waktu`() {
        server.enqueue(MockResponse().setResponseCode(200).setBody("""{"success":true,"status":"accepted"}"""))

        Uploader.send(cfg, event, appVersion = "1.4.2", nowSeconds = 1789036200L)

        val receivedAt = JSONObject(server.takeRequest().body.readUtf8()).getString("received_at")
        assertTrue(
            "received_at = $receivedAt",
            receivedAt.matches(Regex("""^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}([+-]\d{2}:\d{2}|Z)$"""))
        )
    }

    @Test
    fun `amount_hint null dikirim sebagai null, bukan dihilangkan`() {
        server.enqueue(MockResponse().setResponseCode(200).setBody("""{"success":true,"status":"accepted"}"""))

        Uploader.send(cfg, event.copy(amountHint = null), appVersion = "1.4.2", nowSeconds = 1789036200L)

        val body = JSONObject(server.takeRequest().body.readUtf8())
        assertTrue(body.has("amount_hint"))
        assertTrue(body.isNull("amount_hint"))
    }

    @Test
    fun `response duplicate diperlakukan sebagai terkirim`() {
        server.enqueue(MockResponse().setResponseCode(200).setBody("""{"success":true,"status":"duplicate"}"""))

        val out = Uploader.send(cfg, event, appVersion = "1.4.2", nowSeconds = 1789036200L)

        assertTrue(out is UploadOutcome.Sent)
        assertEquals("duplicate", (out as UploadOutcome.Sent).backendStatus)
    }

    @Test
    fun `backend tidak dapat dihubungi menjadi Retry, bukan Failed`() {
        server.shutdown()

        val out = Uploader.send(cfg, event, appVersion = "1.4.2", nowSeconds = 1789036200L)

        assertTrue("jaringan mati selalu dapat dipulihkan", out is UploadOutcome.Retry)
        assertEquals("network", (out as UploadOutcome.Retry).error)
    }

    @Test
    fun `heartbeat mengirim kondisi perangkat dan ditandatangani`() {
        server.enqueue(MockResponse().setResponseCode(200).setBody("""{"success":true,"status":"ok"}"""))

        val ok = Uploader.heartbeat(
            cfg = cfg,
            appVersion = "1.4.2",
            report = Uploader.HeartbeatReport(
                androidVersion = "13",
                listenerConnected = false,
                pendingCount = 3,
                failedCount = 1,
            ),
            nowSeconds = 1789036200L,
        )

        assertTrue(ok)

        val req = server.takeRequest()
        assertEquals("POST", req.method)
        assertEquals("/api/v1/devices/heartbeat", req.path)
        assertEquals("1.4.2", req.getHeader("X-App-Version"))

        val sentBytes = req.body.readByteArray()
        assertEquals(
            "heartbeat harus ditandatangani atas byte body yang persis terkirim",
            Signer.sign(cfg.deviceSecret, Signer.signingString(cfg.deviceId, 1789036200L, sentBytes)),
            req.getHeader("X-Signature"),
        )

        val body = JSONObject(String(sentBytes))
        assertEquals("13", body.getString("android_version"))
        assertEquals(false, body.getBoolean("listener_connected"))
        assertEquals(3, body.getInt("pending_count"))
        assertEquals(1, body.getInt("failed_count"))
    }

    @Test
    fun `heartbeat melaporkan gagal saat backend tidak dapat dihubungi`() {
        server.shutdown()

        val ok = Uploader.heartbeat(
            cfg = cfg,
            appVersion = "1.4.2",
            report = Uploader.HeartbeatReport("13", true, 0, 0),
            nowSeconds = 1789036200L,
        )

        assertEquals(false, ok)
    }

    @Test
    fun `heartbeat melaporkan gagal saat backend menolak`() {
        server.enqueue(MockResponse().setResponseCode(401).setBody("""{"error":"invalid_signature"}"""))

        val ok = Uploader.heartbeat(
            cfg = cfg,
            appVersion = "1.4.2",
            report = Uploader.HeartbeatReport("13", true, 0, 0),
            nowSeconds = 1789036200L,
        )

        assertEquals(false, ok)
    }

    @Test
    fun `health membaca jam server`() {
        server.enqueue(MockResponse().setResponseCode(200).setBody("""{"status":"ok","server_time":1789036200}"""))

        val (reachable, serverTime) = Uploader.health(cfg.backendUrl)

        assertTrue(reachable)
        assertEquals(1789036200L, serverTime)
        assertEquals("/api/v1/health", server.takeRequest().path)
    }

    @Test
    fun `health melaporkan tidak terhubung saat backend mati`() {
        server.shutdown()

        val (reachable, serverTime) = Uploader.health(cfg.backendUrl)

        assertEquals(false, reachable)
        assertEquals(null, serverTime)
    }

    @Test
    fun `deviceMe menandatangani body kosong dan mengembalikan nama device`() {
        server.enqueue(MockResponse().setResponseCode(200).setBody("""{"device_id":"dev_01ABC","name":"HP GoPay Utama"}"""))

        val result = Uploader.deviceMe(cfg, appVersion = "1.4.2", nowSeconds = 1789036200L)

        assertEquals(true, result["ok"])
        assertEquals("HP GoPay Utama", result["deviceName"])

        val req = server.takeRequest()
        assertEquals("GET", req.method)
        assertEquals(
            Signer.sign(cfg.deviceSecret, Signer.signingString(cfg.deviceId, 1789036200L, ByteArray(0))),
            req.getHeader("X-Signature")
        )
    }

    @Test
    fun `deviceMe meneruskan kode error backend apa adanya`() {
        server.enqueue(MockResponse().setResponseCode(401).setBody("""{"success":false,"error":"clock_skew"}"""))

        val result = Uploader.deviceMe(cfg, appVersion = "1.4.2", nowSeconds = 1789036200L)

        assertEquals(false, result["ok"])
        assertEquals("clock_skew", result["error"])
    }
}
