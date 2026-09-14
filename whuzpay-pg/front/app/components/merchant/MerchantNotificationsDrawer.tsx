"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import toast from "react-hot-toast";
import {
  AlertTriangleIcon,
  CheckCircle2Icon,
  CircleAlertIcon,
  Clock3Icon,
} from "lucide-react";
import {
  fetchMerchantNotifications,
  type MerchantNotification,
} from "@/lib/merchant-api";
import {
  Drawer,
  DrawerContent,
  DrawerDescription,
  DrawerHeader,
  DrawerTitle,
} from "@/components/ui/drawer";
import { formatDateTime } from "../admin/ui";

type Props = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  environment: string;
  onCountChange?: (count: number) => void;
};

function kindMeta(kind: string) {
  switch (kind) {
    case "webhook_error":
      return {
        icon: CircleAlertIcon,
        tone: "bg-[#fff1ed] text-[#e85d3b]",
        label: "Callback error",
      };
    case "payment_failed":
      return {
        icon: AlertTriangleIcon,
        tone: "bg-[#fff1ed] text-[#e85d3b]",
        label: "Payment failed",
      };
    case "payment_expired":
      return {
        icon: Clock3Icon,
        tone: "bg-[#eef2f6] text-[#6b7c93]",
        label: "Expired",
      };
    default:
      return {
        icon: CheckCircle2Icon,
        tone: "bg-[#e8f8ee] text-[#2f9e5a]",
        label: "Update",
      };
  }
}

export default function MerchantNotificationsDrawer({
  open,
  onOpenChange,
  environment,
  onCountChange,
}: Props) {
  const [loading, setLoading] = useState(false);
  const [items, setItems] = useState<MerchantNotification[]>([]);

  useEffect(() => {
    let cancelled = false;
    async function load() {
      if (open) setLoading(true);
      try {
        const data = await fetchMerchantNotifications(environment, 30);
        if (cancelled) return;
        setItems(data.items ?? []);
        onCountChange?.(data.attention_count ?? 0);
      } catch (err) {
        if (!cancelled && open) {
          toast.error(
            err instanceof Error ? err.message : "Failed to load notifications.",
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
  }, [open, environment, onCountChange]);

  return (
    <Drawer open={open} onOpenChange={onOpenChange} direction="right">
      <DrawerContent className="ml-auto h-full w-full max-w-md rounded-none border-l border-[#e8eef4] bg-white">
        <DrawerHeader className="border-b border-[#eef2f6] text-left">
          <DrawerTitle className="text-[#1f2a37]">Notifications</DrawerTitle>
          <DrawerDescription className="text-[#8a97a8]">
            Alerts for {environment}: failed callbacks, failed & expired payments.
          </DrawerDescription>
        </DrawerHeader>
        <div className="flex-1 overflow-y-auto p-4">
          {loading ? (
            <p className="text-[13px] text-[#8a97a8]">Loading...</p>
          ) : items.length === 0 ? (
            <p className="text-[13px] text-[#8a97a8]">No notifications right now.</p>
          ) : (
            <ul className="space-y-2">
              {items.map((n) => {
                const meta = kindMeta(n.kind);
                const Icon = meta.icon;
                const body = (
                  <div className="flex gap-3 rounded-xl border border-[#eef2f6] bg-[#fbfcfe] p-3">
                    <span
                      className={`flex size-9 shrink-0 items-center justify-center rounded-lg ${meta.tone}`}
                    >
                      <Icon className="size-4" />
                    </span>
                    <div className="min-w-0 flex-1">
                      <p className="text-[13px] font-semibold text-[#1f2a37]">
                        {n.title}
                      </p>
                      <p className="mt-0.5 text-[12.5px] text-[#6b7c93]">{n.body}</p>
                      <p className="mt-1 text-[11px] text-[#8a97a8]">
                        {formatDateTime(n.created_at)} · {meta.label}
                      </p>
                    </div>
                  </div>
                );
                return (
                  <li key={n.id}>
                    {n.href ? (
                      <Link href={n.href} onClick={() => onOpenChange(false)}>
                        {body}
                      </Link>
                    ) : (
                      body
                    )}
                  </li>
                );
              })}
            </ul>
          )}
        </div>
      </DrawerContent>
    </Drawer>
  );
}
