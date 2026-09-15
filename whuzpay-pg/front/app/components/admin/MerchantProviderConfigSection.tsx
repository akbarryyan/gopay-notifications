"use client";

import { useEffect, useState } from "react";
import toast from "react-hot-toast";
import { PlusIcon, Trash2Icon } from "lucide-react";
import {
  deleteAdminMerchantProviderConfig,
  fetchAdminMerchantProviderConfigs,
  fetchAdminProviders,
  upsertAdminMerchantProviderConfig,
  type AdminMerchantProviderConfig,
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
  LoadingBlock,
  TableShell,
} from "./ui";

type Props = {
  merchantId: string;
};

const emptyForm = {
  provider_name: "gopay",
  payment_method: "qris",
  priority: 1,
  weight: 100,
  failover_enabled: true,
  is_enabled: true,
};

export default function MerchantProviderConfigSection({ merchantId }: Props) {
  const [items, setItems] = useState<AdminMerchantProviderConfig[]>([]);
  const [providerNames, setProviderNames] = useState<string[]>(["gopay"]);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [showForm, setShowForm] = useState(false);
  const [form, setForm] = useState(emptyForm);

  async function load() {
    setLoading(true);
    try {
      const [configs, providers] = await Promise.all([
        fetchAdminMerchantProviderConfigs(merchantId),
        fetchAdminProviders().catch(() => ({ items: [] as { name: string }[] })),
      ]);
      setItems(configs.items ?? []);
      const names = (providers.items ?? []).map((p) => p.name);
      if (names.length > 0) {
        setProviderNames(names);
        setForm((f) =>
          names.includes(f.provider_name)
            ? f
            : { ...f, provider_name: names[0] },
        );
      }
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Failed to load provider configs.",
      );
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    // `load` synchronously calls setLoading(true) before its first await —
    // redundant here since `loading` already starts true, but real for the
    // manual refresh call sites below. The linter can't see the await
    // boundary through the function call.
    // eslint-disable-next-line react-hooks/set-state-in-effect
    void load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [merchantId]);

  async function handleSave(e: React.FormEvent) {
    e.preventDefault();
    setSaving(true);
    try {
      const next = await upsertAdminMerchantProviderConfig(merchantId, {
        provider_name: form.provider_name.trim(),
        payment_method: form.payment_method.trim().toLowerCase(),
        priority: Number(form.priority) || 1,
        weight: Number(form.weight) || 100,
        failover_enabled: form.failover_enabled,
        is_enabled: form.is_enabled,
      });
      setItems(next);
      setShowForm(false);
      toast.success("Provider config saved.");
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Failed to save provider config.",
      );
    } finally {
      setSaving(false);
    }
  }

  async function handleDelete(row: AdminMerchantProviderConfig) {
    if (
      !window.confirm(
        `Remove ${row.provider_name} / ${row.payment_method} from this merchant?`,
      )
    ) {
      return;
    }
    try {
      await deleteAdminMerchantProviderConfig(
        merchantId,
        row.provider_name,
        row.payment_method,
      );
      setItems((prev) =>
        prev.filter(
          (x) =>
            !(
              x.provider_name === row.provider_name &&
              x.payment_method === row.payment_method
            ),
        ),
      );
      toast.success("Provider config removed.");
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Failed to delete provider config.",
      );
    }
  }

  function startEdit(row: AdminMerchantProviderConfig) {
    setForm({
      provider_name: row.provider_name,
      payment_method: row.payment_method,
      priority: row.priority,
      weight: row.weight,
      failover_enabled: row.failover_enabled,
      is_enabled: row.is_enabled,
    });
    setShowForm(true);
  }

  return (
    <Card>
      <div className="flex flex-col gap-3 border-b border-[#eef2f6] px-5 py-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h3 className="text-[15px] font-semibold text-[#1f2a37]">
            Provider routing
          </h3>
          <p className="mt-0.5 text-[12.5px] text-[#8a97a8]">
            Priority (lower first), weight, and failover for this merchant.
          </p>
        </div>
        <Button
          type="button"
          size="sm"
          className="h-9 rounded-full bg-[#ff5e16] px-4 text-[12.5px] font-semibold text-white shadow-none hover:bg-[#e8530f]"
          onClick={() => {
            setForm({
              ...emptyForm,
              provider_name: providerNames[0] ?? "gopay",
            });
            setShowForm((v) => !v);
          }}
        >
          <PlusIcon className="size-4" />
          {showForm ? "Close form" : "Add / update"}
        </Button>
      </div>

      {showForm && (
        <form
          onSubmit={(e) => void handleSave(e)}
          className="grid grid-cols-1 gap-3 border-b border-[#eef2f6] bg-[#fafbfc] px-5 py-4 sm:grid-cols-2 lg:grid-cols-3"
        >
          <label className="flex flex-col gap-1.5">
            <span className="text-[11px] font-semibold uppercase tracking-wide text-[#8a97a8]">
              Provider
            </span>
            <select
              value={form.provider_name}
              onChange={(e) =>
                setForm((f) => ({ ...f, provider_name: e.target.value }))
              }
              className="h-9 rounded-lg border border-[#e8eef4] bg-white px-3 text-[13px] text-[#1f2a37] outline-none focus:border-[#3b9eff] focus:ring-2 focus:ring-[#3b9eff]/15"
            >
              {providerNames.map((n) => (
                <option key={n} value={n}>
                  {n}
                </option>
              ))}
              {!providerNames.includes(form.provider_name) && (
                <option value={form.provider_name}>{form.provider_name}</option>
              )}
            </select>
          </label>

          <label className="flex flex-col gap-1.5">
            <span className="text-[11px] font-semibold uppercase tracking-wide text-[#8a97a8]">
              Payment method
            </span>
            <input
              value={form.payment_method}
              onChange={(e) =>
                setForm((f) => ({ ...f, payment_method: e.target.value }))
              }
              className="h-9 rounded-lg border border-[#e8eef4] bg-white px-3 text-[13px] text-[#1f2a37] outline-none focus:border-[#3b9eff] focus:ring-2 focus:ring-[#3b9eff]/15"
              placeholder="qris"
              required
            />
          </label>

          <label className="flex flex-col gap-1.5">
            <span className="text-[11px] font-semibold uppercase tracking-wide text-[#8a97a8]">
              Priority
            </span>
            <input
              type="number"
              min={1}
              value={form.priority}
              onChange={(e) =>
                setForm((f) => ({ ...f, priority: Number(e.target.value) }))
              }
              className="h-9 rounded-lg border border-[#e8eef4] bg-white px-3 text-[13px] text-[#1f2a37] outline-none focus:border-[#3b9eff] focus:ring-2 focus:ring-[#3b9eff]/15"
            />
          </label>

          <label className="flex flex-col gap-1.5">
            <span className="text-[11px] font-semibold uppercase tracking-wide text-[#8a97a8]">
              Weight
            </span>
            <input
              type="number"
              min={1}
              value={form.weight}
              onChange={(e) =>
                setForm((f) => ({ ...f, weight: Number(e.target.value) }))
              }
              className="h-9 rounded-lg border border-[#e8eef4] bg-white px-3 text-[13px] text-[#1f2a37] outline-none focus:border-[#3b9eff] focus:ring-2 focus:ring-[#3b9eff]/15"
            />
          </label>

          <label className="flex items-center gap-2 pt-5 text-[13px] text-[#3d4b5c]">
            <input
              type="checkbox"
              checked={form.failover_enabled}
              onChange={(e) =>
                setForm((f) => ({
                  ...f,
                  failover_enabled: e.target.checked,
                }))
              }
              className="size-4 rounded border-[#d0d7e2]"
            />
            Failover enabled
          </label>

          <label className="flex items-center gap-2 pt-5 text-[13px] text-[#3d4b5c]">
            <input
              type="checkbox"
              checked={form.is_enabled}
              onChange={(e) =>
                setForm((f) => ({ ...f, is_enabled: e.target.checked }))
              }
              className="size-4 rounded border-[#d0d7e2]"
            />
            Enabled
          </label>

          <div className="flex items-end sm:col-span-2 lg:col-span-3">
            <Button
              type="submit"
              disabled={saving}
              className="h-9 rounded-full bg-[#06163a] px-5 text-[12.5px] font-semibold text-white shadow-none hover:bg-[#0b2048]"
            >
              {saving ? "Saving..." : "Save config"}
            </Button>
          </div>
        </form>
      )}

      {loading ? (
        <LoadingBlock />
      ) : items.length === 0 ? (
        <EmptyState message="No provider configs for this merchant." />
      ) : (
        <TableShell>
          <table className="w-full min-w-[800px] text-left text-[12.5px]">
            <thead>
              <tr className="border-b border-[#eef2f6] text-[11px] font-semibold uppercase tracking-wide text-[#8a97a8]">
                <th className="px-5 py-3">Provider</th>
                <th className="px-5 py-3">Method</th>
                <th className="px-5 py-3">Priority</th>
                <th className="px-5 py-3">Weight</th>
                <th className="px-5 py-3">Failover</th>
                <th className="px-5 py-3">Status</th>
                <th className="px-5 py-3">Updated</th>
                <th className="px-5 py-3 text-right">Actions</th>
              </tr>
            </thead>
            <tbody>
              {items.map((row) => (
                <tr
                  key={row.id}
                  className="border-b border-[#f3f6f9] text-[#3d4b5c] last:border-0"
                >
                  <td className="px-5 py-3.5 font-medium capitalize text-[#1f2a37]">
                    {row.provider_name}
                  </td>
                  <td className="px-5 py-3.5 capitalize">{row.payment_method}</td>
                  <td className="px-5 py-3.5">{row.priority}</td>
                  <td className="px-5 py-3.5">{row.weight}</td>
                  <td className="px-5 py-3.5">
                    {row.failover_enabled ? "Yes" : "No"}
                  </td>
                  <td className="px-5 py-3.5">
                    <span
                      className={`inline-flex rounded-full px-2.5 py-0.5 text-[11px] font-semibold ${
                        row.is_enabled
                          ? "bg-[#e8f8ee] text-[#2f9e5a]"
                          : "bg-[#fff1ed] text-[#e85d3b]"
                      }`}
                    >
                      {row.is_enabled ? "Enabled" : "Disabled"}
                    </span>
                  </td>
                  <td className="px-5 py-3.5 text-[#6b7c93]">
                    {formatDateTime(row.updated_at)}
                  </td>
                  <td className="px-5 py-3.5">
                    <div className="flex items-center justify-end gap-1.5">
                      <button
                        type="button"
                        className="rounded-full px-3 py-1 text-[12px] font-semibold text-[#ff5e16] hover:bg-[#fff5f0]"
                        onClick={() => startEdit(row)}
                      >
                        Edit
                      </button>
                      <TooltipProvider delayDuration={200}>
                        <Tooltip>
                          <TooltipTrigger asChild>
                            <Button
                              type="button"
                              variant="outline"
                              size="icon"
                              className="size-8 rounded-full border-[#ffd4c8] bg-white text-[#e85d3b] shadow-none hover:bg-[#fff5f2]"
                              aria-label="Delete config"
                              onClick={() => void handleDelete(row)}
                            >
                              <Trash2Icon className="size-3.5" />
                            </Button>
                          </TooltipTrigger>
                          <TooltipContent side="bottom" sideOffset={6}>
                            Delete
                          </TooltipContent>
                        </Tooltip>
                      </TooltipProvider>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </TableShell>
      )}
    </Card>
  );
}
