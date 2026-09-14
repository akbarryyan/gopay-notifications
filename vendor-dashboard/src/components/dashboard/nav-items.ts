import type { LucideIcon } from "lucide-react";
import { LayoutDashboard, Receipt, Settings, ShieldCheck, Users } from "lucide-react";

export interface NavItem {
  label: string;
  href: string;
  icon: LucideIcon;
}

// Vendor Console cuma dipakai Akbar sendiri, jadi tidak butuh pengelompokan
// bertingkat seperti Customer Dashboard (Monitoring/Integration/Gateway/
// System) -- halaman ini sudah seluruh isinya. Dashboard jadi halaman
// pertama (di "/", tujuan redirect setelah login) -- Accounts pindah ke
// "/accounts" supaya ada satu tempat "at a glance" sebelum masuk ke daftar.
export const NAV_ITEMS: NavItem[] = [
  { label: "Dashboard", href: "/", icon: LayoutDashboard },
  { label: "Accounts", href: "/accounts", icon: Users },
  { label: "Transactions", href: "/transactions", icon: Receipt },
  { label: "Audit Log", href: "/audit-log", icon: ShieldCheck },
  { label: "Settings", href: "/settings", icon: Settings },
];
