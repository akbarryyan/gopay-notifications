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
 * Hanya tiga halaman ini yang datanya sungguhan ada: heartbeat dan event
 * sudah mengalir dari perangkat sejak M2–M4. Transactions, Webhooks, License,
 * dan sisanya menunggu sub-project 3 dan sistem lisensi — menampilkannya
 * sebagai link aktif sekarang berarti membangun halaman untuk data yang
 * bentuknya belum pasti.
 */
export const MONITORING_ITEMS: NavItem[] = [
  { label: "Overview", href: "/", icon: LayoutDashboard },
  { label: "Events", href: "/events", icon: Bell },
];

export const INTEGRATION_ITEMS: NavItem[] = [
  { label: "Devices", href: "/devices", icon: Smartphone },
];

/**
 * Ditampilkan, tidak disembunyikan — supaya bentuk akhir produk tetap
 * terlihat (sesuai dashboard-spec §16, pola yang sama dipakai untuk
 * connector DANA/OVO yang belum diimplementasikan).
 */
export const COMING_SOON_ITEMS: { label: string; icon: LucideIcon }[] = [
  { label: "Transactions", icon: Receipt },
  { label: "Webhooks", icon: Webhook },
  { label: "API Keys", icon: KeyRound },
  { label: "License", icon: ShieldCheck },
  { label: "Logs", icon: ScrollText },
  { label: "Settings", icon: Settings },
];
