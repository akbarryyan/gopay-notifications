import Link from "next/link";
import { ArrowUpRight, Bell, CreditCard, Wifi } from "lucide-react";

export function Hero() {
  return (
    <section className="relative overflow-hidden bg-white">
      <div className="mx-auto grid max-w-6xl gap-14 px-4 py-16 sm:px-6 lg:grid-cols-2 lg:items-center lg:gap-10 lg:px-8 lg:py-24">
        {/* Kolom kiri: copy + form ringkas */}
        <div className="animate-in fade-in-0 slide-in-from-bottom-4 flex flex-col gap-6 duration-700 ease-out">
          <h1 className="font-(--font-lp-heading) text-4xl leading-[1.08] font-semibold tracking-tight text-slate-900 text-balance sm:text-5xl lg:text-[3.1rem]">
            Terima notifikasi pembayaran, otomatis masuk ke sistem kamu.
          </h1>
          <p className="max-w-md text-base text-slate-500 sm:text-lg">
            Mendukung usaha kecil sampai besar dengan pencocokan invoice
            otomatis, webhook siap pakai, dan dashboard yang selalu realtime.
          </p>

          <form
            action="/register"
            className="flex max-w-md flex-col gap-2 sm:flex-row sm:items-center"
          >
            <input
              type="email"
              placeholder="Email bisnis kamu"
              className="h-11 flex-1 rounded-full border border-slate-200 bg-slate-50 px-4 text-sm text-slate-900 outline-none transition-colors duration-200 placeholder:text-slate-400 focus:border-teal-500 focus:bg-white focus:ring-2 focus:ring-teal-500/20"
            />
            <Link
              href="/register"
              className="group flex h-11 items-center justify-center gap-1 rounded-full bg-slate-900 px-5 text-sm font-medium text-white transition-all duration-200 hover:-translate-y-0.5 hover:bg-slate-800 hover:shadow-lg hover:shadow-slate-900/20 active:translate-y-0 active:scale-[0.97]"
            >
              Mulai Gratis
              <ArrowUpRight className="size-4 transition-transform duration-200 group-hover:translate-x-0.5 group-hover:-translate-y-0.5" />
            </Link>
          </form>

          <div className="flex flex-wrap items-center gap-x-6 gap-y-2 pt-2 text-sm font-medium text-slate-400">
            <span>Hosted, tanpa server sendiri</span>
            <span className="hidden sm:inline">·</span>
            <span>Webhook siap pakai</span>
            <span className="hidden sm:inline">·</span>
            <span>Multi-device</span>
          </div>
        </div>

        {/* Kolom kanan: komposisi kartu produk */}
        <div className="animate-in fade-in-0 slide-in-from-bottom-6 relative mx-auto w-full max-w-sm duration-700 ease-out delay-150 fill-mode-both lg:mx-0 lg:ml-auto">
          <div className="relative rounded-2xl border border-slate-100 bg-white p-4 shadow-xl shadow-slate-900/5 transition-shadow duration-300 hover:shadow-2xl hover:shadow-slate-900/10">
            <div className="flex items-center justify-between border-b border-slate-100 pb-3">
              <div>
                <p className="text-xs text-slate-400">Invoice masuk</p>
                <p className="text-sm font-semibold text-slate-900">ORDER-48213</p>
              </div>
              <span className="rounded-full bg-emerald-50 px-2.5 py-1 text-xs font-medium text-emerald-600">
                Lunas
              </span>
            </div>
            <p className="pt-3 text-2xl font-semibold text-slate-900">Rp 1.876.580</p>
            <p className="text-xs text-slate-400">Dibayar lewat GoPay Merchant</p>

            <div className="mt-4 flex items-center gap-2 rounded-xl border border-slate-100 bg-slate-50 px-3 py-2 text-xs text-slate-500">
              <Bell className="size-3.5 text-slate-400" />
              Notifikasi tercatat 12:04:02
            </div>
          </div>

          {/* Kartu mengambang -- device terhubung, meniru komposisi kartu kredit di referensi */}
          <div className="absolute -right-4 -bottom-8 w-52 rounded-2xl bg-linear-to-br from-slate-900 to-teal-900 p-4 text-white shadow-xl shadow-slate-900/20 transition-transform duration-300 hover:-translate-y-1 sm:-right-8">
            <div className="flex items-center justify-between">
              <Wifi className="size-4 rotate-90 text-teal-300" />
              <CreditCard className="size-5 text-white/70" />
            </div>
            <p className="mt-4 text-xs text-white/60">Device terhubung</p>
            <p className="text-sm font-semibold">HP Toko · Online</p>
          </div>
        </div>
      </div>
    </section>
  );
}
