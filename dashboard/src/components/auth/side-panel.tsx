import { Bell, Check, Smartphone, Zap } from "lucide-react";

/**
 * Panel kiri halaman /login dan /register -- ilustrasi UI dashboard
 * customer kita sendiri (Overview + Devices), bukan tangkapan layar
 * sungguhan dan bukan data customer asli. Angka/nama di dalamnya cuma
 * contoh tampilan, sama seperti mockup di landing page (DashboardPreview,
 * Hero). Warna tetap palet kita (slate/teal), bukan meniru warna referensi.
 * Latar sengaja solid (bukan gradien) atas permintaan Akbar.
 */
export function AuthSidePanel({
  eyebrow,
  title,
  subtitle,
}: {
  eyebrow: string;
  title: string;
  subtitle: string;
}) {
  return (
    <div className="relative hidden overflow-hidden bg-slate-900 lg:flex lg:w-1/2 lg:flex-col lg:px-12 lg:py-12 xl:px-16">
      <div
        aria-hidden
        className="pointer-events-none absolute inset-0 bg-[radial-gradient(circle,rgb(255_255_255/0.06)_1px,transparent_1px)] bg-size-[24px_24px]"
      />
      <div
        aria-hidden
        className="pointer-events-none absolute -top-24 -right-24 size-96 rounded-full bg-teal-400/10 blur-3xl"
      />

      <div className="relative flex items-center gap-2 font-semibold text-white">
        <span className="flex size-8 items-center justify-center rounded-lg bg-white/10 ring-1 ring-white/15">
          <Zap className="size-4" fill="currentColor" strokeWidth={0} />
        </span>
        Payment Bridge
      </div>

      <div className="relative mt-10 max-w-lg">
        <span className="text-xs font-semibold tracking-widest text-teal-300 uppercase">
          {eyebrow}
        </span>
        <h1 className="font-(--font-lp-heading) mt-3 text-3xl leading-tight font-semibold tracking-tight text-balance text-white">
          {title}
        </h1>
        <p className="mt-3 text-sm text-white/60">{subtitle}</p>
      </div>

      {/* Ilustrasi dashboard customer -- kartu utama + kartu device mengambang */}
      <div className="relative mt-12 flex flex-1 items-center justify-center">
        <div className="relative w-full max-w-lg">
          <div className="overflow-hidden rounded-2xl bg-white shadow-2xl shadow-slate-950/40 ring-1 ring-white/10">
            <div className="flex items-center gap-1.5 border-b border-slate-100 px-5 py-3.5">
              <span className="size-2.5 rounded-full bg-red-300" />
              <span className="size-2.5 rounded-full bg-amber-300" />
              <span className="size-2.5 rounded-full bg-teal-400" />
              <span className="ml-3 text-xs text-slate-400">Overview</span>
            </div>
            <div className="p-6">
              <p className="text-base font-semibold text-slate-900">Halo, Toko Kamu 👋</p>
              <p className="text-xs text-slate-400">Sabtu, 13 September 2026</p>

              <div className="mt-5 grid grid-cols-3 gap-3">
                <div className="rounded-lg bg-slate-50 p-3 ring-1 ring-slate-900/6">
                  <p className="text-[11px] text-slate-400">Hari ini</p>
                  <p className="font-mono text-sm font-semibold tabular-nums text-slate-900">
                    Rp 1,8jt
                  </p>
                </div>
                <div className="rounded-lg bg-slate-50 p-3 ring-1 ring-slate-900/6">
                  <p className="text-[11px] text-slate-400">Transaksi</p>
                  <p className="font-mono text-sm font-semibold tabular-nums text-slate-900">24</p>
                </div>
                <div className="rounded-lg bg-slate-50 p-3 ring-1 ring-slate-900/6">
                  <p className="text-[11px] text-slate-400">Device</p>
                  <p className="font-mono text-sm font-semibold tabular-nums text-slate-900">
                    3/3 Online
                  </p>
                </div>
              </div>

              <div className="mt-5 flex h-20 items-end gap-2">
                {[40, 65, 45, 80, 55, 95, 60, 75].map((h, i) => (
                  <span
                    key={i}
                    style={{ height: `${h}%` }}
                    className="flex-1 rounded-sm bg-teal-100"
                  />
                ))}
              </div>

              <div className="mt-5 flex flex-col gap-2">
                <div className="flex items-center justify-between rounded-lg bg-slate-50 px-3.5 py-2.5 ring-1 ring-slate-900/4">
                  <span className="flex items-center gap-2 text-sm text-slate-600">
                    <Bell className="size-4 text-slate-400" />
                    Pembayaran diterima
                  </span>
                  <span className="font-mono text-sm text-slate-400">Rp 125.000</span>
                </div>
                <div className="flex items-center justify-between rounded-lg bg-slate-50 px-3.5 py-2.5 ring-1 ring-slate-900/4">
                  <span className="flex items-center gap-2 text-sm text-slate-600">
                    <Bell className="size-4 text-slate-400" />
                    Pembayaran diterima
                  </span>
                  <span className="font-mono text-sm text-slate-400">Rp 75.000</span>
                </div>
              </div>
            </div>
          </div>

          {/* Kartu mengambang -- daftar device */}
          <div className="absolute -right-8 -bottom-10 w-60 rounded-xl bg-white p-4 shadow-xl shadow-slate-950/30 ring-1 ring-slate-900/6">
            <div className="flex items-center justify-between">
              <p className="text-sm font-semibold text-slate-900">Device</p>
              <span className="rounded-full bg-teal-50 px-2 py-0.5 text-xs font-medium text-teal-700 ring-1 ring-teal-700/10">
                3
              </span>
            </div>
            <div className="mt-3 flex flex-col gap-2">
              {["HP Toko 1", "HP Toko 2", "HP Toko 3"].map((name) => (
                <div key={name} className="flex items-center gap-2 text-xs text-slate-600">
                  <Smartphone className="size-3.5 text-slate-400" />
                  <span className="flex-1 truncate">{name}</span>
                  <Check className="size-3.5 text-teal-600" />
                </div>
              ))}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
