"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import QRCode from "react-qr-code";
import type { GopayDeviceInfo } from "@/lib/merchant-auth";

function readPairingDevice(): GopayDeviceInfo | null {
  if (typeof window === "undefined") return null;
  const raw = sessionStorage.getItem("gopay_pairing_device");
  if (!raw) return null;
  try {
    return JSON.parse(raw) as GopayDeviceInfo;
  } catch {
    return null;
  }
}

export default function PairDevicePage() {
  const router = useRouter();
  const [device] = useState<GopayDeviceInfo | null>(readPairingDevice);

  useEffect(() => {
    // Sekali pakai -- kalau halaman ini di-refresh, QR tidak bisa
    // ditampilkan ulang (device secret memang sengaja tidak pernah
    // disimpan, lihat spec 2026-09-17-whuzpay-pg-unified-onboarding-design.md §6).
    sessionStorage.removeItem("gopay_pairing_device");
    if (!device) {
      router.replace("/dashboard");
    }
  }, [device, router]);

  if (!device) return null;

  const qrPayload = JSON.stringify({
    v: 1,
    backend_url: device.backend_url,
    device_id: device.device_id,
    device_secret: device.device_secret,
  });

  return (
    <main className="mx-auto flex min-h-screen max-w-lg flex-col items-center justify-center gap-6 px-6 text-center">
      <h1 className="text-2xl font-extrabold text-brand-navy">Pasangkan HP kamu</h1>
      <p className="text-sm text-slate-500">
        Unduh aplikasi Android bridge, lalu pindai QR ini dari menu Pengaturan
        di aplikasi untuk menghubungkan HP dengan akun kamu.
      </p>

      <div className="rounded-xl border border-slate-200 bg-white p-4 shadow-sm">
        <QRCode value={qrPayload} size={220} />
      </div>

      <p className="text-xs text-rose-500">
        QR ini memuat kunci perangkat -- jangan screenshot atau bagikan ke orang lain.
      </p>

      <a
        href="https://whuzpay.com/download-app"
        target="_blank"
        rel="noreferrer"
        className="rounded-md bg-brand-yellow px-6 py-3 text-sm font-bold uppercase tracking-wide text-brand-navy-dark hover:bg-brand-yellow-dark"
      >
        Unduh Aplikasi Android
      </a>

      <Link href="/dashboard" className="text-sm font-medium text-brand-navy hover:text-brand-navy-light">
        Lewati, lanjut ke dashboard →
      </Link>
    </main>
  );
}
