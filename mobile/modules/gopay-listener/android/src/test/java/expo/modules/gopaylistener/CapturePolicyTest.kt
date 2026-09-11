package expo.modules.gopaylistener

import org.junit.Assert.assertEquals
import org.junit.Test

class CapturePolicyTest {

    private val merchant = "com.gojek.gopaymerchant"
    private val monitored = listOf(merchant)
    private val title = "Pembayaran QRIS statis diterima"
    private val text = "Rp 1 di AKBAR RAYYAN AL GHIFARI, Digital & Kreatif."

    private fun decide(
        pkg: String = merchant,
        t: String? = title,
        x: String? = text,
        packages: List<String> = monitored,
        ignore: List<String> = emptyList(),
    ) = CapturePolicy.decide(pkg, t, x, packages, ignore)

    @Test
    fun `notifikasi merchant ditangkap`() {
        assertEquals(CaptureDecision.Capture, decide())
    }

    @Test
    fun `aplikasi lain dilewati tanpa dicatat`() {
        assertEquals(CaptureDecision.Skip, decide(pkg = "com.whatsapp"))
    }

    @Test
    fun `aplikasi GoPay pribadi dilewati selama tidak dipantau`() {
        // Keduanya melaporkan pembayaran yang SAMA. Memantau keduanya membuat
        // satu pembayaran menghasilkan dua event_id dan terhitung dua kali.
        assertEquals(CaptureDecision.Skip, decide(pkg = "com.gojek.gopay"))
    }

    @Test
    fun `kata-diabaikan menandai IGNORED, bukan membuang`() {
        assertEquals(
            CaptureDecision.Ignore,
            decide(t = "Cashback menantimu", x = "Rp 10.000 cashback", ignore = listOf("cashback"))
        )
    }

    @Test
    fun `kata-diabaikan tidak peka huruf besar kecil`() {
        assertEquals(
            CaptureDecision.Ignore,
            decide(t = "PROMO SPESIAL", x = null, ignore = listOf("promo"))
        )
    }

    @Test
    fun `kata-diabaikan dicocokkan ke judul maupun isi`() {
        assertEquals(CaptureDecision.Ignore, decide(t = "ada voucher", x = null, ignore = listOf("voucher")))
        assertEquals(CaptureDecision.Ignore, decide(t = null, x = "ada voucher", ignore = listOf("voucher")))
    }

    @Test
    fun `daftar kata-diabaikan kosong tidak menyaring apa pun`() {
        assertEquals(CaptureDecision.Capture, decide(ignore = emptyList()))
    }

    @Test
    fun `entri kosong di daftar tidak menyaring semua notifikasi`() {
        // String kosong cocok dengan teks apa pun. Tanpa penjagaan, satu entri
        // kosong akan membuat SELURUH pembayaran tersaring diam-diam.
        assertEquals(CaptureDecision.Capture, decide(ignore = listOf("")))
    }

    @Test
    fun `judul dan isi null tetap ditangkap bila package cocok`() {
        assertEquals(CaptureDecision.Capture, decide(t = null, x = null))
    }

    @Test
    fun `daftar pantau kosong berarti tidak ada yang ditangkap`() {
        assertEquals(CaptureDecision.Skip, decide(packages = emptyList()))
    }

    @Test
    fun `memakai Notification when bila tersedia`() {
        assertEquals(1_789_051_832_829L, CapturePolicy.effectiveWhen(1_789_051_832_829L, 9_999L))
    }

    @Test
    fun `jatuh ke postTime hanya bila when bernilai nol`() {
        assertEquals(9_999L, CapturePolicy.effectiveWhen(0L, 9_999L))
    }
}
