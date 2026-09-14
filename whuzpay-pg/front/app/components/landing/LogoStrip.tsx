import Container from "./Container";

/**
 * Placeholder client logos.
 *
 * Every mark here is drawn locally and the names are invented — this strip
 * is a slot waiting for real customers, not a claim that these companies
 * are ones. It replaces a version that hot-linked real brands' logos from
 * a competitor's CDN.
 */
const marks = {
  circle: <circle cx="12" cy="12" r="8" />,
  square: <rect x="5" y="5" width="14" height="14" rx="3" />,
  diamond: <path d="M12 3 21 12 12 21 3 12Z" />,
  triangle: <path d="M12 4 21 20H3Z" />,
} as const;

const companies: { name: string; mark: keyof typeof marks }[] = [
  { name: "Nusantara Retail", mark: "circle" },
  { name: "Bahari Logistik", mark: "square" },
  { name: "Cakrawala Media", mark: "diamond" },
  { name: "Dwipa Store", mark: "triangle" },
  { name: "Ekana Digital", mark: "circle" },
  { name: "Fajar Grosir", mark: "square" },
  { name: "Gapura Tech", mark: "diamond" },
  { name: "Harmoni Group", mark: "triangle" },
];

// Duplicated so the marquee can loop seamlessly at -50%.
const track = [...companies, ...companies];

function LogoMark({ name, mark }: { name: string; mark: keyof typeof marks }) {
  return (
    <div className="flex shrink-0 items-center gap-2.5 text-slate-400 transition-colors hover:text-slate-600">
      <svg viewBox="0 0 24 24" fill="currentColor" className="h-6 w-6 opacity-70">
        {marks[mark]}
      </svg>
      <span className="whitespace-nowrap text-base font-bold tracking-tight">
        {name}
      </span>
    </div>
  );
}

export default function LogoStrip() {
  return (
    <section className="bg-white py-12" aria-labelledby="logo-strip-heading">
      <Container>
        <h2
          id="logo-strip-heading"
          className="text-center text-sm font-semibold uppercase tracking-wider text-slate-400"
        >
          Dipakai tim yang menangani transaksi setiap hari
        </h2>

        <div className="mt-8 overflow-hidden [mask-image:linear-gradient(to_right,transparent,black_8%,black_92%,transparent)]">
          <div className="flex w-max animate-[marquee_36s_linear_infinite] items-center gap-16 hover:[animation-play-state:paused] motion-reduce:animate-none">
            {track.map((company, i) => (
              <LogoMark key={`${company.name}-${i}`} {...company} />
            ))}
          </div>
        </div>
      </Container>
    </section>
  );
}
