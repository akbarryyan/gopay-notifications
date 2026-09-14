"use client";

import { useEffect, useState, type FormEvent } from "react";
import toast from "react-hot-toast";
import {
  createAdminPayment,
  fetchAdminMerchants,
  type AdminMerchant,
} from "@/lib/admin-api";
import { Button } from "@/components/ui/button";
import {
  Drawer,
  DrawerClose,
  DrawerContent,
  DrawerDescription,
  DrawerFooter,
  DrawerHeader,
  DrawerTitle,
} from "@/components/ui/drawer";

type CreatePaymentDrawerProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onCreated: () => void;
};

function ButtonSpinner() {
  return (
    <svg className="size-4 animate-spin" viewBox="0 0 24 24" fill="none" aria-hidden>
      <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="3" />
      <path className="opacity-90" fill="currentColor" d="M4 12a8 8 0 0 1 8-8v3a5 5 0 0 0-5 5H4Z" />
    </svg>
  );
}

export default function CreatePaymentDrawer({
  open,
  onOpenChange,
  onCreated,
}: CreatePaymentDrawerProps) {
  const [merchants, setMerchants] = useState<AdminMerchant[]>([]);
  const [merchantID, setMerchantID] = useState("");
  const [amount, setAmount] = useState("");
  const [description, setDescription] = useState("");
  const [customerName, setCustomerName] = useState("");
  const [customerEmail, setCustomerEmail] = useState("");
  const [expiresIn, setExpiresIn] = useState("30");
  const [loadingMerchants, setLoadingMerchants] = useState(false);
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    if (!open) return;
    let cancelled = false;

    async function loadMerchants() {
      setLoadingMerchants(true);
      try {
        const data = await fetchAdminMerchants({
          status: "active",
          limit: 100,
          offset: 0,
        });
        if (cancelled) return;
        setMerchants(data.items ?? []);
      } catch (err) {
        if (!cancelled) {
          toast.error(
            err instanceof Error ? err.message : "Failed to load merchants.",
          );
        }
      } finally {
        if (!cancelled) setLoadingMerchants(false);
      }
    }

    void loadMerchants();
    return () => {
      cancelled = true;
    };
  }, [open]);

  function resetForm() {
    setMerchantID("");
    setAmount("");
    setDescription("");
    setCustomerName("");
    setCustomerEmail("");
    setExpiresIn("30");
  }

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();

    const amountValue = Number(amount);
    if (!merchantID) {
      toast.error("Merchant is required.");
      return;
    }
    if (!Number.isFinite(amountValue) || amountValue <= 0) {
      toast.error("Amount must be greater than 0.");
      return;
    }
    if (!description.trim()) {
      toast.error("Description is required.");
      return;
    }

    setSubmitting(true);
    try {
      await createAdminPayment({
        merchant_id: merchantID,
        amount: Math.round(amountValue),
        description: description.trim(),
        customer_name: customerName.trim() || undefined,
        customer_email: customerEmail.trim() || undefined,
        expires_in_minutes: Number(expiresIn) || 30,
        payment_method: "qris",
        currency: "IDR",
      });
      toast.success("Payment created successfully.");
      resetForm();
      onOpenChange(false);
      onCreated();
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Failed to create payment.",
      );
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <Drawer
      open={open}
      onOpenChange={(next) => {
        if (submitting) return;
        onOpenChange(next);
        if (!next) resetForm();
      }}
      direction="right"
    >
      <DrawerContent className="data-[vaul-drawer-direction=right]:w-full data-[vaul-drawer-direction=right]:sm:max-w-md border-[#e8eef4] bg-white">
        <DrawerHeader className="border-b border-[#eef2f6] text-left">
          <DrawerTitle className="text-[18px] font-semibold text-[#1f2a37]">
            Create payment
          </DrawerTitle>
          <DrawerDescription className="text-[13px] text-[#8a97a8]">
            Create a QRIS payment for an active merchant.
          </DrawerDescription>
        </DrawerHeader>

        <form
          id="create-payment-form"
          onSubmit={handleSubmit}
          className="flex flex-1 flex-col overflow-y-auto px-4 py-4"
        >
          <div className="space-y-4">
            <div>
              <label
                htmlFor="payment-merchant"
                className="mb-1.5 block text-[13px] font-medium text-[#3d4b5c]"
              >
                Merchant <span className="text-[#e85d3b]">*</span>
              </label>
              <select
                id="payment-merchant"
                value={merchantID}
                onChange={(e) => setMerchantID(e.target.value)}
                disabled={loadingMerchants}
                className="h-10 w-full rounded-lg border border-[#e8eef4] bg-[#f8fafc] px-3 text-[13.5px] text-[#1f2a37] outline-none focus:border-[#3b9eff] focus:bg-white focus:ring-2 focus:ring-[#3b9eff]/15"
              >
                <option value="">
                  {loadingMerchants ? "Loading merchants..." : "Select merchant"}
                </option>
                {merchants.map((m) => (
                  <option key={m.id} value={m.id}>
                    {m.business_name} ({m.email})
                  </option>
                ))}
              </select>
            </div>

            <div>
              <label
                htmlFor="payment-amount"
                className="mb-1.5 block text-[13px] font-medium text-[#3d4b5c]"
              >
                Amount (IDR) <span className="text-[#e85d3b]">*</span>
              </label>
              <input
                id="payment-amount"
                type="number"
                min={1}
                step={1}
                value={amount}
                onChange={(e) => setAmount(e.target.value)}
                placeholder="50000"
                className="h-10 w-full rounded-lg border border-[#e8eef4] bg-[#f8fafc] px-3 text-[13.5px] text-[#1f2a37] outline-none focus:border-[#3b9eff] focus:bg-white focus:ring-2 focus:ring-[#3b9eff]/15"
              />
            </div>

            <div>
              <label
                htmlFor="payment-description"
                className="mb-1.5 block text-[13px] font-medium text-[#3d4b5c]"
              >
                Description <span className="text-[#e85d3b]">*</span>
              </label>
              <input
                id="payment-description"
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                placeholder="Order #123"
                className="h-10 w-full rounded-lg border border-[#e8eef4] bg-[#f8fafc] px-3 text-[13.5px] text-[#1f2a37] outline-none focus:border-[#3b9eff] focus:bg-white focus:ring-2 focus:ring-[#3b9eff]/15"
              />
            </div>

            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
              <div>
                <label
                  htmlFor="payment-customer-name"
                  className="mb-1.5 block text-[13px] font-medium text-[#3d4b5c]"
                >
                  Customer name
                </label>
                <input
                  id="payment-customer-name"
                  value={customerName}
                  onChange={(e) => setCustomerName(e.target.value)}
                  placeholder="Budi"
                  className="h-10 w-full rounded-lg border border-[#e8eef4] bg-[#f8fafc] px-3 text-[13.5px] text-[#1f2a37] outline-none focus:border-[#3b9eff] focus:bg-white focus:ring-2 focus:ring-[#3b9eff]/15"
                />
              </div>
              <div>
                <label
                  htmlFor="payment-customer-email"
                  className="mb-1.5 block text-[13px] font-medium text-[#3d4b5c]"
                >
                  Customer email
                </label>
                <input
                  id="payment-customer-email"
                  type="email"
                  value={customerEmail}
                  onChange={(e) => setCustomerEmail(e.target.value)}
                  placeholder="budi@email.com"
                  className="h-10 w-full rounded-lg border border-[#e8eef4] bg-[#f8fafc] px-3 text-[13.5px] text-[#1f2a37] outline-none focus:border-[#3b9eff] focus:bg-white focus:ring-2 focus:ring-[#3b9eff]/15"
                />
              </div>
            </div>

            <div>
              <label
                htmlFor="payment-expires"
                className="mb-1.5 block text-[13px] font-medium text-[#3d4b5c]"
              >
                Expires in (minutes)
              </label>
              <input
                id="payment-expires"
                type="number"
                min={1}
                max={1440}
                value={expiresIn}
                onChange={(e) => setExpiresIn(e.target.value)}
                className="h-10 w-full rounded-lg border border-[#e8eef4] bg-[#f8fafc] px-3 text-[13.5px] text-[#1f2a37] outline-none focus:border-[#3b9eff] focus:bg-white focus:ring-2 focus:ring-[#3b9eff]/15"
              />
            </div>

            <div className="rounded-lg bg-[#f8fafc] px-3 py-2.5 text-[12.5px] text-[#6b7c93]">
              Method: <span className="font-semibold text-[#1f2a37]">QRIS</span>
              {" · "}
              Currency: <span className="font-semibold text-[#1f2a37]">IDR</span>
            </div>
          </div>
        </form>

        <DrawerFooter className="border-t border-[#eef2f6] sm:flex-row sm:justify-end">
          <DrawerClose asChild>
            <Button
              type="button"
              variant="outline"
              disabled={submitting}
              className="border-[#e8eef4] text-[#3d4b5c]"
            >
              Cancel
            </Button>
          </DrawerClose>
          <Button
            type="submit"
            form="create-payment-form"
            disabled={submitting || loadingMerchants}
            className="bg-[#ff5e16] text-white hover:bg-[#ef5510]"
          >
            {submitting ? (
              <>
                <ButtonSpinner />
                Creating...
              </>
            ) : (
              "Create payment"
            )}
          </Button>
        </DrawerFooter>
      </DrawerContent>
    </Drawer>
  );
}
