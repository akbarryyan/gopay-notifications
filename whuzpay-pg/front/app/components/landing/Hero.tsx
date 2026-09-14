import Link from "next/link";
import Container from "./Container";
import DashboardMockup from "./DashboardMockup";
import { btnPrimary, btnOnNavy, focusRingDark } from "./styles";

const trustPoints = [
  "Sandbox gratis",
  "Tanpa kartu kredit",
  "Integrasi dalam hitungan menit",
];

export default function Hero() {
  return (
    <section id="beranda" className="relative overflow-hidden bg-brand-navy">
      <div className="absolute inset-0 bg-linear-to-br from-brand-navy-light via-brand-navy to-brand-navy-dark" />
      {/* Soft glow, purely decorative — keeps the flat navy from reading as a slab. */}
      <div className="pointer-events-none absolute -right-24 -top-24 h-96 w-96 rounded-full bg-brand-navy-light/40 blur-3xl" />

      <Container className="relative grid items-center gap-14 pb-32 pt-32 sm:pb-40 sm:pt-40 lg:grid-cols-2 lg:gap-12">
        <div>
          <span className="inline-flex items-center gap-2 rounded-full border border-white/20 bg-white/10 px-4 py-1.5 text-xs font-semibold uppercase tracking-wider text-white/80">
            <span className="h-1.5 w-1.5 rounded-full bg-brand-yellow" />
            Payment gateway QRIS
          </span>

          <h1 className="mt-6 text-4xl font-extrabold leading-tight tracking-tight text-white sm:text-5xl lg:text-6xl">
            Satu integrasi, semua pembayaran.
          </h1>

          <p className="mt-6 max-w-xl text-base leading-relaxed text-white/75 sm:text-lg">
            Hubungkan bisnis Anda dengan berbagai metode pembayaran lewat satu
            platform yang dirancang untuk kecepatan, kemudahan, dan
            skalabilitas.
          </p>

          <div className="mt-9 flex flex-wrap gap-3">
            <Link href="/register" className={`${btnPrimary} ${focusRingDark}`}>
              Mulai Gratis
            </Link>
            <a href="#produk" className={`${btnOnNavy} ${focusRingDark}`}>
              Lihat Solusi
            </a>
          </div>

          <ul className="mt-8 flex flex-wrap items-center gap-x-6 gap-y-2">
            {trustPoints.map((point) => (
              <li
                key={point}
                className="flex items-center gap-2 text-sm text-white/60"
              >
                <svg
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth={2.5}
                  className="h-3.5 w-3.5 text-brand-yellow"
                  aria-hidden="true"
                >
                  <path d="m5 13 4 4L19 7" strokeLinecap="round" strokeLinejoin="round" />
                </svg>
                {point}
              </li>
            ))}
          </ul>
        </div>

        <DashboardMockup />
      </Container>

      {/* Curved hand-off into the next (white) section, replacing the diagonal
          clip-path that used to cut across the dashboard mockup. */}
      <svg
        className="absolute inset-x-0 bottom-0 w-full text-white"
        viewBox="0 0 1440 120"
        preserveAspectRatio="none"
        fill="currentColor"
        aria-hidden="true"
      >
        <path d="M0 64C240 120 480 8 720 32s480 88 720 48v40H0Z" />
      </svg>
    </section>
  );
}
