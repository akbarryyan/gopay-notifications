"use client";

import { useId, useLayoutEffect, useMemo, useRef, useState } from "react";
import { formatShortDate } from "@/lib/format";

interface Point {
  date: string;
  count: number;
}

const WIDTH = 600;
const HEIGHT = 220;
const PAD_TOP = 16;
const PAD_BOTTOM = 28;
const PAD_X = 4;

function catmullRomPath(pts: { x: number; y: number }[]): string {
  if (pts.length === 0) return "";
  if (pts.length === 1) return `M ${pts[0].x},${pts[0].y}`;

  let d = `M ${pts[0].x},${pts[0].y}`;
  for (let i = 0; i < pts.length - 1; i++) {
    const p0 = pts[i === 0 ? i : i - 1];
    const p1 = pts[i];
    const p2 = pts[i + 1];
    const p3 = pts[i + 2 < pts.length ? i + 2 : i + 1];

    const c1x = p1.x + (p2.x - p0.x) / 6;
    const c1y = p1.y + (p2.y - p0.y) / 6;
    const c2x = p2.x - (p3.x - p1.x) / 6;
    const c2y = p2.y - (p3.y - p1.y) / 6;

    d += ` C ${c1x},${c1y} ${c2x},${c2y} ${p2.x},${p2.y}`;
  }
  return d;
}

/**
 * Grafik tren event 14 hari, satu series — jadi tanpa kotak legenda (judul
 * kartu sudah menyebut apa yang ditampilkan). Kurva dihaluskan lewat
 * Catmull-Rom (bukan garis lurus antar titik), fill area ~diberi gradasi
 * tipis dari hue --primary, dan digambar dengan animasi "menggambar" satu
 * kali saat dimuat — dihormati prefers-reduced-motion.
 */
export function EventsTrendChart({ data }: { data: Point[] }) {
  const gradientId = useId();
  const [showTable, setShowTable] = useState(false);
  const [hoverIndex, setHoverIndex] = useState<number | null>(null);
  const wrapRef = useRef<HTMLDivElement>(null);
  const pathRef = useRef<SVGPathElement>(null);
  const [drawn, setDrawn] = useState(false);

  const plotWidth = WIDTH - PAD_X * 2;
  const maxCount = Math.max(1, ...data.map((d) => d.count));

  const points = useMemo(
    () =>
      data.map((d, i) => ({
        x: PAD_X + (data.length === 1 ? plotWidth / 2 : (i / (data.length - 1)) * plotWidth),
        y: PAD_TOP + (1 - d.count / maxCount) * (HEIGHT - PAD_TOP - PAD_BOTTOM),
      })),
    [data, maxCount, plotWidth],
  );

  const linePath = useMemo(() => catmullRomPath(points), [points]);
  const areaPath = useMemo(() => {
    if (points.length === 0) return "";
    const baseline = HEIGHT - PAD_BOTTOM;
    return `${linePath} L ${points[points.length - 1].x},${baseline} L ${points[0].x},${baseline} Z`;
  }, [points, linePath]);

  const [pathLength, setPathLength] = useState<number | null>(null);

  // Diukur di layout effect (sebelum browser sempat mengecat frame), bukan
  // dibaca langsung saat render — ref belum terpasang pada render pertama,
  // dan mengukur setelah paint akan sempat mengedipkan garis penuh sesaat.
  useLayoutEffect(() => {
    setPathLength(pathRef.current?.getTotalLength() ?? null);
  }, [linePath]);

  // Animasi "menggambar" garis satu kali setelah panjangnya diketahui:
  // dashoffset dari panjang path penuh ke 0. Dilewati (langsung tampil
  // penuh) bila pengguna meminta lebih sedikit gerakan.
  useLayoutEffect(() => {
    if (pathLength === null) return;
    const reduceMotion = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
    if (reduceMotion) {
      // Menyinkronkan ke preferensi sistem eksternal (matchMedia) — bukan
      // cascading render biasa, dan tidak ada pengganti yang lebih sederhana
      // untuk melewati animasi saat pengguna meminta lebih sedikit gerakan.
      // eslint-disable-next-line react-hooks/set-state-in-effect
      setDrawn(true);
      return;
    }
    const raf = requestAnimationFrame(() => setDrawn(true));
    return () => cancelAnimationFrame(raf);
  }, [pathLength]);

  function onPointerMove(e: React.PointerEvent<HTMLDivElement>) {
    const rect = wrapRef.current?.getBoundingClientRect();
    if (!rect || data.length === 0) return;
    const fraction = Math.min(1, Math.max(0, (e.clientX - rect.left) / rect.width));
    const index = Math.round(fraction * (data.length - 1));
    setHoverIndex(index);
  }

  const hovered = hoverIndex !== null ? data[hoverIndex] : null;
  const hoveredPoint = hoverIndex !== null ? points[hoverIndex] : null;
  const hoveredLeftPct = hoveredPoint ? (hoveredPoint.x / WIDTH) * 100 : null;

  if (showTable) {
    return (
      <TableView data={data} onBack={() => setShowTable(false)} />
    );
  }

  return (
    <div className="flex flex-col gap-2">
      <div
        ref={wrapRef}
        className="relative"
        onPointerMove={onPointerMove}
        onPointerLeave={() => setHoverIndex(null)}
      >
        <svg
          viewBox={`0 0 ${WIDTH} ${HEIGHT}`}
          preserveAspectRatio="none"
          className="h-48 w-full"
          role="img"
          aria-label="Grafik jumlah event per hari, 14 hari terakhir"
        >
          <defs>
            <linearGradient id={gradientId} x1="0" y1="0" x2="0" y2="1">
              <stop offset="0%" style={{ stopColor: "var(--primary)", stopOpacity: 0.18 }} />
              <stop offset="100%" style={{ stopColor: "var(--primary)", stopOpacity: 0 }} />
            </linearGradient>
          </defs>

          {/* Gridlines horizontal: garis rambut, resesif — jumlahnya sedikit
              dan tanpa label per titik (label diselipkan hanya di sumbu Y
              lewat teks terpisah di bawah, bukan di sini). */}
          {[0.25, 0.5, 0.75].map((f) => {
            const y = PAD_TOP + f * (HEIGHT - PAD_TOP - PAD_BOTTOM);
            return (
              <line
                key={f}
                x1={PAD_X}
                y1={y}
                x2={WIDTH - PAD_X}
                y2={y}
                stroke="var(--border)"
                strokeWidth={1}
                vectorEffect="non-scaling-stroke"
              />
            );
          })}

          <path d={areaPath} fill={`url(#${gradientId})`} className="transition-opacity duration-700" style={{ opacity: drawn ? 1 : 0 }} />

          <path
            ref={pathRef}
            d={linePath}
            fill="none"
            stroke="var(--primary)"
            strokeWidth={2}
            strokeLinecap="round"
            strokeLinejoin="round"
            vectorEffect="non-scaling-stroke"
            style={
              pathLength === null
                ? { visibility: "hidden" }
                : {
                    strokeDasharray: pathLength,
                    strokeDashoffset: drawn ? 0 : pathLength,
                    transition: "stroke-dashoffset 900ms ease-out",
                  }
            }
          />

          {/* Titik akhir: penanda >=8px dengan cincin warna surface, supaya
              tetap terbaca walau menumpuk di ujung garis. */}
          {points.length > 0 && (
            <g
              className="transition-opacity duration-500"
              style={{ opacity: drawn ? 1 : 0, transitionDelay: "700ms" }}
            >
              <circle
                cx={points[points.length - 1].x}
                cy={points[points.length - 1].y}
                r={5}
                fill="var(--card)"
                vectorEffect="non-scaling-stroke"
              />
              <circle
                cx={points[points.length - 1].x}
                cy={points[points.length - 1].y}
                r={4}
                fill="var(--primary)"
                vectorEffect="non-scaling-stroke"
              />
            </g>
          )}

          {/* Crosshair: hairline vertikal yang mengikuti pointer, snap ke
              titik data terdekat. */}
          {hoveredPoint && (
            <g>
              <line
                x1={hoveredPoint.x}
                y1={PAD_TOP}
                x2={hoveredPoint.x}
                y2={HEIGHT - PAD_BOTTOM}
                stroke="var(--foreground)"
                strokeOpacity={0.15}
                strokeWidth={1}
                vectorEffect="non-scaling-stroke"
              />
              <circle
                cx={hoveredPoint.x}
                cy={hoveredPoint.y}
                r={4}
                fill="var(--card)"
                stroke="var(--primary)"
                strokeWidth={2}
                vectorEffect="non-scaling-stroke"
              />
            </g>
          )}
        </svg>

        {/* Label sumbu-X selektif: hanya titik pertama dan terakhir, bukan
            satu label per hari (14 label akan berdesakan dan tak terbaca). */}
        {data.length > 0 && (
          <div className="flex justify-between px-1 text-xs text-muted-foreground">
            <span>{formatShortDate(data[0].date)}</span>
            <span>{formatShortDate(data[data.length - 1].date)}</span>
          </div>
        )}

        {/* Tooltip: nilai lebih menonjol daripada tanggal, mengikuti posisi
            horizontal titik yang di-hover. */}
        {hovered && hoveredLeftPct !== null && (
          <div
            className="pointer-events-none absolute top-2 z-10 -translate-x-1/2 rounded-lg border bg-popover px-2.5 py-1.5 text-xs whitespace-nowrap text-popover-foreground shadow-md ring-1 ring-foreground/10"
            style={{ left: `${hoveredLeftPct}%` }}
          >
            <div className="font-semibold">{hovered.count} event</div>
            <div className="text-muted-foreground">{formatShortDate(hovered.date)}</div>
          </div>
        )}
      </div>

      <button
        type="button"
        onClick={() => setShowTable(true)}
        className="self-start text-xs text-muted-foreground underline-offset-2 hover:text-foreground hover:underline"
      >
        Lihat sebagai tabel
      </button>
    </div>
  );
}

function TableView({ data, onBack }: { data: Point[]; onBack: () => void }) {
  return (
    <div className="flex flex-col gap-2">
      <div className="max-h-56 overflow-y-auto rounded-lg border">
        <table className="w-full text-sm">
          <thead className="sticky top-0 bg-muted/60">
            <tr>
              <th className="px-3 py-1.5 text-left font-medium text-muted-foreground">Tanggal</th>
              <th className="px-3 py-1.5 text-right font-medium text-muted-foreground">Jumlah event</th>
            </tr>
          </thead>
          <tbody>
            {data.map((d) => (
              <tr key={d.date} className="border-t">
                <td className="px-3 py-1.5">{formatShortDate(d.date)}</td>
                <td className="px-3 py-1.5 text-right font-medium">{d.count}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      <button
        type="button"
        onClick={onBack}
        className="self-start text-xs text-muted-foreground underline-offset-2 hover:text-foreground hover:underline"
      >
        Lihat sebagai grafik
      </button>
    </div>
  );
}
