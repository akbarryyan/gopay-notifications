package expo.modules.gopaylistener

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotEquals
import org.junit.Assert.assertTrue
import org.junit.Test

class EventIdBuilderTest {

    private val pkg = "com.gojek.gopay"
    private val title = "Transfer masuk"
    private val text = "Rp1 dari icaangg udah masuk ke GoPay kamu."
    private val whenMs = 1_789_051_832_829L

    private val shape = Regex("^evt_[0-9a-f]{32}$")

    @Test
    fun `berbentuk evt_ diikuti 32 heksadesimal`() {
        val id = EventIdBuilder.build(pkg, title, text, whenMs)
        assertTrue("id = $id", shape.matches(id))
    }

    @Test
    fun `deterministik untuk masukan yang sama`() {
        assertEquals(
            EventIdBuilder.build(pkg, title, text, whenMs),
            EventIdBuilder.build(pkg, title, text, whenMs)
        )
    }

    @Test
    fun `berubah bila satu bahan berubah`() {
        val base = EventIdBuilder.build(pkg, title, text, whenMs)

        assertNotEquals(base, EventIdBuilder.build("com.gojek.app", title, text, whenMs))
        assertNotEquals(base, EventIdBuilder.build(pkg, "Transfer keluar", text, whenMs))
        assertNotEquals(base, EventIdBuilder.build(pkg, title, text + " ", whenMs))
        assertNotEquals(base, EventIdBuilder.build(pkg, title, text, whenMs + 1))
    }

    @Test
    fun `title null dan string kosong diperlakukan sama`() {
        val nullTitle = EventIdBuilder.build(pkg, null, text, whenMs)
        val emptyTitle = EventIdBuilder.build(pkg, "", text, whenMs)

        assertTrue(shape.matches(nullTitle))
        assertEquals(
            "keduanya berarti tidak ada judul",
            nullTitle,
            emptyTitle
        )
    }

    @Test
    fun `dua transfer identik pada waktu berbeda menghasilkan id berbeda`() {
        val a = EventIdBuilder.build(pkg, title, text, whenMs)
        val b = EventIdBuilder.build(pkg, title, text, whenMs + 1000)
        assertNotEquals("dua transfer sungguhan tidak boleh dianggap satu", a, b)
    }
}
