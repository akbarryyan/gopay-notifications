package expo.modules.gopaylistener

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test

class AmountParserTest {

    @Test
    fun `mengurai notifikasi transfer masuk sungguhan dari perangkat`() {
        assertEquals(
            1L,
            AmountParser.parse("Rp1 dari icaangg udah masuk ke GoPay kamu.")
        )
    }

    @Test
    fun `mengurai pemisah ribuan bergaya Indonesia`() {
        assertEquals(25_000L, AmountParser.parse("Pembayaran diterima Rp25.000"))
        assertEquals(1_234_567L, AmountParser.parse("Saldo kamu Rp1.234.567"))
    }

    @Test
    fun `mengurai varian spasi dan koma`() {
        assertEquals(25_000L, AmountParser.parse("Rp 25.000"))
        assertEquals(25_000L, AmountParser.parse("Rp25,000"))
    }

    @Test
    fun `mengabaikan sen di belakang koma`() {
        assertEquals(25_000L, AmountParser.parse("Rp25.000,50"))
    }

    @Test
    fun `mengambil kemunculan pertama bila ada lebih dari satu`() {
        assertEquals(25_000L, AmountParser.parse("Rp25.000 dari Rp50.000"))
    }

    @Test
    fun `menghasilkan null bila tidak ada Rp`() {
        assertNull(AmountParser.parse("Transfer masuk"))
        assertNull(AmountParser.parse("Ref 12345678"))
        assertNull(AmountParser.parse("Cashback 50% untuk kamu"))
    }

    @Test
    fun `menghasilkan null untuk masukan kosong atau null`() {
        assertNull(AmountParser.parse(null))
        assertNull(AmountParser.parse(""))
        assertNull(AmountParser.parse("Rp"))
        assertNull(AmountParser.parse("Rp "))
    }

    @Test
    fun `tidak memperlakukan angka biasa sebagai nominal`() {
        assertNull(AmountParser.parse("Diskon 25.000 poin menanti"))
        assertNull(AmountParser.parse("Kode OTP kamu 123456"))
    }
}
