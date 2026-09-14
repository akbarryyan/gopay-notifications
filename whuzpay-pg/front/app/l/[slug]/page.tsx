"use client";

import { useParams, useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import {
  fetchPublicPaymentLink,
  initiatePaymentLinkCheckout,
  type PublicPaymentLink,
} from "@/lib/merchant-api";
import { formatIDR } from "@/app/components/admin/ui";

export default function PublicPaymentLinkPage() {
  const params = useParams<{ slug: string }>();
  const router = useRouter();
  const slug = params?.slug ? decodeURIComponent(params.slug) : "";

  const [link, setLink] = useState<PublicPaymentLink | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const [amount, setAmount] = useState("");
  const [customerName, setCustomerName] = useState("");
  const [customerEmail, setCustomerEmail] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [submitError, setSubmitError] = useState<string | null>(null);

  useEffect(() => {
    if (!slug) return;
    let cancelled = false;
    async function load() {
      setLoading(true);
      try {
        const data = await fetchPublicPaymentLink(slug);
        if (!cancelled) {
          setLink(data);
          setError(null);
        }
      } catch (err) {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : "Payment link not found.");
          setLink(null);
        }
      } finally {
        if (!cancelled) setLoading(false);
      }
    }
    void load();
    return () => {
      cancelled = true;
    };
  }, [slug]);

  async function handlePay() {
    if (!link) return;
    setSubmitError(null);

    let numericAmount: number | undefined;
    if (link.amount_type === "open") {
      numericAmount = Number(amount);
      if (!amount || Number.isNaN(numericAmount) || numericAmount <= 0) {
        setSubmitError("Enter a valid amount.");
        return;
      }
      if (link.min_amount && numericAmount < link.min_amount) {
        setSubmitError(`Minimum amount is ${formatIDR(link.min_amount, link.currency)}.`);
        return;
      }
      if (link.max_amount && numericAmount > link.max_amount) {
        setSubmitError(`Maximum amount is ${formatIDR(link.max_amount, link.currency)}.`);
        return;
      }
    }

    setSubmitting(true);
    try {
      const payment = await initiatePaymentLinkCheckout(slug, {
        amount: link.amount_type === "open" ? numericAmount : undefined,
        customer_name: customerName.trim() || undefined,
        customer_email: customerEmail.trim() || undefined,
      });
      router.push(`/pay/${encodeURIComponent(payment.reference)}`);
    } catch (err) {
      setSubmitError(
        err instanceof Error ? err.message : "Failed to start checkout.",
      );
      setSubmitting(false);
    }
  }

  return (
    <main className="flex min-h-screen items-center justify-center bg-[#eff4f8] px-4 py-10">
      <div className="w-full max-w-md overflow-hidden rounded-2xl border border-[#e8eef4] bg-white shadow-[0_16px_48px_rgba(16,38,73,0.08)]">
        <div className="bg-[#06163a] px-6 py-5 text-white">
          <p className="text-[13px] font-medium text-white/70">whuzpay · Payment link</p>
          <h1 className="mt-1 text-[20px] font-semibold tracking-tight">
            {link?.title || "Payment link"}
          </h1>
        </div>

        <div className="space-y-5 p-6">
          {loading ? (
            <p className="text-center text-[13px] text-[#8a97a8]">Loading...</p>
          ) : error ? (
            <p className="text-center text-[13px] text-[#e85d3b]">{error}</p>
          ) : link && !link.is_available ? (
            <div className="rounded-xl border border-[#ffd4c8] bg-[#fff5f2] px-4 py-3 text-center text-[13px] font-medium text-[#e85d3b]">
              This link is no longer available
              {link.reason ? `: ${link.reason}` : "."}
            </div>
          ) : link ? (
            <>
              {link.description && (
                <p className="text-[13px] text-[#6b7c93]">{link.description}</p>
              )}

              {link.amount_type === "fixed" && link.amount ? (
                <div>
                  <p className="text-[12px] text-[#8a97a8]">Amount</p>
                  <p className="mt-0.5 text-[28px] font-bold tracking-tight text-[#1f2a37]">
                    {formatIDR(link.amount, link.currency)}
                  </p>
                </div>
              ) : (
                <div className="space-y-1.5">
                  <label className="text-[11px] font-semibold uppercase tracking-wide text-[#8a97a8]">
                    Amount ({link.currency})
                  </label>
                  <input
                    type="number"
                    className="h-11 w-full rounded-lg border border-[#e8eef4] bg-[#f8fafc] px-3 text-[15px] text-[#1f2a37] outline-none focus:border-[#ff5e16] focus:bg-white"
                    value={amount}
                    onChange={(e) => setAmount(e.target.value)}
                    placeholder="Enter amount"
                  />
                  <p className="text-[11.5px] text-[#8a97a8]">
                    {link.min_amount ? formatIDR(link.min_amount, link.currency) : "Any amount"}
                    {" – "}
                    {link.max_amount ? formatIDR(link.max_amount, link.currency) : "no maximum"}
                  </p>
                </div>
              )}

              <div className="space-y-3">
                <div className="space-y-1.5">
                  <label className="text-[11px] font-semibold uppercase tracking-wide text-[#8a97a8]">
                    Name (optional)
                  </label>
                  <input
                    className="h-10 w-full rounded-lg border border-[#e8eef4] bg-[#f8fafc] px-3 text-[13.5px] text-[#1f2a37] outline-none focus:border-[#ff5e16] focus:bg-white"
                    value={customerName}
                    onChange={(e) => setCustomerName(e.target.value)}
                  />
                </div>
                <div className="space-y-1.5">
                  <label className="text-[11px] font-semibold uppercase tracking-wide text-[#8a97a8]">
                    Email (optional)
                  </label>
                  <input
                    type="email"
                    className="h-10 w-full rounded-lg border border-[#e8eef4] bg-[#f8fafc] px-3 text-[13.5px] text-[#1f2a37] outline-none focus:border-[#ff5e16] focus:bg-white"
                    value={customerEmail}
                    onChange={(e) => setCustomerEmail(e.target.value)}
                  />
                </div>
              </div>

              {submitError && (
                <p className="text-center text-[12.5px] text-[#e85d3b]">
                  {submitError}
                </p>
              )}

              <button
                type="button"
                disabled={submitting}
                onClick={() => void handlePay()}
                className="h-11 w-full rounded-full bg-[#ff5e16] text-[14px] font-semibold text-white transition hover:bg-[#ef5510] disabled:opacity-60"
              >
                {submitting ? "Generating QR..." : "Pay now"}
              </button>

              <p className="text-center text-[11px] text-[#8a97a8]">
                Environment: {link.environment} · A fresh QRIS is generated for
                this payment.
              </p>
            </>
          ) : null}
        </div>
      </div>
    </main>
  );
}
