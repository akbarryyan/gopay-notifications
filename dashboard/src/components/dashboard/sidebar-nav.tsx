"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { cn } from "@/lib/utils";
import {
  GATEWAY_ITEMS,
  INTEGRATION_ITEMS,
  MONITORING_ITEMS,
  SYSTEM_ITEMS,
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
      {SYSTEM_ITEMS.map((item) => (
        <NavLink
          key={item.href}
          item={item}
          active={pathname === item.href}
          collapsed={collapsed}
          onNavigate={onNavigate}
        />
      ))}
    </nav>
  );
}
