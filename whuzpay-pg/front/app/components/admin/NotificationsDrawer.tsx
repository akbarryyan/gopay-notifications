"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import toast from "react-hot-toast";
import {
  AlertTriangleIcon,
  BellIcon,
  CheckCircle2Icon,
  CircleAlertIcon,
  Clock3Icon,
} from "lucide-react";
import {
  fetchAdminNotifications,
  type AdminNotification,
} from "@/lib/admin-api";
import { Button } from "@/components/ui/button";
import {
  Drawer,
  DrawerContent,
  DrawerDescription,
  DrawerHeader,
  DrawerTitle,
} from "@/components/ui/drawer";
import { formatDateTime } from "./ui";

type NotificationsDrawerProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onCountChange?: (count: number) => void;
};

function kindMeta(kind: AdminNotification["kind"]) {
  switch (kind) {
    case "webhook_error":
      return {
        icon: CircleAlertIcon,
        tone: "bg-[#fff1ed] text-[#e85d3b]",
        label: "Webhook error",
      };
    case "webhook_pending":
      return {
        icon: Clock3Icon,
        tone: "bg-[#fff7e6] text-[#c27a00]",
        label: "Needs attention",
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
        label: "Payment expired",
      };
    default:
      return {
        icon: CheckCircle2Icon,
        tone: "bg-[#e8f8ee] text-[#2f9e5a]",
        label: "Update",
      };
  }
}

export default function NotificationsDrawer({
  open,
  onOpenChange,
  onCountChange,
}: NotificationsDrawerProps) {
  const [loading, setLoading] = useState(false);
  const [notifications, setNotifications] = useState<AdminNotification[]>([]);

  useEffect(() => {
    let cancelled = false;

    async function load() {
      if (open) setLoading(true);
      try {
        const data = await fetchAdminNotifications(30);
        if (cancelled) return;
        setNotifications(data.items ?? []);
        onCountChange?.(data.attention_count ?? 0);
      } catch (err) {
        if (!cancelled && open) {
          toast.error(
            err instanceof Error
              ? err.message
              : "Failed to load notifications.",
          );
        }
      } finally {
        if (!cancelled) setLoading(false);
      }
    }

    // Prefetch badge on mount; refresh when drawer opens.
    void load();
    return () => {
      cancelled = true;
    };
  }, [open, onCountChange]);

  return (
    <Drawer open={open} onOpenChange={onOpenChange} direction="right">
      <DrawerContent className="data-[vaul-drawer-direction=right]:w-full data-[vaul-drawer-direction=right]:sm:max-w-md border-[#e8eef4] bg-white">
        <DrawerHeader className="border-b border-[#eef2f6] text-left">
          <DrawerTitle className="flex items-center gap-2 text-[18px] font-semibold text-[#1f2a37]">
            <BellIcon className="size-5 text-[#ff5e16]" />
            Notifications
          </DrawerTitle>
          <DrawerDescription className="text-[13px] text-[#8a97a8]">
            Operational alerts from webhooks and payment failures. No secrets are
            shown.
          </DrawerDescription>
        </DrawerHeader>

        <div className="flex-1 overflow-y-auto px-4 py-3">
          {loading ? (
            <div className="py-12 text-center text-[13px] text-[#8a97a8]">
              Loading notifications...
            </div>
          ) : notifications.length === 0 ? (
            <div className="py-12 text-center">
              <div className="mx-auto mb-3 flex size-12 items-center justify-center rounded-full bg-[#f4f7fb] text-[#8a97a8]">
                <BellIcon className="size-5" />
              </div>
              <p className="text-[13.5px] font-medium text-[#1f2a37]">
                All clear
              </p>
              <p className="mt-1 text-[12.5px] text-[#8a97a8]">
                No webhook errors or failed/expired payments right now.
              </p>
            </div>
          ) : (
            <ul className="space-y-2.5">
              {notifications.map((n) => {
                const meta = kindMeta(n.kind);
                const Icon = meta.icon;
                return (
                  <li key={n.id}>
                    <Link
                      href={n.href || "/admin/logs"}
                      onClick={() => onOpenChange(false)}
                      className="flex gap-3 rounded-xl border border-[#eef2f6] bg-[#fbfcfe] p-3 transition hover:border-[#dbe4ee] hover:bg-white"
                    >
                      <span
                        className={`mt-0.5 flex size-9 shrink-0 items-center justify-center rounded-full ${meta.tone}`}
                      >
                        <Icon className="size-4" />
                      </span>
                      <div className="min-w-0 flex-1">
                        <div className="flex items-start justify-between gap-2">
                          <p className="truncate text-[13px] font-semibold text-[#1f2a37]">
                            {n.title}
                          </p>
                          <span className="shrink-0 text-[11px] text-[#8a97a8]">
                            {formatDateTime(n.created_at)}
                          </span>
                        </div>
                        <p className="mt-0.5 text-[11px] font-medium text-[#8a97a8]">
                          {meta.label}
                        </p>
                        <p className="mt-1 line-clamp-2 text-[12.5px] text-[#3d4b5c]">
                          {n.body}
                        </p>
                      </div>
                    </Link>
                  </li>
                );
              })}
            </ul>
          )}
        </div>

        <div className="border-t border-[#eef2f6] p-4">
          <Button
            type="button"
            variant="outline"
            className="w-full border-[#e8eef4] text-[#3d4b5c]"
            asChild
          >
            <Link href="/admin/logs" onClick={() => onOpenChange(false)}>
              View all logs
            </Link>
          </Button>
        </div>
      </DrawerContent>
    </Drawer>
  );
}
