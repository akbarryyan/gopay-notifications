"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import toast from "react-hot-toast";
import MerchantChartsPanel from "@/app/components/merchant/MerchantCharts";
import {
  fetchMerchantDashboardCharts,
  fetchMerchantDashboardSummary,
  fetchMerchantPayments,
  type MerchantCharts,
  type MerchantDashboardSummary,
  type MerchantPayment,
} from "@/lib/merchant-api";
import { getMerchantProfile } from "@/lib/merchant-auth";
import { useMerchantEnvironment } from "@/lib/use-merchant-environment";
import {
  Card,
  EmptyState,
  formatDateTime,
  formatIDR,
  LoadingBlock,
  PageHeader,
  StatusBadge,
  TableShell,
} from "../components/admin/ui";

export default function MerchantDashboardPage() {
  const { environment } = useMerchantEnvironment();
  const [summary, setSummary] = useState<MerchantDashboardSummary | null>(null);
  const [recent, setRecent] = useState<MerchantPayment[]>([]);
  const [charts, setCharts] = useState<MerchantCharts | null>(null);
  const [chartDays, setChartDays] = useState(14);
  const [loading, setLoading] = useState(true);
  const profile = typeof window !== "undefined" ? getMerchantProfile() : null;

  useEffect(() => {
    let cancelled = false;
    async function load() {
      setLoading(true);
      try {
        const [s, p, c] = await Promise.all([
          fetchMerchantDashboardSummary(environment),
          fetchMerchantPayments({ limit: 8, offset: 0, environment }),
          fetchMerchantDashboardCharts(chartDays, environment),
        ]);
        if (cancelled) return;
        setSummary(s);
        setRecent(p.items ?? []);
        setCharts(c);
      } catch (err) {
        if (!cancelled) {
          toast.error(
            err instanceof Error ? err.message : "Failed to load dashboard.",
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
  }, [environment, chartDays]);

  const cards = summary
    ? [
        {
          label: "Total payments",
          value: summary.total_payments.toLocaleString("id-ID"),
          tone: "text-[#1f2a37]",
          href: "/dashboard/payments",
        },
        {
          label: "Paid",
          value: summary.paid_payments.toLocaleString("id-ID"),
          tone: "text-[#2f9e5a]",
          href: "/dashboard/payments?status=paid",
        },
        {
          label: "Pending",
          value: summary.pending_payments.toLocaleString("id-ID"),
          tone: "text-[#c27a00]",
          href: "/dashboard/payments?status=pending",
        },
        {
          label: "Failed / expired",
          value: (
            summary.failed_payments + summary.expired_payments
          ).toLocaleString("id-ID"),
          tone: "text-[#e85d3b]",
          href: "/dashboard/payments?status=failed",
        },
        {
          label: "Paid amount",
          value: formatIDR(summary.paid_amount),
          tone: "text-[#1f2a37]",
          href: "/dashboard/payments?status=paid",
        },
      ]
    : [];

  return (
    <div className="w-full space-y-5">
      <PageHeader
        title="Dashboard"
        description={
          profile?.business_name
            ? `${profile.business_name} · ${environment} overview`
            : `Overview of your ${environment} payments and revenue.`
        }
        actions={
          <Link
            href="/dashboard/payments"
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

          {charts && (
            <MerchantChartsPanel
              data={charts}
              days={chartDays}
              onDaysChange={setChartDays}
            />
          )}

          <Card>
            <div className="flex items-center justify-between border-b border-[#eef2f6] px-5 py-4">
              <div>
                <h2 className="text-[15px] font-semibold text-[#1f2a37]">
                  Recent payments
                </h2>
                <p className="mt-0.5 text-[12px] text-[#8a97a8]">
                  Latest transactions for your store
                </p>
              </div>
              <Link
                href="/dashboard/payments"
                className="text-[12.5px] font-semibold text-[#ff5e16] hover:text-[#ef5510]"
              >
                See all
              </Link>
            </div>
            {recent.length === 0 ? (
              <EmptyState message="No payments yet." />
            ) : (
              <TableShell>
                <table className="w-full min-w-180 text-left text-[12.5px]">
                  <thead>
                    <tr className="border-b border-[#eef2f6] text-[11px] font-semibold uppercase tracking-wide text-[#8a97a8]">
                      <th className="px-5 py-3">Reference</th>
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
                            href={`/dashboard/payments/${p.id}`}
                            className="font-medium text-[#1f2a37] hover:text-[#ff5e16]"
                          >
                            {p.reference}
                          </Link>
                          <p className="mt-0.5 text-[11px] text-[#8a97a8]">
                            {p.provider_name} · {p.payment_method}
                          </p>
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
