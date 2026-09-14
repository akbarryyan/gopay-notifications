"use client";

import { useState, type FormEvent } from "react";
import toast from "react-hot-toast";
import {
  updateAdminMerchant,
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

type EditMerchantDrawerProps = {
  open: boolean;
  merchant: AdminMerchant | null;
  onOpenChange: (open: boolean) => void;
  onUpdated: (merchant: AdminMerchant) => void;
};

function ButtonSpinner() {
  return (
    <svg className="size-4 animate-spin" viewBox="0 0 24 24" fill="none" aria-hidden>
      <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="3" />
      <path className="opacity-90" fill="currentColor" d="M4 12a8 8 0 0 1 8-8v3a5 5 0 0 0-5 5H4Z" />
    </svg>
  );
}

export default function EditMerchantDrawer({
  open,
  merchant,
  onOpenChange,
  onUpdated,
}: EditMerchantDrawerProps) {
  const [name, setName] = useState("");
  const [businessName, setBusinessName] = useState("");
  const [phone, setPhone] = useState("");
  const [webhookURL, setWebhookURL] = useState("");
  const [submitting, setSubmitting] = useState(false);

  // Reset the form fields whenever the drawer opens for a merchant —
  // adjusted during render (React's documented pattern for resetting state
  // when an input changes) instead of an effect+setState round-trip.
  const resetKey = open ? (merchant?.id ?? null) : null;
  const [prevResetKey, setPrevResetKey] = useState(resetKey);
  if (prevResetKey !== resetKey) {
    setPrevResetKey(resetKey);
    if (resetKey && merchant) {
      setName(merchant.name ?? "");
      setBusinessName(merchant.business_name ?? "");
      setPhone(merchant.phone ?? "");
      setWebhookURL(merchant.webhook_url ?? "");
    }
  }

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    if (!merchant) return;

    if (!name.trim() || !businessName.trim()) {
      toast.error("Owner name and business name are required.");
      return;
    }

    setSubmitting(true);
    try {
      const updated = await updateAdminMerchant(merchant.id, {
        name: name.trim(),
        business_name: businessName.trim(),
        phone: phone.trim(),
        webhook_url: webhookURL.trim() || null,
      });
      toast.success("Merchant updated successfully.");
      onUpdated(updated);
      onOpenChange(false);
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Failed to update merchant.",
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
      }}
      direction="right"
    >
      <DrawerContent className="data-[vaul-drawer-direction=right]:w-full data-[vaul-drawer-direction=right]:sm:max-w-md border-[#e8eef4] bg-white">
        <DrawerHeader className="border-b border-[#eef2f6] text-left">
          <DrawerTitle className="text-[18px] font-semibold text-[#1f2a37]">
            Edit merchant
          </DrawerTitle>
          <DrawerDescription className="text-[13px] text-[#8a97a8]">
            Update merchant profile. Email cannot be changed here.
          </DrawerDescription>
        </DrawerHeader>

        <form
          id="edit-merchant-form"
          onSubmit={handleSubmit}
          className="flex flex-1 flex-col overflow-y-auto px-4 py-4"
        >
          <div className="space-y-4">
            <div>
              <label className="mb-1.5 block text-[13px] font-medium text-[#3d4b5c]">
                Email
              </label>
              <input
                readOnly
                value={merchant?.email ?? ""}
                className="h-10 w-full rounded-lg border border-[#e8eef4] bg-[#f1f5f9] px-3 text-[13.5px] text-[#6b7c93] outline-none"
              />
            </div>

            <div>
              <label
                htmlFor="edit-business-name"
                className="mb-1.5 block text-[13px] font-medium text-[#3d4b5c]"
              >
                Business name <span className="text-[#e85d3b]">*</span>
              </label>
              <input
                id="edit-business-name"
                value={businessName}
                onChange={(e) => setBusinessName(e.target.value)}
                className="h-10 w-full rounded-lg border border-[#e8eef4] bg-[#f8fafc] px-3 text-[13.5px] text-[#1f2a37] outline-none focus:border-[#3b9eff] focus:bg-white focus:ring-2 focus:ring-[#3b9eff]/15"
              />
            </div>

            <div>
              <label
                htmlFor="edit-owner-name"
                className="mb-1.5 block text-[13px] font-medium text-[#3d4b5c]"
              >
                Owner name <span className="text-[#e85d3b]">*</span>
              </label>
              <input
                id="edit-owner-name"
                value={name}
                onChange={(e) => setName(e.target.value)}
                className="h-10 w-full rounded-lg border border-[#e8eef4] bg-[#f8fafc] px-3 text-[13.5px] text-[#1f2a37] outline-none focus:border-[#3b9eff] focus:bg-white focus:ring-2 focus:ring-[#3b9eff]/15"
              />
            </div>

            <div>
              <label
                htmlFor="edit-phone"
                className="mb-1.5 block text-[13px] font-medium text-[#3d4b5c]"
              >
                Phone
              </label>
              <input
                id="edit-phone"
                value={phone}
                onChange={(e) => setPhone(e.target.value)}
                className="h-10 w-full rounded-lg border border-[#e8eef4] bg-[#f8fafc] px-3 text-[13.5px] text-[#1f2a37] outline-none focus:border-[#3b9eff] focus:bg-white focus:ring-2 focus:ring-[#3b9eff]/15"
              />
            </div>

            <div>
              <label
                htmlFor="edit-webhook"
                className="mb-1.5 block text-[13px] font-medium text-[#3d4b5c]"
              >
                Webhook URL
              </label>
              <input
                id="edit-webhook"
                value={webhookURL}
                onChange={(e) => setWebhookURL(e.target.value)}
                className="h-10 w-full rounded-lg border border-[#e8eef4] bg-[#f8fafc] px-3 text-[13.5px] text-[#1f2a37] outline-none focus:border-[#3b9eff] focus:bg-white focus:ring-2 focus:ring-[#3b9eff]/15"
              />
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
            form="edit-merchant-form"
            disabled={submitting}
            className="bg-[#ff5e16] text-white hover:bg-[#ef5510]"
          >
            {submitting ? (
              <>
                <ButtonSpinner />
                Saving...
              </>
            ) : (
              "Save changes"
            )}
          </Button>
        </DrawerFooter>
      </DrawerContent>
    </Drawer>
  );
}
