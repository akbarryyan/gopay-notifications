"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import {
  IconBook,
  IconChart,
  IconFile,
  IconGrid,
  IconSend,
  IconTable,
  IconUser,
  IconWebhook,
} from "../admin/icons";

const navItems = [
  { label: "Dashboard", href: "/dashboard", icon: IconGrid },
  { label: "Payments", href: "/dashboard/payments", icon: IconTable },
  { label: "Payment Links", href: "/dashboard/payment-links", icon: IconSend },
  { label: "Reports", href: "/dashboard/reports", icon: IconChart },
  { label: "Webhook Logs", href: "/dashboard/webhooks", icon: IconWebhook },
  { label: "API keys", href: "/dashboard/api-keys", icon: IconFile },
  { label: "API Docs", href: "/dashboard/api-docs", icon: IconBook },
  { label: "Settings", href: "/dashboard/settings", icon: IconUser },
];

type Props = { collapsed: boolean };

function isActivePath(pathname: string, href: string) {
  if (href === "/dashboard") return pathname === "/dashboard";
  return pathname === href || pathname.startsWith(`${href}/`);
}

export default function MerchantSidebar({ collapsed }: Props) {
  const pathname = usePathname();

  return (
    <aside
      className={`admin-sidebar sticky top-0 z-40 flex h-screen shrink-0 flex-col bg-[#06163a] text-white transition-[width] duration-200 ease-in-out ${
        collapsed ? "w-18" : "w-65"
      }`}
    >
      <div
        className={`flex h-16 shrink-0 items-center ${
          collapsed ? "justify-center px-2" : "gap-2.5 px-5"
        }`}
      >
        <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-md bg-[#ff5e16]">
          <svg viewBox="0 0 24 24" className="h-4 w-4 text-white" fill="currentColor">
            <path d="M4 12c3-6 6-8 8-8s5 2 8 8c-3 6-6 8-8 8s-5-2-8-8Z" />
            <path
              d="M9 12c1.2-2.2 2.3-3 3-3s1.8.8 3 3c-1.2 2.2-2.3 3-3 3s-1.8-.8-3-3Z"
              fill="#06163a"
            />
          </svg>
        </span>
        <span
          className={`overflow-hidden whitespace-nowrap text-[22px] font-semibold tracking-tight transition-all duration-200 ${
            collapsed ? "w-0 opacity-0" : "w-auto opacity-100"
          }`}
        >
          Lector<span className="text-[#ff5e16]">.</span>
        </span>
      </div>

      <nav className="admin-sidebar-scroll flex-1 overflow-y-auto overflow-x-hidden px-0 pb-6 pt-1">
        <ul className="flex flex-col">
          {navItems.map((item) => {
            const active = isActivePath(pathname, item.href);
            const Icon = item.icon;
            const className = `group relative flex items-center transition-colors ${
              collapsed
                ? "justify-center px-0 py-3"
                : "gap-3 px-5 py-2.75 text-[13.5px]"
            } ${
              active
                ? "bg-white/5 font-medium text-white"
                : "text-white/65 hover:bg-white/5 hover:text-white"
            }`;

            return (
              <li key={item.href} className="relative">
                <Link
                  href={item.href}
                  className={className}
                  title={collapsed ? item.label : undefined}
                >
                  {active && (
                    <span className="absolute top-1 bottom-1 left-0 w-0.75 rounded-r bg-[#ff5e16]" />
                  )}
                  <Icon className="h-4.5 w-4.5 shrink-0 opacity-90" />
                  <span
                    className={`overflow-hidden whitespace-nowrap transition-all duration-200 ${
                      collapsed ? "w-0 opacity-0" : "w-auto opacity-100"
                    }`}
                  >
                    {item.label}
                  </span>
                  {collapsed && (
                    <span className="pointer-events-none absolute left-full z-50 ml-2 hidden whitespace-nowrap rounded-md bg-[#0b2048] px-2.5 py-1.5 text-[12px] font-medium text-white shadow-lg group-hover:block">
                      {item.label}
                    </span>
                  )}
                </Link>
              </li>
            );
          })}
        </ul>

        {!collapsed && (
          <div className="mx-5 mt-6 rounded-lg border border-white/10 bg-white/5 px-3 py-3">
            <p className="text-[11px] font-semibold uppercase tracking-wide text-white/45">
              Merchant
            </p>
            <p className="mt-1 text-[12px] leading-relaxed text-white/60">
              Payments, API keys, and business settings for your store.
            </p>
          </div>
        )}
      </nav>
    </aside>
  );
}
