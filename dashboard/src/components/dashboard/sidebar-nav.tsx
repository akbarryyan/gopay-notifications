"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { cn } from "@/lib/utils";
import {
  COMING_SOON_ITEMS,
  GATEWAY_ITEMS,
  INTEGRATION_ITEMS,
  MONITORING_ITEMS,
  type NavItem,
} from "./nav-items";

function SectionLabel({ children, collapsed }: { children: string; collapsed?: boolean }) {
  if (collapsed) {
    return <div className="mx-3 mt-4 border-t border-sidebar-border first:mt-0" />;
  }
  return (
    <p className="px-3 pb-1.5 pt-5 text-[11px] font-semibold uppercase tracking-wider text-sidebar-foreground/45 first:pt-0">
      {children}
    </p>
  );
}

function NavLink({
  item,
  active,
  collapsed,
  onNavigate,
}: {
  item: NavItem;
  active: boolean;
  collapsed?: boolean;
  onNavigate?: () => void;
}) {
  return (
    <Link
      href={item.href}
      onClick={onNavigate}
      title={collapsed ? item.label : undefined}
      className={cn(
        "flex items-center gap-2.5 rounded-xl px-3 py-2 text-sm font-medium transition-colors",
        collapsed && "justify-center px-2",
        active
          ? "bg-sidebar-accent text-sidebar-accent-foreground"
          : "text-sidebar-foreground/65 hover:bg-white/5 hover:text-sidebar-foreground",
      )}
    >
      <item.icon className="size-4 shrink-0" />
      {!collapsed && item.label}
    </Link>
  );
}

export function SidebarNav({
  onNavigate,
  collapsed,
}: {
  onNavigate?: () => void;
  collapsed?: boolean;
}) {
  const pathname = usePathname();

  return (
    <nav className="flex flex-col gap-0.5 px-3 py-2">
      <SectionLabel collapsed={collapsed}>Monitoring</SectionLabel>
      {MONITORING_ITEMS.map((item) => (
        <NavLink
          key={item.href}
          item={item}
          active={pathname === item.href}
          collapsed={collapsed}
          onNavigate={onNavigate}
        />
      ))}

      <SectionLabel collapsed={collapsed}>Integration</SectionLabel>
      {INTEGRATION_ITEMS.map((item) => (
        <NavLink
          key={item.href}
          item={item}
          active={pathname === item.href}
          collapsed={collapsed}
          onNavigate={onNavigate}
        />
      ))}

      <SectionLabel collapsed={collapsed}>Gateway</SectionLabel>
      {GATEWAY_ITEMS.map((item) => (
        <NavLink
          key={item.href}
          item={item}
          active={pathname === item.href}
          collapsed={collapsed}
          onNavigate={onNavigate}
        />
      ))}

      <SectionLabel collapsed={collapsed}>System</SectionLabel>
      {COMING_SOON_ITEMS.map((item) => (
        <div
          key={item.label}
          title={collapsed ? `${item.label} — belum tersedia` : "Belum tersedia"}
          className={cn(
            "flex cursor-not-allowed items-center gap-2.5 rounded-xl px-3 py-2 text-sm text-sidebar-foreground/30",
            collapsed && "justify-center px-2",
          )}
        >
          <item.icon className="size-4 shrink-0" />
          {!collapsed && (
            <>
              <span className="flex-1">{item.label}</span>
              <span className="rounded-full bg-white/10 px-1.5 py-0.5 text-[10px] font-medium uppercase tracking-wide text-sidebar-foreground/50">
                Segera
              </span>
            </>
          )}
        </div>
      ))}
    </nav>
  );
}
