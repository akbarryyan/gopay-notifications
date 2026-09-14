import { IconBarChart, IconCrown, IconDollar, IconMedal } from "../icons";

const metrics = [
  {
    label: "Wallet Balance",
    value: "$4,567.53",
    icon: IconCrown,
    iconBg: "bg-[#ffe4e6]",
    iconColor: "text-[#f43f5e]",
  },
  {
    label: "Referral Earning",
    value: "$1689.53",
    icon: IconMedal,
    iconBg: "bg-[#ede9fe]",
    iconColor: "text-[#8b5cf6]",
  },
  {
    label: "Estimate Sales",
    value: "$2851.53",
    icon: IconDollar,
    iconBg: "bg-[#dcfce7]",
    iconColor: "text-[#22c55e]",
  },
  {
    label: "Earning",
    value: "$52,567.53",
    icon: IconBarChart,
    iconBg: "bg-[#fce7f3]",
    iconColor: "text-[#ec4899]",
  },
];

export default function MetricCards() {
  return (
    <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
      {metrics.map((m) => {
        const Icon = m.icon;
        return (
          <div
            key={m.label}
            className="flex items-center gap-4 rounded-xl bg-white px-5 py-4 shadow-[0_1px_3px_rgba(16,38,73,0.06)]"
          >
            <span
              className={`flex h-11 w-11 shrink-0 items-center justify-center rounded-full ${m.iconBg} ${m.iconColor}`}
            >
              <Icon className="h-5 w-5" />
            </span>
            <div className="min-w-0">
              <p className="truncate text-[12px] text-[#8a97a8]">{m.label}</p>
              <p className="mt-0.5 text-[18px] font-bold tracking-tight text-[#1f2a37]">
                {m.value}
              </p>
            </div>
          </div>
        );
      })}
    </div>
  );
}
