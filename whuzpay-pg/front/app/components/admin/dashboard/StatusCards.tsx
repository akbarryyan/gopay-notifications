const cards = [
  {
    title: "Revenue Status",
    value: "$432",
    period: "Jan 01 - Jan 10",
    bg: "bg-[#def2fd]",
    accent: "text-[#1d8fd8]",
    chart: "bars-blue" as const,
  },
  {
    title: "Page View",
    value: "$432",
    period: null,
    bg: "bg-[#fef7db]",
    accent: "text-[#d4a017]",
    chart: "line-yellow" as const,
  },
  {
    title: "Bounce Rate",
    value: "$432",
    period: "Monthly",
    bg: "bg-[#fee8dd]",
    accent: "text-[#e85d3b]",
    chart: "line-orange" as const,
  },
  {
    title: "Revenue Status",
    value: "$432",
    period: "Jan 01 - Jan 10",
    bg: "bg-[#f3e8ff]",
    accent: "text-[#9b5de5]",
    chart: "bars-purple" as const,
  },
];

function MiniBars({ color }: { color: string }) {
  const heights = [10, 16, 12, 20, 14, 18, 11];
  return (
    <div className="flex h-10 items-end gap-[3px]">
      {heights.map((h, i) => (
        <span
          key={i}
          className="w-[5px] rounded-sm opacity-80"
          style={{ height: h, backgroundColor: color }}
        />
      ))}
    </div>
  );
}

function MiniLine({ color }: { color: string }) {
  return (
    <svg viewBox="0 0 80 32" className="h-10 w-20" aria-hidden>
      <path
        d="M2 22 C12 20 18 10 28 14 C38 18 42 8 52 10 C62 12 68 18 78 12"
        fill="none"
        stroke={color}
        strokeWidth="2"
        strokeLinecap="round"
      />
      <circle cx="52" cy="10" r="2.5" fill={color} />
    </svg>
  );
}

export default function StatusCards() {
  return (
    <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
      {cards.map((card, idx) => (
        <div
          key={`${card.title}-${idx}`}
          className={`flex items-center justify-between rounded-xl px-5 py-4 ${card.bg}`}
        >
          <div className="min-w-0">
            <p className="text-[12px] font-medium text-[#5a6a7e]">{card.title}</p>
            <div className="mt-2 flex items-center gap-1">
              <span className={`text-[13px] font-semibold ${card.accent}`}>↑</span>
              <span className={`text-[22px] font-bold leading-none ${card.accent}`}>
                {card.value}
              </span>
            </div>
            {card.period && (
              <p className="mt-1.5 text-[11px] text-[#8a97a8]">
                {card.chart === "line-orange" ? (
                  <span className="inline-flex items-center gap-1 rounded bg-white/60 px-1.5 py-0.5">
                    {card.period}
                    <span className="text-[9px]">▾</span>
                  </span>
                ) : (
                  card.period
                )}
              </p>
            )}
          </div>

          <div className="shrink-0 opacity-90">
            {card.chart === "bars-blue" && <MiniBars color="#3b9eff" />}
            {card.chart === "bars-purple" && <MiniBars color="#9b5de5" />}
            {card.chart === "line-yellow" && <MiniLine color="#d4a017" />}
            {card.chart === "line-orange" && <MiniLine color="#e85d3b" />}
          </div>
        </div>
      ))}
    </div>
  );
}
