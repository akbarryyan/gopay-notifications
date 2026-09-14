"use client";

import { useRouter } from "next/navigation";
import { useState, type FormEvent } from "react";
import toast from "react-hot-toast";
import { createMerchantPaymentLink } from "@/lib/merchant-api";
import { Button } from "@/components/ui/button";
import { DatePicker } from "@/components/ui/date-picker";
import { Card, PageHeader } from "@/app/components/admin/ui";

const PLATFORM_MIN_AMOUNT = 2000;
const PLATFORM_MAX_AMOUNT = 10000000;

type AmountType = "fixed" | "open";

function inputClass() {
  return "h-10 w-full rounded-lg border border-[#e8eef4] bg-[#f8fafc] px-3 text-[13.5px] text-[#1f2a37] outline-none focus:border-[#ff5e16] focus:bg-white";
}

export default function NewPaymentLinkPage() {
  const router = useRouter();
  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [amountType, setAmountType] = useState<AmountType>("fixed");
  const [amount, setAmount] = useState("");
  const [minAmount, setMinAmount] = useState("");
  const [maxAmount, setMaxAmount] = useState("");
  const [expiresAt, setExpiresAt] = useState("");
  const [submitting, setSubmitting] = useState(false);

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    if (!title.trim()) {
      toast.error("Title is required.");
      return;
    }
    if (amountType === "fixed" && (!amount || Number(amount) <= 0)) {
      toast.error("Enter a fixed amount greater than zero.");
      return;
    }

    setSubmitting(true);
    try {
      const link = await createMerchantPaymentLink({
        title: title.trim(),
        description: description.trim() || undefined,
        amount_type: amountType,
        amount: amountType === "fixed" ? Number(amount) : undefined,
        min_amount:
          amountType === "open" && minAmount ? Number(minAmount) : undefined,
        max_amount:
          amountType === "open" && maxAmount ? Number(maxAmount) : undefined,
        expires_at: expiresAt ? new Date(expiresAt).toISOString() : undefined,
      });
      toast.success("Payment link created.");
      router.push(`/dashboard/payment-links/${link.id}`);
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Failed to create payment link.",
      );
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <div className="w-full max-w-2xl space-y-5">
      <PageHeader
        title="New payment link"
        description="A reusable checkout URL — every payment made through it is a fresh, independent transaction."
      />

      <Card className="p-5 sm:p-6">
        <form className="space-y-5" onSubmit={handleSubmit}>
          <div className="space-y-1.5">
            <label className="text-[11px] font-semibold uppercase tracking-wide text-[#8a97a8]">
              Title
            </label>
            <input
              className={inputClass()}
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              placeholder="e.g. Donation, Invoice #204, Store checkout"
              maxLength={150}
            />
          </div>

          <div className="space-y-1.5">
            <label className="text-[11px] font-semibold uppercase tracking-wide text-[#8a97a8]">
              Description <span className="normal-case text-[#b7c0cc]">(optional)</span>
            </label>
            <textarea
              className={`${inputClass()} h-20 resize-none py-2`}
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="Shown to the customer on the pay page"
              maxLength={500}
            />
          </div>

          <div className="space-y-1.5">
            <span className="text-[11px] font-semibold uppercase tracking-wide text-[#8a97a8]">
              Amount type
            </span>
            <div className="flex gap-2">
              <button
                type="button"
                onClick={() => setAmountType("fixed")}
                className={`flex-1 rounded-lg border px-3 py-2.5 text-left text-[13px] font-medium transition ${
                  amountType === "fixed"
                    ? "border-[#ff5e16] bg-[#fff4ee] text-[#ff5e16]"
                    : "border-[#e8eef4] bg-white text-[#3d4b5c] hover:bg-[#f8fafc]"
                }`}
              >
                Fixed amount
                <p className="mt-0.5 text-[11.5px] font-normal text-[#8a97a8]">
                  You set one price for everyone.
                </p>
              </button>
              <button
                type="button"
                onClick={() => setAmountType("open")}
                className={`flex-1 rounded-lg border px-3 py-2.5 text-left text-[13px] font-medium transition ${
                  amountType === "open"
                    ? "border-[#ff5e16] bg-[#fff4ee] text-[#ff5e16]"
                    : "border-[#e8eef4] bg-white text-[#3d4b5c] hover:bg-[#f8fafc]"
                }`}
              >
                Customer sets amount
                <p className="mt-0.5 text-[11.5px] font-normal text-[#8a97a8]">
                  Customer enters how much to pay.
                </p>
              </button>
            </div>
          </div>

          {amountType === "fixed" ? (
            <div className="space-y-1.5">
              <label className="text-[11px] font-semibold uppercase tracking-wide text-[#8a97a8]">
                Amount (IDR)
              </label>
              <input
                type="number"
                min={PLATFORM_MIN_AMOUNT}
                max={PLATFORM_MAX_AMOUNT}
                className={inputClass()}
                value={amount}
                onChange={(e) => setAmount(e.target.value)}
                placeholder="50000"
              />
              <p className="text-[11.5px] text-[#8a97a8]">
                Must be between Rp{PLATFORM_MIN_AMOUNT.toLocaleString("id-ID")}{" "}
                and Rp{PLATFORM_MAX_AMOUNT.toLocaleString("id-ID")}.
              </p>
            </div>
          ) : (
            <div className="grid grid-cols-2 gap-3">
              <div className="space-y-1.5">
                <label className="text-[11px] font-semibold uppercase tracking-wide text-[#8a97a8]">
                  Min amount <span className="normal-case text-[#b7c0cc]">(optional)</span>
                </label>
                <input
                  type="number"
                  min={PLATFORM_MIN_AMOUNT}
                  max={PLATFORM_MAX_AMOUNT}
                  className={inputClass()}
                  value={minAmount}
                  onChange={(e) => setMinAmount(e.target.value)}
                  placeholder={String(PLATFORM_MIN_AMOUNT)}
                />
              </div>
              <div className="space-y-1.5">
                <label className="text-[11px] font-semibold uppercase tracking-wide text-[#8a97a8]">
                  Max amount <span className="normal-case text-[#b7c0cc]">(optional)</span>
                </label>
                <input
                  type="number"
                  min={PLATFORM_MIN_AMOUNT}
                  max={PLATFORM_MAX_AMOUNT}
                  className={inputClass()}
                  value={maxAmount}
                  onChange={(e) => setMaxAmount(e.target.value)}
                  placeholder={String(PLATFORM_MAX_AMOUNT)}
                />
              </div>
              <p className="col-span-2 text-[11.5px] text-[#8a97a8]">
                Leave blank to allow the full platform range: Rp
                {PLATFORM_MIN_AMOUNT.toLocaleString("id-ID")}–Rp
                {PLATFORM_MAX_AMOUNT.toLocaleString("id-ID")}.
              </p>
            </div>
          )}

          <DatePicker
            label="Expires on (optional)"
            value={expiresAt}
            onChange={setExpiresAt}
            placeholder="No expiry"
          />

          <div className="flex items-center gap-2 pt-1">
            <Button
              type="submit"
              disabled={submitting}
              className="h-10 rounded-full bg-[#ff5e16] px-5 text-[13px] font-semibold text-white hover:bg-[#ef5510]"
            >
              {submitting ? "Creating..." : "Create link"}
            </Button>
            <Button
              type="button"
              variant="outline"
              className="h-10 rounded-full border-[#e8eef4] px-5 text-[13px] shadow-none"
              onClick={() => router.push("/dashboard/payment-links")}
            >
              Cancel
            </Button>
          </div>
        </form>
      </Card>
    </div>
  );
}
