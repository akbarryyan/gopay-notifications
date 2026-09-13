import type { CSSProperties, ReactNode } from "react";
import { SectionGrid } from "./section-grid";
import {
  AlertTriangle,
  ArrowRight,
  Bell,
  Check,
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

function Eyebrow({ children, invert }: { children: string; invert?: boolean }) {
  return (
    <span
      className={`inline-flex items-center gap-2 text-xs font-semibold tracking-widest uppercase ${
        invert ? "text-teal-400" : "text-teal-700"
      }`}
    >
      <span className={`h-px w-4 ${invert ? "bg-teal-400/60" : "bg-teal-700/40"}`} />
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
    <section className="relative bg-white py-16 sm:py-20">
      <SectionGrid />
      <div className="relative mx-auto max-w-6xl px-4 sm:px-6 lg:px-8">
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
              <div key={point.title} className="group flex flex-col gap-2">
                <span className="flex size-9 items-center justify-center rounded-lg bg-white text-teal-700 shadow-sm ring-1 ring-slate-900/6 transition-all duration-300 group-hover:-translate-y-0.5 group-hover:shadow-md">
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
        <Eyebrow invert>Cara kerja</Eyebrow>
        <h2 className="font-(--font-lp-heading) mt-3 max-w-md text-3xl leading-tight font-semibold tracking-tight text-balance sm:text-4xl">
          Dari notifikasi sampai ke sistem kamu.
        </h2>

        <div className="mt-12 grid grid-cols-1 gap-2 sm:grid-cols-[1fr_auto_1fr_auto_1fr] sm:items-stretch sm:gap-4">
          {STEPS.flatMap((step, i) => {
            const card = (
              <div
                key={step.title}
                className="rounded-2xl border border-white/10 bg-white/5 p-6 transition-all duration-300 hover:-translate-y-1 hover:border-white/20 hover:bg-white/10"
              >
                <span className="text-4xl font-semibold text-white/15">
                  {String(i + 1).padStart(2, "0")}
                </span>
                <step.icon className="mt-4 size-5 text-teal-400" />
                <p className="mt-3 text-sm font-semibold">{step.title}</p>
                <p className="mt-1 text-sm text-white/60">{step.desc}</p>
              </div>
            );
            if (i === STEPS.length - 1) return [card];
            return [card, <StepConnector key={`connector-${step.title}`} index={i} />];
          })}
        </div>
      </div>
    </section>
  );
}

/**
 * Konektor antar kartu langkah -- garis melengkung (bukan lurus) yang
 * bentuknya diam total. Yang beranimasi cuma warnanya: sebuah overlay
 * path dengan stroke-dashoffset (lp-line-fill di globals.css) "mengisi"
 * warna di atas garis pemandu pudar yang selalu ada, lalu memudar lagi --
 * bukan garis yang bergerak atau berganti bentuk. Delay dibuat bertingkat
 * per index biar tidak berdenyut serempak di semua konektor.
 */
function StepConnector({ index }: { index: number }) {
  const stroke = index % 2 === 0 ? "rgb(45 212 191 / 0.9)" : "rgb(148 163 184 / 0.85)";
  const delayStyle = { "--lp-flow-delay": `${index * 0.5}s` } as CSSProperties;
  return (
    <div className="-my-2 flex items-center justify-center sm:my-0 sm:-mx-4 sm:items-stretch">
      {/* Mobile: melebar ke atas/bawah sejauh gap grid (-my-2) supaya garis
          benar-benar menyentuh tepi kartu, bukan berhenti di tengah celah. */}
      <svg
        aria-hidden
        viewBox="0 0 40 56"
        preserveAspectRatio="none"
        className="h-14 w-10 sm:hidden"
        fill="none"
      >
        <path
          d="M20 0 C4 12 36 20 20 28 C4 36 36 44 20 56"
          stroke={stroke}
          strokeOpacity={0.18}
          strokeWidth="2"
          strokeLinecap="round"
          vectorEffect="non-scaling-stroke"
        />
        <path
          d="M20 0 C4 12 36 20 20 28 C4 36 36 44 20 56"
          stroke={stroke}
          strokeWidth="2"
          strokeLinecap="round"
          vectorEffect="non-scaling-stroke"
          pathLength={1}
          className="lp-line-fill"
          style={delayStyle}
        />
      </svg>
      {/* Desktop: melebar ke kiri/kanan sejauh gap grid (-mx-4), alasan sama. */}
      <svg
        aria-hidden
        viewBox="0 0 56 40"
        preserveAspectRatio="none"
        className="hidden h-full w-20 sm:block"
        fill="none"
      >
        <path
          d="M0 20 C12 4 20 36 28 20 C36 4 44 36 56 20"
          stroke={stroke}
          strokeOpacity={0.18}
          strokeWidth="2"
          strokeLinecap="round"
          vectorEffect="non-scaling-stroke"
        />
        <path
          d="M0 20 C12 4 20 36 28 20 C36 4 44 36 56 20"
          stroke={stroke}
          strokeWidth="2"
          strokeLinecap="round"
          vectorEffect="non-scaling-stroke"
          pathLength={1}
          className="lp-line-fill"
          style={delayStyle}
        />
      </svg>
    </div>
  );
}

/* ---------------------------------------------------------------------- */
/* Features (bento grid)                                                  */
/* ---------------------------------------------------------------------- */

export function FeaturesSection() {
  return (
    <section id="fitur" className="relative bg-white py-16 sm:py-20">
      <SectionGrid />
      <div className="relative mx-auto max-w-6xl px-4 sm:px-6 lg:px-8">
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
            highlight
          >
            <div className="mt-5 flex flex-wrap items-center gap-2 font-mono text-xs tabular-nums">
              <span className="rounded-md bg-slate-50 px-2.5 py-1.5 text-slate-500 ring-1 ring-slate-900/6">
                Ditagih Rp 125.000
              </span>
              <ArrowRight className="size-3.5 shrink-0 text-slate-300" />
              <span className="rounded-md bg-teal-50 px-2.5 py-1.5 text-teal-700 ring-1 ring-teal-700/15">
                Diterima Rp 125.483
              </span>
              <span className="ml-auto flex items-center gap-1 rounded-full bg-emerald-50 px-2 py-1 text-emerald-600 ring-1 ring-emerald-600/10">
                <Check className="size-3" />
                Cocok
              </span>
            </div>
          </FeatureCard>
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
            highlight
          >
            <div className="mt-5 flex items-center justify-between rounded-md bg-slate-900 px-3 py-2 font-mono text-xs text-slate-300 ring-1 ring-slate-900/10">
              <span className="tracking-wide">
                sk_live_<span className="text-white/30">••••••••••••</span>7f2a
              </span>
              <span className="rounded-sm bg-white/10 px-1.5 py-0.5 text-[10px] text-white/60">
                Cabut
              </span>
            </div>
          </FeatureCard>
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
  highlight,
  children,
}: {
  icon: typeof Repeat;
  title: string;
  desc: string;
  className?: string;
  highlight?: boolean;
  children?: ReactNode;
}) {
  return (
    <div
      className={`group relative overflow-hidden rounded-2xl bg-white p-6 shadow-sm ring-1 ring-slate-900/6 transition-all duration-300 hover:-translate-y-1 hover:shadow-lg hover:shadow-slate-900/5 hover:ring-slate-900/10 ${className ?? ""}`}
    >
      {highlight && (
        <span
          aria-hidden
          className="pointer-events-none absolute -top-10 -right-10 size-36 rounded-full bg-teal-500/[0.07] blur-2xl transition-opacity duration-300 group-hover:opacity-80"
        />
      )}
      <span className="relative mb-4 flex size-10 items-center justify-center rounded-lg bg-teal-50 text-teal-700 ring-1 ring-teal-700/10 transition-all duration-300 group-hover:scale-105 group-hover:bg-teal-100">
        <Icon className="size-5" />
      </span>
      <p className="relative text-sm font-semibold text-slate-900">{title}</p>
      <p className="relative mt-1 text-sm text-slate-500">{desc}</p>
      {children}
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
              <div className="group flex w-full items-center gap-3 rounded-xl bg-white px-4 py-3 text-left shadow-sm ring-1 ring-slate-900/6 transition-all duration-300 hover:-translate-x-0.5 hover:shadow-md hover:ring-slate-900/10">
                <span className="flex size-9 items-center justify-center rounded-lg bg-teal-50 text-teal-700 ring-1 ring-teal-700/10 transition-colors duration-300 group-hover:bg-teal-100">
                  <layer.icon className="size-4.5" />
                </span>
                <span className="text-sm font-semibold text-slate-900">{layer.label}</span>
              </div>
              {i < layers.length - 1 && <FlowConnector index={i} />}
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}

/**
 * Konektor antar layer arsitektur -- garis gradien halus dengan titik yang
 * mengalir turun terus-menerus (dijeda otomatis kalau prefers-reduced-motion
 * aktif, lihat globals.css). Warna gradien berselang-seling teal/slate dan
 * delay animasinya bertingkat per index supaya beberapa segmen tidak
 * berdenyut serempak -- kesannya satu aliran yang menyambung, bukan
 * sekumpulan segmen terpisah yang identik.
 */
function FlowConnector({ index }: { index: number }) {
  const tealTint = index % 2 === 0;
  return (
    <div className="relative flex h-10 w-8 items-center justify-center">
      <span
        aria-hidden
        className={`h-full w-px bg-linear-to-b ${
          tealTint
            ? "from-teal-400/0 via-teal-500/50 to-slate-300/0"
            : "from-teal-400/0 via-slate-400/60 to-slate-300/0"
        }`}
      />
      <span
        aria-hidden
        className="lp-flow-dot absolute left-1/2 size-1.5 -translate-x-1/2 rounded-full bg-teal-400 shadow-[0_0_8px_1px_rgb(45_212_191/0.55)]"
        style={{ "--lp-flow-delay": `${index * 0.35}s` } as CSSProperties}
      />
    </div>
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
    <section className="relative bg-white py-16 sm:py-20">
      <SectionGrid />
      <div className="relative mx-auto max-w-6xl px-4 sm:px-6 lg:px-8">
        <Eyebrow>Use case</Eyebrow>
        <h2 className="font-(--font-lp-heading) mt-3 max-w-md text-3xl leading-tight font-semibold tracking-tight text-slate-900 sm:text-4xl">
          Dipakai untuk apa saja.
        </h2>
        <div className="mt-10 grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {USE_CASES.map((uc) => (
            <div
              key={uc.title}
              className="rounded-2xl bg-slate-50 p-5 ring-1 ring-slate-900/6 transition-all duration-300 hover:-translate-y-1 hover:bg-white hover:shadow-md hover:ring-teal-700/15"
            >
              <uc.icon className="mb-3 size-5 text-teal-700" />
              <p className="text-sm font-semibold text-slate-900">{uc.title}</p>
              <p className="mt-1 text-sm text-slate-500">{uc.desc}</p>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}
