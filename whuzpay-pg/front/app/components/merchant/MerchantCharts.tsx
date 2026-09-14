"use client";

import {
  Area,
  AreaChart,
  Bar,
  BarChart,
  CartesianGrid,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";
import type { MerchantCharts } from "@/lib/merchant-api";
import { Card, formatIDR } from "../admin/ui";

const SERIES = {
  paid: "#22c55e",
  amount: "#ff5e16",
  pending: "#f59e0b",
};

function Tip({
  active,
  payload,
  label,
  money,
}: {
  active?: boolean;
  payload?: Array<{ name?: string; value?: number; color?: string }>;
  label?: string;
  money?: boolean;
}) {
  if (!active || !payload?.length) return null;
  return (
    <div className="rounded-lg border border-[#e8eef4] bg-white px-3 py-2 text-[12px] shadow-lg">
      {label && <p className="mb-1 font-semibold text-[#1f2a37]">{label}</p>}
      {payload.map((item) => (
        <p key={String(item.name)} className="text-[#6b7c93]">
          {item.name}:{" "}
          <span className="font-medium text-[#1f2a37]">
            {money
              ? formatIDR(Number(item.value ?? 0))
              : Number(item.value ?? 0).toLocaleString("id-ID")}
          </span>
        </p>
      ))}
    </div>
  );
}

export default function MerchantChartsPanel({
  data,
  days,
  onDaysChange,
}: {
  data: MerchantCharts;
  days: number;
  onDaysChange: (d: number) => void;
}) {
  const daily = data.daily ?? [];
  const hasData = daily.some((d) => d.total > 0 || d.paid_amount > 0);

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <p className="text-[12.5px] text-[#8a97a8]">Chart range</p>
        <div className="inline-flex rounded-full border border-[#e8eef4] bg-white p-1">
          {[7, 14, 30].map((d) => (
            <button
              key={d}
              type="button"
              onClick={() => onDaysChange(d)}
              className={`rounded-full px-3 py-1.5 text-[12px] font-semibold transition ${
                days === d
                  ? "bg-[#06163a] text-white"
                  : "text-[#6b7c93] hover:bg-[#f4f7fb]"
              }`}
            >
              {d}d
            </button>
          ))}
        </div>
      </div>

      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        <Card className="p-5">
          <h2 className="text-[15px] font-semibold text-[#1f2a37]">
            Paid amount
          </h2>
          <p className="mt-0.5 text-[12px] text-[#8a97a8]">
            Last {data.days} days
          </p>
          <div className="mt-3 h-[220px] w-full">
            {!hasData ? (
              <div className="flex h-full items-center justify-center text-[13px] text-[#8a97a8]">
                No chart data yet.
              </div>
            ) : (
              <ResponsiveContainer width="100%" height="100%">
                <AreaChart data={daily}>
                  <defs>
                    <linearGradient id="mPaidFill" x1="0" y1="0" x2="0" y2="1">
                      <stop offset="0%" stopColor={SERIES.amount} stopOpacity={0.35} />
                      <stop offset="100%" stopColor={SERIES.amount} stopOpacity={0.02} />
                    </linearGradient>
                  </defs>
                  <CartesianGrid strokeDasharray="3 3" stroke="#eef2f6" vertical={false} />
                  <XAxis dataKey="label" tick={{ fill: "#8a97a8", fontSize: 11 }} axisLine={false} tickLine={false} />
                  <YAxis
                    tick={{ fill: "#8a97a8", fontSize: 11 }}
                    axisLine={false}
                    tickLine={false}
                    width={48}
                    tickFormatter={(v) =>
                      v >= 1_000_000
                        ? `${(v / 1_000_000).toFixed(1)}jt`
                        : v >= 1000
                          ? `${Math.round(v / 1000)}rb`
                          : String(v)
                    }
                  />
                  <Tooltip content={<Tip money />} />
                  <Area
                    type="monotone"
                    dataKey="paid_amount"
                    name="Paid amount"
                    stroke={SERIES.amount}
                    strokeWidth={2}
                    fill="url(#mPaidFill)"
                    dot={false}
                  />
                </AreaChart>
              </ResponsiveContainer>
            )}
          </div>
        </Card>

        <Card className="p-5">
          <h2 className="text-[15px] font-semibold text-[#1f2a37]">
            Daily volume
          </h2>
          <p className="mt-0.5 text-[12px] text-[#8a97a8]">
            Paid vs pending counts
          </p>
          <div className="mt-3 h-[220px] w-full">
            {!hasData ? (
              <div className="flex h-full items-center justify-center text-[13px] text-[#8a97a8]">
                No chart data yet.
              </div>
            ) : (
              <ResponsiveContainer width="100%" height="100%">
                <BarChart data={daily}>
                  <CartesianGrid strokeDasharray="3 3" stroke="#eef2f6" vertical={false} />
                  <XAxis dataKey="label" tick={{ fill: "#8a97a8", fontSize: 11 }} axisLine={false} tickLine={false} />
                  <YAxis allowDecimals={false} tick={{ fill: "#8a97a8", fontSize: 11 }} axisLine={false} tickLine={false} width={28} />
                  <Tooltip content={<Tip />} />
                  <Bar dataKey="paid" name="Paid" fill={SERIES.paid} radius={[3, 3, 0, 0]} />
                  <Bar dataKey="pending" name="Pending" fill={SERIES.pending} radius={[3, 3, 0, 0]} />
                </BarChart>
              </ResponsiveContainer>
            )}
          </div>
        </Card>
      </div>
    </div>
  );
}
