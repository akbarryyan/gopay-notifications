import Container from "./Container";
import { btnPrimary, btnOnNavy, focusRingDark } from "./styles";

export default function CtaRow() {
  return (
    <section id="hubungi-sales" className="scroll-mt-24 bg-brand-navy py-20 sm:py-24">
      <Container className="flex flex-col items-start justify-between gap-8 lg:flex-row lg:items-center">
        <div>
          <h2 className="max-w-2xl text-3xl font-extrabold leading-tight tracking-tight text-white sm:text-4xl">
            Siap menata alur pembayaran bisnis Anda?
          </h2>
          <p className="mt-4 max-w-xl text-base leading-relaxed text-white/70 sm:text-lg">
            Ceritakan kebutuhan bisnis Anda, lalu kita susun alur integrasi,
            checkout, dan routing pembayaran yang paling relevan.
          </p>
        </div>

        <div className="flex shrink-0 flex-wrap gap-3">
          <a href="#" className={`${btnPrimary} ${focusRingDark}`}>
            Hubungi Sales
          </a>
          <a href="#" className={`${btnOnNavy} ${focusRingDark}`}>
            Konsultasi Sekarang
          </a>
        </div>
      </Container>
    </section>
  );
}
