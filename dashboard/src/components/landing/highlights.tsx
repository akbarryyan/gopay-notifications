"use client";

import { useEffect, useRef, useState } from "react";
import {
  ChevronLeft,
  ChevronRight,
  KeyRound,
  Radar,
  Repeat,
  Search,
  ShieldCheck,
  Webhook,
} from "lucide-react";
import { SectionGrid } from "./section-grid";

/**
 * Bukan testimoni pelanggan -- produk ini baru mulai, belum ada customer
 * asli untuk dikutip (lihat standing rule "jangan mengarang testimoni" di
 * sesi ini). Ini rotator value-prop: sorotan fitur yang sudah benar-benar
 * ada, cuma dikemas bergilir seperti carousel.
 */
const HIGHLIGHTS = [
  {
    icon: Repeat,
    title: "Nominal unik, otomatis cocok",
    desc: "Tiap invoice dapat kombinasi nominal berbeda, jadi pembayaran yang masuk otomatis cocok tanpa konfirmasi manual.",
  },
  {
    icon: Webhook,
    title: "Webhook & retry siap pakai",
    desc: "invoice.paid dan invoice.expired terkirim otomatis ke sistem kamu, dengan retry berjenjang kalau pengiriman sempat gagal.",
  },
  {
    icon: ShieldCheck,
    title: "Hosted, tanpa server sendiri",
    desc: "Daftar, pasang aplikasi Android, langsung pakai -- infrastrukturnya kami yang urus sepenuhnya.",
  },
  {
    icon: Search,
    title: "Konsol pengecualian",
    desc: "Transaksi yang nominalnya tidak cocok otomatis tetap kelihatan di dashboard, tidak pernah hilang diam-diam.",
  },
  {
    icon: Radar,
    title: "Dashboard realtime",
    desc: "Overview, Events, dan Transactions semua ter-update otomatis begitu pembayaran masuk -- tidak perlu refresh manual.",
  },
  {
    icon: KeyRound,
    title: "API key yang bisa dicabut",
    desc: "Amankan integrasi dari sistem kamu ke Payment Bridge dengan key yang bisa kamu cabut kapan saja tanpa mengganggu yang lain.",
  },
];

const AUTO_ADVANCE_MS = 5000;

export function HighlightsSection() {
  const [active, setActive] = useState(0);
  const [paused, setPaused] = useState(false);
  const count = HIGHLIGHTS.length;

  const goTo = (i: number) => setActive(((i % count) + count) % count);
  const next = () => goTo(active + 1);
  const prev = () => goTo(active - 1);

  const timerRef = useRef<ReturnType<typeof setInterval> | null>(null);
  useEffect(() => {
    if (paused) return;
    timerRef.current = setInterval(() => {
      setActive((i) => (i + 1) % count);
    }, AUTO_ADVANCE_MS);
    return () => {
      if (timerRef.current) clearInterval(timerRef.current);
    };
  }, [paused, count]);

  return (
    <section
      className="relative overflow-hidden bg-slate-50 py-16 sm:py-20"
      onMouseEnter={() => setPaused(true)}
      onMouseLeave={() => setPaused(false)}
    >
      <SectionGrid />
      <div className="relative mx-auto max-w-3xl px-4 sm:px-6 lg:px-8">
        <div className="text-center">
          <span className="inline-flex items-center justify-center gap-2 text-xs font-semibold tracking-widest text-teal-700 uppercase">
            <span className="h-px w-4 bg-teal-700/40" />
            Kenapa Payment Bridge
            <span className="h-px w-4 bg-teal-700/40" />
          </span>
          <h2 className="font-(--font-lp-heading) mt-3 text-3xl leading-tight font-semibold tracking-tight text-slate-900 sm:text-4xl">
            Dirancang seputar satu masalah: kecocokan pembayaran.
          </h2>
        </div>

        <div className="relative mt-10">
          <div className="overflow-hidden rounded-2xl bg-white shadow-sm ring-1 ring-slate-900/6">
            <div
              className="flex transition-transform duration-500 ease-out motion-reduce:transition-none"
              style={{ transform: `translateX(-${active * 100}%)` }}
            >
              {HIGHLIGHTS.map((h) => (
                <div key={h.title} className="w-full shrink-0 px-8 py-10 text-center sm:px-14">
                  <span className="mx-auto flex size-12 items-center justify-center rounded-xl bg-teal-50 text-teal-700 ring-1 ring-teal-700/10">
                    <h.icon className="size-6" />
                  </span>
                  <p className="mt-4 text-lg font-semibold text-slate-900">{h.title}</p>
                  <p className="mx-auto mt-2 max-w-md text-sm text-slate-500">{h.desc}</p>
                </div>
              ))}
            </div>
          </div>

          {/* Panah manual -- desktop saja, di mobile cukup swipe area + dots */}
          <button
            type="button"
            onClick={prev}
            aria-label="Sorotan sebelumnya"
            className="absolute top-1/2 left-0 hidden size-9 -translate-x-4 -translate-y-1/2 items-center justify-center rounded-full bg-white text-slate-500 shadow-sm ring-1 ring-slate-900/6 transition-colors duration-200 hover:text-slate-900 sm:flex"
          >
            <ChevronLeft className="size-4" />
          </button>
          <button
            type="button"
            onClick={next}
            aria-label="Sorotan berikutnya"
            className="absolute top-1/2 right-0 hidden size-9 -translate-y-1/2 translate-x-4 items-center justify-center rounded-full bg-white text-slate-500 shadow-sm ring-1 ring-slate-900/6 transition-colors duration-200 hover:text-slate-900 sm:flex"
          >
            <ChevronRight className="size-4" />
          </button>
        </div>

        <div className="mt-6 flex items-center justify-center gap-2">
          {HIGHLIGHTS.map((h, i) => (
            <button
              key={h.title}
              type="button"
              onClick={() => goTo(i)}
              aria-label={`Ke sorotan "${h.title}"`}
              aria-current={i === active}
              className={`h-1.5 rounded-full transition-all duration-300 ${
                i === active ? "w-6 bg-teal-600" : "w-1.5 bg-slate-300 hover:bg-slate-400"
              }`}
            />
          ))}
        </div>
      </div>
    </section>
  );
}
