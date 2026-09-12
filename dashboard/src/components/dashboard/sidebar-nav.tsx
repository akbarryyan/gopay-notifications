"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { cn } from "@/lib/utils";
import { COMING_SOON_ITEMS, INTEGRATION_ITEMS, MONITORING_ITEMS } from "./nav-items";

function SectionLabel({ children }: { children: string }) {
  return (
    <p className="px-3 pb-1 pt-4 text-[11px] font-semibold uppercase tracking-wider text-muted-foreground first:pt-0">
      {children}
    </p>
  );
}

export function SidebarNav({ onNavigate }: { onNavigate?: () => void }) {
  const pathname = usePathname();

  return (
    <nav className="flex flex-col gap-1 px-2 py-4">
      <SectionLabel>Monitoring</SectionLabel>
      {MONITORING_ITEMS.map((item) => (
        <Link
          key={item.href}
          href={item.href}
          onClick={onNavigate}
          className={cn(
            "flex items-center gap-2.5 rounded-md px-3 py-2 text-sm font-medium transition-colors",
            pathname === item.href
              ? "bg-secondary text-secondary-foreground"
              : "text-muted-foreground hover:bg-secondary/60 hover:text-foreground",
          )}
        >
          <item.icon className="size-4 shrink-0" />
          {item.label}
        </Link>
      ))}

      <SectionLabel>Integration</SectionLabel>
      {INTEGRATION_ITEMS.map((item) => (
        <Link
          key={item.href}
          href={item.href}
          onClick={onNavigate}
          className={cn(
            "flex items-center gap-2.5 rounded-md px-3 py-2 text-sm font-medium transition-colors",
            pathname === item.href
              ? "bg-secondary text-secondary-foreground"
              : "text-muted-foreground hover:bg-secondary/60 hover:text-foreground",
          )}
        >
          <item.icon className="size-4 shrink-0" />
          {item.label}
        </Link>
      ))}

      <SectionLabel>System</SectionLabel>
      {COMING_SOON_ITEMS.map((item) => (
        <div
          key={item.label}
          title="Belum tersedia"
          className="flex cursor-not-allowed items-center gap-2.5 rounded-md px-3 py-2 text-sm text-muted-foreground/50"
        >
          <item.icon className="size-4 shrink-0" />
          <span className="flex-1">{item.label}</span>
          <span className="text-[10px] font-medium uppercase tracking-wide">Segera</span>
        </div>
      ))}
    </nav>
  );
}
