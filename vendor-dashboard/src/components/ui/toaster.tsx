"use client";

import { Toaster as HotToaster } from "react-hot-toast";

/**
 * Pembungkus tema react-hot-toast -- menggantikan sonner sepenuhnya (lihat
 * git log). Animasi masuk/keluar bawaan react-hot-toast (slide + fade,
 * ~350ms) sudah halus tanpa konfigurasi tambahan, jadi tidak ada logic
 * animasi kustom di sini.
 *
 * Warna dibaca langsung dari CSS custom property tema (--popover, dst) via
 * `style` -- bukan lewat prop `theme` seperti sonner (yang butuh next-themes
 * untuk tahu mode aktif). react-hot-toast tidak punya konsep tema sama
 * sekali, cuma merender style yang diberi, jadi ia otomatis ikut light/dark
 * begitu saja karena browser me-resolve var(--popover) sesuai class `.dark`
 * yang aktif saat itu -- tidak perlu JS mendeteksi tema secara terpisah.
 */
export function Toaster() {
  return (
    <HotToaster
      position="bottom-right"
      toastOptions={{
        duration: 4000,
        style: {
          background: "var(--popover)",
          color: "var(--popover-foreground)",
          border: "1px solid var(--border)",
          borderRadius: "var(--radius)",
          fontSize: "0.875rem",
          boxShadow: "0 4px 12px rgb(0 0 0 / 0.08)",
        },
        success: {
          iconTheme: { primary: "#16a34a", secondary: "var(--popover)" },
        },
        error: {
          iconTheme: { primary: "#dc2626", secondary: "var(--popover)" },
        },
      }}
    />
  );
}
