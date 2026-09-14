"use client";

import { useState } from "react";

const periods = ["DAILY", "WEEKLY", "MONTHLY", "YEARLY"] as const;

/** Approximate multi-series area chart matching the reference dashboard */
const bluePath =
  "M0,95 C40,90 55,70 80,72 C110,75 120,45 150,50 C180,55 195,30 230,35 C265,40 280,55 310,48 C340,40 360,20 400,28 C430,34 450,55 480,42 C500,34 520,50 540,55";
const orangePath =
  "M0,110 C35,105 60,85 90,88 C120,92 140,70 170,74 C200,78 220,58 255,62 C285,66 300,80 335,72 C365,64 385,45 420,52 C450,58 470,70 500,60 C520,54 535,65 540,70";
const areaBlue =
  "M0,95 C40,90 55,70 80,72 C110,75 120,45 150,50 C180,55 195,30 230,35 C265,40 280,55 310,48 C340,40 360,20 400,28 C430,34 450,55 480,42 C500,34 520,50 540,55 L540,140 L0,140 Z";
const areaOrange =
  "M0,110 C35,105 60,85 90,88 C120,92 140,70 170,74 C200,78 220,58 255,62 C285,66 300,80 335,72 C365,64 385,45 420,52 C450,58 470,70 500,60 C520,54 535,65 540,70 L540,140 L0,140 Z";

export default function SalesChart() {
  const [period, setPeriod] = useState<(typeof periods)[number]>("DAILY");

  return (
    <div className="flex h-full flex-col rounded-xl bg-white p-5 shadow-[0_1px_3px_rgba(16,38,73,0.06)]">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="flex items-center gap-5 text-[11px] font-semibold tracking-wide text-[#9aa8b8]">
          {periods.map((p) => (
            <button
              key={p}
              type="button"
              onClick={() => setPeriod(p)}
              className={`relative pb-1 transition ${
                period === p ? "text-[#ff5e16]" : "hover:text-[#6b7c93]"
              }`}
            >
              {p}
              {period === p && (
                <span className="absolute inset-x-0 -bottom-0.5 h-[2px] rounded bg-[#ff5e16]" />
              )}
            </button>
          ))}
        </div>

        <div className="flex items-center gap-4 text-[11px] text-[#6b7c93]">
          <span className="inline-flex items-center gap-1.5">
            <span className="h-2 w-2 rounded-full bg-[#3b9eff]" />
            Online
          </span>
          <span className="inline-flex items-center gap-1.5">
            <span className="h-2 w-2 rounded-full bg-[#f0a04b]" />
            Store
          </span>
        </div>
      </div>

      <div className="mt-4 flex min-h-[180px] flex-1 gap-2">
        <div className="flex w-6 shrink-0 flex-col justify-between py-0.5 text-right text-[10px] leading-none text-[#b0bbc8]">
          {[35, 30, 25, 20, 15, 10, 5].map((n) => (
            <span key={n}>{n}</span>
          ))}
        </div>

        <div className="relative min-w-0 flex-1">
          <svg
            viewBox="0 0 560 150"
            className="h-full w-full"
            preserveAspectRatio="none"
            aria-hidden
          >
            {[20, 45, 70, 95, 120].map((y) => (
              <line
                key={y}
                x1="0"
                y1={y}
                x2="560"
                y2={y}
                stroke="#eef2f6"
                strokeWidth="1"
              />
            ))}

            <path d={areaBlue} fill="url(#blueFill)" opacity="0.35" />
            <path d={areaOrange} fill="url(#orangeFill)" opacity="0.28" />
            <path d={bluePath} fill="none" stroke="#3b9eff" strokeWidth="2.2" />
            <path d={orangePath} fill="none" stroke="#f0a04b" strokeWidth="2.2" />

            <defs>
              <linearGradient id="blueFill" x1="0" y1="0" x2="0" y2="1">
                <stop offset="0%" stopColor="#3b9eff" stopOpacity="0.45" />
                <stop offset="100%" stopColor="#3b9eff" stopOpacity="0" />
              </linearGradient>
              <linearGradient id="orangeFill" x1="0" y1="0" x2="0" y2="1">
                <stop offset="0%" stopColor="#f0a04b" stopOpacity="0.4" />
                <stop offset="100%" stopColor="#f0a04b" stopOpacity="0" />
              </linearGradient>
            </defs>
          </svg>

          <div className="mt-1 flex justify-between text-[11px] text-[#b0bbc8]">
            {[1, 2, 3, 4, 5, 6].map((n) => (
              <span key={n}>{n}</span>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}
