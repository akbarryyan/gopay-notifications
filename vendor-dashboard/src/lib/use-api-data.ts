"use client";

import { useCallback, useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { ApiError, logout } from "@/lib/api";

interface State<T> {
  data: T | null;
  error: string | null;
  loading: boolean;
}

/**
 * Pola ambil-data yang dipakai berulang di Overview, Devices, dan Events:
 * loading di awal, error yang bisa dicoba ulang, dan penanganan sesi yang
 * sudah tidak valid.
 *
 * `fetcher` harus stabil antar render (referensi ke fungsi top-level seperti
 * `getDevices`, atau hasil `useCallback` di sisi pemanggil bila bergantung
 * pada state seperti nomor halaman) — react-hooks/exhaustive-deps menuntut
 * daftar dependensi berupa array literal, sehingga hook ini sengaja tidak
 * menerima parameter `deps` terpisah.
 */
export function useApiData<T>(fetcher: () => Promise<T>) {
  const router = useRouter();
  const [state, setState] = useState<State<T>>({ data: null, error: null, loading: true });

  const load = useCallback(() => {
    setState((s) => ({ ...s, loading: true, error: null }));
    fetcher()
      .then((data) => setState({ data, error: null, loading: false }))
      .catch((err) => {
        if (err instanceof ApiError && err.status === 401) {
          // Cookie sesi HttpOnly tidak bisa dihapus lewat JavaScript. Tanpa
          // memanggil logout() dulu, proxy.ts masih melihat cookie itu ADA
          // (walau sudah tidak valid) dan langsung memantulkan navigasi ke
          // /login kembali ke halaman ini — pantulan tanpa henti.
          logout().finally(() => router.push("/login"));
          return;
        }
        const message = err instanceof Error ? err.message : "Terjadi kesalahan tidak terduga.";
        setState({ data: null, error: message, loading: false });
      });
  }, [fetcher, router]);

  useEffect(() => {
    // react-hooks/set-state-in-effect (ditujukan untuk model React Compiler
    // / Suspense) menandai pola fetch-saat-mount ini sebagai berisiko
    // cascading render. Untuk dashboard yang murni memuat data lewat client
    // fetch — tanpa Server Components untuk data ini — pola ini memang benar
    // dan tidak ada gantinya yang lebih sederhana pada skala MVP ini.
    // eslint-disable-next-line react-hooks/set-state-in-effect
    load();
  }, [load]);

  return { ...state, reload: load };
}
