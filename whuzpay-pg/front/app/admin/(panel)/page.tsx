"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import toast from "react-hot-toast";
import DashboardCharts from "@/app/components/admin/DashboardCharts";
import {
  fetchAdminPayments,
  fetchDashboardCharts,
  fetchDashboardSummary,
  type AdminPayment,
  type DashboardCharts as DashboardChartsData,
  type DashboardSummary,
} from "@/lib/admin-api";
import {
  Card,
  EmptyState,
  formatDateTime,
  formatIDR,
  LoadingBlock,
  PageHeader,
  StatusBadge,
  TableShell,
} from "../../components/admin/ui";

const CHART_DAY_OPTIONS = [7, 14, 30] as const;

export default function AdminDashboardPage() {
  const [summary, setSummary] = useState<DashboardSummary | null>(null);
  const [charts, setCharts] = useState<DashboardChartsData | null>(null);
  const [recent, setRecent] = useState<AdminPayment[]>([]);
  const [loading, setLoading] = useState(true);
  const [chartDays, setChartDays] =
    useState<(typeof CHART_DAY_OPTIONS)[number]>(14);
  const [chartsLoading, setChartsLoading] = useState(false);

  // Summary + recent once on mount
  useEffect(() => {
    let cancelled = false;

    async function load() {
      setLoading(true);
      try {
        const [summaryData, paymentsData] = await Promise.all([
          fetchDashboardSummary(),
          fetchAdminPayments({ limit: 8, offset: 0 }),
        ]);
        if (cancelled) return;
        setSummary(summaryData);
        setRecent(paymentsData.items ?? []);
      } catch (err) {
        if (cancelled) return;
        toast.error(
          err instanceof Error ? err.message : "Failed to load dashboard.",
        );
      } finally {
        if (!cancelled) setLoading(false);
      }
    }

    void load();
    return () => {
      cancelled = true;
    };
  }, []);

  // Charts when range changes
  useEffect(() => {
    let cancelled = false;

    async function loadCharts() {
      setChartsLoading(true);
      try {
        const chartsData = await fetchDashboardCharts(chartDays);
        if (!cancelled) setCharts(chartsData);
      } catch (err) {
        if (!cancelled) {
          toast.error(
            err instanceof Error ? err.message : "Failed to load charts.",
          );
        }
      } finally {
        if (!cancelled) setChartsLoading(false);
      }
    }

    void loadCharts();
    return () => {
      cancelled = true;
    };
  }, [chartDays]);

  const cards = summary
    ? [
        {
          label: "Total payments",
          value: summary.total_payments.toLocaleString("id-ID"),
          tone: "text-[#1f2a37]",
          href: "/admin/payments",
        },
        {
          label: "Paid",
          value: summary.paid_payments.toLocaleString("id-ID"),
          tone: "text-[#2f9e5a]",
          href: "/admin/payments?status=paid",
        },
        {
          label: "Pending",
          value: summary.pending_payments.toLocaleString("id-ID"),
          tone: "text-[#c27a00]",
          href: "/admin/payments?status=pending",
        },
        {
          label: "Failed / expired",
          value: (
            summary.failed_payments + summary.expired_payments
          ).toLocaleString("id-ID"),
          tone: "text-[#e85d3b]",
          href: "/admin/payments?status=failed",
        },
        {
          label: "Paid amount",
          value: formatIDR(summary.paid_amount),
          tone: "text-[#1f2a37]",
          href: "/admin/payments?status=paid",
        },
        {
          label: "Merchants",
          value: summary.total_merchants.toLocaleString("id-ID"),
          tone: "text-[#1f2a37]",
          href: "/admin/merchants",
        },
      ]
    : [];

  return (
    <div className="w-full space-y-5">
      <PageHeader
        title="Dashboard"
        description="Platform overview for payments, merchants, and operational health."
        actions={
          <Link
            href="/admin/payments"
            className="inline-flex rounded-full bg-[#ff5e16] px-4 py-2 text-[12.5px] font-semibold text-white shadow-[0_4px_14px_rgba(255,94,22,0.25)] transition hover:bg-[#ef5510]"
          >
            View payments
          </Link>
        }
      />

      {loading ? (
        <Card>
          <LoadingBlock label="Loading dashboard..." />
        </Card>
      ) : (
        <>
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-3">
            {cards.map((card) => (
              <Link key={card.label} href={card.href} className="group block">
                <Card className="h-full p-5 transition-shadow group-hover:shadow-md group-hover:ring-1 group-hover:ring-[#e8eef4]">
                  <p className="text-[12px] text-[#8a97a8]">{card.label}</p>
                  <p
                    className={`mt-2 text-[22px] font-bold tracking-tight ${card.tone}`}
                  >
                    {card.value}
                  </p>
                  <p className="mt-2 text-[11.5px] font-semibold text-[#ff5e16]">
                    View details →
                  </p>
                </Card>
              </Link>
            ))}
          </div>

          <div className="flex flex-wrap items-center justify-between gap-3">
            <p className="text-[12.5px] text-[#8a97a8]">
              Chart range
              {chartsLoading ? " · updating..." : ""}
            </p>
            <div className="inline-flex rounded-full border border-[#e8eef4] bg-white p-1">
              {CHART_DAY_OPTIONS.map((d) => (
                <button
                  key={d}
                  type="button"
                  onClick={() => setChartDays(d)}
                  className={`rounded-full px-3.5 py-1.5 text-[12px] font-semibold transition ${
                    chartDays === d
                      ? "bg-[#06163a] text-white"
                      : "text-[#6b7c93] hover:bg-[#f4f7fb]"
                  }`}
                >
                  {d}d
                </button>
              ))}
            </div>
          </div>

          {charts ? (
            <DashboardCharts data={charts} />
          ) : (
            <Card>
              <LoadingBlock label="Loading charts..." />
            </Card>
          )}

          <Card>
            <div className="flex items-center justify-between border-b border-[#eef2f6] px-5 py-4">
              <div>
                <h2 className="text-[15px] font-semibold text-[#1f2a37]">
                  Recent payments
                </h2>
                <p className="mt-0.5 text-[12px] text-[#8a97a8]">
                  Latest transactions across all merchants
                </p>
              </div>
              <Link
                href="/admin/payments"
                className="text-[12.5px] font-semibold text-[#ff5e16] hover:text-[#ef5510]"
              >
                See all
              </Link>
            </div>

            {recent.length === 0 ? (
              <EmptyState message="No payments yet." />
            ) : (
              <TableShell>
                <table className="w-full min-w-[720px] text-left text-[12.5px]">
                  <thead>
                    <tr className="border-b border-[#eef2f6] text-[11px] font-semibold uppercase tracking-wide text-[#8a97a8]">
                      <th className="px-5 py-3">Reference</th>
                      <th className="px-5 py-3">Merchant</th>
                      <th className="px-5 py-3">Amount</th>
                      <th className="px-5 py-3">Status</th>
                      <th className="px-5 py-3">Created</th>
                    </tr>
                  </thead>
                  <tbody>
                    {recent.map((p) => (
                      <tr
                        key={p.id}
                        className="border-b border-[#f3f6f9] text-[#3d4b5c] last:border-0"
                      >
                        <td className="px-5 py-3.5">
                          <Link
                            href={`/admin/payments/${p.id}`}
                            className="font-medium text-[#1f2a37] hover:text-[#ff5e16]"
                          >
                            {p.reference}
                          </Link>
                          <p className="mt-0.5 text-[11px] text-[#8a97a8]">
                            {p.provider_name} · {p.payment_method}
                          </p>
                        </td>
                        <td className="px-5 py-3.5">
                          {p.merchant_name || "—"}
                        </td>
                        <td className="px-5 py-3.5 font-medium text-[#1f2a37]">
                          {formatIDR(p.amount, p.currency)}
                        </td>
                        <td className="px-5 py-3.5">
                          <StatusBadge status={p.status} />
                        </td>
                        <td className="px-5 py-3.5 text-[#6b7c93]">
                          {formatDateTime(p.created_at)}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </TableShell>
            )}
          </Card>
        </>
      )}
    </div>
  );
}
