"use client";

import { useState, type FormEvent } from "react";
import toast from "react-hot-toast";
import { createAdminMerchant } from "@/lib/admin-api";
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

type CreateMerchantDrawerProps = {
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

export default function CreateMerchantDrawer({
  open,
  onOpenChange,
  onCreated,
}: CreateMerchantDrawerProps) {
  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [phone, setPhone] = useState("");
  const [businessName, setBusinessName] = useState("");
  const [webhookURL, setWebhookURL] = useState("");
  const [submitting, setSubmitting] = useState(false);

  function resetForm() {
    setName("");
    setEmail("");
    setPhone("");
    setBusinessName("");
    setWebhookURL("");
  }

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();

    if (!name.trim() || !email.trim() || !businessName.trim()) {
      toast.error("Name, email, and business name are required.");
      return;
    }

    setSubmitting(true);
    try {
      await createAdminMerchant({
        name: name.trim(),
        email: email.trim(),
        phone: phone.trim() || undefined,
        business_name: businessName.trim(),
        webhook_url: webhookURL.trim() || undefined,
      });
      toast.success("Merchant created successfully.");
      resetForm();
      onOpenChange(false);
      onCreated();
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Failed to create merchant.",
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
            Add merchant
          </DrawerTitle>
          <DrawerDescription className="text-[13px] text-[#8a97a8]">
            Create a new merchant that can receive payments on the platform.
          </DrawerDescription>
        </DrawerHeader>

        <form
          id="create-merchant-form"
          onSubmit={handleSubmit}
          className="flex flex-1 flex-col overflow-y-auto px-4 py-4"
        >
          <div className="space-y-4">
            <div>
              <label
                htmlFor="merchant-business-name"
                className="mb-1.5 block text-[13px] font-medium text-[#3d4b5c]"
              >
                Business name <span className="text-[#e85d3b]">*</span>
              </label>
              <input
                id="merchant-business-name"
                value={businessName}
                onChange={(e) => setBusinessName(e.target.value)}
                placeholder="Demo Merchant Store"
                className="h-10 w-full rounded-lg border border-[#e8eef4] bg-[#f8fafc] px-3 text-[13.5px] text-[#1f2a37] outline-none focus:border-[#3b9eff] focus:bg-white focus:ring-2 focus:ring-[#3b9eff]/15"
              />
            </div>

            <div>
              <label
                htmlFor="merchant-owner-name"
                className="mb-1.5 block text-[13px] font-medium text-[#3d4b5c]"
              >
                Owner name <span className="text-[#e85d3b]">*</span>
              </label>
              <input
                id="merchant-owner-name"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="John Doe"
                className="h-10 w-full rounded-lg border border-[#e8eef4] bg-[#f8fafc] px-3 text-[13.5px] text-[#1f2a37] outline-none focus:border-[#3b9eff] focus:bg-white focus:ring-2 focus:ring-[#3b9eff]/15"
              />
            </div>

            <div>
              <label
                htmlFor="merchant-email"
                className="mb-1.5 block text-[13px] font-medium text-[#3d4b5c]"
              >
                Email <span className="text-[#e85d3b]">*</span>
              </label>
              <input
                id="merchant-email"
                type="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                placeholder="owner@business.com"
                className="h-10 w-full rounded-lg border border-[#e8eef4] bg-[#f8fafc] px-3 text-[13.5px] text-[#1f2a37] outline-none focus:border-[#3b9eff] focus:bg-white focus:ring-2 focus:ring-[#3b9eff]/15"
              />
            </div>

            <div>
              <label
                htmlFor="merchant-phone"
                className="mb-1.5 block text-[13px] font-medium text-[#3d4b5c]"
              >
                Phone
              </label>
              <input
                id="merchant-phone"
                value={phone}
                onChange={(e) => setPhone(e.target.value)}
                placeholder="081234567890"
                className="h-10 w-full rounded-lg border border-[#e8eef4] bg-[#f8fafc] px-3 text-[13.5px] text-[#1f2a37] outline-none focus:border-[#3b9eff] focus:bg-white focus:ring-2 focus:ring-[#3b9eff]/15"
              />
            </div>

            <div>
              <label
                htmlFor="merchant-webhook"
                className="mb-1.5 block text-[13px] font-medium text-[#3d4b5c]"
              >
                Webhook URL
              </label>
              <input
                id="merchant-webhook"
                value={webhookURL}
                onChange={(e) => setWebhookURL(e.target.value)}
                placeholder="https://merchant.example.com/webhooks"
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
            form="create-merchant-form"
            disabled={submitting}
            className="bg-[#ff5e16] text-white hover:bg-[#ef5510]"
          >
            {submitting ? (
              <>
                <ButtonSpinner />
                Creating...
              </>
            ) : (
              "Create merchant"
            )}
          </Button>
        </DrawerFooter>
      </DrawerContent>
    </Drawer>
  );
}
