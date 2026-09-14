"use client";

import { useParams } from "next/navigation";
import { useEffect, useState } from "react";
import {
  fetchPublicPaymentByReference,
  type PublicPayment,
} from "@/lib/merchant-api";
import { formatDateTime, formatIDR } from "@/app/components/admin/ui";

function statusStyle(status: string) {
  switch (status) {
    case "paid":
      return "bg-[#e8f8ee] text-[#2f9e5a]";
    case "pending":
      return "bg-[#fff7e6] text-[#c27a00]";
    case "expired":
    case "failed":
      return "bg-[#fff1ed] text-[#e85d3b]";
    default:
      return "bg-[#eef2f6] text-[#6b7c93]";
  }
}

export default function PublicPayPage() {
  const params = useParams<{ reference: string }>();
  const reference = params?.reference ? decodeURIComponent(params.reference) : "";
  const [payment, setPayment] = useState<PublicPayment | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [now, setNow] = useState(() => Date.now());

  useEffect(() => {
    const t = window.setInterval(() => setNow(Date.now()), 1000);
    return () => window.clearInterval(t);
  }, []);

  useEffect(() => {
    if (!reference) return;
    let cancelled = false;
    async function load() {
      setLoading(true);
      try {
        const data = await fetchPublicPaymentByReference(reference);
        if (!cancelled) {
          setPayment(data);
          setError(null);
        }
      } catch (err) {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : "Payment not found.");
          setPayment(null);
        }
      } finally {
        if (!cancelled) setLoading(false);
      }
    }
    void load();
    // Poll while pending
    const interval = window.setInterval(() => {
      if (!cancelled) void load();
    }, 8000);
    return () => {
      cancelled = true;
      window.clearInterval(interval);
    };
  }, [reference]);

  const remaining = (() => {
    if (!payment?.expires_at) return null;
    const ms = new Date(payment.expires_at).getTime() - now;
    if (ms <= 0) return "Expired";
    const m = Math.floor(ms / 60000);
    const s = Math.floor((ms % 60000) / 1000);
    return `${m}:${String(s).padStart(2, "0")}`;
  })();

  const isImageQr =
    payment?.qris_data?.startsWith("data:image") ||
    payment?.qris_data?.startsWith("http");

  return (
    <main className="flex min-h-screen items-center justify-center bg-[#eff4f8] px-4 py-10">
      <div className="w-full max-w-md overflow-hidden rounded-2xl border border-[#e8eef4] bg-white shadow-[0_16px_48px_rgba(16,38,73,0.08)]">
        <div className="bg-[#06163a] px-6 py-5 text-white">
          <p className="text-[13px] font-medium text-white/70">whuzpay · Checkout</p>
          <h1 className="mt-1 text-[20px] font-semibold tracking-tight">
            {payment?.description || "Payment"}
          </h1>
        </div>

        <div className="space-y-5 p-6">
          {loading && !payment ? (
            <p className="text-center text-[13px] text-[#8a97a8]">Loading...</p>
          ) : error ? (
            <p className="text-center text-[13px] text-[#e85d3b]">{error}</p>
          ) : payment ? (
            <>
              <div className="flex items-start justify-between gap-3">
                <div>
                  <p className="text-[12px] text-[#8a97a8]">Amount</p>
                  <p className="mt-0.5 text-[28px] font-bold tracking-tight text-[#1f2a37]">
                    {formatIDR(payment.amount, payment.currency)}
                  </p>
                  <p className="mt-1 text-[12px] text-[#8a97a8]">
                    Ref · {payment.reference}
                  </p>
                </div>
                <span
                  className={`inline-flex rounded-full px-2.5 py-0.5 text-[11px] font-semibold capitalize ${statusStyle(
                    payment.status,
                  )}`}
                >
                  {payment.status}
                </span>
              </div>

              {payment.status === "pending" && payment.qris_data && (
                <div className="rounded-xl border border-[#eef2f6] bg-[#fafbfc] p-4 text-center">
                  {isImageQr ? (
                    // eslint-disable-next-line @next/next/no-img-element
                    <img
                      src={payment.qris_data}
                      alt="QRIS"
                      className="mx-auto max-h-56 max-w-full rounded-lg"
                    />
                  ) : (
                    <pre className="max-h-40 overflow-auto break-all text-left text-[11px] text-[#3d4b5c]">
                      {payment.qris_data}
                    </pre>
                  )}
                  {remaining && (
                    <p className="mt-3 text-[12.5px] text-[#6b7c93]">
                      Expires in{" "}
                      <span className="font-semibold text-[#1f2a37]">
                        {remaining}
                      </span>
                    </p>
                  )}
                </div>
              )}

              {payment.status === "paid" && (
                <div className="rounded-xl border border-[#c8efd4] bg-[#f0faf3] px-4 py-3 text-center text-[13px] font-medium text-[#2f9e5a]">
                  Payment received
                  {payment.paid_at
                    ? ` · ${formatDateTime(payment.paid_at)}`
                    : ""}
                </div>
              )}

              {(payment.status === "expired" || payment.status === "failed") && (
                <div className="rounded-xl border border-[#ffd4c8] bg-[#fff5f2] px-4 py-3 text-center text-[13px] font-medium text-[#e85d3b]">
                  This payment is {payment.status}. Contact the merchant if you
                  need a new link.
                </div>
              )}

              <p className="text-center text-[11px] text-[#8a97a8]">
                Environment: {payment.environment || "production"} · Do not share
                this page with payment secrets.
              </p>
            </>
          ) : null}
        </div>
      </div>
    </main>
  );
}
