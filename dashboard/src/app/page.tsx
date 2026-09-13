import Link from "next/link";
import { Zap, Smartphone, Repeat, Webhook } from "lucide-react";
import { buttonVariants } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

const FEATURES = [
  {
    icon: Repeat,
    title: "Nominal unik",
    desc: "Tiap invoice dapat nominal yang sedikit berbeda -- tidak ada lagi tebak-tebak siapa yang bayar berapa.",
  },
  {
    icon: Webhook,
    title: "Webhook otomatis",
    desc: "invoice.paid dan invoice.expired terkirim otomatis ke website kamu, lengkap dengan retry kalau gagal.",
  },
  {
    icon: Smartphone,
    title: "Dashboard realtime",
    desc: "Overview, Devices, Events, Transactions -- semua ter-update otomatis begitu notifikasi GoPay masuk.",
  },
  {
    icon: Zap,
    title: "Konsol pengecualian",
    desc: "Transaksi yang nominalnya tidak cocok otomatis, tidak hilang begitu saja -- tetap kelihatan dan bisa dicocokkan manual.",
  },
];

const STEPS = [
  "Pasang aplikasi Android di HP yang menerima notifikasi GoPay Merchant.",
  "Notifikasi pembayaran otomatis tercatat dan dikirim ke sistem kami.",
  "Invoice otomatis lunas begitu nominal cocok, webhook langsung terkirim ke website kamu.",
];

export default function LandingPage() {
  return (
    <div className="flex min-h-screen flex-col">
      <header className="flex items-center justify-between px-6 py-4">
        <span className="flex items-center gap-2 font-semibold">
          <span className="flex size-8 items-center justify-center rounded-lg bg-primary text-primary-foreground">
            <Zap className="size-4" fill="currentColor" strokeWidth={0} />
          </span>
          Payment Bridge
        </span>
        <nav className="flex items-center gap-2">
          <Link href="/login" className={buttonVariants({ variant: "ghost", size: "sm" })}>
            Masuk
          </Link>
          <Link href="/register" className={buttonVariants({ size: "sm" })}>
            Daftar Gratis 3 Hari
          </Link>
        </nav>
      </header>

      <main className="flex-1">
        {/* Hero */}
        <section className="mx-auto flex max-w-2xl flex-col items-center gap-6 px-6 py-20 text-center">
          <h1 className="text-4xl font-semibold tracking-tight sm:text-5xl">
            Pantau pembayaran GoPay tanpa daftar ke Midtrans
          </h1>
          <p className="text-lg text-muted-foreground">
            Notifikasi GoPay Merchant di HP kamu jadi invoice otomatis lunas
            dan webhook ke website kamu — tanpa integrasi payment gateway
            yang ribet.
          </p>
          <div className="flex flex-wrap items-center justify-center gap-3">
            <Link href="/register" className={buttonVariants({ size: "lg" })}>
              Daftar Gratis 3 Hari
            </Link>
            <Link href="/login" className={buttonVariants({ variant: "outline", size: "lg" })}>
              Masuk
            </Link>
          </div>
        </section>

        {/* Cara kerja */}
        <section className="mx-auto max-w-3xl px-6 py-12">
          <h2 className="mb-8 text-center text-2xl font-semibold">Cara kerja</h2>
          <ol className="flex flex-col gap-6 sm:flex-row">
            {STEPS.map((step, i) => (
              <li key={i} className="flex flex-1 flex-col items-center gap-3 text-center">
                <span className="flex size-9 items-center justify-center rounded-full bg-primary text-sm font-semibold text-primary-foreground">
                  {i + 1}
                </span>
                <p className="text-sm text-muted-foreground">{step}</p>
              </li>
            ))}
          </ol>
        </section>

        {/* Fitur */}
        <section className="mx-auto max-w-4xl px-6 py-12">
          <h2 className="mb-8 text-center text-2xl font-semibold">Fitur</h2>
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            {FEATURES.map((f) => (
              <Card key={f.title} className="border-none shadow-sm ring-1 ring-border/60">
                <CardHeader className="flex-row items-center gap-3 space-y-0">
                  <span className="flex size-9 items-center justify-center rounded-lg bg-secondary">
                    <f.icon className="size-4.5" />
                  </span>
                  <CardTitle className="text-base">{f.title}</CardTitle>
                </CardHeader>
                <CardContent className="text-sm text-muted-foreground">{f.desc}</CardContent>
              </Card>
            ))}
          </div>
        </section>

        {/* CTA penutup */}
        <section className="mx-auto flex max-w-2xl flex-col items-center gap-4 px-6 py-20 text-center">
          <h2 className="text-2xl font-semibold">Coba sekarang, gratis 3 hari</h2>
          <Link href="/register" className={buttonVariants({ size: "lg" })}>
            Daftar Gratis 3 Hari
          </Link>
        </section>
      </main>

      <footer className="border-t border-border/60 px-6 py-6 text-center text-sm text-muted-foreground">
        Payment Bridge © {new Date().getFullYear()}
      </footer>
    </div>
  );
}
