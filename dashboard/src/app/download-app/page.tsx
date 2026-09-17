import Link from "next/link";
import { Download, ShieldCheck, Smartphone } from "lucide-react";
import { LandingNavbar } from "@/components/landing/navbar";
import { LandingFooter } from "@/components/landing/pricing-faq-footer";

const STEPS = [
  {
    icon: Download,
    title: "1. Unduh APK",
    desc: "Klik tombol di bawah. Browser akan menandai file ini sebagai \"tidak dikenal\" -- itu wajar untuk aplikasi di luar Play Store.",
  },
  {
    icon: ShieldCheck,
    title: "2. Izinkan pemasangan",
    desc: "Saat memasang, Android akan meminta izin \"Pasang aplikasi tidak dikenal\". Aktifkan untuk browser yang kamu pakai mengunduh tadi.",
  },
  {
    icon: Smartphone,
    title: "3. Izinkan akses notifikasi",
    desc: "Setelah terpasang, buka aplikasinya dan izinkan Notification Access saat diminta -- ini yang dipakai aplikasi membaca notifikasi pembayaran GoPay.",
  },
];

export const metadata = {
  title: "Unduh Aplikasi Android — GoPay Notification Bridge",
};

export default function DownloadAppPage() {
  return (
    <div className="font-(--font-lp-body) flex min-h-screen flex-col">
      <LandingNavbar />
      <main className="flex-1 bg-white py-16 sm:py-20">
        <div className="mx-auto flex max-w-2xl flex-col items-center px-4 text-center">
          <span className="mb-4 flex size-11 items-center justify-center rounded-xl bg-slate-900 text-white ring-1 ring-slate-900/10">
            <Smartphone className="size-5" />
          </span>
          <h1 className="text-2xl font-semibold text-slate-900 sm:text-3xl">
            Unduh Aplikasi Android
          </h1>
          <p className="mt-3 text-sm text-slate-500 sm:text-base">
            Aplikasi ini yang memantau notifikasi GoPay di HP kamu dan melaporkannya secara
            otomatis. Wajib Android 8.0 (API 26) ke atas.
          </p>

          <Link
            href="/downloads/gopay-bridge.apk"
            className="group mt-8 flex h-12 items-center justify-center gap-1.5 rounded-lg bg-slate-900 px-8 text-base font-medium text-white transition-all duration-200 hover:-translate-y-0.5 hover:bg-slate-800 hover:shadow-lg hover:shadow-slate-900/20 active:translate-y-0 active:scale-[0.97]"
          >
            <Download className="size-4" />
            Unduh APK
          </Link>

          <div className="mt-14 grid w-full gap-6 text-left sm:grid-cols-3">
            {STEPS.map((step) => (
              <div key={step.title} className="flex flex-col gap-2">
                <span className="flex size-9 items-center justify-center rounded-lg bg-teal-50 text-teal-700 ring-1 ring-teal-600/20">
                  <step.icon className="size-4" />
                </span>
                <h2 className="text-sm font-semibold text-slate-900">{step.title}</h2>
                <p className="text-sm text-slate-500">{step.desc}</p>
              </div>
            ))}
          </div>
        </div>
      </main>
      <LandingFooter />
    </div>
  );
}
