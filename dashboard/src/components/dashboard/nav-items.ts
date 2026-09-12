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
} from "lucide-react";

export interface NavItem {
  label: string;
  href: string;
  icon: LucideIcon;
}

/**
 * Halaman-halaman ini datanya sungguhan ada: heartbeat dan event sejak
 * M2–M4, invoice/nominal unik/matching sejak sub-project 3 fase 1 (lihat
 * docs/superpowers/specs/2026-09-12-invoice-nominal-matching-design.md).
 * Webhooks, License, dan Settings menunggu fase 2 dan sistem lisensi —
 * menampilkannya sebagai link aktif sekarang berarti membangun halaman
 * untuk data yang bentuknya belum pasti.
 */
export const MONITORING_ITEMS: NavItem[] = [
  { label: "Overview", href: "/", icon: LayoutDashboard },
  { label: "Events", href: "/events", icon: Bell },
];

export const INTEGRATION_ITEMS: NavItem[] = [
  { label: "Devices", href: "/devices", icon: Smartphone },
];

export const GATEWAY_ITEMS: NavItem[] = [
  { label: "Transactions", href: "/transactions", icon: Receipt },
  { label: "API Keys", href: "/api-keys", icon: KeyRound },
];

/**
 * Ditampilkan, tidak disembunyikan — supaya bentuk akhir produk tetap
 * terlihat (sesuai dashboard-spec §16, pola yang sama dipakai untuk
 * connector DANA/OVO yang belum diimplementasikan).
 */
export const COMING_SOON_ITEMS: { label: string; icon: LucideIcon }[] = [
  { label: "Webhooks", icon: Webhook },
  { label: "License", icon: ShieldCheck },
  { label: "Logs", icon: ScrollText },
  { label: "Settings", icon: Settings },
];
