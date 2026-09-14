"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import toast from "react-hot-toast";
import { ArrowRightIcon } from "lucide-react";
import { fetchAdminProviders, type AdminProvider } from "@/lib/admin-api";
import {
  Card,
  EmptyState,
  formatDateTime,
  LoadingBlock,
  PageHeader,
} from "../../../components/admin/ui";

function healthStyle(status: string) {
  switch (status) {
    case "healthy":
      return "bg-[#e8f8ee] text-[#2f9e5a]";
    case "degraded":
      return "bg-[#fff7e6] text-[#c27a00]";
    case "unhealthy":
      return "bg-[#fff1ed] text-[#e85d3b]";
    default:
      return "bg-[#eef2f6] text-[#6b7c93]";
  }
}

export default function AdminProvidersPage() {
  const [items, setItems] = useState<AdminProvider[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;

    async function load() {
      setLoading(true);
      try {
        const data = await fetchAdminProviders();
        if (!cancelled) setItems(data.items ?? []);
      } catch (err) {
        if (!cancelled) {
          toast.error(
            err instanceof Error ? err.message : "Failed to load providers.",
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

  return (
    <div className="w-full space-y-5">
      <PageHeader
        title="Providers"
        description="Registered payment providers and health status. Credentials stay in server environment variables."
      />

      {loading ? (
        <Card>
          <LoadingBlock />
        </Card>
      ) : items.length === 0 ? (
        <Card>
          <EmptyState message="No providers registered." />
        </Card>
      ) : (
        <div className="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-3">
          {items.map((p) => (
            <Card key={p.name} className="p-5 transition-shadow hover:shadow-md">
              <div className="flex items-start justify-between gap-3">
                <div>
                  <h2 className="text-[16px] font-semibold capitalize text-[#1f2a37]">
                    {p.name}
                  </h2>
                  <p className="mt-1 text-[12.5px] text-[#8a97a8]">
                    {p.is_registered ? "Registered adapter" : "Not registered"}
                  </p>
                </div>
                <span
                  className={`inline-flex rounded-full px-2.5 py-0.5 text-[11px] font-semibold capitalize ${healthStyle(
                    p.health?.status ?? "unknown",
                  )}`}
                >
                  {p.health?.status ?? "unknown"}
                </span>
              </div>

              <div className="mt-4 space-y-3">
                <div className="rounded-lg bg-[#f8fafc] px-3 py-2.5">
                  <p className="text-[11px] font-semibold uppercase tracking-wide text-[#8a97a8]">
                    Payment methods
                  </p>
                  <p className="mt-1 text-[13px] font-medium capitalize text-[#1f2a37]">
                    {(p.payment_methods ?? []).length > 0
                      ? p.payment_methods.join(", ")
                      : "—"}
                  </p>
                </div>

                {p.health?.reason && (
                  <div className="rounded-lg bg-[#f8fafc] px-3 py-2.5">
                    <p className="text-[11px] font-semibold uppercase tracking-wide text-[#8a97a8]">
                      Reason
                    </p>
                    <p className="mt-1 text-[13px] text-[#3d4b5c]">
                      {p.health.reason}
                    </p>
                  </div>
                )}

                <div className="rounded-lg bg-[#f8fafc] px-3 py-2.5">
                  <p className="text-[11px] font-semibold uppercase tracking-wide text-[#8a97a8]">
                    Health updated
                  </p>
                  <p className="mt-1 text-[13px] text-[#3d4b5c]">
                    {formatDateTime(p.health?.updated_at)}
                  </p>
                </div>

                <Link
                  href={`/admin/providers/${encodeURIComponent(p.name)}`}
                  className="inline-flex items-center gap-1.5 text-[12.5px] font-semibold text-[#ff5e16] hover:underline"
                >
                  View detail
                  <ArrowRightIcon className="size-3.5" />
                </Link>
              </div>
            </Card>
          ))}
        </div>
      )}
    </div>
  );
}
