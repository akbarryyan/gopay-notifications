import type { Account } from "@/lib/api";

// Dipakai bersama oleh halaman Accounts dan Dashboard -- satu tempat supaya
// warna status tidak diam-diam berbeda antara dua halaman yang menampilkan
// account yang sama.
export const STATUS_BADGE: Record<Account["status"], string> = {
  active: "border-transparent bg-emerald-500/15 text-emerald-700 dark:text-emerald-400",
  expiring: "border-transparent bg-amber-500/15 text-amber-700 dark:text-amber-400",
  expired: "border-transparent bg-red-500/15 text-red-700 dark:text-red-400",
  suspended: "border-transparent bg-red-500/15 text-red-700 dark:text-red-400",
  revoked: "border-transparent bg-red-500/15 text-red-700 dark:text-red-400",
};
