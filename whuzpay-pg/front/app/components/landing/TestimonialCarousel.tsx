"use client";

import { useState } from "react";
import Section from "./Section";
import { focusRing } from "./styles";

/**
 * Placeholder testimonials — invented companies and people, kept as a slot
 * to fill once there are real customers to quote.
 */
const testimonials = [
  {
    company: "Vocaria Games",
    quote:
      "Kami butuh pembayaran yang statusnya bisa dipercaya. Sejak pindah, tim operasional tidak lagi menebak-nebak transaksi mana yang benar-benar masuk.",
    name: "Hardi Wijaya",
    role: "Chief Commercial Officer",
  },
  {
    company: "Nusantara Retail",
    quote:
      "Integrasinya selesai dalam satu sore. Yang paling membantu justru webhook-nya — kami tidak perlu lagi polling status tiap beberapa detik.",
    name: "Sinta Marlina",
    role: "Engineering Lead",
  },
  {
    company: "Bahari Logistik",
    quote:
      "Rekonsiliasi harian yang dulu memakan waktu dua jam sekarang tinggal ekspor. Riwayat webhook-nya lengkap, jadi selisih gampang ditelusuri.",
    name: "Rizal Kurniawan",
    role: "Finance Manager",
  },
];

function ArrowButton({
  direction,
  onClick,
}: {
  direction: "left" | "right";
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      aria-label={direction === "left" ? "Testimoni sebelumnya" : "Testimoni berikutnya"}
      className={`flex h-10 w-10 shrink-0 items-center justify-center rounded-full border border-slate-200 bg-white text-slate-400 transition-colors hover:border-brand-navy hover:text-brand-navy ${focusRing}`}
    >
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2} className="h-4 w-4">
        {direction === "left" ? (
          <path d="m15 6-6 6 6 6" strokeLinecap="round" strokeLinejoin="round" />
        ) : (
          <path d="m9 6 6 6-6 6" strokeLinecap="round" strokeLinejoin="round" />
        )}
      </svg>
    </button>
  );
}

export default function TestimonialCarousel() {
  const [index, setIndex] = useState(0);
  const active = testimonials[index];

  const go = (delta: number) =>
    setIndex((i) => (i + delta + testimonials.length) % testimonials.length);

  return (
    <Section background="slate" className="py-16 sm:py-20">
      <div
        role="group"
        aria-roledescription="carousel"
        aria-label="Testimoni pelanggan"
        onKeyDown={(e) => {
          if (e.key === "ArrowLeft") go(-1);
          if (e.key === "ArrowRight") go(1);
        }}
        tabIndex={0}
        className={`mx-auto max-w-3xl rounded-2xl border border-slate-200 bg-white p-8 text-center shadow-sm sm:p-12 ${focusRing}`}
      >
        <div aria-live="polite">
          <p className="text-base font-bold uppercase tracking-wider text-brand-navy-light">
            {active.company}
          </p>
          <blockquote className="mt-6 text-lg leading-relaxed text-slate-600 sm:text-xl">
            &ldquo;{active.quote}&rdquo;
          </blockquote>
          <p className="mt-6 text-base font-bold text-slate-800">{active.name}</p>
          <p className="text-sm text-slate-400">
            {active.role} · {active.company}
          </p>
        </div>

        <div className="mt-8 flex items-center justify-center gap-4">
          <ArrowButton direction="left" onClick={() => go(-1)} />

          <div className="flex gap-2">
            {testimonials.map((testimonial, i) => (
              <button
                key={testimonial.company}
                type="button"
                onClick={() => setIndex(i)}
                aria-label={`Testimoni ${i + 1} dari ${testimonials.length}`}
                aria-current={i === index}
                className={`h-2 rounded-full transition-all ${focusRing} ${
                  i === index ? "w-6 bg-brand-navy" : "w-2 bg-slate-300 hover:bg-slate-400"
                }`}
              />
            ))}
          </div>

          <ArrowButton direction="right" onClick={() => go(1)} />
        </div>
      </div>
    </Section>
  );
}
