"use client";

import { useState } from "react";
import {
  CartesianGrid,
  Legend,
  Line,
  LineChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";
import type { TooltipContentProps } from "recharts";
import { formatRupiah, formatShortDate } from "@/lib/format";

interface Point {
  date: string;
  count: number;
  paid_amount_rp: number;
}

/**
 * Grafik tren event 14 hari -- biaxial (dua sumbu-Y): jumlah event di kiri,
 * nominal lunas (Rp) di kanan. Dua metrik ini skalanya jauh berbeda (satuan
 * vs jutaan rupiah), makanya butuh dua sumbu -- disatukan ke satu sumbu
 * saja bikin salah satu garis nyaris rata tak berarti.
 *
 * Pakai Recharts, bukan SVG tangan sendiri seperti sebelumnya -- axis,
 * gridline, dan tooltip yang benar/terbaca tanpa menghitung ulang semuanya
 * manual.
 */
export function EventsTrendChart({ data }: { data: Point[] }) {
  const [showTable, setShowTable] = useState(false);

  if (showTable) {
    return <TableView data={data} onBack={() => setShowTable(false)} />;
  }

  return (
    <div className="flex flex-col gap-2">
      <div className="h-64 w-full">
        <ResponsiveContainer width="100%" height="100%">
          <LineChart data={data} margin={{ top: 8, right: 8, bottom: 0, left: 0 }}>
            <CartesianGrid strokeDasharray="3 3" stroke="var(--border)" vertical={false} />
            <XAxis
              dataKey="date"
              tickFormatter={(v: string) => formatShortDate(v)}
              tick={{ fontSize: 11, fill: "var(--muted-foreground)" }}
              tickLine={false}
              axisLine={{ stroke: "var(--border)" }}
              minTickGap={24}
            />
            <YAxis
              yAxisId="count"
              allowDecimals={false}
              tick={{ fontSize: 11, fill: "var(--muted-foreground)" }}
              tickLine={false}
              axisLine={false}
              width={28}
            />
            <YAxis
              yAxisId="amount"
              orientation="right"
              tickFormatter={(v: number) => formatCompactRupiah(v)}
              tick={{ fontSize: 11, fill: "var(--muted-foreground)" }}
              tickLine={false}
              axisLine={false}
              width={48}
            />
            <Tooltip content={ChartTooltip} />
            <Legend
              formatter={(value: string) =>
                value === "count" ? "Jumlah event" : "Nominal lunas"
              }
              wrapperStyle={{ fontSize: 12 }}
            />
            <Line
              yAxisId="count"
              type="monotone"
              dataKey="count"
              name="count"
              stroke="var(--primary)"
              strokeWidth={2}
              dot={{ r: 2.5 }}
              activeDot={{ r: 4 }}
              isAnimationActive={false}
            />
            <Line
              yAxisId="amount"
              type="monotone"
              dataKey="paid_amount_rp"
              name="paid_amount_rp"
              stroke="var(--chart-2)"
              strokeWidth={2}
              dot={{ r: 2.5 }}
              activeDot={{ r: 4 }}
              isAnimationActive={false}
            />
          </LineChart>
        </ResponsiveContainer>
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

function formatCompactRupiah(value: number): string {
  if (value >= 1_000_000) return `${(value / 1_000_000).toFixed(1)}jt`;
  if (value >= 1_000) return `${Math.round(value / 1_000)}rb`;
  return `${value}`;
}

function ChartTooltip({ active, payload, label }: TooltipContentProps) {
  if (!active || !payload || payload.length === 0) return null;
  const count = payload.find((p) => p.dataKey === "count")?.value ?? 0;
  const amount = payload.find((p) => p.dataKey === "paid_amount_rp")?.value ?? 0;
  return (
    <div className="rounded-lg border bg-popover px-3 py-2 text-xs text-popover-foreground shadow-md ring-1 ring-foreground/10">
      <div className="mb-1 font-medium">{formatShortDate(String(label))}</div>
      <div className="flex items-center justify-between gap-4">
        <span className="text-muted-foreground">Jumlah event</span>
        <span className="font-semibold">{count}</span>
      </div>
      <div className="flex items-center justify-between gap-4">
        <span className="text-muted-foreground">Nominal lunas</span>
        <span className="font-semibold">{formatRupiah(Number(amount))}</span>
      </div>
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
              <th className="px-3 py-1.5 text-right font-medium text-muted-foreground">
                Jumlah event
              </th>
              <th className="px-3 py-1.5 text-right font-medium text-muted-foreground">
                Nominal lunas
              </th>
            </tr>
          </thead>
          <tbody>
            {data.map((d) => (
              <tr key={d.date} className="border-t">
                <td className="px-3 py-1.5">{formatShortDate(d.date)}</td>
                <td className="px-3 py-1.5 text-right font-medium">{d.count}</td>
                <td className="px-3 py-1.5 text-right font-medium">
                  {formatRupiah(d.paid_amount_rp)}
                </td>
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
