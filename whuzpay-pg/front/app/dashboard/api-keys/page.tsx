"use client";

import { useEffect, useMemo, useState } from "react";
import toast from "react-hot-toast";
import { CopyIcon, RefreshCwIcon } from "lucide-react";
import FilterDropdown from "@/app/components/admin/FilterDropdown";
import {
  deleteMerchantAPIKey,
  fetchMerchantAPIKeys,
  upsertMerchantAPIKey,
  type MerchantAPIKey,
} from "@/lib/merchant-api";
import { API_URL as API_BASE } from "@/lib/config";
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
  EmptyState,
  formatDateTime,
  LoadingBlock,
  PageHeader,
  TableShell,
} from "../../components/admin/ui";

const KEY_ENV_OPTIONS = [
  { value: "sandbox", label: "Sandbox" },
  { value: "production", label: "Production" },
];

export default function MerchantAPIKeysPage() {
  const [items, setItems] = useState<MerchantAPIKey[]>([]);
  const [loading, setLoading] = useState(true);
  const [environment, setEnvironment] = useState("sandbox");
  const [password, setPassword] = useState("");
  const [saving, setSaving] = useState(false);
  const [revealedSecret, setRevealedSecret] = useState<string | null>(null);
  const [hint, setHint] = useState<string | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<MerchantAPIKey | null>(null);
  const [deletePassword, setDeletePassword] = useState("");
  const [deleting, setDeleting] = useState(false);

  const existing = useMemo(
    () => items.find((k) => k.name === environment && k.is_active),
    [items, environment],
  );

  async function load() {
    setLoading(true);
    try {
      const data = await fetchMerchantAPIKeys();
      setItems(data.items ?? []);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to load keys.");
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
  }, []);

  async function handleUpdate() {
    if (!password.trim()) {
      toast.error("Password is required.");
      return;
    }
    setSaving(true);
    try {
      const result = await upsertMerchantAPIKey({
        environment,
        password: password.trim(),
      });
      setRevealedSecret(result.secret);
      setHint(result.hint);
      setPassword("");
      setItems((prev) => {
        const rest = prev.filter((k) => k.name !== result.environment);
        return [result.key, ...rest];
      });
      toast.success(
        result.rotated ? "API key rotated." : "API key created.",
      );
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to update key.");
    } finally {
      setSaving(false);
    }
  }

  async function handleDelete() {
    if (!deleteTarget || !deletePassword.trim()) return;
    setDeleting(true);
    try {
      await deleteMerchantAPIKey(deleteTarget.id, deletePassword.trim());
      setItems((prev) => prev.filter((k) => k.id !== deleteTarget.id));
      toast.success("API key deleted.");
      setDeleteTarget(null);
      setDeletePassword("");
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to delete key.");
    } finally {
      setDeleting(false);
    }
  }

  async function copyText(text: string, label: string) {
    try {
      await navigator.clipboard.writeText(text);
      toast.success(`${label} copied.`);
    } catch {
      toast.error("Failed to copy.");
    }
  }

  const exampleCurl = revealedSecret
    ? `curl -X POST ${API_BASE}/api/v1/payments \\\n  -H "Authorization: Bearer ${revealedSecret}" \\\n  -H "Content-Type: application/json" \\\n  -d '{"amount":10000,"currency":"IDR","payment_method":"qris","description":"Test payment"}'`
    : "";

  return (
    <div className="w-full space-y-5">
      <PageHeader
        title="API keys"
        description="One key per environment. Update rotates the secret — confirm with your account password."
      />

      <Card>
        <div className="grid grid-cols-1 gap-3 border-b border-[#eef2f6] bg-[#fafbfc] px-5 py-4 sm:grid-cols-[180px_1fr_auto] sm:items-end">
          <FilterDropdown
            label="Environment"
            value={environment}
            options={KEY_ENV_OPTIONS}
            onChange={setEnvironment}
          />
          <label className="flex flex-col gap-1.5">
            <span className="text-[11px] font-semibold uppercase tracking-wide text-[#8a97a8]">
              Account password
            </span>
            <input
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder="Confirm password"
              className="h-9 rounded-lg border border-[#e8eef4] bg-white px-3 text-[13px] outline-none focus:border-[#3b9eff] focus:ring-2 focus:ring-[#3b9eff]/15"
            />
          </label>
          <Button
            type="button"
            disabled={saving || !password.trim()}
            onClick={() => void handleUpdate()}
            className="h-9 rounded-full bg-[#ff5e16] px-4 text-[12.5px] font-semibold text-white shadow-none hover:bg-[#e8530f] disabled:opacity-50"
          >
            <RefreshCwIcon className={`size-4 ${saving ? "animate-spin" : ""}`} />
            {existing ? "Update API key" : "Create API key"}
          </Button>
        </div>

        {revealedSecret && (
          <div className="border-b border-[#ffe0cc] bg-[#fff8f3] px-5 py-4">
            <p className="text-[12.5px] font-semibold text-[#9a4b1a]">
              {hint || "Secret (copy now)"}
            </p>
            <div className="mt-2 flex flex-col gap-2 sm:flex-row sm:items-center">
              <code className="flex-1 break-all rounded-lg border border-[#ffd4b0] bg-white px-3 py-2 text-[12.5px]">
                {revealedSecret}
              </code>
              <Button
                type="button"
                variant="outline"
                className="h-9 rounded-full shadow-none"
                onClick={() => void copyText(revealedSecret, "Secret")}
              >
                <CopyIcon className="size-3.5" />
                Copy
              </Button>
            </div>
            <pre className="mt-3 overflow-x-auto whitespace-pre-wrap break-all rounded-lg border border-[#ffd4b0] bg-white px-3 py-2.5 text-[11.5px]">
              {exampleCurl}
            </pre>
          </div>
        )}

        {loading ? (
          <LoadingBlock />
        ) : items.length === 0 ? (
          <EmptyState message="No API keys yet." />
        ) : (
          <TableShell>
            <table className="w-full min-w-160 text-left text-[12.5px]">
              <thead>
                <tr className="border-b border-[#eef2f6] text-[11px] font-semibold uppercase tracking-wide text-[#8a97a8]">
                  <th className="px-5 py-3">Environment</th>
                  <th className="px-5 py-3">Prefix</th>
                  <th className="px-5 py-3">Last used</th>
                  <th className="px-5 py-3">Created</th>
                  <th className="px-5 py-3 text-right">Actions</th>
                </tr>
              </thead>
              <tbody>
                {items.map((key) => (
                  <tr
                    key={key.id}
                    className="border-b border-[#f3f6f9] last:border-0"
                  >
                    <td className="px-5 py-3.5 font-medium capitalize">
                      {key.name}
                    </td>
                    <td className="px-5 py-3.5">
                      <code className="rounded bg-[#f4f7fb] px-1.5 py-0.5 text-[12px]">
                        {key.key_prefix}…
                      </code>
                    </td>
                    <td className="px-5 py-3.5 text-[#6b7c93]">
                      {key.last_used_at
                        ? formatDateTime(key.last_used_at)
                        : "—"}
                    </td>
                    <td className="px-5 py-3.5 text-[#6b7c93]">
                      {formatDateTime(key.created_at)}
                    </td>
                    <td className="px-5 py-3.5 text-right">
                      <button
                        type="button"
                        className="rounded-full px-2.5 py-1 text-[12px] font-semibold text-[#e85d3b] hover:bg-[#fff5f2]"
                        onClick={() => {
                          setDeletePassword("");
                          setDeleteTarget(key);
                        }}
                      >
                        Delete
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </TableShell>
        )}
      </Card>

      <AlertDialog
        open={!!deleteTarget}
        onOpenChange={(o) => {
          if (!o && !deleting) {
            setDeleteTarget(null);
            setDeletePassword("");
          }
        }}
      >
        <AlertDialogContent className="max-w-sm border border-[#e8eef4] bg-white">
          <AlertDialogHeader>
            <AlertDialogTitle>Delete API key?</AlertDialogTitle>
            <AlertDialogDescription>
              Enter your account password to permanently delete this key.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <input
            type="password"
            value={deletePassword}
            onChange={(e) => setDeletePassword(e.target.value)}
            placeholder="Password"
            className="h-9 w-full rounded-lg border border-[#e8eef4] px-3 text-[13px] outline-none focus:border-[#3b9eff] focus:ring-2 focus:ring-[#3b9eff]/15"
          />
          <AlertDialogFooter>
            <AlertDialogCancel disabled={deleting}>Cancel</AlertDialogCancel>
            <AlertDialogAction
              disabled={deleting || !deletePassword.trim()}
              className="!bg-[#e85d3b] !text-white hover:!bg-[#d64f30]"
              onClick={(e) => {
                e.preventDefault();
                void handleDelete();
              }}
            >
              {deleting ? "Deleting..." : "Delete key"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
