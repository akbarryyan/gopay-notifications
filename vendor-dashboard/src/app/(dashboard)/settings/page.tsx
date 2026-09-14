"use client";

import { NotificationSettingsCard } from "@/components/dashboard/notification-settings-card";

/**
 * Pengaturan notifikasi ke customer (SMTP + bot Telegram, disimpan di
 * database). Ganti password vendor pindah ke halaman Profile, lewat menu
 * avatar di header.
 */
export default function SettingsPage() {
  return (
    <div className="flex flex-col gap-6">
      <div>
        <h1 className="text-2xl font-semibold">Settings</h1>
        <p className="text-sm text-muted-foreground">
          Pengaturan pengiriman notifikasi ke customer.
        </p>
      </div>

      <div className="max-w-3xl">
        <NotificationSettingsCard />
      </div>
    </div>
  );
}
