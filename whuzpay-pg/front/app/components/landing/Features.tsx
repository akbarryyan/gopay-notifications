import Section from "./Section";
import { cardHeading, focusRing } from "./styles";

function CardIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2} className="h-5 w-5">
      <rect x="3" y="6" width="18" height="12" rx="2" />
      <path d="M3 10h18" />
    </svg>
  );
}

function ChartIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2} className="h-5 w-5">
      <path d="M4 20V10M10 20V4M16 20v-7M22 20H2" strokeLinecap="round" />
    </svg>
  );
}

function HeadsetIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2} className="h-5 w-5">
      <path d="M4 13a8 8 0 0 1 16 0" strokeLinecap="round" />
      <rect x="3" y="13" width="4" height="6" rx="1.5" />
      <rect x="17" y="13" width="4" height="6" rx="1.5" />
      <path d="M20 19v1a3 3 0 0 1-3 3h-3" strokeLinecap="round" />
    </svg>
  );
}

function ShieldIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2} className="h-5 w-5">
      <path d="M12 3l7 3v6c0 4.5-3 7.5-7 9-4-1.5-7-4.5-7-9V6l7-3Z" strokeLinejoin="round" />
      <path d="m9 12 2 2 4-4" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  );
}

const features = [
  {
    icon: <CardIcon />,
    iconClass: "bg-teal-500",
    title: "Penerimaan pembayaran",
    description:
      "Bangun alur pembayaran yang sederhana untuk pelanggan dan mudah dikontrol untuk tim Anda — checkout, invoice, maupun tagihan operasional dengan status yang jelas.",
    tags: ["QRIS", "Checkout", "Invoice", "API", "Webhook"],
  },
  {
    icon: <ChartIcon />,
    iconClass: "bg-rose-400",
    title: "Monitoring & transparansi",
    description:
      "Pantau setiap transaksi secara real-time. Status pembayaran, pengiriman webhook, dan riwayatnya tercatat rapi dalam satu dashboard.",
    tags: ["Real-time", "Webhook", "Logs", "Ekspor CSV"],
  },
  {
    icon: <HeadsetIcon />,
    iconClass: "bg-brand-navy-light",
    title: "Support yang responsif",
    description:
      "Tim kami membantu saat onboarding, menguji integrasi, atau menelusuri transaksi bermasalah. Pembayaran tidak boleh menunggu terlalu lama.",
    tags: ["Telepon", "Email", "Live chat"],
  },
  {
    icon: <ShieldIcon />,
    iconClass: "bg-brand-yellow",
    title: "Dibangun untuk operasional",
    description:
      "Bisnis butuh lebih dari sekadar tombol bayar: status yang akurat, webhook yang rapi, dan kontrol provider yang tetap andal saat transaksi bertambah.",
    tags: ["Multi-provider", "Failover", "Rekonsiliasi"],
  },
];

export default function Features() {
  return (
    <Section
      id="solusi"
      eyebrow="Solusi"
      title="Dirancang untuk transaksi hari ini dan pertumbuhan berikutnya"
      subtitle="Bukan sekadar menerima pembayaran, tapi membangun fondasi operasional yang lebih rapi."
    >
      <div className="grid gap-6 sm:grid-cols-2">
        {features.map((feature) => (
          <article
            key={feature.title}
            className="flex flex-col rounded-2xl border border-slate-200 bg-white p-8 shadow-sm transition-shadow hover:shadow-md"
          >
            <span
              className={`flex h-11 w-11 items-center justify-center rounded-xl ${feature.iconClass} text-white`}
            >
              {feature.icon}
            </span>
            <h3 className={`mt-5 ${cardHeading}`}>{feature.title}</h3>
            <p className="mt-3 flex-1 text-base leading-relaxed text-slate-600">
              {feature.description}
            </p>
            <div className="mt-6 flex flex-wrap gap-2">
              {feature.tags.map((tag) => (
                <span
                  key={tag}
                  className="rounded-full bg-slate-100 px-3 py-1 text-xs font-semibold text-slate-600"
                >
                  {tag}
                </span>
              ))}
            </div>
          </article>
        ))}
      </div>

      <div className="mt-10 text-center">
        <a
          href="#integrasi"
          className={`inline-flex items-center gap-2 text-base font-semibold text-brand-navy hover:underline ${focusRing} rounded-md`}
        >
          Lihat cara mengintegrasikannya
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2} className="h-4 w-4">
            <path d="M5 12h14M13 6l6 6-6 6" strokeLinecap="round" strokeLinejoin="round" />
          </svg>
        </a>
      </div>
    </Section>
  );
}
