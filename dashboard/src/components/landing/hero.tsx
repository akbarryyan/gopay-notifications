import Link from "next/link";
import { ArrowRight, Check, Smartphone, Webhook, Zap } from "lucide-react";
import { buttonVariants } from "@/components/ui/button";

const TRUST_POINTS = ["Hosted, tanpa server sendiri", "Webhook siap pakai", "Multi-device"];

export function Hero() {
  return (
    <section className="relative overflow-hidden border-b border-border/60">
      <div className="mx-auto grid max-w-6xl gap-12 px-4 py-20 sm:px-6 lg:grid-cols-2 lg:items-center lg:gap-8 lg:px-8 lg:py-28">
        {/* Kolom kiri: copy */}
        <div className="flex flex-col gap-6">
          <h1 className="font-(--font-lp-heading) text-4xl leading-[1.05] font-semibold tracking-tight text-balance sm:text-5xl lg:text-[3.25rem]">
            Notifikasi pembayaran, terhubung ke sistem kamu.
          </h1>
          <p className="max-w-lg text-lg text-muted-foreground">
            Ubah notifikasi pembayaran GoPay jadi event terstruktur dan webhook
            otomatis — dengan infrastruktur yang sudah kami operasikan, bukan
            yang harus kamu kelola sendiri.
          </p>
          <div className="flex flex-wrap items-center gap-3">
            <Link href="/register" className={buttonVariants({ size: "lg" })}>
              Mulai Gratis 3 Hari
              <ArrowRight className="ml-1.5 size-4" />
            </Link>
            <a href="#cara-kerja" className={buttonVariants({ variant: "outline", size: "lg" })}>
              Lihat Cara Kerja
            </a>
          </div>
          <ul className="flex flex-wrap gap-x-6 gap-y-2 pt-2">
            {TRUST_POINTS.map((point) => (
              <li key={point} className="flex items-center gap-1.5 text-sm text-muted-foreground">
                <Check className="size-4 text-primary" />
                {point}
              </li>
            ))}
          </ul>
        </div>

        {/* Kolom kanan: visual produk -- diagram alur, bukan ilustrasi generik */}
        <div className="relative">
          <div className="rounded-2xl border border-border/60 bg-card p-5 shadow-sm sm:p-6">
            <FlowNode icon={Smartphone} label="Aplikasi Pembayaran" sublabel="Notifikasi GoPay masuk" />
            <FlowArrow />
            <FlowNode icon={Zap} label="Payment Bridge" sublabel="Event diproses & dicocokkan" active />
            <FlowArrow />
            <FlowNode icon={Webhook} label="Webhook" sublabel="Terkirim ke sistem kamu" />

            <div className="mt-5 flex items-center justify-between rounded-lg border border-border/60 bg-secondary/40 px-3 py-2 text-xs">
              <span className="flex items-center gap-1.5 font-medium text-emerald-700 dark:text-emerald-400">
                <span className="size-1.5 rounded-full bg-emerald-500" />
                invoice.paid terkirim
              </span>
              <span className="font-mono text-muted-foreground">200 OK · 142ms</span>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}

function FlowNode({
  icon: Icon,
  label,
  sublabel,
  active,
}: {
  icon: typeof Smartphone;
  label: string;
  sublabel: string;
  active?: boolean;
}) {
  return (
    <div
      className={`flex items-center gap-3 rounded-xl border px-4 py-3 ${
        active ? "border-primary/40 bg-primary/5" : "border-border/60 bg-background"
      }`}
    >
      <span
        className={`flex size-9 shrink-0 items-center justify-center rounded-lg ${
          active ? "bg-primary text-primary-foreground" : "bg-secondary text-foreground"
        }`}
      >
        <Icon className="size-4.5" />
      </span>
      <div className="min-w-0">
        <p className="truncate text-sm font-semibold">{label}</p>
        <p className="truncate text-xs text-muted-foreground">{sublabel}</p>
      </div>
    </div>
  );
}

function FlowArrow() {
  return (
    <div className="flex justify-start pl-[1.15rem]">
      <div className="h-4 w-px bg-border" />
    </div>
  );
}
