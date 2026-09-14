"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { useEffect, useState, type ReactNode } from "react";
import toast from "react-hot-toast";
import { CopyIcon, ExternalLinkIcon } from "lucide-react";
import {
  fetchMerchantPaymentLink,
  fetchMerchantPaymentLinkPayments,
  setMerchantPaymentLinkActive,
  updateMerchantPaymentLink,
  type MerchantPayment,
  type MerchantPaymentLink,
} from "@/lib/merchant-api";
import { Button } from "@/components/ui/button";
import { DatePicker } from "@/components/ui/date-picker";
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
  EmptyState,
  formatDateTime,
  formatIDR,
  LoadingBlock,
  PageHeader,
  StatusBadge,
  TableShell,
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

function toYmd(iso?: string | null) {
  if (!iso) return "";
  return iso.slice(0, 10);
}

export default function PaymentLinkDetailPage() {
  const params = useParams<{ id: string }>();
  const id = params?.id;

  const [link, setLink] = useState<MerchantPaymentLink | null>(null);
  const [loading, setLoading] = useState(true);
  const [payments, setPayments] = useState<MerchantPayment[]>([]);
  const [paymentsTotal, setPaymentsTotal] = useState(0);
  const [paymentsLoading, setPaymentsLoading] = useState(true);

  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [expiresAt, setExpiresAt] = useState("");
  const [saving, setSaving] = useState(false);
  const [confirmOpen, setConfirmOpen] = useState(false);
  const [togglingActive, setTogglingActive] = useState(false);

  useEffect(() => {
    if (!id) return;
    let cancelled = false;
    async function load() {
      setLoading(true);
      try {
        const data = await fetchMerchantPaymentLink(id);
        if (cancelled) return;
        setLink(data);
        setTitle(data.title);
        setDescription(data.description ?? "");
        setExpiresAt(toYmd(data.expires_at));
      } catch (err) {
        if (!cancelled) {
          toast.error(
            err instanceof Error ? err.message : "Failed to load payment link.",
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

  useEffect(() => {
    if (!id) return;
    let cancelled = false;
    async function loadPayments() {
      setPaymentsLoading(true);
      try {
        const data = await fetchMerchantPaymentLinkPayments(id, { limit: 20 });
        if (cancelled) return;
        setPayments(data.items ?? []);
        setPaymentsTotal(data.total ?? 0);
      } catch (err) {
        if (!cancelled) {
          toast.error(
            err instanceof Error ? err.message : "Failed to load payment history.",
          );
        }
      } finally {
        if (!cancelled) setPaymentsLoading(false);
      }
    }
    void loadPayments();
    return () => {
      cancelled = true;
    };
  }, [id]);

  async function copy(text: string, label: string) {
    try {
      await navigator.clipboard.writeText(text);
      toast.success(`${label} copied.`);
    } catch {
      toast.error("Failed to copy.");
    }
  }

  async function handleSave() {
    if (!link) return;
    if (!title.trim()) {
      toast.error("Title is required.");
      return;
    }
    setSaving(true);
    try {
      const updated = await updateMerchantPaymentLink(link.id, {
        title: title.trim(),
        description: description.trim() || undefined,
        expires_at: expiresAt ? new Date(expiresAt).toISOString() : null,
      });
      setLink(updated);
      toast.success("Payment link updated.");
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to save changes.");
    } finally {
      setSaving(false);
    }
  }

  async function handleToggleActive(nextActive: boolean) {
    if (!link) return;
    setTogglingActive(true);
    try {
      await setMerchantPaymentLinkActive(link.id, nextActive);
      setLink({ ...link, is_active: nextActive });
      toast.success(nextActive ? "Link activated." : "Link deactivated.");
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to update status.");
    } finally {
      setTogglingActive(false);
      setConfirmOpen(false);
    }
  }

  const publicPath = link ? `/l/${link.slug}` : "";
  const publicUrl =
    typeof window !== "undefined" && link
      ? `${window.location.origin}${publicPath}`
      : link?.public_url ?? "";

  return (
    <div className="w-full space-y-5">
      <PageHeader
        title="Payment link detail"
        description="Reusable checkout URL — edit its details or review payments spawned from it."
        actions={
          <Link
            href="/dashboard/payment-links"
            className="text-[13px] font-semibold text-[#6b7c93] hover:text-[#06163a]"
          >
            ← Back to payment links
          </Link>
        }
      />

      {loading ? (
        <Card>
          <LoadingBlock />
        </Card>
      ) : !link ? (
        <Card className="p-6 text-[13px] text-[#8a97a8]">
          Payment link not found.
        </Card>
      ) : (
        <>
          <Card className="p-5 sm:p-6">
            <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
              <div>
                <p className="text-[12px] text-[#8a97a8]">Slug</p>
                <h2 className="mt-1 text-[18px] font-semibold text-[#1f2a37]">
                  {link.title}
                </h2>
                <p className="mt-1 font-mono text-[12.5px] text-[#6b7c93]">
                  /{link.slug}
                </p>
              </div>
              <span
                className={`inline-flex rounded-full px-3 py-1 text-[11px] font-semibold ${
                  link.is_active
                    ? "bg-[#e8f8ee] text-[#2f9e5a]"
                    : "bg-[#eef2f6] text-[#6b7c93]"
                }`}
              >
                {link.is_active ? "Active" : "Inactive"}
              </span>
            </div>
            <div className="mt-5 grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-3">
              <Field
                label="Type"
                value={<span className="capitalize">{link.amount_type}</span>}
              />
              <Field
                label="Amount"
                value={
                  link.amount_type === "fixed" && link.amount
                    ? formatIDR(link.amount, link.currency)
                    : `Customer sets amount${
                        link.min_amount || link.max_amount
                          ? ` (${
                              link.min_amount
                                ? formatIDR(link.min_amount, link.currency)
                                : "no min"
                            } – ${
                              link.max_amount
                                ? formatIDR(link.max_amount, link.currency)
                                : "no max"
                            })`
                          : ""
                      }`
                }
              />
              <Field
                label="Environment"
                value={<span className="capitalize">{link.environment}</span>}
              />
              <Field label="Created" value={formatDateTime(link.created_at)} />
              <Field label="Updated" value={formatDateTime(link.updated_at)} />
              <Field
                label="Expires"
                value={link.expires_at ? formatDateTime(link.expires_at) : "Never"}
              />
            </div>
          </Card>

          <Card className="p-5 sm:p-6">
            <h3 className="text-[15px] font-semibold text-[#1f2a37]">
              Shareable link
            </h3>
            <p className="mt-1 text-[12.5px] text-[#8a97a8]">
              Anyone with this URL can pay through this link while it is active.
            </p>
            <div className="mt-3 flex flex-col gap-2 sm:flex-row sm:items-center">
              <code className="flex-1 break-all rounded-lg border border-[#e8eef4] bg-[#f8fafc] px-3 py-2 text-[12.5px] text-[#1f2a37]">
                {publicUrl}
              </code>
              <Button
                type="button"
                variant="outline"
                className="h-9 shrink-0 rounded-full border-[#e8eef4] shadow-none"
                onClick={() => void copy(publicUrl, "Link")}
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
                <a href={publicPath} target="_blank" rel="noreferrer">
                  <ExternalLinkIcon className="size-3.5" />
                  Open
                </a>
              </Button>
            </div>
          </Card>

          <Card className="p-5 sm:p-6">
            <h3 className="text-[15px] font-semibold text-[#1f2a37]">Edit details</h3>
            <div className="mt-4 space-y-4">
              <div className="space-y-1.5">
                <label className="text-[11px] font-semibold uppercase tracking-wide text-[#8a97a8]">
                  Title
                </label>
                <input
                  className="h-10 w-full rounded-lg border border-[#e8eef4] bg-[#f8fafc] px-3 text-[13.5px] text-[#1f2a37] outline-none focus:border-[#ff5e16] focus:bg-white"
                  value={title}
                  onChange={(e) => setTitle(e.target.value)}
                  maxLength={150}
                />
              </div>
              <div className="space-y-1.5">
                <label className="text-[11px] font-semibold uppercase tracking-wide text-[#8a97a8]">
                  Description
                </label>
                <textarea
                  className="h-20 w-full resize-none rounded-lg border border-[#e8eef4] bg-[#f8fafc] px-3 py-2 text-[13.5px] text-[#1f2a37] outline-none focus:border-[#ff5e16] focus:bg-white"
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                  maxLength={500}
                />
              </div>
              <DatePicker
                label="Expires on"
                value={expiresAt}
                onChange={setExpiresAt}
                placeholder="No expiry"
              />
              <div className="flex flex-wrap items-center gap-2 pt-1">
                <Button
                  type="button"
                  disabled={saving}
                  className="h-10 rounded-full bg-[#ff5e16] px-5 text-[13px] font-semibold text-white hover:bg-[#ef5510]"
                  onClick={() => void handleSave()}
                >
                  {saving ? "Saving..." : "Save changes"}
                </Button>
                {link.is_active ? (
                  <Button
                    type="button"
                    variant="outline"
                    className="h-10 rounded-full border-[#e8eef4] px-5 text-[13px] text-[#c0442a] shadow-none hover:bg-[#fdf0ed] hover:text-[#c0442a]"
                    onClick={() => setConfirmOpen(true)}
                  >
                    Deactivate link
                  </Button>
                ) : (
                  <Button
                    type="button"
                    variant="outline"
                    disabled={togglingActive}
                    className="h-10 rounded-full border-[#e8eef4] px-5 text-[13px] shadow-none"
                    onClick={() => void handleToggleActive(true)}
                  >
                    {togglingActive ? "Activating..." : "Activate link"}
                  </Button>
                )}
              </div>
            </div>
          </Card>

          <Card>
            <div className="border-b border-[#eef2f6] px-5 py-4">
              <h3 className="text-[15px] font-semibold text-[#1f2a37]">
                Payments from this link
              </h3>
              <p className="mt-0.5 text-[12.5px] text-[#8a97a8]">
                {paymentsLoading
                  ? "Loading..."
                  : `${paymentsTotal} payment(s) spawned from this link`}
              </p>
            </div>
            {paymentsLoading ? (
              <LoadingBlock />
            ) : payments.length === 0 ? (
              <EmptyState message="No payments through this link yet." />
            ) : (
              <TableShell>
                <table className="w-full min-w-175 text-left text-[12.5px]">
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
                            href={`/dashboard/payments/${p.id}`}
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
        </>
      )}

      <AlertDialog open={confirmOpen} onOpenChange={setConfirmOpen}>
        <AlertDialogContent className="max-w-sm border border-[#e8eef4] bg-white">
          <AlertDialogHeader>
            <AlertDialogTitle>Deactivate this link?</AlertDialogTitle>
            <AlertDialogDescription>
              Customers won&apos;t be able to pay through it anymore. Payments
              already made through this link are unaffected and stay in your
              history.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={togglingActive}>
              Cancel
            </AlertDialogCancel>
            <AlertDialogAction
              disabled={togglingActive}
              className="bg-[#e85d3b]! text-white! hover:bg-[#d64f30]!"
              onClick={(e) => {
                e.preventDefault();
                void handleToggleActive(false);
              }}
            >
              {togglingActive ? "Deactivating..." : "Deactivate"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
