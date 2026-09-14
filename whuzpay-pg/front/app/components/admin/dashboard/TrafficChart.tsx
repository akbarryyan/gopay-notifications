const segments = [
  { label: "Facebook", value: 34, color: "#3b9eff" },
  { label: "Youtube", value: 55, color: "#ff5e16" },
  { label: "Direct Search", value: 11, color: "#f0c14b" },
];

/** Donut chart via conic-gradient: Facebook 34% blue, Youtube 55% orange, Direct 11% yellow */
export default function TrafficChart() {
  return (
    <div className="flex h-full flex-col rounded-xl bg-white p-5 shadow-[0_1px_3px_rgba(16,38,73,0.06)]">
      <h3 className="text-[15px] font-semibold text-[#1f2a37]">Traffic</h3>

      <div className="flex flex-1 flex-col items-center justify-center py-3">
        <div
          className="relative h-[140px] w-[140px] rounded-full"
          style={{
            background:
              "conic-gradient(#3b9eff 0% 34%, #f0c14b 34% 45%, #ff5e16 45% 100%)",
          }}
          aria-hidden
        >
          <div className="absolute inset-[28px] rounded-full bg-white" />
        </div>
      </div>

      <div className="mt-1 grid grid-cols-3 gap-2 text-center">
        {segments.map((s) => (
          <div key={s.label}>
            <p className="text-[18px] font-bold leading-none text-[#1f2a37]">
              {s.value}%
            </p>
            <p className="mt-1.5 flex items-center justify-center gap-1 text-[11px] text-[#8a97a8]">
              <span
                className="inline-block h-1.5 w-1.5 rounded-full"
                style={{ backgroundColor: s.color }}
              />
              {s.label}
            </p>
          </div>
        ))}
      </div>
    </div>
  );
}
