import Link from "next/link";
import { ArrowRight, Check, Zap } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { buttonVariants } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

function SectionHeading({ eyebrow, title }: { eyebrow: string; title: string }) {
  return (
    <div className="mx-auto max-w-2xl text-center">
      <Badge variant="secondary" className="mb-4">
        {eyebrow}
      </Badge>
      <h2 className="font-(--font-lp-heading) text-3xl font-semibold tracking-tight text-balance sm:text-4xl">
        {title}
      </h2>
    </div>
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
    <section className="border-b border-border/60 py-20 sm:py-28">
      <div className="mx-auto max-w-4xl px-4 sm:px-6 lg:px-8">
        <SectionHeading eyebrow="Dashboard" title="Semua transaksi, satu layar" />
        <div className="mt-14 overflow-hidden rounded-2xl border border-border/60 bg-card shadow-lg">
          <div className="flex items-center gap-1.5 border-b border-border/60 px-4 py-3">
            <span className="size-2.5 rounded-full bg-red-400" />
            <span className="size-2.5 rounded-full bg-amber-400" />
            <span className="size-2.5 rounded-full bg-emerald-400" />
            <span className="ml-3 text-xs text-muted-foreground">Overview</span>
          </div>
          <div className="grid grid-cols-2 gap-px bg-border/60 sm:grid-cols-4">
            {stats.map((s) => (
              <div key={s.label} className="bg-card p-4">
                <p className="text-xs text-muted-foreground">{s.label}</p>
                <p className="mt-1 text-lg font-semibold">{s.value}</p>
              </div>
            ))}
          </div>
          <div className="border-t border-border/60 p-4">
            <p className="mb-3 text-xs font-medium text-muted-foreground">Event terbaru</p>
            <div className="flex flex-col gap-2">
              {recent.map((r, i) => (
                <div
                  key={i}
                  className="flex items-center justify-between rounded-lg bg-secondary/40 px-3 py-2 text-sm"
                >
                  <span className="flex items-center gap-2">
                    <Check className="size-3.5 text-emerald-600 dark:text-emerald-400" />
                    Pembayaran diterima
                  </span>
                  <span className="font-mono text-xs text-muted-foreground">{r.amount}</span>
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
/* Pricing                                                                */
/* ---------------------------------------------------------------------- */

const PLANS = [
  {
    name: "Starter",
    desc: "Untuk usaha kecil yang baru mulai.",
    devices: "3 device",
    highlight: false,
  },
  {
    name: "Business",
    desc: "Untuk operasional yang sedang berkembang.",
    devices: "10 device",
    highlight: true,
  },
  {
    name: "Enterprise",
    desc: "Untuk kebutuhan volume tinggi.",
    devices: "Device tanpa batas",
    highlight: false,
  },
];

export function PricingSection() {
  return (
    <section id="harga" className="border-b border-border/60 bg-secondary/20 py-20 sm:py-28">
      <div className="mx-auto max-w-5xl px-4 sm:px-6 lg:px-8">
        <SectionHeading eyebrow="Harga" title="Satu paket untuk tiap skala usaha" />
        <div className="mt-14 grid gap-6 sm:grid-cols-3">
          {PLANS.map((plan) => (
            <Card
              key={plan.name}
              className={
                plan.highlight
                  ? "border-primary/40 shadow-md ring-2 ring-primary/30"
                  : "border-none shadow-sm ring-1 ring-border/60"
              }
            >
              <CardHeader>
                {plan.highlight && (
                  <Badge className="mb-2 w-fit border-transparent bg-primary text-primary-foreground">
                    Direkomendasikan
                  </Badge>
                )}
                <CardTitle className="text-lg">{plan.name}</CardTitle>
                <p className="text-sm text-muted-foreground">{plan.desc}</p>
              </CardHeader>
              <CardContent className="flex flex-col gap-4">
                <p className="text-sm font-medium">{plan.devices}</p>
                <ul className="flex flex-col gap-2 text-sm text-muted-foreground">
                  <li className="flex items-center gap-2">
                    <Check className="size-4 text-primary" /> Webhook & retry
                  </li>
                  <li className="flex items-center gap-2">
                    <Check className="size-4 text-primary" /> Dashboard realtime
                  </li>
                  <li className="flex items-center gap-2">
                    <Check className="size-4 text-primary" /> Konsol pengecualian
                  </li>
                </ul>
                <Link
                  href="/register"
                  className={buttonVariants({ variant: plan.highlight ? "default" : "outline" })}
                >
                  Mulai Gratis 3 Hari
                </Link>
              </CardContent>
            </Card>
          ))}
        </div>
        <p className="mt-8 text-center text-sm text-muted-foreground">
          Butuh paket khusus?{" "}
          <a href="#" className="font-medium text-foreground hover:underline">
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
    <section id="faq" className="border-b border-border/60 py-20 sm:py-28">
      <div className="mx-auto max-w-2xl px-4 sm:px-6 lg:px-8">
        <SectionHeading eyebrow="FAQ" title="Pertanyaan yang sering ditanyakan" />
        <div className="mt-12 flex flex-col divide-y divide-border/60 rounded-2xl border border-border/60">
          {FAQS.map((faq) => (
            <details key={faq.q} className="group px-5 py-4 open:pb-4">
              <summary className="flex cursor-pointer list-none items-center justify-between gap-4 text-sm font-medium marker:content-none">
                {faq.q}
                <span className="shrink-0 text-muted-foreground transition-transform group-open:rotate-45">
                  +
                </span>
              </summary>
              <p className="mt-3 text-sm text-muted-foreground">{faq.a}</p>
            </details>
          ))}
        </div>
      </div>
    </section>
  );
}

/* ---------------------------------------------------------------------- */
/* Final CTA                                                              */
/* ---------------------------------------------------------------------- */

export function FinalCtaSection() {
  return (
    <section className="border-b border-border/60 py-20 sm:py-28">
      <div className="mx-auto flex max-w-2xl flex-col items-center gap-6 px-4 text-center sm:px-6 lg:px-8">
        <h2 className="font-(--font-lp-heading) text-3xl font-semibold tracking-tight text-balance sm:text-4xl">
          Hubungkan pembayaran ke sistem yang sudah kamu pakai
        </h2>
        <p className="text-muted-foreground">
          Mulai bangun alur kerja pembayaran yang lebih otomatis, dengan
          infrastruktur yang sudah kami siapkan.
        </p>
        <div className="flex flex-wrap items-center justify-center gap-3">
          <Link href="/register" className={buttonVariants({ size: "lg" })}>
            Mulai Gratis 3 Hari
            <ArrowRight className="ml-1.5 size-4" />
          </Link>
          <a href="#fitur" className={buttonVariants({ variant: "outline", size: "lg" })}>
            Lihat Fitur
          </a>
        </div>
      </div>
    </section>
  );
}

/* ---------------------------------------------------------------------- */
/* Footer                                                                 */
/* ---------------------------------------------------------------------- */

export function LandingFooter() {
  return (
    <footer className="py-10">
      <div className="mx-auto flex max-w-6xl flex-col items-center gap-4 px-4 text-center sm:px-6 lg:flex-row lg:justify-between lg:text-left lg:px-8">
        <div className="flex items-center gap-2 font-semibold">
          <span className="flex size-7 items-center justify-center rounded-lg bg-primary text-primary-foreground">
            <Zap className="size-3.5" fill="currentColor" strokeWidth={0} />
          </span>
          Payment Bridge
        </div>
        <nav className="flex flex-wrap items-center justify-center gap-x-6 gap-y-2 text-sm text-muted-foreground">
          <a href="#fitur" className="hover:text-foreground">
            Fitur
          </a>
          <a href="#cara-kerja" className="hover:text-foreground">
            Cara Kerja
          </a>
          <a href="#harga" className="hover:text-foreground">
            Harga
          </a>
          <a href="#faq" className="hover:text-foreground">
            FAQ
          </a>
          <Link href="/login" className="hover:text-foreground">
            Masuk
          </Link>
        </nav>
        <p className="text-xs text-muted-foreground">
          © {new Date().getFullYear()} Payment Bridge
        </p>
      </div>
    </footer>
  );
}
