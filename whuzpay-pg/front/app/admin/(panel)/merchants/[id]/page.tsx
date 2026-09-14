"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { useEffect, useState } from "react";
import toast from "react-hot-toast";
import { PencilIcon, PowerIcon } from "lucide-react";
import EditMerchantDrawer from "@/app/components/admin/EditMerchantDrawer";
import MerchantAPIKeysSection from "@/app/components/admin/MerchantAPIKeysSection";
import MerchantProviderConfigSection from "@/app/components/admin/MerchantProviderConfigSection";
import {
  fetchAdminMerchant,
  fetchAdminMerchantPayments,
  setAdminMerchantActive,
  type AdminMerchant,
  type AdminPayment,
} from "@/lib/admin-api";
import { Button } from "@/components/ui/button";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import {
  Card,
  EmptyState,
  formatDateTime,
  formatIDR,
  LoadingBlock,
  PageHeader,
  StatusBadge,
  TableShell,
} from "../../../../components/admin/ui";

export default function AdminMerchantDetailPage() {
  const params = useParams<{ id: string }>();
  const id = params?.id;
  const [merchant, setMerchant] = useState<AdminMerchant | null>(null);
  const [payments, setPayments] = useState<AdminPayment[]>([]);
  const [loading, setLoading] = useState(true);
  const [editOpen, setEditOpen] = useState(false);
  const [toggling, setToggling] = useState(false);

  useEffect(() => {
    if (!id) return;
    let cancelled = false;

    async function load() {
      setLoading(true);
      try {
        const [m, p] = await Promise.all([
          fetchAdminMerchant(id),
          fetchAdminMerchantPayments(id, { limit: 20, offset: 0 }),
        ]);
        if (cancelled) return;
        setMerchant(m);
        setPayments(p.items ?? []);
      } catch (err) {
        if (!cancelled) {
          toast.error(
            err instanceof Error ? err.message : "Failed to load merchant.",
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

  async function handleToggleActive() {
    if (!merchant) return;
    setToggling(true);
    try {
      const updated = await setAdminMerchantActive(
        merchant.id,
        !merchant.is_active,
      );
      setMerchant(updated);
      toast.success(
        updated.is_active ? "Merchant activated." : "Merchant deactivated.",
      );
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Failed to update status.",
      );
    } finally {
      setToggling(false);
    }
  }

  return (
    <div className="w-full space-y-5">
      <PageHeader
        title="Merchant detail"
        description="Merchant profile and related payments."
        actions={
          <Link
            href="/admin/merchants"
            className="text-[13px] font-semibold text-[#6b7c93] hover:text-[#06163a]"
          >
            ← Back to merchants
          </Link>
        }
      />

      {loading ? (
        <Card>
          <LoadingBlock />
        </Card>
      ) : !merchant ? (
        <Card className="p-6 text-[13px] text-[#8a97a8]">Merchant not found.</Card>
      ) : (
        <>
          <Card className="p-5 sm:p-6">
            <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
              <div>
                <h2 className="text-[18px] font-semibold text-[#1f2a37]">
                  {merchant.business_name}
                </h2>
                <p className="mt-1 text-[13px] text-[#6b7c93]">{merchant.name}</p>
              </div>
              <div className="flex flex-wrap items-center gap-2">
                <span
                  className={`inline-flex rounded-full px-2.5 py-0.5 text-[11px] font-semibold ${
                    merchant.is_active
                      ? "bg-[#e8f8ee] text-[#2f9e5a]"
                      : "bg-[#fff1ed] text-[#e85d3b]"
                  }`}
                >
                  {merchant.is_active ? "Active" : "Inactive"}
                </span>

                <TooltipProvider delayDuration={200}>
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <Button
                        type="button"
                        variant="outline"
                        size="icon"
                        className="size-9 rounded-full border-[#e8eef4] bg-white text-[#3d4b5c] shadow-none hover:bg-[#f4f7fb]"
                        aria-label="Edit merchant"
                        onClick={() => setEditOpen(true)}
                      >
                        <PencilIcon />
                      </Button>
                    </TooltipTrigger>
                    <TooltipContent side="bottom" sideOffset={6}>
                      Edit merchant
                    </TooltipContent>
                  </Tooltip>

                  <Tooltip>
                    <TooltipTrigger asChild>
                      <Button
                        type="button"
                        variant="outline"
                        size="icon"
                        disabled={toggling}
                        className={`size-9 rounded-full shadow-none ${
                          merchant.is_active
                            ? "border-[#ffd4c8] bg-white text-[#e85d3b] hover:bg-[#fff5f2]"
                            : "border-[#c8efd4] bg-white text-[#2f9e5a] hover:bg-[#f0faf3]"
                        }`}
                        aria-label={
                          merchant.is_active
                            ? "Deactivate merchant"
                            : "Activate merchant"
                        }
                        onClick={() => void handleToggleActive()}
                      >
                        <PowerIcon className={toggling ? "animate-pulse" : undefined} />
                      </Button>
                    </TooltipTrigger>
                    <TooltipContent side="bottom" sideOffset={6}>
                      {merchant.is_active ? "Deactivate" : "Activate"}
                    </TooltipContent>
                  </Tooltip>
                </TooltipProvider>
              </div>
            </div>

            <div className="mt-5 grid grid-cols-1 gap-3 sm:grid-cols-2">
              <div className="rounded-lg bg-[#f8fafc] px-4 py-3">
                <p className="text-[11px] font-semibold uppercase tracking-wide text-[#8a97a8]">
                  Email
                </p>
                <p className="mt-1 text-[13.5px] font-medium text-[#1f2a37]">
                  {merchant.email}
                </p>
              </div>
              <div className="rounded-lg bg-[#f8fafc] px-4 py-3">
                <p className="text-[11px] font-semibold uppercase tracking-wide text-[#8a97a8]">
                  Phone
                </p>
                <p className="mt-1 text-[13.5px] font-medium text-[#1f2a37]">
                  {merchant.phone || "—"}
                </p>
              </div>
              <div className="rounded-lg bg-[#f8fafc] px-4 py-3 sm:col-span-2">
                <p className="text-[11px] font-semibold uppercase tracking-wide text-[#8a97a8]">
                  Webhook URL
                </p>
                <p className="mt-1 break-all text-[13.5px] font-medium text-[#1f2a37]">
                  {merchant.webhook_url || "—"}
                </p>
              </div>
              <div className="rounded-lg bg-[#f8fafc] px-4 py-3">
                <p className="text-[11px] font-semibold uppercase tracking-wide text-[#8a97a8]">
                  Created
                </p>
                <p className="mt-1 text-[13.5px] font-medium text-[#1f2a37]">
                  {formatDateTime(merchant.created_at)}
                </p>
              </div>
              <div className="rounded-lg bg-[#f8fafc] px-4 py-3">
                <p className="text-[11px] font-semibold uppercase tracking-wide text-[#8a97a8]">
                  Updated
                </p>
                <p className="mt-1 text-[13.5px] font-medium text-[#1f2a37]">
                  {formatDateTime(merchant.updated_at)}
                </p>
              </div>
            </div>
          </Card>

          <MerchantAPIKeysSection merchantId={merchant.id} />

          <MerchantProviderConfigSection merchantId={merchant.id} />

          <Card>
            <div className="border-b border-[#eef2f6] px-5 py-4">
              <h3 className="text-[15px] font-semibold text-[#1f2a37]">
                Recent payments
              </h3>
            </div>
            {payments.length === 0 ? (
              <EmptyState message="No payments for this merchant." />
            ) : (
              <TableShell>
                <table className="w-full min-w-[720px] text-left text-[12.5px]">
                  <thead>
                    <tr className="border-b border-[#eef2f6] text-[11px] font-semibold uppercase tracking-wide text-[#8a97a8]">
                      <th className="px-5 py-3">Reference</th>
                      <th className="px-5 py-3">Amount</th>
                      <th className="px-5 py-3">Status</th>
                      <th className="px-5 py-3">Created</th>
                    </tr>
                  </thead>
                  <tbody>
                    {payments.map((p) => (
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
                        </td>
                        <td className="px-5 py-3.5">
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

          <EditMerchantDrawer
            open={editOpen}
            merchant={merchant}
            onOpenChange={setEditOpen}
            onUpdated={setMerchant}
          />
        </>
      )}
    </div>
  );
}
