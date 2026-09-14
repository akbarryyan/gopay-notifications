import type { LucideIcon } from "lucide-react";
import {
  LayoutDashboard,
  Smartphone,
  Bell,
  Receipt,
  Webhook,
  KeyRound,
  ShieldCheck,
  Settings,
  ScrollText,
  AlertTriangle,
  BookOpen,
} from "lucide-react";

export interface NavItem {
  label: string;
  href: string;
  icon: LucideIcon;
}

/**
 * Halaman-halaman ini datanya sungguhan ada: heartbeat dan event sejak
 * M2–M4, invoice/nominal unik/matching/API key sejak sub-project 3 fase 1,
 * webhook sejak fase 2, konsol pengecualian sejak fase 4, lisensi sejak
 * sistem lisensi offline (lihat
 * docs/superpowers/specs/2026-09-12-invoice-nominal-matching-design.md,
 * 2026-09-13-webhook-delivery-design.md,
 * 2026-09-13-exception-console-design.md, dan
 * 2026-09-13-license-system-design.md). Logs masih menunggu —
 * menampilkannya sebagai link aktif sekarang berarti membangun halaman
 * untuk data yang bentuknya belum pasti.
 */
export const MONITORING_ITEMS: NavItem[] = [
  { label: "Overview", href: "/overview", icon: LayoutDashboard },
  { label: "Events", href: "/events", icon: Bell },
];

export const INTEGRATION_ITEMS: NavItem[] = [
  { label: "Devices", href: "/devices", icon: Smartphone },
];

export const GATEWAY_ITEMS: NavItem[] = [
  { label: "Transactions", href: "/transactions", icon: Receipt },
  { label: "API Keys", href: "/api-keys", icon: KeyRound },
  { label: "Webhooks", href: "/webhooks", icon: Webhook },
  { label: "Exceptions", href: "/exceptions", icon: AlertTriangle },
  { label: "API Docs", href: "/api-docs", icon: BookOpen },
];

export const SYSTEM_ITEMS: NavItem[] = [
  { label: "License", href: "/license", icon: ShieldCheck },
  { label: "Settings", href: "/settings", icon: Settings },
];

/**
 * Ditampilkan, tidak disembunyikan — supaya bentuk akhir produk tetap
 * terlihat (sesuai dashboard-spec §16, pola yang sama dipakai untuk
 * connector DANA/OVO yang belum diimplementasikan).
 */
export const COMING_SOON_ITEMS: { label: string; icon: LucideIcon }[] = [
  { label: "Logs", icon: ScrollText },
];
