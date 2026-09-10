package expo.modules.gopaylistener

/**
 * Mengekstrak nominal dari teks notifikasi.
 *
 * Hasilnya DISPLAY-ONLY. Backend melakukan ekstraksi otoritatifnya sendiri dari
 * teks mentah; parser ini boleh salah tanpa merusak apa pun.
 *
 * Prinsipnya: jangan pernah menebak. Angka yang tidak didahului "Rp" bukan nominal.
 */
object AmountParser {

    // Hanya menerima pengelompokan yang berbentuk wajar: "1", "25.000",
    // "1.234.567", "25,000". Rangkaian seperti "25.00" atau "1.2345" tidak
    // cocok sebagai kelompok ribuan, sehingga angka yang kebetulan berdekatan
    // dengan "Rp" tidak ikut terambil utuh.
    private val PATTERN = Regex("""Rp\s?(\d{1,3}(?:[.,]\d{3})*|\d+)""")

    fun parse(text: String?): Long? {
        if (text.isNullOrEmpty()) return null

        val match = PATTERN.find(text) ?: return null
        val digits = match.groupValues[1].replace(".", "").replace(",", "")
        if (digits.isEmpty()) return null

        return digits.toLongOrNull()
    }
}
