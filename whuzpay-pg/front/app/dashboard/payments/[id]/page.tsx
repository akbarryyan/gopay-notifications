"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { useEffect, useState, type ReactNode } from "react";
import toast from "react-hot-toast";
import { CopyIcon, ExternalLinkIcon } from "lucide-react";
import {
  fetchMerchantPayment,
  type MerchantPayment,
} from "@/lib/merchant-api";
import { Button } from "@/components/ui/button";
import {
  Card,
  formatDateTime,
  formatIDR,
  LoadingBlock,
  PageHeader,
  StatusBadge,
} from "../../../components/admin/ui";

function Field({ label, value }: { label: string; value: ReactNode }) {
  return (
    <div className="rounded-lg bg-[#f8fafc] px-4 py-3">
      <p className="text-[11px] font-semibold uppercase tracking-wide text-[#8a97a8]">
        {label}
      </p>
      <div className="mt-1 text-[13.5px] font-medium break-all text-[#1f2a37]">
        {value || "—"}
      </div>
    </div>
  );
}

export default function MerchantPaymentDetailPage() {
  const params = useParams<{ id: string }>();
  const id = params?.id;
  const [payment, setPayment] = useState<MerchantPayment | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (!id) return;
    let cancelled = false;
    async function load() {
      setLoading(true);
      try {
        const data = await fetchMerchantPayment(id);
        if (!cancelled) setPayment(data);
      } catch (err) {
        if (!cancelled) {
          toast.error(
            err instanceof Error ? err.message : "Failed to load payment.",
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
  }, [id]);

  const payPath = payment ? `/pay/${encodeURIComponent(payment.reference)}` : "";
  const payUrl =
    typeof window !== "undefined" && payPath
      ? `${window.location.origin}${payPath}`
      : payPath;

  async function copy(text: string, label: string) {
    try {
      await navigator.clipboard.writeText(text);
      toast.success(`${label} copied.`);
    } catch {
      toast.error("Failed to copy.");
    }
  }

  const isImageQr =
    payment?.qris_data?.startsWith("data:image") ||
    payment?.qris_data?.startsWith("http");

  return (
    <div className="w-full space-y-5">
      <PageHeader
        title="Payment detail"
        description="Transaction details, QR, and shareable checkout link."
        actions={
          <Link
            href="/dashboard/payments"
            className="text-[13px] font-semibold text-[#6b7c93] hover:text-[#06163a]"
          >
            ← Back to payments
          </Link>
        }
      />

      {loading ? (
        <Card>
          <LoadingBlock />
        </Card>
      ) : !payment ? (
        <Card className="p-6 text-[13px] text-[#8a97a8]">Payment not found.</Card>
      ) : (
        <>
          <Card className="p-5 sm:p-6">
            <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
              <div>
                <p className="text-[12px] text-[#8a97a8]">Reference</p>
                <h2 className="mt-1 text-[18px] font-semibold text-[#1f2a37]">
                  {payment.reference}
                </h2>
                <p className="mt-1 text-[13px] text-[#6b7c93]">
                  {payment.description}
                </p>
              </div>
              <StatusBadge status={payment.status} />
            </div>
            <div className="mt-5 grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-3">
              <Field
                label="Amount"
                value={formatIDR(payment.amount, payment.currency)}
              />
              <Field label="Method" value={payment.payment_method} />
              <Field label="Provider" value={payment.provider_name} />
              <Field
                label="Environment"
                value={
                  <span className="capitalize">
                    {payment.environment || "production"}
                  </span>
                }
              />
              <Field
                label="Provider reference"
                value={payment.provider_reference || "—"}
              />
              <Field label="Customer" value={payment.customer_name || "—"} />
              <Field label="Email" value={payment.customer_email || "—"} />
              <Field label="Created" value={formatDateTime(payment.created_at)} />
              <Field label="Expires" value={formatDateTime(payment.expires_at)} />
              <Field label="Paid at" value={formatDateTime(payment.paid_at)} />
            </div>
          </Card>

          <Card className="p-5 sm:p-6">
            <h3 className="text-[15px] font-semibold text-[#1f2a37]">
              Payment link
            </h3>
            <p className="mt-1 text-[12.5px] text-[#8a97a8]">
              Share this public checkout page with your customer.
            </p>
            <div className="mt-3 flex flex-col gap-2 sm:flex-row sm:items-center">
              <code className="flex-1 break-all rounded-lg border border-[#e8eef4] bg-[#f8fafc] px-3 py-2 text-[12.5px] text-[#1f2a37]">
                {payUrl}
              </code>
              <Button
                type="button"
                variant="outline"
                className="h-9 shrink-0 rounded-full border-[#e8eef4] shadow-none"
                onClick={() => void copy(payUrl, "Payment link")}
              >
                <CopyIcon className="size-3.5" />
                Copy link
              </Button>
              <Button
                type="button"
                variant="outline"
                className="h-9 shrink-0 rounded-full border-[#e8eef4] shadow-none"
                asChild
              >
                <a href={payPath} target="_blank" rel="noreferrer">
                  <ExternalLinkIcon className="size-3.5" />
                  Open
                </a>
              </Button>
            </div>
          </Card>

          {payment.qris_data && (
            <Card className="p-5 sm:p-6">
              <h3 className="text-[15px] font-semibold text-[#1f2a37]">
                QRIS
              </h3>
              <div className="mt-4 flex flex-col items-start gap-4 sm:flex-row">
                {isImageQr ? (
                  // eslint-disable-next-line @next/next/no-img-element
                  <img
                    src={payment.qris_data}
                    alt="QRIS"
                    className="max-h-52 rounded-xl border border-[#eef2f6]"
                  />
                ) : (
                  <pre className="max-h-48 max-w-full overflow-auto rounded-lg bg-[#f8fafc] p-3 text-[11px] break-all text-[#3d4b5c]">
                    {payment.qris_data}
                  </pre>
                )}
                <Button
                  type="button"
                  variant="outline"
                  className="h-9 rounded-full border-[#e8eef4] shadow-none"
                  onClick={() =>
                    void copy(payment.qris_data || "", "QRIS data")
                  }
                >
                  <CopyIcon className="size-3.5" />
                  Copy QR data
                </Button>
              </div>
            </Card>
          )}
        </>
      )}
    </div>
  );
}
