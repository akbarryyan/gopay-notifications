"use client";

import { useEffect, useState, type FormEvent } from "react";
import toast from "react-hot-toast";
import { CopyIcon, EyeIcon, EyeOffIcon, RefreshCwIcon } from "lucide-react";
import {
  fetchMerchantBusiness,
  fetchMerchantWebhookSecret,
  regenerateMerchantWebhookSecret,
  updateMerchantBusiness,
  type MerchantBusiness,
} from "@/lib/merchant-api";
import { Button } from "@/components/ui/button";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import {
  Card,
  LoadingBlock,
  PageHeader,
} from "../../components/admin/ui";

function WebhookSecretCard() {
  const [secret, setSecret] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [revealed, setRevealed] = useState(false);
  const [confirmOpen, setConfirmOpen] = useState(false);
  const [regenerating, setRegenerating] = useState(false);

  useEffect(() => {
    let cancelled = false;
    async function load() {
      setLoading(true);
      try {
        const data = await fetchMerchantWebhookSecret();
        if (!cancelled) setSecret(data.webhook_secret);
      } catch (err) {
        if (!cancelled) {
          toast.error(
            err instanceof Error
              ? err.message
              : "Failed to load webhook secret.",
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
  }, []);

  async function copySecret() {
    if (!secret) return;
    try {
      await navigator.clipboard.writeText(secret);
      toast.success("Copied.");
    } catch {
      toast.error("Failed to copy.");
    }
  }

  async function handleRegenerate() {
    setRegenerating(true);
    try {
      const data = await regenerateMerchantWebhookSecret();
      setSecret(data.webhook_secret);
      setRevealed(true);
      setConfirmOpen(false);
      toast.success("Webhook secret regenerated.");
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Failed to regenerate secret.",
      );
    } finally {
      setRegenerating(false);
    }
  }

  return (
    <Card className="p-5 sm:p-6">
      <h2 className="text-[14px] font-semibold text-[#1f2a37]">
        Webhook signing secret
      </h2>
      <p className="mt-1 text-[12.5px] leading-relaxed text-[#6b7c93]">
        Used to verify the <code className="font-mono">X-PG-Signature</code>{" "}
        header on payment webhooks sent to your URL above — see{" "}
        <a href="/dashboard/api-docs#webhooks" className="text-[#3b9eff] hover:underline">
          API docs
        </a>{" "}
        for the verification example.
      </p>

      {loading ? (
        <div className="mt-3 text-[12.5px] text-[#8a97a8]">Loading...</div>
      ) : secret ? (
        <>
          <div className="mt-3 flex items-center gap-2">
            <code className="flex-1 truncate rounded-lg border border-[#e8eef4] bg-[#f8fafc] px-3 py-2 text-[12.5px] text-[#1f2a37]">
              {revealed ? secret : "•".repeat(40)}
            </code>
            <Button
              type="button"
              variant="outline"
              size="icon"
              className="size-9 shrink-0 rounded-full border-[#e8eef4] bg-white text-[#3d4b5c] shadow-none hover:bg-[#f4f7fb]"
              aria-label={revealed ? "Hide secret" : "Show secret"}
              onClick={() => setRevealed((v) => !v)}
            >
              {revealed ? <EyeOffIcon /> : <EyeIcon />}
            </Button>
            <Button
              type="button"
              variant="outline"
              size="icon"
              className="size-9 shrink-0 rounded-full border-[#e8eef4] bg-white text-[#3d4b5c] shadow-none hover:bg-[#f4f7fb]"
              aria-label="Copy secret"
              onClick={() => void copySecret()}
            >
              <CopyIcon />
            </Button>
          </div>
          <button
            type="button"
            onClick={() => setConfirmOpen(true)}
            className="mt-3 flex items-center gap-1.5 text-[12.5px] font-semibold text-[#e85d3b] hover:underline"
          >
            <RefreshCwIcon className="size-3.5" />
            Regenerate secret
          </button>
        </>
      ) : (
        <p className="mt-3 text-[12.5px] text-[#8a97a8]">
          Failed to load. Refresh the page to try again.
        </p>
      )}

      <AlertDialog open={confirmOpen} onOpenChange={setConfirmOpen}>
        <AlertDialogContent className="max-w-sm border border-[#e8eef4] bg-white">
          <AlertDialogHeader>
            <AlertDialogTitle>Regenerate webhook secret?</AlertDialogTitle>
            <AlertDialogDescription>
              The current secret stops working immediately. Any webhook
              verification code still using it will start rejecting valid
              deliveries until you update it with the new one.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={regenerating}>
              Cancel
            </AlertDialogCancel>
            <AlertDialogAction
              disabled={regenerating}
              className="bg-[#e85d3b]! text-white! hover:bg-[#d64f30]!"
              onClick={(e) => {
                e.preventDefault();
                void handleRegenerate();
              }}
            >
              {regenerating ? "Regenerating..." : "Regenerate"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </Card>
  );
}

export default function MerchantSettingsPage() {
  const [business, setBusiness] = useState<MerchantBusiness | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [name, setName] = useState("");
  const [businessName, setBusinessName] = useState("");
  const [phone, setPhone] = useState("");
  const [webhookURL, setWebhookURL] = useState("");

  useEffect(() => {
    let cancelled = false;
    async function load() {
      setLoading(true);
      try {
        const data = await fetchMerchantBusiness();
        if (cancelled) return;
        setBusiness(data);
        setName(data.name ?? "");
        setBusinessName(data.business_name ?? "");
        setPhone(data.phone ?? "");
        setWebhookURL(data.webhook_url ?? "");
      } catch (err) {
        if (!cancelled) {
          toast.error(
            err instanceof Error ? err.message : "Failed to load settings.",
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
  }, []);

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setSaving(true);
    try {
      const updated = await updateMerchantBusiness({
        name: name.trim(),
        business_name: businessName.trim(),
        phone: phone.trim(),
        webhook_url: webhookURL.trim() || null,
      });
      setBusiness(updated);
      toast.success("Business settings saved.");
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Failed to save settings.",
      );
    } finally {
      setSaving(false);
    }
  }

  return (
    <div className="w-full space-y-5">
      <PageHeader
        title="Settings"
        description="Business profile and merchant webhook URL for payment notifications."
      />

      {loading ? (
        <Card>
          <LoadingBlock />
        </Card>
      ) : (
        <Card className="p-5 sm:p-6">
          <form onSubmit={(e) => void handleSubmit(e)} className="space-y-4">
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
              <label className="flex flex-col gap-1.5">
                <span className="text-[11px] font-semibold uppercase tracking-wide text-[#8a97a8]">
                  Contact name
                </span>
                <input
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  className="h-10 rounded-lg border border-[#e8eef4] px-3 text-[13px] outline-none focus:border-[#3b9eff] focus:ring-2 focus:ring-[#3b9eff]/15"
                  required
                />
              </label>
              <label className="flex flex-col gap-1.5">
                <span className="text-[11px] font-semibold uppercase tracking-wide text-[#8a97a8]">
                  Business name
                </span>
                <input
                  value={businessName}
                  onChange={(e) => setBusinessName(e.target.value)}
                  className="h-10 rounded-lg border border-[#e8eef4] px-3 text-[13px] outline-none focus:border-[#3b9eff] focus:ring-2 focus:ring-[#3b9eff]/15"
                  required
                />
              </label>
              <label className="flex flex-col gap-1.5">
                <span className="text-[11px] font-semibold uppercase tracking-wide text-[#8a97a8]">
                  Phone
                </span>
                <input
                  value={phone}
                  onChange={(e) => setPhone(e.target.value)}
                  className="h-10 rounded-lg border border-[#e8eef4] px-3 text-[13px] outline-none focus:border-[#3b9eff] focus:ring-2 focus:ring-[#3b9eff]/15"
                />
              </label>
              <label className="flex flex-col gap-1.5 sm:col-span-2">
                <span className="text-[11px] font-semibold uppercase tracking-wide text-[#8a97a8]">
                  Webhook URL
                </span>
                <input
                  value={webhookURL}
                  onChange={(e) => setWebhookURL(e.target.value)}
                  placeholder="https://your-store.com/webhooks/payments"
                  className="h-10 rounded-lg border border-[#e8eef4] px-3 text-[13px] outline-none focus:border-[#3b9eff] focus:ring-2 focus:ring-[#3b9eff]/15"
                />
              </label>
            </div>
            {business && (
              <p className="text-[12.5px] text-[#8a97a8]">
                Login email (account):{" "}
                <span className="font-medium text-[#3d4b5c]">
                  {business.email}
                </span>
              </p>
            )}
            <Button
              type="submit"
              disabled={saving}
              className="h-10 rounded-full bg-[#06163a] px-5 text-[13px] font-semibold text-white shadow-none hover:bg-[#0b2048]"
            >
              {saving ? "Saving..." : "Save settings"}
            </Button>
          </form>
        </Card>
      )}

      <WebhookSecretCard />
    </div>
  );
}
