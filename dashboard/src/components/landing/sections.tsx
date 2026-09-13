import {
  AlertTriangle,
  ArrowRight,
  Bell,
  Fingerprint,
  KeyRound,
  Radar,
  Receipt,
  RefreshCw,
  Repeat,
  Search,
  ShieldCheck,
  Smartphone,
  Webhook,
  Zap,
} from "lucide-react";

function Eyebrow({ children }: { children: string }) {
  return (
    <span className="text-xs font-semibold tracking-widest text-teal-600 uppercase">
      {children}
    </span>
  );
}

/* ---------------------------------------------------------------------- */
/* Experience -- satu kartu besar, eyebrow + heading + 3 kolom ikon        */
/* (mengikuti pola referensi: bukan grid kartu terpisah-pisah)             */
/* ---------------------------------------------------------------------- */

const EXPERIENCE_POINTS = [
  {
    icon: Repeat,
    title: "Nominal unik",
    desc: "Tiap invoice dapat nominal berbeda, jadi pembayaran otomatis cocok tanpa tebak-tebak.",
  },
  {
    icon: Webhook,
    title: "Webhook siap pakai",
    desc: "invoice.paid dan invoice.expired terkirim otomatis ke sistem kamu, lengkap dengan retry.",
  },
  {
    icon: ShieldCheck,
    title: "Konsol pengecualian",
    desc: "Transaksi yang tidak cocok otomatis tetap kelihatan, bisa dicocokkan manual kapan saja.",
  },
];

export function ProblemSection() {
  return (
    <section className="bg-white py-16 sm:py-20">
      <div className="mx-auto max-w-6xl px-4 sm:px-6 lg:px-8">
        <div className="rounded-3xl bg-slate-50 p-8 sm:p-12">
          <div className="flex flex-col gap-6 lg:flex-row lg:items-start lg:justify-between">
            <div className="max-w-sm">
              <Eyebrow>Kenapa Payment Bridge</Eyebrow>
              <h2 className="font-(--font-lp-heading) mt-3 text-3xl leading-tight font-semibold tracking-tight text-slate-900 sm:text-4xl">
                Sistem yang tumbuh bersama bisnismu.
              </h2>
            </div>
            <p className="max-w-sm text-sm text-slate-500 lg:pt-2">
              Notifikasi pembayaran bukan cuma informasi di layar HP — ubah jadi
              alur kerja otomatis yang langsung terhubung ke sistem bisnismu
              sendiri.
            </p>
          </div>

          <div className="mt-10 grid gap-8 border-t border-slate-200 pt-8 sm:grid-cols-3">
            {EXPERIENCE_POINTS.map((point) => (
              <div key={point.title} className="flex flex-col gap-2">
                <span className="flex size-9 items-center justify-center rounded-lg bg-white text-teal-600 shadow-sm">
                  <point.icon className="size-4.5" />
                </span>
                <p className="text-sm font-semibold text-slate-900">{point.title}</p>
                <p className="text-sm text-slate-500">{point.desc}</p>
              </div>
            ))}
          </div>
        </div>
      </div>
    </section>
  );
}

/* ---------------------------------------------------------------------- */
/* How it works -- band navy penuh, 3 langkah bernomor besar              */
/* ---------------------------------------------------------------------- */

const STEPS = [
  {
    icon: Smartphone,
    title: "Pasang aplikasi",
    desc: "Aplikasi Android membaca notifikasi GoPay Merchant di HP kamu.",
  },
  {
    icon: Zap,
    title: "Event diproses",
    desc: "Nominal dicocokkan otomatis ke invoice yang sedang menunggu.",
  },
  {
    icon: Webhook,
    title: "Webhook terkirim",
    desc: "Sistem kamu langsung tahu begitu pembayaran diterima.",
  },
];

export function HowItWorksSection() {
  return (
    <section id="cara-kerja" className="bg-slate-900 py-20 text-white sm:py-24">
      <div className="mx-auto max-w-6xl px-4 sm:px-6 lg:px-8">
        <Eyebrow>Cara kerja</Eyebrow>
        <h2 className="font-(--font-lp-heading) mt-3 max-w-md text-3xl leading-tight font-semibold tracking-tight text-balance sm:text-4xl">
          Dari notifikasi sampai ke sistem kamu.
        </h2>

        <div className="mt-12 grid gap-4 sm:grid-cols-3">
          {STEPS.map((step, i) => (
            <div
              key={step.title}
              className="rounded-2xl border border-white/10 bg-white/5 p-6"
            >
              <span className="text-4xl font-semibold text-white/15">
                {String(i + 1).padStart(2, "0")}
              </span>
              <step.icon className="mt-4 size-5 text-teal-400" />
              <p className="mt-3 text-sm font-semibold">{step.title}</p>
              <p className="mt-1 text-sm text-white/60">{step.desc}</p>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}

/* ---------------------------------------------------------------------- */
/* Features (bento grid)                                                  */
/* ---------------------------------------------------------------------- */

export function FeaturesSection() {
  return (
    <section id="fitur" className="bg-white py-16 sm:py-20">
      <div className="mx-auto max-w-6xl px-4 sm:px-6 lg:px-8">
        <Eyebrow>Fitur</Eyebrow>
        <h2 className="font-(--font-lp-heading) mt-3 max-w-md text-3xl leading-tight font-semibold tracking-tight text-slate-900 sm:text-4xl">
          Semua yang dibutuhkan untuk otomasi pembayaran.
        </h2>
        <div className="mt-10 grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
          <FeatureCard
            icon={Repeat}
            title="Nominal unik"
            desc="Tiap invoice dapat nominal yang sedikit berbeda, jadi pembayaran otomatis cocok tanpa tebak-tebak."
            className="lg:col-span-2"
          />
          <FeatureCard
            icon={Webhook}
            title="Webhook & retry"
            desc="invoice.paid dan invoice.expired terkirim otomatis, dengan retry berjenjang kalau gagal."
          />
          <FeatureCard
            icon={Smartphone}
            title="Manajemen device"
            desc="Pantau HP yang terhubung — status online, versi aplikasi, dan heartbeat terakhir."
          />
          <FeatureCard
            icon={Search}
            title="Konsol pengecualian"
            desc="Transaksi yang nominalnya tidak cocok otomatis tetap kelihatan, bisa dicocokkan manual."
          />
          <FeatureCard
            icon={KeyRound}
            title="API key"
            desc="Amankan integrasi dari sistem kamu ke Payment Bridge dengan key yang bisa dicabut kapan saja."
            className="lg:col-span-2"
          />
          <FeatureCard
            icon={Radar}
            title="Dashboard realtime"
            desc="Overview, Events, Transactions — semua ter-update otomatis begitu pembayaran masuk."
          />
        </div>
      </div>
    </section>
  );
}

function FeatureCard({
  icon: Icon,
  title,
  desc,
  className,
}: {
  icon: typeof Repeat;
  title: string;
  desc: string;
  className?: string;
}) {
  return (
    <div
      className={`rounded-2xl border border-slate-100 bg-white p-6 shadow-sm ${className ?? ""}`}
    >
      <span className="mb-4 flex size-10 items-center justify-center rounded-lg bg-teal-50 text-teal-600">
        <Icon className="size-5" />
      </span>
      <p className="text-sm font-semibold text-slate-900">{title}</p>
      <p className="mt-1 text-sm text-slate-500">{desc}</p>
    </div>
  );
}

/* ---------------------------------------------------------------------- */
/* Architecture                                                           */
/* ---------------------------------------------------------------------- */

export function ArchitectureSection() {
  const layers = [
    { icon: Smartphone, label: "Aplikasi Pembayaran" },
    { icon: Bell, label: "Android Bridge" },
    { icon: Zap, label: "Payment Bridge (hosted)" },
    { icon: Webhook, label: "Webhook" },
    { icon: Receipt, label: "Sistem Bisnis Kamu" },
  ];
  return (
    <section className="bg-slate-50 py-16 sm:py-20">
      <div className="mx-auto max-w-3xl px-4 text-center sm:px-6 lg:px-8">
        <Eyebrow>Arsitektur</Eyebrow>
        <h2 className="font-(--font-lp-heading) mt-3 text-3xl leading-tight font-semibold tracking-tight text-slate-900 sm:text-4xl">
          Dibangun sebagai lapisan integrasi.
        </h2>
        <p className="mx-auto mt-4 max-w-xl text-sm text-slate-500">
          Payment Bridge menjadi penghubung antara notifikasi pembayaran dan
          sistem yang butuh mengonsumsinya — kamu tidak perlu menyiapkan
          server sendiri.
        </p>
        <div className="mx-auto mt-10 flex max-w-sm flex-col items-center">
          {layers.map((layer, i) => (
            <div key={layer.label} className="flex w-full flex-col items-center">
              <div className="flex w-full items-center gap-3 rounded-xl border border-slate-100 bg-white px-4 py-3 text-left shadow-sm">
                <span className="flex size-9 items-center justify-center rounded-lg bg-teal-50 text-teal-600">
                  <layer.icon className="size-4.5" />
                </span>
                <span className="text-sm font-semibold text-slate-900">{layer.label}</span>
              </div>
              {i < layers.length - 1 && (
                <ArrowRight className="my-2 size-4 rotate-90 text-slate-300" />
              )}
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}

/* ---------------------------------------------------------------------- */
/* Use cases                                                              */
/* ---------------------------------------------------------------------- */

const USE_CASES = [
  {
    icon: ShieldCheck,
    title: "Konfirmasi pembayaran otomatis",
    desc: "Sistem internal langsung tahu begitu pembayaran diterima.",
  },
  {
    icon: RefreshCw,
    title: "Otomasi pesanan",
    desc: "Proses order otomatis terpicu setelah pembayaran sukses.",
  },
  {
    icon: Webhook,
    title: "Integrasi webhook kustom",
    desc: "Hubungkan event pembayaran ke API yang sudah kamu punya.",
  },
  {
    icon: Smartphone,
    title: "Operasional multi-device",
    desc: "Pantau beberapa HP penerima pembayaran dari satu dashboard.",
  },
  {
    icon: AlertTriangle,
    title: "Deteksi transaksi ganjil",
    desc: "Nominal yang tidak cocok otomatis tetap tercatat, tidak hilang.",
  },
  {
    icon: Fingerprint,
    title: "Integrasi POS kustom",
    desc: "Sambungkan event pembayaran ke software kasir milikmu sendiri.",
  },
];

export function UseCasesSection() {
  return (
    <section className="bg-white py-16 sm:py-20">
      <div className="mx-auto max-w-6xl px-4 sm:px-6 lg:px-8">
        <Eyebrow>Use case</Eyebrow>
        <h2 className="font-(--font-lp-heading) mt-3 max-w-md text-3xl leading-tight font-semibold tracking-tight text-slate-900 sm:text-4xl">
          Dipakai untuk apa saja.
        </h2>
        <div className="mt-10 grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {USE_CASES.map((uc) => (
            <div key={uc.title} className="rounded-2xl border border-slate-100 bg-slate-50 p-5">
              <uc.icon className="mb-3 size-5 text-teal-600" />
              <p className="text-sm font-semibold text-slate-900">{uc.title}</p>
              <p className="mt-1 text-sm text-slate-500">{uc.desc}</p>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}
