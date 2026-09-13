"use client";

import { useEffect, useRef, useState, type ReactNode } from "react";

/**
 * Fade + slide-up begitu section masuk viewport, sekali saja. Memakai
 * utilitas `tw-animate-css` yang sudah jadi dependency proyek ini (dipakai
 * juga oleh dropdown/dialog di components/ui) -- bukan library animasi baru.
 * Menghormati prefers-reduced-motion: kalau aktif, section langsung tampil
 * tanpa animasi sama sekali.
 */
export function Reveal({
  children,
  className,
  delay,
}: {
  children: ReactNode;
  className?: string;
  delay?: 75 | 100 | 150 | 200 | 300;
}) {
  const ref = useRef<HTMLDivElement>(null);
  // Dibaca sekali lewat lazy initializer (bukan di-set di dalam effect) supaya
  // tidak melanggar react-hooks/set-state-in-effect.
  const [reducedMotion] = useState(
    () =>
      typeof window !== "undefined" &&
      window.matchMedia("(prefers-reduced-motion: reduce)").matches,
  );
  const [visible, setVisible] = useState(reducedMotion);

  useEffect(() => {
    if (reducedMotion) return;
    const node = ref.current;
    if (!node) return;

    const observer = new IntersectionObserver(
      ([entry]) => {
        if (entry.isIntersecting) {
          setVisible(true);
          observer.disconnect();
        }
      },
      { threshold: 0.15, rootMargin: "0px 0px -10% 0px" },
    );
    observer.observe(node);
    return () => observer.disconnect();
  }, [reducedMotion]);

  const delayClass = delay ? `delay-${delay}` : "";

  return (
    <div
      ref={ref}
      className={[
        visible
          ? `animate-in fade-in-0 slide-in-from-bottom-6 duration-700 ease-out fill-mode-both ${delayClass}`
          : "opacity-0",
        className ?? "",
      ].join(" ")}
    >
      {children}
    </div>
  );
}
