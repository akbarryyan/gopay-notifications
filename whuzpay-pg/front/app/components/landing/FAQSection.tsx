"use client";

import { useState } from "react";
import Container from "./Container";
import { focusRing } from "./styles";

const faqs = [
  {
    question: "Apa itu WhuzPay?",
    answer:
      "Platform pembayaran QRIS yang menyatukan beberapa penyedia pembayaran di balik satu API dan satu dashboard. Anda cukup berintegrasi sekali; urusan memilih provider, menormalkan status, dan mengirim notifikasi kami yang tangani.",
  },
  {
    question: "Berapa biaya layanan WhuzPay?",
    answer:
      "Biaya dihitung per transaksi berhasil, tanpa biaya langganan bulanan dan tanpa biaya integrasi. Besarannya menyesuaikan volume transaksi, jadi hubungi tim kami untuk penawaran yang sesuai dengan skala bisnis Anda.",
  },
  {
    question: "Bagaimana cara mulai menggunakan WhuzPay?",
    answer:
      "Daftar, ambil API key sandbox dari dashboard, lalu panggil endpoint pembuatan pembayaran. Environment sandbox berjalan tanpa uang sungguhan dan tanpa kartu kredit, jadi Anda bisa menguji seluruh alur sebelum mengaktifkan akun produksi.",
  },
  {
    question: "Bagaimana saya tahu sebuah pembayaran sudah lunas?",
    answer:
      "WhuzPay mengirim webhook bertanda tangan HMAC ke URL yang Anda daftarkan begitu status berubah, jadi tidak perlu polling. Pengiriman yang gagal dicoba ulang otomatis, dan seluruh riwayatnya bisa dilihat di dashboard.",
  },
];

function DocIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2} className="h-4 w-4">
      <path d="M7 3h7l4 4v14H7z" strokeLinecap="round" strokeLinejoin="round" />
      <path d="M14 3v4h4" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  );
}

function Illustration() {
  return (
    <div className="relative mx-auto aspect-4/3 w-full max-w-sm">
      <div className="absolute inset-6 rounded-full border-2 border-dashed border-brand-navy-light/30" />
      <div className="absolute left-1/2 top-1/2 w-56 -translate-x-1/2 -translate-y-1/2 rounded-2xl border border-slate-100 bg-white p-4 shadow-xl">
        <div className="mb-3 flex items-center gap-2">
          <span className="h-2.5 w-2.5 rounded-full bg-rose-400" />
          <span className="h-2.5 w-2.5 rounded-full bg-brand-yellow" />
          <span className="h-2.5 w-2.5 rounded-full bg-teal-400" />
        </div>
        <div className="space-y-2">
          <div className="h-2 w-3/4 rounded bg-slate-100" />
          <div className="h-2 w-full rounded bg-slate-100" />
          <div className="h-2 w-2/3 rounded bg-slate-100" />
        </div>
      </div>
      <span className="absolute bottom-2 right-6 flex h-11 w-11 items-center justify-center rounded-full bg-teal-500 text-white shadow-lg">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2} className="h-5 w-5">
          <path d="M5 12h14M13 6l6 6-6 6" strokeLinecap="round" strokeLinejoin="round" />
        </svg>
      </span>
    </div>
  );
}

export default function FAQSection() {
  const [open, setOpen] = useState(0);

  return (
    <section id="faq" className="scroll-mt-24 bg-sky-50 py-20 sm:py-28">
      <Container className="grid gap-12 lg:grid-cols-2 lg:items-center lg:gap-16">
        <div>
          <Illustration />
          <h2 className="mt-10 text-3xl font-extrabold leading-tight tracking-tight text-brand-navy sm:text-4xl">
            Pertanyaan yang sering muncul sebelum mulai
          </h2>
          <p className="mt-4 max-w-md text-base leading-relaxed text-slate-600 sm:text-lg">
            Berikut rangkuman hal yang biasanya ingin dipastikan sebelum
            integrasi berjalan. Jika masih ada kebutuhan khusus, tim kami siap
            bantu diskusikan alur yang paling sesuai.
          </p>
        </div>

        <div className="rounded-2xl bg-white p-6 shadow-sm sm:p-8">
          <div className="mb-4 flex items-center gap-2 text-brand-navy">
            <DocIcon />
            <span className="text-sm font-semibold uppercase tracking-wider">
              Pertanyaan yang sering diajukan
            </span>
          </div>

          <div className="divide-y divide-slate-100">
            {faqs.map((faq, i) => {
              const expanded = open === i;
              return (
                <div key={faq.question}>
                  <button
                    type="button"
                    onClick={() => setOpen(expanded ? -1 : i)}
                    aria-expanded={expanded}
                    aria-controls={`faq-answer-${i}`}
                    className={`flex w-full items-center justify-between gap-4 rounded-md py-4 text-left ${focusRing}`}
                  >
                    <span className="text-base font-medium text-slate-800 sm:text-lg">
                      {faq.question}
                    </span>
                    <span
                      className={`flex h-7 w-7 shrink-0 items-center justify-center rounded-full border transition-transform duration-300 ${
                        expanded
                          ? "rotate-45 border-brand-navy text-brand-navy"
                          : "border-slate-200 text-slate-400"
                      }`}
                    >
                      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2} className="h-3.5 w-3.5">
                        <path d="M12 5v14M5 12h14" strokeLinecap="round" />
                      </svg>
                    </span>
                  </button>

                  {/* 0fr → 1fr animates to the answer's natural height, which a
                      plain max-h transition can only approximate. */}
                  <div
                    id={`faq-answer-${i}`}
                    className={`grid transition-all duration-300 ease-out motion-reduce:transition-none ${
                      expanded ? "grid-rows-[1fr] opacity-100" : "grid-rows-[0fr] opacity-0"
                    }`}
                  >
                    <div className="overflow-hidden">
                      <p className="pb-5 pr-10 text-base leading-relaxed text-slate-600">
                        {faq.answer}
                      </p>
                    </div>
                  </div>
                </div>
              );
            })}
          </div>
        </div>
      </Container>
    </section>
  );
}
