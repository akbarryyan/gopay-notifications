"use client";

import { useMemo, useState, useEffect } from "react";
import toast from "react-hot-toast";
import { DownloadIcon } from "lucide-react";
import {
  Area,
  Bar,
  BarChart,
  CartesianGrid,
  ComposedChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";
import {
  fetchMerchantDashboardCharts,
  type MerchantChartDaily,
  type MerchantCharts,
} from "@/lib/merchant-api";
import { useMerchantEnvironment } from "@/lib/use-merchant-environment";
import { Button } from "@/components/ui/button";
import {
  Card,
  EmptyState,
  formatIDR,
  LoadingBlock,
  PageHeader,
  TableShell,
} from "@/app/components/admin/ui";

// Status is a fixed state scale, not a generic categorical series — reuse
// the exact colors StatusBadge already uses everywhere else in the
// dashboard so a "paid" segment here means the same color as a "paid"
// badge on the Payments page.
const STATUS_COLORS: Record<string, string> = {
  paid: "#2f9e5a",
  pending: "#c27a00",
  failed: "#e85d3b",
  expired: "#6b7c93",
  cancelled: "#7c3aed",
};
const STATUS_ORDER = ["paid", "pending", "failed", "expired", "cancelled"];
const REVENUE_COLOR = "#ff5e16";

const RANGE_OPTIONS = [7, 30, 90];

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
  const rows = payload.filter((p) => Number(p.value ?? 0) > 0 || payload.length === 1);
  return (
    <div className="rounded-lg border border-[#e8eef4] bg-white px-3 py-2 text-[12px] shadow-lg">
      {label && <p className="mb-1 font-semibold text-[#1f2a37]">{label}</p>}
      {rows.map((item) => (
        <p key={String(item.name)} className="flex items-center gap-1.5 text-[#6b7c93]">
          <span
            className="inline-block h-[2px] w-3 shrink-0"
            style={{ backgroundColor: item.color }}
          />
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

function StatTile({ label, value, tone }: { label: string; value: string; tone?: string }) {
  return (
    <Card className="p-5">
      <p className="text-[12px] text-[#8a97a8]">{label}</p>
      <p className={`mt-2 text-[22px] font-bold tracking-tight ${tone ?? "text-[#1f2a37]"}`}>
        {value}
      </p>
    </Card>
  );
}

/** Part-to-whole proportional bar for the whole period — see dataviz skill's
 * "part-to-whole → stacked bar" guidance; a single horizontal 100% stack
 * reads more precisely than a pie for 5 categories. */
function StatusBreakdownBar({ breakdown }: { breakdown: MerchantCharts["status_breakdown"] }) {
  const total = breakdown.reduce((sum, b) => sum + b.count, 0);
  if (total === 0) {
    return <EmptyState message="No transactions in this period." />;
  }
  const segments = STATUS_ORDER.map((status) => {
    const found = breakdown.find((b) => b.status === status);
    const count = found?.count ?? 0;
    return { status, count, pct: (count / total) * 100 };
  }).filter((s) => s.count > 0);

  return (
    <div>
      <div className="flex h-8 w-full overflow-hidden rounded-lg" role="img" aria-label="Transaction status breakdown">
        {segments.map((s, i) => (
          <div
            key={s.status}
            title={`${s.status}: ${s.count} (${s.pct.toFixed(1)}%)`}
            className="flex h-full items-center justify-center text-[11px] font-semibold text-white"
            style={{
              width: `${s.pct}%`,
              backgroundColor: STATUS_COLORS[s.status],
              marginLeft: i === 0 ? 0 : "2px",
            }}
          >
            {s.pct >= 8 ? `${s.pct.toFixed(0)}%` : ""}
          </div>
        ))}
      </div>
      <div className="mt-3 flex flex-wrap gap-x-4 gap-y-1.5">
        {segments.map((s) => (
          <div key={s.status} className="flex items-center gap-1.5 text-[12px] text-[#6b7c93]">
            <span
              className="inline-block size-2.5 rounded-full"
              style={{ backgroundColor: STATUS_COLORS[s.status] }}
            />
            <span className="capitalize">{s.status}</span>
            <span className="font-medium text-[#1f2a37]">{s.count}</span>
          </div>
        ))}
      </div>
    </div>
  );
}

function buildCsv(daily: MerchantChartDaily[], environment: string): string {
  const header = [
    "date",
    "total",
    "paid",
    "pending",
    "failed",
    "expired",
    "cancelled",
    "paid_amount",
  ];
  const rows = daily.map((d) =>
    [d.date, d.total, d.paid, d.pending, d.failed, d.expired, d.cancelled, d.paid_amount].join(","),
  );
  return [`# environment: ${environment}`, header.join(","), ...rows].join("\n");
}

function downloadCsv(content: string, filename: string) {
  const blob = new Blob([content], { type: "text/csv;charset=utf-8;" });
  const url = window.URL.createObjectURL(blob);
  const anchor = document.createElement("a");
  anchor.href = url;
  anchor.download = filename;
  document.body.appendChild(anchor);
  anchor.click();
  anchor.remove();
  window.URL.revokeObjectURL(url);
}

export default function MerchantReportsPage() {
  const { environment } = useMerchantEnvironment();
  const [days, setDays] = useState(30);
  const [charts, setCharts] = useState<MerchantCharts | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;
    async function load() {
      setLoading(true);
      try {
        const c = await fetchMerchantDashboardCharts(days, environment);
        if (!cancelled) setCharts(c);
      } catch (err) {
        if (!cancelled) {
          toast.error(
            err instanceof Error ? err.message : "Failed to load report.",
          );
        }
      } finally {
        if (!cancelled) setLoading(false);
      }
    }
    void load();
    return () => {
      cancelled = true;
    };
  }, [days, environment]);

  const daily = useMemo(() => charts?.daily ?? [], [charts]);

  const totals = useMemo(() => {
    const totalTransactions = daily.reduce((s, d) => s + d.total, 0);
    const totalPaid = daily.reduce((s, d) => s + d.paid, 0);
    const totalRevenue = daily.reduce((s, d) => s + d.paid_amount, 0);
    const successRate = totalTransactions > 0 ? (totalPaid / totalTransactions) * 100 : 0;
    const avgTransaction = totalPaid > 0 ? totalRevenue / totalPaid : 0;
    return { totalTransactions, totalPaid, totalRevenue, successRate, avgTransaction };
  }, [daily]);

  const hasData = daily.some((d) => d.total > 0);

  function handleExport() {
    if (!charts || daily.length === 0) {
      toast.error("Nothing to export yet.");
      return;
    }
    downloadCsv(
      buildCsv(daily, environment),
      `report-${environment}-${days}d-${new Date().toISOString().slice(0, 10)}.csv`,
    );
    toast.success("Report exported.");
  }

  return (
    <div className="w-full space-y-5">
      <PageHeader
        title="Reports"
        description={`Revenue and transaction trends for your ${environment} store.`}
        actions={
          <Button
            type="button"
            variant="outline"
            disabled={loading || !hasData}
            onClick={handleExport}
            className="h-9 gap-1.5 rounded-full border-[#e8eef4] bg-white px-4 text-[12.5px] font-semibold text-[#3d4b5c] shadow-none hover:bg-[#f4f7fb]"
          >
            <DownloadIcon className="size-4" />
            Export CSV
          </Button>
        }
      />

      <Card className="flex flex-wrap items-center justify-between gap-2 p-3">
        <p className="px-1.5 text-[12.5px] text-[#8a97a8]">Date range</p>
        <div className="inline-flex rounded-full border border-[#e8eef4] bg-white p-1">
          {RANGE_OPTIONS.map((d) => (
            <button
              key={d}
              type="button"
              onClick={() => setDays(d)}
              className={`rounded-full px-3.5 py-1.5 text-[12px] font-semibold transition ${
                days === d
                  ? "bg-[#06163a] text-white"
                  : "text-[#6b7c93] hover:bg-[#f4f7fb]"
              }`}
            >
              {d}d
            </button>
          ))}
        </div>
      </Card>

      {loading && !charts ? (
        <Card>
          <LoadingBlock label="Loading report..." />
        </Card>
      ) : !hasData ? (
        <Card>
          <EmptyState message="No transactions in this period yet." />
        </Card>
      ) : (
        <>
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
            <StatTile label="Total revenue" value={formatIDR(totals.totalRevenue)} />
            <StatTile
              label="Total transactions"
              value={totals.totalTransactions.toLocaleString("id-ID")}
            />
            <StatTile
              label="Success rate"
              value={`${totals.successRate.toFixed(1)}%`}
              tone="text-[#2f9e5a]"
            />
            <StatTile
              label="Avg. paid transaction"
              value={formatIDR(Math.round(totals.avgTransaction))}
            />
          </div>

          <Card className="p-5">
            <h2 className="text-[15px] font-semibold text-[#1f2a37]">Revenue trend</h2>
            <p className="mt-0.5 text-[12px] text-[#8a97a8]">
              Paid amount per day · last {days} days
            </p>
            <div className="mt-3 h-[240px] w-full">
              <ResponsiveContainer width="100%" height="100%">
                <ComposedChart data={daily}>
                  <defs>
                    <linearGradient id="revenueFill" x1="0" y1="0" x2="0" y2="1">
                      <stop offset="0%" stopColor={REVENUE_COLOR} stopOpacity={0.3} />
                      <stop offset="100%" stopColor={REVENUE_COLOR} stopOpacity={0.02} />
                    </linearGradient>
                  </defs>
                  <CartesianGrid strokeDasharray="3 3" stroke="#eef2f6" vertical={false} />
                  <XAxis dataKey="label" tick={{ fill: "#8a97a8", fontSize: 11 }} axisLine={false} tickLine={false} />
                  <YAxis
                    tick={{ fill: "#8a97a8", fontSize: 11 }}
                    axisLine={false}
                    tickLine={false}
                    width={52}
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
                    name="Revenue"
                    stroke={REVENUE_COLOR}
                    strokeWidth={2}
                    fill="url(#revenueFill)"
                    dot={false}
                  />
                </ComposedChart>
              </ResponsiveContainer>
            </div>
          </Card>

          <Card className="p-5">
            <h2 className="text-[15px] font-semibold text-[#1f2a37]">
              Transactions by status
            </h2>
            <p className="mt-0.5 text-[12px] text-[#8a97a8]">
              Daily count, broken down by outcome
            </p>
            <div className="mt-3 h-[260px] w-full">
              <ResponsiveContainer width="100%" height="100%">
                <BarChart data={daily} barGap={2} barCategoryGap="20%">
                  <CartesianGrid strokeDasharray="3 3" stroke="#eef2f6" vertical={false} />
                  <XAxis dataKey="label" tick={{ fill: "#8a97a8", fontSize: 11 }} axisLine={false} tickLine={false} />
                  <YAxis allowDecimals={false} tick={{ fill: "#8a97a8", fontSize: 11 }} axisLine={false} tickLine={false} width={28} />
                  <Tooltip content={<Tip />} />
                  {STATUS_ORDER.map((status, i) => (
                    <Bar
                      key={status}
                      dataKey={status}
                      name={status.charAt(0).toUpperCase() + status.slice(1)}
                      stackId="status"
                      fill={STATUS_COLORS[status]}
                      stroke="#ffffff"
                      strokeWidth={2}
                      maxBarSize={24}
                      radius={i === STATUS_ORDER.length - 1 ? [3, 3, 0, 0] : 0}
                    />
                  ))}
                </BarChart>
              </ResponsiveContainer>
            </div>
            <div className="mt-3 flex flex-wrap gap-x-4 gap-y-1.5 border-t border-[#eef2f6] pt-3">
              {STATUS_ORDER.map((status) => (
                <div key={status} className="flex items-center gap-1.5 text-[12px] text-[#6b7c93]">
                  <span
                    className="inline-block size-2.5 rounded-sm"
                    style={{ backgroundColor: STATUS_COLORS[status] }}
                  />
                  <span className="capitalize">{status}</span>
                </div>
              ))}
            </div>
          </Card>

          {charts && (
            <Card className="p-5">
              <h2 className="text-[15px] font-semibold text-[#1f2a37]">
                Status breakdown
              </h2>
              <p className="mt-0.5 text-[12px] text-[#8a97a8]">
                Share of total transactions · last {days} days
              </p>
              <div className="mt-4">
                <StatusBreakdownBar breakdown={charts.status_breakdown ?? []} />
              </div>
            </Card>
          )}

          <Card>
            <div className="border-b border-[#eef2f6] px-5 py-4">
              <h2 className="text-[15px] font-semibold text-[#1f2a37]">Daily detail</h2>
              <p className="mt-0.5 text-[12px] text-[#8a97a8]">
                Same data as the charts above, in table form
              </p>
            </div>
            <TableShell>
              <table className="w-full min-w-150 text-left text-[12.5px]">
                <thead>
                  <tr className="border-b border-[#eef2f6] text-[11px] font-semibold uppercase tracking-wide text-[#8a97a8]">
                    <th className="px-5 py-3">Date</th>
                    <th className="px-5 py-3">Total</th>
                    <th className="px-5 py-3">Paid</th>
                    <th className="px-5 py-3">Pending</th>
                    <th className="px-5 py-3">Failed</th>
                    <th className="px-5 py-3">Expired</th>
                    <th className="px-5 py-3">Cancelled</th>
                    <th className="px-5 py-3">Revenue</th>
                  </tr>
                </thead>
                <tbody>
                  {[...daily].reverse().map((d) => (
                    <tr key={d.date} className="border-b border-[#f3f6f9] text-[#3d4b5c] last:border-0">
                      <td className="px-5 py-3 text-[#6b7c93]">{d.label}</td>
                      <td className="px-5 py-3 font-medium text-[#1f2a37]">{d.total}</td>
                      <td className="px-5 py-3">{d.paid}</td>
                      <td className="px-5 py-3">{d.pending}</td>
                      <td className="px-5 py-3">{d.failed}</td>
                      <td className="px-5 py-3">{d.expired}</td>
                      <td className="px-5 py-3">{d.cancelled}</td>
                      <td className="px-5 py-3 font-medium text-[#1f2a37]">
                        {formatIDR(d.paid_amount)}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </TableShell>
          </Card>
        </>
      )}
    </div>
  );
}
