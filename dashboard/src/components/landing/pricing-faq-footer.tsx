"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { ArrowRight, Check, Zap } from "lucide-react";
import { SectionGrid } from "./section-grid";

function Eyebrow({ children, invert }: { children: string; invert?: boolean }) {
  return (
    <span
      className={`inline-flex items-center justify-center gap-2 text-xs font-semibold tracking-widest uppercase ${
        invert ? "text-teal-400" : "text-teal-700"
      }`}
    >
      <span className={`h-px w-4 ${invert ? "bg-teal-400/60" : "bg-teal-700/40"}`} />
      {children}
      <span className={`h-px w-4 ${invert ? "bg-teal-400/60" : "bg-teal-700/40"}`} />
    </span>
  );
}

/* ---------------------------------------------------------------------- */
/* Dashboard preview                                                      */
/* ---------------------------------------------------------------------- */

export function DashboardPreviewSection() {
  const stats = [
    { label: "Hari ini", value: "Rp 12.450.000" },
    { label: "Events", value: "1.284" },
    { label: "Devices", value: "8 / 8 Online" },
    { label: "Webhook", value: "99,8% Sukses" },
  ];
  const recent = [
    { amount: "Rp 125.000" },
    { amount: "Rp 75.000" },
    { amount: "Rp 250.000" },
    { amount: "Rp 50.000" },
  ];

  return (
    <section className="bg-slate-50 py-16 sm:py-20">
      <div className="mx-auto max-w-4xl px-4 sm:px-6 lg:px-8">
        <div className="text-center">
          <Eyebrow>Dashboard</Eyebrow>
          <h2 className="font-(--font-lp-heading) mt-3 text-3xl leading-tight font-semibold tracking-tight text-slate-900 sm:text-4xl">
            Semua transaksi, satu layar.
          </h2>
        </div>
        <div className="mt-12 overflow-hidden rounded-2xl bg-white shadow-xl shadow-slate-900/5 ring-1 ring-slate-900/6">
          <div className="flex items-center gap-1.5 border-b border-slate-100/80 px-4 py-3">
            <span className="size-2.5 rounded-full bg-red-300" />
            <span className="size-2.5 rounded-full bg-amber-300" />
            <span className="size-2.5 rounded-full bg-teal-400" />
            <span className="ml-3 text-xs text-slate-400">Overview</span>
          </div>
          <div className="grid grid-cols-2 gap-px bg-slate-100 sm:grid-cols-4">
            {stats.map((s) => (
              <div key={s.label} className="bg-white p-4">
                <p className="text-xs text-slate-400">{s.label}</p>
                <p className="mt-1 text-lg font-semibold text-slate-900">{s.value}</p>
              </div>
            ))}
          </div>
          <div className="border-t border-slate-100 p-4">
            <p className="mb-3 text-xs font-medium text-slate-400">Event terbaru</p>
            <div className="flex flex-col gap-2">
              {recent.map((r, i) => (
                <div
                  key={i}
                  className="flex items-center justify-between rounded-lg bg-slate-50 px-3 py-2 text-sm"
                >
                  <span className="flex items-center gap-2 text-slate-700">
                    <Check className="size-3.5 text-teal-700" />
                    Pembayaran diterima
                  </span>
                  <span className="font-mono text-xs text-slate-400">{r.amount}</span>
                </div>
              ))}
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}

/* ---------------------------------------------------------------------- */
/* Pricing -- data dari GET /api/v1/pricing-plans (diatur vendor lewat     */
/* halaman Plans di Vendor Dashboard), bukan lagi array tetap di sini.     */
/* Layout/styling kartu SENGAJA dipertahankan persis sama.                */
/* ---------------------------------------------------------------------- */

interface PublicPlan {
  name: string;
  price_label: string;
  price_period: string;
  description: string;
  device_label: string;
  features: string[];
  highlighted: boolean;
}

function PricingCardSkeleton({ highlight }: { highlight?: boolean }) {
  return (
    <div
      className={
        highlight
          ? "flex animate-pulse flex-col rounded-2xl bg-slate-900/5 p-6 ring-1 ring-slate-900/6"
          : "flex animate-pulse flex-col rounded-2xl bg-slate-100 p-6"
      }
    >
      <div className="h-5 w-24 rounded bg-slate-200" />
      <div className="mt-2 h-4 w-40 rounded bg-slate-200" />
      <div className="mt-6 h-4 w-20 rounded bg-slate-200" />
      <div className="mt-4 flex flex-col gap-2">
        <div className="h-3 w-full rounded bg-slate-200" />
        <div className="h-3 w-full rounded bg-slate-200" />
        <div className="h-3 w-2/3 rounded bg-slate-200" />
      </div>
      <div className="mt-6 h-10 rounded-lg bg-slate-200" />
    </div>
  );
}

export function PricingSection() {
  // window/fetch tidak boleh berjalan saat prerender -- null berarti belum
  // selesai dimuat sama sekali (beda dari [] yang berarti vendor memang
  // belum mengaktifkan plan apa pun), jadi tiga keadaan ini dibedakan
  // dengan jelas di bawah: loading / kosong / terisi.
  const [plans, setPlans] = useState<PublicPlan[] | null>(null);

  useEffect(() => {
    let cancelled = false;
    fetch("/api/v1/pricing-plans")
      .then((res) => (res.ok ? res.json() : Promise.reject(new Error("gagal"))))
      .then((data: { plans: PublicPlan[] }) => {
        if (!cancelled) setPlans(data.plans);
      })
      .catch(() => {
        if (!cancelled) setPlans([]);
      });
    return () => {
      cancelled = true;
    };
  }, []);

  return (
    <section id="harga" className="relative bg-white py-16 sm:py-20">
      <SectionGrid />
      <div className="relative mx-auto max-w-5xl px-4 sm:px-6 lg:px-8">
        <div className="text-center">
          <Eyebrow>Harga</Eyebrow>
          <h2 className="font-(--font-lp-heading) mt-3 text-3xl leading-tight font-semibold tracking-tight text-slate-900 sm:text-4xl">
            Satu paket untuk tiap skala usaha.
          </h2>
        </div>
        {plans === null ? (
          <div className="mt-12 grid gap-6 sm:grid-cols-3">
            <PricingCardSkeleton />
            <PricingCardSkeleton highlight />
            <PricingCardSkeleton />
          </div>
        ) : plans.length === 0 ? (
          <p className="mt-12 text-center text-sm text-slate-500">
            Info paket sedang disiapkan. Hubungi kami untuk detail harga terbaru.
          </p>
        ) : (
          <div className="mt-12 grid gap-6 sm:grid-cols-3">
            {plans.map((plan) => (
              <div
                key={plan.name}
                className={
                  plan.highlighted
                    ? "relative flex flex-col overflow-hidden rounded-2xl bg-slate-900 p-6 text-white shadow-xl shadow-slate-900/20 ring-1 ring-slate-900/10 transition-transform duration-300 hover:-translate-y-4 sm:-translate-y-3"
                    : "flex flex-col rounded-2xl bg-white p-6 shadow-sm ring-1 ring-slate-900/6 transition-all duration-300 hover:-translate-y-1 hover:shadow-lg hover:shadow-slate-900/5 hover:ring-slate-900/10"
                }
              >
                {plan.highlighted && (
                  <>
                    <span aria-hidden className="absolute inset-x-0 top-0 h-1 bg-teal-400" />
                    <span className="mb-2 w-fit rounded-md bg-teal-400/15 px-2.5 py-1 text-xs font-semibold text-teal-300 ring-1 ring-teal-400/20">
                      Direkomendasikan
                    </span>
                  </>
                )}
                <p
                  className={`text-lg font-semibold ${plan.highlighted ? "text-white" : "text-slate-900"}`}
                >
                  {plan.name}
                </p>
                {plan.description && (
                  <p className={`text-sm ${plan.highlighted ? "text-white/70" : "text-slate-500"}`}>
                    {plan.description}
                  </p>
                )}
                {plan.price_label && (
                  <p
                    className={`mt-4 text-3xl font-semibold ${plan.highlighted ? "text-white" : "text-slate-900"}`}
                  >
                    {plan.price_label}
                    {plan.price_period && (
                      <span
                        className={`text-sm font-normal ${plan.highlighted ? "text-white/60" : "text-slate-400"}`}
                      >
                        {" "}
                        {plan.price_period}
                      </span>
                    )}
                  </p>
                )}
                <p
                  className={`mt-4 text-sm font-medium ${plan.highlighted ? "text-white" : "text-slate-900"}`}
                >
                  {plan.device_label}
                </p>
                {plan.features.length > 0 && (
                  <ul
                    className={`mt-4 flex flex-col gap-2 text-sm ${plan.highlighted ? "text-white/80" : "text-slate-500"}`}
                  >
                    {plan.features.map((feature) => (
                      <li key={feature} className="flex items-center gap-2">
                        <Check
                          className={`size-4 shrink-0 ${plan.highlighted ? "text-teal-300" : "text-teal-700"}`}
                        />
                        {feature}
                      </li>
                    ))}
                  </ul>
                )}
                <Link
                  href="/register"
                  className={
                    plan.highlighted
                      ? "mt-6 flex h-10 items-center justify-center rounded-lg bg-teal-400 text-sm font-semibold text-slate-900 transition-all duration-200 hover:bg-teal-300 active:scale-[0.97]"
                      : "mt-6 flex h-10 items-center justify-center rounded-lg text-sm font-medium text-slate-700 ring-1 ring-slate-900/10 transition-all duration-200 hover:bg-slate-50 active:scale-[0.97]"
                  }
                >
                  Mulai Gratis 3 Hari
                </Link>
              </div>
            ))}
          </div>
        )}
        <p className="mt-8 text-center text-sm text-slate-500">
          Butuh paket khusus?{" "}
          <a href="#" className="font-medium text-slate-900 hover:underline">
            Hubungi kami
          </a>
          .
        </p>
      </div>
    </section>
  );
}

/* ---------------------------------------------------------------------- */
/* FAQ -- <details>/<summary> native, tanpa JS tambahan                    */
/* ---------------------------------------------------------------------- */

const FAQS = [
  {
    q: "Apa itu Payment Bridge?",
    a: "Layanan yang mengubah notifikasi pembayaran GoPay di HP kamu jadi event terstruktur dan webhook otomatis ke sistem bisnismu.",
  },
  {
    q: "Bagaimana cara kerja Android Bridge-nya?",
    a: "Aplikasi pendamping membaca notifikasi pembayaran yang masuk di HP, lalu mengirimkannya ke Payment Bridge secara otomatis dan aman.",
  },
  {
    q: "Apakah saya perlu menyiapkan server sendiri?",
    a: "Tidak. Payment Bridge sepenuhnya hosted — kamu cukup daftar, pasang aplikasi Android, dan mulai memakai dashboard serta webhook.",
  },
  {
    q: "Apa yang terjadi setelah masa trial 3 hari?",
    a: "Akun tetap ada, tapi endpoint pengiriman data akan nonaktif sampai paket dipilih dan diaktifkan kembali.",
  },
  {
    q: "Bisa pakai lebih dari satu HP?",
    a: "Bisa, sesuai jumlah device pada paket yang kamu pilih (lihat bagian Harga).",
  },
  {
    q: "Bisa dihubungkan ke sistem yang sudah saya punya?",
    a: "Bisa. Daftarkan endpoint webhook kamu, dan event invoice.paid/invoice.expired akan dikirim otomatis ke sana.",
  },
  {
    q: "Sumber pembayaran apa saja yang didukung?",
    a: "Saat ini GoPay Merchant. Dukungan sumber pembayaran lain sedang kami eksplorasi.",
  },
];

export function FaqSection() {
  return (
    <section id="faq" className="bg-slate-50 py-16 sm:py-20">
      <div className="mx-auto max-w-2xl px-4 sm:px-6 lg:px-8">
        <div className="text-center">
          <Eyebrow>FAQ</Eyebrow>
          <h2 className="font-(--font-lp-heading) mt-3 text-3xl leading-tight font-semibold tracking-tight text-slate-900 sm:text-4xl">
            Pertanyaan yang sering ditanyakan.
          </h2>
        </div>
        <div className="mt-10 flex flex-col divide-y divide-slate-100 rounded-2xl bg-white ring-1 ring-slate-900/6">
          {FAQS.map((faq) => (
            <details key={faq.q} className="group px-5 py-4 open:pb-4">
              <summary className="flex cursor-pointer list-none items-center justify-between gap-4 text-sm font-medium text-slate-900 transition-colors duration-200 marker:content-none hover:text-teal-700">
                {faq.q}
                <span className="shrink-0 text-slate-400 transition-transform duration-300 group-open:rotate-45">
                  +
                </span>
              </summary>
              <p className="animate-in fade-in-0 slide-in-from-top-1 mt-3 text-sm text-slate-500 duration-300">
                {faq.a}
              </p>
            </details>
          ))}
        </div>
      </div>
    </section>
  );
}

/* ---------------------------------------------------------------------- */
/* Final CTA -- band navy penuh                                          */
/* ---------------------------------------------------------------------- */

export function FinalCtaSection() {
  return (
    <section className="relative overflow-hidden bg-slate-900 py-20 text-white sm:py-24">
      <div
        aria-hidden
        className="pointer-events-none absolute inset-0 bg-[linear-gradient(to_right,rgb(255_255_255/0.04)_1px,transparent_1px),linear-gradient(to_bottom,rgb(255_255_255/0.04)_1px,transparent_1px)] bg-size-[40px_40px] mask-[radial-gradient(ellipse_60%_80%_at_50%_50%,#000_30%,transparent_100%)]"
      />
      <div
        aria-hidden
        className="pointer-events-none absolute top-1/2 left-1/2 size-144 -translate-x-1/2 -translate-y-1/2 rounded-full bg-teal-500/10 blur-3xl"
      />

      <div className="relative mx-auto flex max-w-2xl flex-col items-center gap-6 px-4 text-center sm:px-6 lg:px-8">
        <span className="rounded-full bg-white/5 px-3 py-1 text-xs font-medium text-white/70 ring-1 ring-white/10">
          Aktif dalam hitungan menit, tanpa kartu kredit
        </span>

        <h2 className="font-(--font-lp-heading) text-3xl leading-tight font-semibold tracking-tight text-balance sm:text-4xl">
          Hubungkan pembayaran ke sistem yang sudah kamu pakai.
        </h2>
        <p className="text-white/60">
          Mulai bangun alur kerja pembayaran yang lebih otomatis, dengan infrastruktur yang sudah
          kami siapkan.
        </p>
        <div className="flex flex-wrap items-center justify-center gap-3">
          <Link
            href="/register"
            className="group flex h-11 items-center gap-1.5 rounded-lg bg-teal-400 px-5 text-sm font-semibold text-slate-900 transition-all duration-200 hover:-translate-y-0.5 hover:bg-teal-300 hover:shadow-lg hover:shadow-teal-400/20 active:translate-y-0 active:scale-[0.97]"
          >
            Mulai Gratis 3 Hari
            <ArrowRight className="size-4 transition-transform duration-200 group-hover:translate-x-0.5" />
          </Link>
          <a
            href="#fitur"
            className="flex h-11 items-center rounded-lg px-5 text-sm font-medium text-white/80 ring-1 ring-white/15 transition-all duration-200 hover:-translate-y-0.5 hover:bg-white/5"
          >
            Lihat Fitur
          </a>
        </div>
      </div>
    </section>
  );
}

/* ---------------------------------------------------------------------- */
/* Footer -- layout kolom                                                */
/* ---------------------------------------------------------------------- */

export function LandingFooter() {
  return (
    <footer className="relative bg-white py-12">
      <SectionGrid />
      <div className="relative mx-auto max-w-6xl px-4 sm:px-6 lg:px-8">
        <div className="flex flex-col gap-8 sm:flex-row sm:justify-between">
          <div className="flex flex-col gap-3">
            <div className="flex items-center gap-2 font-semibold text-slate-900">
              <span className="flex size-7 items-center justify-center rounded-lg bg-slate-900 text-white ring-1 ring-slate-900/10">
                <Zap className="size-3.5" fill="currentColor" strokeWidth={0} />
              </span>
              Payment Bridge
            </div>
            <p className="max-w-xs text-sm text-slate-500">
              Notifikasi pembayaran GoPay, otomatis jadi event dan webhook ke sistem bisnismu.
            </p>
          </div>

          <div className="grid grid-cols-2 gap-8 sm:grid-cols-2">
            <div className="flex flex-col gap-2">
              <p className="text-xs font-semibold tracking-widest text-slate-400 uppercase">
                Produk
              </p>
              <a
                href="#fitur"
                className="text-sm text-slate-500 transition-colors duration-200 hover:text-slate-900"
              >
                Fitur
              </a>
              <a
                href="#cara-kerja"
                className="text-sm text-slate-500 transition-colors duration-200 hover:text-slate-900"
              >
                Cara Kerja
              </a>
              <a
                href="#harga"
                className="text-sm text-slate-500 transition-colors duration-200 hover:text-slate-900"
              >
                Harga
              </a>
              <a
                href="#faq"
                className="text-sm text-slate-500 transition-colors duration-200 hover:text-slate-900"
              >
                FAQ
              </a>
            </div>
            <div className="flex flex-col gap-2">
              <p className="text-xs font-semibold tracking-widest text-slate-400 uppercase">Akun</p>
              <Link
                href="/login"
                className="text-sm text-slate-500 transition-colors duration-200 hover:text-slate-900"
              >
                Masuk
              </Link>
              <Link
                href="/register"
                className="text-sm text-slate-500 transition-colors duration-200 hover:text-slate-900"
              >
                Daftar
              </Link>
            </div>
          </div>
        </div>

        <p className="mt-10 border-t border-slate-100 pt-6 text-xs text-slate-400">
          © {new Date().getFullYear()} Payment Bridge
        </p>
      </div>
    </footer>
  );
}
