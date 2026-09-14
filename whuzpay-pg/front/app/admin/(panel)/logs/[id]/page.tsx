"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { useEffect, useState } from "react";
import toast from "react-hot-toast";
import { fetchAdminLog, type AdminLogDetail } from "@/lib/admin-api";
import {
  Card,
  formatDateTime,
  LoadingBlock,
  PageHeader,
  StatusBadge,
} from "../../../../components/admin/ui";

export default function AdminLogDetailPage() {
  const params = useParams<{ id: string }>();
  const id = params?.id;
  const [log, setLog] = useState<AdminLogDetail | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (!id) return;
    let cancelled = false;

    async function load() {
      setLoading(true);
      try {
        const data = await fetchAdminLog(id);
        if (!cancelled) setLog(data);
      } catch (err) {
        if (!cancelled) {
          toast.error(err instanceof Error ? err.message : "Failed to load log.");
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

  return (
    <div className="w-full space-y-5">
      <PageHeader
        title="Log detail"
        description="Webhook event detail for debugging. Sensitive keys are redacted."
        actions={
          <Link
            href="/admin/logs"
            className="text-[13px] font-semibold text-[#6b7c93] hover:text-[#06163a]"
          >
            ← Back to logs
          </Link>
        }
      />

      {loading ? (
        <Card>
          <LoadingBlock />
        </Card>
      ) : !log ? (
        <Card className="p-6 text-[13px] text-[#8a97a8]">Log not found.</Card>
      ) : (
        <>
          <Card className="p-5 sm:p-6">
            <div className="flex flex-wrap items-start justify-between gap-3">
              <div>
                <h2 className="text-[18px] font-semibold text-[#1f2a37]">
                  {log.event_type}
                </h2>
                <p className="mt-1 text-[13px] capitalize text-[#6b7c93]">
                  {log.provider_name}
                </p>
              </div>
              <StatusBadge status={log.status} />
            </div>

            <div className="mt-5 grid grid-cols-1 gap-3 sm:grid-cols-2">
              <div className="rounded-lg bg-[#f8fafc] px-4 py-3">
                <p className="text-[11px] font-semibold uppercase tracking-wide text-[#8a97a8]">
                  Created
                </p>
                <p className="mt-1 text-[13.5px] font-medium text-[#1f2a37]">
                  {formatDateTime(log.created_at)}
                </p>
              </div>
              <div className="rounded-lg bg-[#f8fafc] px-4 py-3">
                <p className="text-[11px] font-semibold uppercase tracking-wide text-[#8a97a8]">
                  Processed
                </p>
                <p className="mt-1 text-[13.5px] font-medium text-[#1f2a37]">
                  {log.is_processed
                    ? formatDateTime(log.processed_at) || "Yes"
                    : "No"}
                </p>
              </div>
              <div className="rounded-lg bg-[#f8fafc] px-4 py-3">
                <p className="text-[11px] font-semibold uppercase tracking-wide text-[#8a97a8]">
                  Provider reference
                </p>
                <p className="mt-1 break-all text-[13.5px] font-medium text-[#1f2a37]">
                  {log.provider_reference || "—"}
                </p>
              </div>
              <div className="rounded-lg bg-[#f8fafc] px-4 py-3">
                <p className="text-[11px] font-semibold uppercase tracking-wide text-[#8a97a8]">
                  Payment
                </p>
                <p className="mt-1 text-[13.5px] font-medium text-[#1f2a37]">
                  {log.payment_id ? (
                    <Link
                      href={`/admin/payments/${log.payment_id}`}
                      className="text-[#ff5e16] hover:underline"
                    >
                      View payment
                    </Link>
                  ) : (
                    "—"
                  )}
                </p>
              </div>
              {log.processing_error && (
                <div className="rounded-lg bg-[#fff5f2] px-4 py-3 sm:col-span-2">
                  <p className="text-[11px] font-semibold uppercase tracking-wide text-[#e85d3b]">
                    Processing error
                  </p>
                  <p className="mt-1 text-[13.5px] text-[#e85d3b]">
                    {log.processing_error}
                  </p>
                </div>
              )}
            </div>
          </Card>

          <Card className="p-5 sm:p-6">
            <h3 className="text-[15px] font-semibold text-[#1f2a37]">
              Raw payload
            </h3>
            <p className="mt-1 text-[12.5px] text-[#8a97a8]">
              Sensitive keys (secret, token, signature, api_key, etc.) are redacted.
            </p>
            <pre className="mt-4 max-h-[420px] overflow-auto rounded-lg bg-[#0b1c2c] p-4 text-[12px] leading-relaxed text-[#e8eef4]">
              {JSON.stringify(log.raw_payload ?? {}, null, 2)}
            </pre>
          </Card>
        </>
      )}
    </div>
  );
}
