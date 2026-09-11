package expo.modules.gopaylistener

import android.os.Bundle

/**
 * Jalur satu arah dari service ke module, dipakai hanya untuk menyegarkan UI
 * saat aplikasi sedang dibuka.
 *
 * Murni kosmetik. Bila UI mati tidak ada observer terdaftar dan tidak ada yang
 * hilang — pengiriman tetap berjalan lewat WorkManager.
 *
 * Observer disimpan di objek statis, sehingga module WAJIB mendaftarkannya
 * lewat WeakReference. Tanpa itu instance module tertahan di memori setelah
 * UI ditutup, dan kebocorannya baru terasa setelah aplikasi dibuka-tutup
 * berkali-kali.
 */
object CaptureBus {

    private val observers = mutableSetOf<(Bundle) -> Unit>()

    @Synchronized
    fun register(observer: (Bundle) -> Unit) {
        observers.add(observer)
    }

    @Synchronized
    fun unregister(observer: (Bundle) -> Unit) {
        observers.remove(observer)
    }

    @Synchronized
    fun emit(payload: Bundle) {
        observers.forEach { it(payload) }
    }
}
