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
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

function SectionHeading({
  eyebrow,
  title,
  description,
}: {
  eyebrow: string;
  title: string;
  description?: string;
}) {
  return (
    <div className="mx-auto max-w-2xl text-center">
      <Badge variant="secondary" className="mb-4">
        {eyebrow}
      </Badge>
      <h2 className="font-(--font-lp-heading) text-3xl font-semibold tracking-tight text-balance sm:text-4xl">
        {title}
      </h2>
      {description && <p className="mt-4 text-muted-foreground">{description}</p>}
    </div>
  );
}

/* ---------------------------------------------------------------------- */
/* Problem / Value proposition                                            */
/* ---------------------------------------------------------------------- */

export function ProblemSection() {
  return (
    <section className="border-b border-border/60 bg-secondary/20 py-20 sm:py-28">
      <div className="mx-auto max-w-5xl px-4 sm:px-6 lg:px-8">
        <SectionHeading
          eyebrow="Kenapa Payment Bridge"
          title="Notifikasi pembayaran kamu berhenti di HP saja"
        />
        <div className="mt-12 grid gap-6 sm:grid-cols-2">
          <Card className="border-none shadow-sm ring-1 ring-border/60">
            <CardHeader>
              <CardTitle className="text-base text-muted-foreground">Tanpa Payment Bridge</CardTitle>
            </CardHeader>
            <CardContent>
              <ol className="flex flex-col gap-4">
                {["Aplikasi pembayaran", "Notifikasi masuk", "Dicek manual satu per satu"].map(
                  (step, i) => (
                    <li key={step} className="flex items-center gap-3 text-sm">
                      <span className="flex size-6 shrink-0 items-center justify-center rounded-full bg-muted text-xs font-medium text-muted-foreground">
                        {i + 1}
                      </span>
                      {step}
                    </li>
                  ),
                )}
              </ol>
            </CardContent>
          </Card>
          <Card className="border-primary/30 shadow-sm ring-1 ring-primary/20">
            <CardHeader>
              <CardTitle className="text-base text-primary">Dengan Payment Bridge</CardTitle>
            </CardHeader>
            <CardContent>
              <ol className="flex flex-col gap-4">
                {[
                  "Aplikasi pembayaran",
                  "Notifikasi masuk",
                  "Event terstruktur otomatis",
                  "Webhook ke sistem kamu",
                ].map((step, i) => (
                  <li key={step} className="flex items-center gap-3 text-sm">
                    <span className="flex size-6 shrink-0 items-center justify-center rounded-full bg-primary text-xs font-medium text-primary-foreground">
                      {i + 1}
                    </span>
                    {step}
                  </li>
                ))}
              </ol>
            </CardContent>
          </Card>
        </div>
      </div>
    </section>
  );
}

/* ---------------------------------------------------------------------- */
/* How it works                                                           */
/* ---------------------------------------------------------------------- */

const STEPS = [
  { icon: Smartphone, title: "Aplikasi pembayaran", desc: "Notifikasi GoPay Merchant masuk di HP kamu." },
  { icon: Bell, title: "Android Bridge", desc: "Aplikasi pendamping membaca notifikasi itu secara otomatis." },
  { icon: Zap, title: "Pemrosesan event", desc: "Nominal dicocokkan ke invoice yang sedang menunggu." },
  { icon: Webhook, title: "Webhook", desc: "invoice.paid dikirim ke endpoint yang kamu daftarkan." },
  { icon: Receipt, title: "Sistem kamu", desc: "Order otomatis diproses tanpa pengecekan manual." },
];

export function HowItWorksSection() {
  return (
    <section id="cara-kerja" className="border-b border-border/60 py-20 sm:py-28">
      <div className="mx-auto max-w-6xl px-4 sm:px-6 lg:px-8">
        <SectionHeading eyebrow="Cara kerja" title="Dari notifikasi sampai ke sistem kamu" />
        <div className="mt-14 grid gap-8 sm:grid-cols-2 lg:grid-cols-5 lg:gap-4">
          {STEPS.map((step, i) => (
            <div key={step.title} className="relative flex flex-col items-center gap-3 text-center">
              {i < STEPS.length - 1 && (
                <div className="absolute top-6 left-1/2 hidden h-px w-full bg-border lg:block" />
              )}
              <span className="relative z-10 flex size-12 items-center justify-center rounded-full border border-border/60 bg-background text-sm font-semibold">
                {String(i + 1).padStart(2, "0")}
              </span>
              <step.icon className="size-5 text-primary" />
              <p className="text-sm font-semibold">{step.title}</p>
              <p className="text-sm text-muted-foreground">{step.desc}</p>
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
    <section id="fitur" className="border-b border-border/60 bg-secondary/20 py-20 sm:py-28">
      <div className="mx-auto max-w-6xl px-4 sm:px-6 lg:px-8">
        <SectionHeading
          eyebrow="Fitur"
          title="Semua yang dibutuhkan untuk otomasi pembayaran"
        />
        <div className="mt-14 grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
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
    <Card className={`border-none shadow-sm ring-1 ring-border/60 ${className ?? ""}`}>
      <CardHeader>
        <span className="mb-2 flex size-10 items-center justify-center rounded-lg bg-primary/10 text-primary">
          <Icon className="size-5" />
        </span>
        <CardTitle className="text-base">{title}</CardTitle>
      </CardHeader>
      <CardContent className="text-sm text-muted-foreground">{desc}</CardContent>
    </Card>
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
    <section className="border-b border-border/60 py-20 sm:py-28">
      <div className="mx-auto max-w-3xl px-4 sm:px-6 lg:px-8">
        <SectionHeading
          eyebrow="Arsitektur"
          title="Dibangun sebagai lapisan integrasi"
          description="Payment Bridge menjadi penghubung antara notifikasi pembayaran dan sistem yang butuh mengonsumsinya — kamu tidak perlu menyiapkan server sendiri."
        />
        <div className="mx-auto mt-12 flex max-w-sm flex-col items-center">
          {layers.map((layer, i) => (
            <div key={layer.label} className="flex w-full flex-col items-center">
              <div className="flex w-full items-center gap-3 rounded-xl border border-border/60 bg-card px-4 py-3 shadow-sm">
                <span className="flex size-9 items-center justify-center rounded-lg bg-secondary">
                  <layer.icon className="size-4.5" />
                </span>
                <span className="text-sm font-semibold">{layer.label}</span>
              </div>
              {i < layers.length - 1 && (
                <ArrowRight className="my-2 size-4 rotate-90 text-muted-foreground" />
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
    <section className="border-b border-border/60 bg-secondary/20 py-20 sm:py-28">
      <div className="mx-auto max-w-6xl px-4 sm:px-6 lg:px-8">
        <SectionHeading eyebrow="Use case" title="Dipakai untuk apa saja" />
        <div className="mt-14 grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {USE_CASES.map((uc) => (
            <div key={uc.title} className="rounded-xl border border-border/60 bg-card p-5">
              <uc.icon className="mb-3 size-5 text-primary" />
              <p className="text-sm font-semibold">{uc.title}</p>
              <p className="mt-1 text-sm text-muted-foreground">{uc.desc}</p>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}
