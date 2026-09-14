import Section from "./Section";
import DashboardMockup from "./DashboardMockup";
import CheckoutMockup from "./CheckoutMockup";
import { cardHeading, bodyText } from "./styles";

function CheckIcon() {
  return (
    <svg
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth={2.5}
      className="mt-0.5 h-4 w-4 shrink-0 text-teal-500"
      aria-hidden="true"
    >
      <path d="m5 13 4 4L19 7" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  );
}

function Row({
  eyebrow,
  title,
  description,
  points,
  visual,
  reverse = false,
}: {
  eyebrow: string;
  title: string;
  description: string;
  points: string[];
  visual: React.ReactNode;
  reverse?: boolean;
}) {
  return (
    <div className="grid items-center gap-12 lg:grid-cols-2 lg:gap-16">
      <div className={reverse ? "lg:order-2" : ""}>
        <p className="text-sm font-semibold uppercase tracking-wider text-brand-navy-light">
          {eyebrow}
        </p>
        <h3 className={`mt-3 ${cardHeading} sm:text-2xl`}>{title}</h3>
        <p className={`mt-4 ${bodyText}`}>{description}</p>
        <ul className="mt-6 flex flex-col gap-3">
          {points.map((point) => (
            <li key={point} className="flex gap-3 text-base text-slate-600">
              <CheckIcon />
              <span>{point}</span>
            </li>
          ))}
        </ul>
      </div>

      <div className={reverse ? "lg:order-1" : ""}>{visual}</div>
    </div>
  );
}

export default function ProductShowcase() {
  return (
    <Section
      id="produk"
      background="slate"
      eyebrow="Produk"
      title="Dua sisi yang sama-sama perlu rapi"
      subtitle="Yang dilihat tim Anda saat memantau transaksi, dan yang dilihat pelanggan saat membayar."
      containerClassName="max-w-6xl"
    >
      <div className="flex flex-col gap-20 sm:gap-28">
        <Row
          eyebrow="Dashboard merchant"
          title="Semua transaksi dalam satu layar"
          description="Pantau pembayaran yang masuk, yang tertunda, dan yang gagal tanpa perlu bertanya ke siapa pun. Setiap perubahan status tercatat lengkap dengan waktunya."
          points={[
            "Ringkasan harian: sukses, pending, dan gagal terpisah jelas",
            "Riwayat pengiriman webhook beserta percobaan ulangnya",
            "Ekspor CSV untuk rekonsiliasi dengan pembukuan",
          ]}
          visual={<DashboardMockup />}
        />

        <Row
          reverse
          eyebrow="Halaman checkout"
          title="Pelanggan cukup memindai lalu selesai"
          description="Halaman pembayaran siap pakai yang bisa langsung dibagikan lewat tautan — tanpa perlu membangun antarmuka checkout sendiri."
          points={[
            "Status pembayaran diperbarui otomatis tanpa refresh",
            "Hitung mundur kedaluwarsa terlihat jelas oleh pembayar",
            "Tautan pembayaran sekali pakai maupun yang bisa dipakai berulang",
          ]}
          visual={<CheckoutMockup />}
        />
      </div>
    </Section>
  );
}
