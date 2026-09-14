"use client";

import {
  Area,
  AreaChart,
  Bar,
  BarChart,
  CartesianGrid,
  Cell,
  Legend,
  Pie,
  PieChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";
import type { DashboardCharts as DashboardChartsData } from "@/lib/admin-api";
import { Card, formatIDR } from "./ui";

const STATUS_COLORS: Record<string, string> = {
  paid: "#22c55e",
  pending: "#f59e0b",
  failed: "#ef4444",
  expired: "#94a3b8",
  cancelled: "#8b5cf6",
};

const SERIES = {
  paid: "#22c55e",
  pending: "#f59e0b",
  failed: "#ef4444",
  expired: "#94a3b8",
  total: "#3b9eff",
  amount: "#ff5e16",
};

function ChartTooltipBox({
  active,
  payload,
  label,
  valueFormatter,
}: {
  active?: boolean;
  payload?: Array<{ name?: string; value?: number; color?: string }>;
  label?: string;
  valueFormatter?: (value: number) => string;
}) {
  if (!active || !payload?.length) return null;
  return (
    <div className="rounded-lg border border-[#e8eef4] bg-white px-3 py-2 text-[12px] shadow-lg">
      {label && <p className="mb-1 font-semibold text-[#1f2a37]">{label}</p>}
      <div className="space-y-0.5">
        {payload.map((item) => (
          <p key={String(item.name)} className="text-[#6b7c93]">
            <span
              className="mr-1.5 inline-block size-2 rounded-full"
              style={{ backgroundColor: item.color }}
            />
            {item.name}:{" "}
            <span className="font-medium text-[#1f2a37]">
              {valueFormatter
                ? valueFormatter(Number(item.value ?? 0))
                : Number(item.value ?? 0).toLocaleString("id-ID")}
            </span>
          </p>
        ))}
      </div>
    </div>
  );
}

export default function DashboardCharts({ data }: { data: DashboardChartsData }) {
  const daily = data.daily ?? [];
  const statusData = (data.status_breakdown ?? [])
    .filter((s) => s.count > 0)
    .map((s) => ({
      name: s.status.charAt(0).toUpperCase() + s.status.slice(1),
      value: s.count,
      status: s.status,
    }));

  const hasDaily = daily.some((d) => d.total > 0 || d.paid_amount > 0);
  const hasStatus = statusData.length > 0;

  return (
    <div className="grid grid-cols-1 gap-4 xl:grid-cols-3">
      {/* Paid amount area chart */}
      <Card className="p-5 xl:col-span-2">
        <div className="mb-4">
          <h2 className="text-[15px] font-semibold text-[#1f2a37]">
            Paid amount trend
          </h2>
          <p className="mt-0.5 text-[12px] text-[#8a97a8]">
            Last {data.days} days · successful payments volume
          </p>
        </div>
        <div className="h-[280px] w-full">
          {!hasDaily ? (
            <div className="flex h-full items-center justify-center text-[13px] text-[#8a97a8]">
              No chart data yet.
            </div>
          ) : (
            <ResponsiveContainer width="100%" height="100%">
              <AreaChart data={daily} margin={{ top: 8, right: 8, left: 0, bottom: 0 }}>
                <defs>
                  <linearGradient id="paidAmountFill" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="0%" stopColor={SERIES.amount} stopOpacity={0.35} />
                    <stop offset="100%" stopColor={SERIES.amount} stopOpacity={0.02} />
                  </linearGradient>
                </defs>
                <CartesianGrid strokeDasharray="3 3" stroke="#eef2f6" vertical={false} />
                <XAxis
                  dataKey="label"
                  tick={{ fill: "#8a97a8", fontSize: 11 }}
                  axisLine={false}
                  tickLine={false}
                  minTickGap={16}
                />
                <YAxis
                  tick={{ fill: "#8a97a8", fontSize: 11 }}
                  axisLine={false}
                  tickLine={false}
                  width={56}
                  tickFormatter={(v) =>
                    v >= 1_000_000
                      ? `${(v / 1_000_000).toFixed(1)}jt`
                      : v >= 1000
                        ? `${Math.round(v / 1000)}rb`
                        : String(v)
                  }
                />
                <Tooltip
                  content={
                    <ChartTooltipBox valueFormatter={(v) => formatIDR(v)} />
                  }
                />
                <Area
                  type="monotone"
                  dataKey="paid_amount"
                  name="Paid amount"
                  stroke={SERIES.amount}
                  strokeWidth={2.2}
                  fill="url(#paidAmountFill)"
                  dot={false}
                  activeDot={{ r: 4 }}
                />
              </AreaChart>
            </ResponsiveContainer>
          )}
        </div>
      </Card>

      {/* Status pie */}
      <Card className="p-5">
        <div className="mb-4">
          <h2 className="text-[15px] font-semibold text-[#1f2a37]">
            Status breakdown
          </h2>
          <p className="mt-0.5 text-[12px] text-[#8a97a8]">
            Share of payments by status
          </p>
        </div>
        <div className="h-[280px] w-full">
          {!hasStatus ? (
            <div className="flex h-full items-center justify-center text-[13px] text-[#8a97a8]">
              No status data yet.
            </div>
          ) : (
            <ResponsiveContainer width="100%" height="100%">
              <PieChart>
                <Pie
                  data={statusData}
                  dataKey="value"
                  nameKey="name"
                  innerRadius={58}
                  outerRadius={88}
                  paddingAngle={3}
                  stroke="#fff"
                  strokeWidth={2}
                >
                  {statusData.map((entry) => (
                    <Cell
                      key={entry.status}
                      fill={STATUS_COLORS[entry.status] ?? "#94a3b8"}
                    />
                  ))}
                </Pie>
                <Tooltip content={<ChartTooltipBox />} />
                <Legend
                  verticalAlign="bottom"
                  height={36}
                  iconType="circle"
                  wrapperStyle={{ fontSize: 12, color: "#6b7c93" }}
                />
              </PieChart>
            </ResponsiveContainer>
          )}
        </div>
      </Card>

      {/* Daily volume bar chart */}
      <Card className="p-5 xl:col-span-3">
        <div className="mb-4">
          <h2 className="text-[15px] font-semibold text-[#1f2a37]">
            Daily payment volume
          </h2>
          <p className="mt-0.5 text-[12px] text-[#8a97a8]">
            Count of payments by day and status (last {data.days} days)
          </p>
        </div>
        <div className="h-[300px] w-full">
          {!hasDaily ? (
            <div className="flex h-full items-center justify-center text-[13px] text-[#8a97a8]">
              No chart data yet.
            </div>
          ) : (
            <ResponsiveContainer width="100%" height="100%">
              <BarChart data={daily} margin={{ top: 8, right: 8, left: 0, bottom: 0 }}>
                <CartesianGrid strokeDasharray="3 3" stroke="#eef2f6" vertical={false} />
                <XAxis
                  dataKey="label"
                  tick={{ fill: "#8a97a8", fontSize: 11 }}
                  axisLine={false}
                  tickLine={false}
                  minTickGap={12}
                />
                <YAxis
                  allowDecimals={false}
                  tick={{ fill: "#8a97a8", fontSize: 11 }}
                  axisLine={false}
                  tickLine={false}
                  width={36}
                />
                <Tooltip content={<ChartTooltipBox />} />
                <Legend
                  iconType="circle"
                  wrapperStyle={{ fontSize: 12, color: "#6b7c93" }}
                />
                <Bar dataKey="paid" name="Paid" stackId="a" fill={SERIES.paid} radius={[0, 0, 0, 0]} />
                <Bar dataKey="pending" name="Pending" stackId="a" fill={SERIES.pending} />
                <Bar dataKey="failed" name="Failed" stackId="a" fill={SERIES.failed} />
                <Bar
                  dataKey="expired"
                  name="Expired"
                  stackId="a"
                  fill={SERIES.expired}
                  radius={[4, 4, 0, 0]}
                />
              </BarChart>
            </ResponsiveContainer>
          )}
        </div>
      </Card>
    </div>
  );
}
