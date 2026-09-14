import Container from "./Container";

type Background = "white" | "slate" | "sky" | "navy" | "navy-dark";

const backgroundClass: Record<Background, string> = {
  white: "bg-white",
  slate: "bg-slate-50",
  sky: "bg-sky-50",
  navy: "bg-brand-navy",
  "navy-dark": "bg-brand-navy-dark",
};

const isDark = (background: Background) =>
  background === "navy" || background === "navy-dark";

type Props = {
  id?: string;
  eyebrow?: string;
  title?: string;
  subtitle?: string;
  background?: Background;
  align?: "left" | "center";
  /** Extra classes on the <section>, e.g. a tighter vertical rhythm. */
  className?: string;
  /** Extra classes on the inner Container. */
  containerClassName?: string;
  children?: React.ReactNode;
};

/**
 * One vertical band of the landing page: background, vertical rhythm, and
 * the optional eyebrow/title/subtitle header, all on a single scale.
 *
 * Sections needing a bespoke layout still render their own children — they
 * just inherit the spacing and heading treatment instead of inventing one.
 */
export default function Section({
  id,
  eyebrow,
  title,
  subtitle,
  background = "white",
  align = "center",
  className = "",
  containerClassName = "",
  children,
}: Props) {
  const dark = isDark(background);
  const hasHeader = Boolean(eyebrow || title || subtitle);

  return (
    <section
      id={id}
      className={`${backgroundClass[background]} scroll-mt-24 py-20 sm:py-28 ${className}`}
    >
      <Container className={containerClassName}>
        {hasHeader && (
          <div
            className={
              align === "center" ? "mx-auto max-w-3xl text-center" : "max-w-3xl"
            }
          >
            {eyebrow && (
              <p
                className={`text-sm font-semibold uppercase tracking-wider ${
                  dark ? "text-brand-yellow" : "text-brand-navy-light"
                }`}
              >
                {eyebrow}
              </p>
            )}
            {title && (
              <h2
                className={`text-3xl font-extrabold leading-tight tracking-tight sm:text-4xl ${
                  eyebrow ? "mt-3" : ""
                } ${dark ? "text-white" : "text-brand-navy"}`}
              >
                {title}
              </h2>
            )}
            {subtitle && (
              <p
                className={`mt-4 text-base leading-relaxed sm:text-lg ${
                  dark ? "text-white/70" : "text-slate-600"
                }`}
              >
                {subtitle}
              </p>
            )}
          </div>
        )}

        {children && <div className={hasHeader ? "mt-14" : ""}>{children}</div>}
      </Container>
    </section>
  );
}
