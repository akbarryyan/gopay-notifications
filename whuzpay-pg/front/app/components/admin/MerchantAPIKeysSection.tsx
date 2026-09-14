"use client";

import { useEffect, useMemo, useState } from "react";
import toast from "react-hot-toast";
import { CopyIcon, KeyRoundIcon, RefreshCwIcon } from "lucide-react";
import {
  deleteAdminMerchantAPIKey,
  fetchAdminMerchantAPIKeys,
  upsertAdminMerchantAPIKey,
  type AdminMerchantAPIKey,
} from "@/lib/admin-api";
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
import FilterDropdown from "./FilterDropdown";
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

const KEY_ENV_OPTIONS = [
  { value: "sandbox", label: "Sandbox" },
  { value: "production", label: "Production" },
];

export default function MerchantAPIKeysSection({ merchantId }: Props) {
  const [items, setItems] = useState<AdminMerchantAPIKey[]>([]);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [environment, setEnvironment] = useState("sandbox");
  const [password, setPassword] = useState("");
  const [revealedSecret, setRevealedSecret] = useState<string | null>(null);
  const [lastHint, setLastHint] = useState<string | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<AdminMerchantAPIKey | null>(
    null,
  );
  const [deletePassword, setDeletePassword] = useState("");
  const [deleting, setDeleting] = useState(false);

  const existingForEnv = useMemo(
    () => items.find((k) => k.name === environment && k.is_active),
    [items, environment],
  );

  async function load() {
    setLoading(true);
    try {
      const data = await fetchAdminMerchantAPIKeys(merchantId);
      setItems(data.items ?? []);
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Failed to load API keys.",
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

  async function handleUpdate() {
    if (!password.trim()) {
      toast.error("Admin password is required.");
      return;
    }
    setSaving(true);
    try {
      const result = await upsertAdminMerchantAPIKey(merchantId, {
        environment,
        password: password.trim(),
      });
      setRevealedSecret(result.secret);
      setLastHint(result.hint);
      setPassword("");
      setItems((prev) => {
        const withoutEnv = prev.filter((k) => k.name !== result.environment);
        return [result.key, ...withoutEnv];
      });
      toast.success(
        result.rotated
          ? "API key rotated. Copy the new secret now."
          : "API key created. Copy the secret now.",
      );
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Failed to update API key.",
      );
    } finally {
      setSaving(false);
    }
  }

  async function handleDelete() {
    if (!deleteTarget) return;
    if (!deletePassword.trim()) {
      toast.error("Admin password is required.");
      return;
    }
    setDeleting(true);
    try {
      await deleteAdminMerchantAPIKey(
        merchantId,
        deleteTarget.id,
        deletePassword.trim(),
      );
      setItems((prev) => prev.filter((k) => k.id !== deleteTarget.id));
      toast.success("API key deleted.");
      setDeleteTarget(null);
      setDeletePassword("");
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Failed to delete API key.",
      );
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
    <>
      <Card>
        <div className="flex flex-col gap-3 border-b border-[#eef2f6] px-5 py-4 sm:flex-row sm:items-start sm:justify-between">
          <div>
            <h3 className="flex items-center gap-2 text-[15px] font-semibold text-[#1f2a37]">
              <KeyRoundIcon className="size-4 text-[#ff5e16]" />
              API keys
            </h3>
            <p className="mt-0.5 text-[12.5px] text-[#8a97a8]">
              One key per environment (sandbox / production). Update rotates the
              secret — previous key for that environment stops working. Admin
              password is required.
            </p>
          </div>
        </div>

        <div className="grid grid-cols-1 gap-3 border-b border-[#eef2f6] bg-[#fafbfc] px-5 py-4 sm:grid-cols-[180px_1fr_auto] sm:items-end">
          <FilterDropdown
            label="Environment"
            value={environment}
            options={KEY_ENV_OPTIONS}
            onChange={setEnvironment}
            className="w-full"
          />
          <label className="flex min-w-0 flex-col gap-1.5">
            <span className="text-[11px] font-semibold uppercase tracking-wide text-[#8a97a8]">
              Admin password
            </span>
            <input
              type="password"
              autoComplete="current-password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder="Confirm your password"
              className="h-9 w-full rounded-lg border border-[#e8eef4] bg-white px-3 text-[13px] text-[#1f2a37] outline-none focus:border-[#3b9eff] focus:ring-2 focus:ring-[#3b9eff]/15"
            />
          </label>
          <Button
            type="button"
            disabled={saving || !password.trim()}
            onClick={() => void handleUpdate()}
            className="h-9 rounded-full bg-[#ff5e16] px-4 text-[12.5px] font-semibold text-white shadow-none hover:bg-[#e8530f] disabled:opacity-50"
          >
            <RefreshCwIcon className={`size-4 ${saving ? "animate-spin" : ""}`} />
            {saving
              ? "Updating..."
              : existingForEnv
                ? "Update API key"
                : "Create API key"}
          </Button>
        </div>

        {existingForEnv && (
          <div className="border-b border-[#eef2f6] px-5 py-3 text-[12.5px] text-[#6b7c93]">
            Current{" "}
            <span className="font-semibold capitalize text-[#1f2a37]">
              {environment}
            </span>{" "}
            key:{" "}
            <code className="rounded bg-[#f4f7fb] px-1.5 py-0.5 text-[12px] text-[#1f2a37]">
              {existingForEnv.key_prefix}…
            </code>
            {" · "}
            Updating will invalidate this secret immediately.
          </div>
        )}

        {revealedSecret && (
          <div className="border-b border-[#ffe0cc] bg-[#fff8f3] px-5 py-4">
            <p className="text-[12.5px] font-semibold text-[#9a4b1a]">
              {lastHint || "Secret (copy now — will not be shown again)"}
            </p>
            <div className="mt-2 flex flex-col gap-2 sm:flex-row sm:items-center">
              <code className="block flex-1 break-all rounded-lg border border-[#ffd4b0] bg-white px-3 py-2 text-[12.5px] text-[#1f2a37]">
                {revealedSecret}
              </code>
              <Button
                type="button"
                variant="outline"
                className="h-9 shrink-0 rounded-full border-[#e8eef4] bg-white text-[12.5px] font-semibold shadow-none"
                onClick={() => void copyText(revealedSecret, "Secret")}
              >
                <CopyIcon className="size-3.5" />
                Copy
              </Button>
              <Button
                type="button"
                variant="outline"
                className="h-9 shrink-0 rounded-full border-[#e8eef4] bg-white text-[12.5px] font-semibold shadow-none"
                onClick={() => {
                  setRevealedSecret(null);
                  setLastHint(null);
                }}
              >
                Dismiss
              </Button>
            </div>

            <div className="mt-3">
              <div className="mb-1.5 flex items-center justify-between gap-2">
                <p className="text-[12px] font-semibold text-[#9a4b1a]">
                  Example request
                </p>
                <button
                  type="button"
                  className="text-[12px] font-semibold text-[#ff5e16] hover:underline"
                  onClick={() => void copyText(exampleCurl, "Example curl")}
                >
                  Copy curl
                </button>
              </div>
              <pre className="overflow-x-auto whitespace-pre-wrap break-all rounded-lg border border-[#ffd4b0] bg-white px-3 py-2.5 text-[11.5px] leading-relaxed text-[#1f2a37]">
                {exampleCurl}
              </pre>
            </div>
          </div>
        )}

        {loading ? (
          <LoadingBlock />
        ) : items.length === 0 ? (
          <EmptyState message="No API keys yet. Choose environment, enter password, then create." />
        ) : (
          <TableShell>
            <table className="w-full min-w-[720px] text-left text-[12.5px]">
              <thead>
                <tr className="border-b border-[#eef2f6] text-[11px] font-semibold uppercase tracking-wide text-[#8a97a8]">
                  <th className="px-5 py-3">Environment</th>
                  <th className="px-5 py-3">Prefix</th>
                  <th className="px-5 py-3">Status</th>
                  <th className="px-5 py-3">Last used</th>
                  <th className="px-5 py-3">Created</th>
                  <th className="px-5 py-3 text-right">Actions</th>
                </tr>
              </thead>
              <tbody>
                {items.map((key) => (
                  <tr
                    key={key.id}
                    className="border-b border-[#f3f6f9] text-[#3d4b5c] last:border-0"
                  >
                    <td className="px-5 py-3.5 font-medium capitalize text-[#1f2a37]">
                      {key.name}
                    </td>
                    <td className="px-5 py-3.5">
                      <code className="rounded bg-[#f4f7fb] px-1.5 py-0.5 text-[12px]">
                        {key.key_prefix}…
                      </code>
                    </td>
                    <td className="px-5 py-3.5">
                      <span
                        className={`inline-flex rounded-full px-2.5 py-0.5 text-[11px] font-semibold ${
                          key.is_active
                            ? "bg-[#e8f8ee] text-[#2f9e5a]"
                            : "bg-[#fff1ed] text-[#e85d3b]"
                        }`}
                      >
                        {key.is_active ? "Active" : "Inactive"}
                      </span>
                    </td>
                    <td className="px-5 py-3.5 text-[#6b7c93]">
                      {key.last_used_at
                        ? formatDateTime(key.last_used_at)
                        : "—"}
                    </td>
                    <td className="px-5 py-3.5 text-[#6b7c93]">
                      {formatDateTime(key.created_at)}
                    </td>
                    <td className="px-5 py-3.5">
                      <div className="flex justify-end gap-2">
                        <button
                          type="button"
                          className="rounded-full px-2.5 py-1 text-[12px] font-semibold text-[#6b7c93] hover:bg-[#f4f7fb]"
                          onClick={() =>
                            void copyText(key.key_prefix, "Prefix")
                          }
                        >
                          Copy prefix
                        </button>
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
                      </div>
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
        onOpenChange={(open) => {
          if (!open && !deleting) {
            setDeleteTarget(null);
            setDeletePassword("");
          }
        }}
      >
        <AlertDialogContent className="max-w-sm border border-[#e8eef4] bg-white">
          <AlertDialogHeader>
            <AlertDialogTitle className="text-[#1f2a37]">
              Delete API key?
            </AlertDialogTitle>
            <AlertDialogDescription className="text-[#8a97a8]">
              {deleteTarget
                ? `Key “${deleteTarget.name}” (${deleteTarget.key_prefix}…) will be permanently deleted. Enter your admin password to confirm.`
                : ""}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <div className="px-0">
            <label className="flex flex-col gap-1.5">
              <span className="text-[11px] font-semibold uppercase tracking-wide text-[#8a97a8]">
                Admin password
              </span>
              <input
                type="password"
                autoComplete="current-password"
                value={deletePassword}
                onChange={(e) => setDeletePassword(e.target.value)}
                placeholder="Your password"
                className="h-9 w-full rounded-lg border border-[#e8eef4] bg-white px-3 text-[13px] text-[#1f2a37] outline-none focus:border-[#3b9eff] focus:ring-2 focus:ring-[#3b9eff]/15"
              />
            </label>
          </div>
          <AlertDialogFooter>
            <AlertDialogCancel
              disabled={deleting}
              className="border-[#e8eef4] bg-white"
            >
              Cancel
            </AlertDialogCancel>
            <AlertDialogAction
              disabled={deleting || !deletePassword.trim()}
              variant="default"
              className="!bg-[#e85d3b] !text-white hover:!bg-[#d64f30] disabled:opacity-50"
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
    </>
  );
}
