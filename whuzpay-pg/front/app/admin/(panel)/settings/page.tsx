"use client";

import { useState, type FormEvent } from "react";
import toast from "react-hot-toast";

type ToggleRowProps = {
  title: string;
  description: string;
  checked: boolean;
  onChange: (value: boolean) => void;
};

function ToggleRow({ title, description, checked, onChange }: ToggleRowProps) {
  return (
    <div className="flex items-start justify-between gap-4 border-b border-[#eef2f6] py-4 last:border-0">
      <div className="min-w-0">
        <p className="text-[13.5px] font-semibold text-[#1f2a37]">{title}</p>
        <p className="mt-1 text-[12.5px] leading-relaxed text-[#8a97a8]">
          {description}
        </p>
      </div>
      <button
        type="button"
        role="switch"
        aria-checked={checked}
        onClick={() => onChange(!checked)}
        className={`relative mt-0.5 h-6 w-11 shrink-0 rounded-full transition-colors duration-200 ${
          checked ? "bg-[#ff5e16]" : "bg-[#d5dee8]"
        }`}
      >
        <span
          className={`absolute top-0.5 left-0.5 h-5 w-5 rounded-full bg-white shadow transition-transform duration-200 ease-out ${
            checked ? "translate-x-5" : "translate-x-0"
          }`}
        />
      </button>
    </div>
  );
}

export default function AdminSettingsPage() {
  const [emailAlerts, setEmailAlerts] = useState(true);
  const [paymentAlerts, setPaymentAlerts] = useState(true);
  const [webhookAlerts, setWebhookAlerts] = useState(false);
  const [compactSidebar, setCompactSidebar] = useState(false);
  const [timezone, setTimezone] = useState("Asia/Jakarta");
  const [language, setLanguage] = useState("en");
  const [saving, setSaving] = useState(false);

  async function handleSave(e: FormEvent) {
    e.preventDefault();
    setSaving(true);
    // UI-only for now — settings API not implemented yet
    window.setTimeout(() => {
      setSaving(false);
      toast.success("Settings saved locally. API sync coming soon.");
    }, 900);
  }

  return (
    <div className="w-full space-y-5">
      <div>
        <h1 className="text-[20px] font-semibold tracking-tight text-[#1f2a37]">
          Settings
        </h1>
        <p className="mt-1 text-[13px] text-[#8a97a8]">
          Configure notifications, preferences, and panel behavior.
        </p>
      </div>

      <form onSubmit={handleSave} className="space-y-5">
        <section className="rounded-xl border border-[#e8eef4] bg-white p-5 shadow-[0_1px_3px_rgba(16,38,73,0.06)] sm:p-6">
          <h2 className="text-[15px] font-semibold text-[#1f2a37]">
            Notifications
          </h2>
          <p className="mt-1 text-[12.5px] text-[#8a97a8]">
            Choose which alerts you want to receive as an admin.
          </p>

          <div className="mt-2">
            <ToggleRow
              title="Email alerts"
              description="Receive important platform updates via email."
              checked={emailAlerts}
              onChange={setEmailAlerts}
            />
            <ToggleRow
              title="Payment alerts"
              description="Get notified when payments succeed, fail, or expire."
              checked={paymentAlerts}
              onChange={setPaymentAlerts}
            />
            <ToggleRow
              title="Webhook failure alerts"
              description="Alert when provider webhook processing fails repeatedly."
              checked={webhookAlerts}
              onChange={setWebhookAlerts}
            />
          </div>
        </section>

        <section className="rounded-xl border border-[#e8eef4] bg-white p-5 shadow-[0_1px_3px_rgba(16,38,73,0.06)] sm:p-6">
          <h2 className="text-[15px] font-semibold text-[#1f2a37]">
            Preferences
          </h2>
          <p className="mt-1 text-[12.5px] text-[#8a97a8]">
            Personalize how the admin panel looks and behaves.
          </p>

          <div className="mt-5 grid grid-cols-1 gap-4 sm:grid-cols-2">
            <div>
              <label htmlFor="settings-language" className="mb-1.5 block text-[13px] font-medium text-[#3d4b5c]">
                Language
              </label>
              <select
                id="settings-language"
                value={language}
                onChange={(e) => setLanguage(e.target.value)}
                className="block w-full rounded-lg border border-[#e8eef4] bg-[#f8fafc] px-3.5 py-2.5 text-[13.5px] text-[#1f2a37] outline-none transition focus:border-[#3b9eff] focus:bg-white focus:ring-2 focus:ring-[#3b9eff]/15"
              >
                <option value="en">English</option>
                <option value="id">Bahasa Indonesia</option>
              </select>
            </div>

            <div>
              <label htmlFor="settings-timezone" className="mb-1.5 block text-[13px] font-medium text-[#3d4b5c]">
                Timezone
              </label>
              <select
                id="settings-timezone"
                value={timezone}
                onChange={(e) => setTimezone(e.target.value)}
                className="block w-full rounded-lg border border-[#e8eef4] bg-[#f8fafc] px-3.5 py-2.5 text-[13.5px] text-[#1f2a37] outline-none transition focus:border-[#3b9eff] focus:bg-white focus:ring-2 focus:ring-[#3b9eff]/15"
              >
                <option value="Asia/Jakarta">Asia/Jakarta (WIB)</option>
                <option value="Asia/Makassar">Asia/Makassar (WITA)</option>
                <option value="Asia/Jayapura">Asia/Jayapura (WIT)</option>
                <option value="UTC">UTC</option>
              </select>
            </div>
          </div>

          <div className="mt-2">
            <ToggleRow
              title="Compact sidebar by default"
              description="Start the admin panel with a collapsed icon-only sidebar."
              checked={compactSidebar}
              onChange={setCompactSidebar}
            />
          </div>
        </section>

        <section className="rounded-xl border border-[#e8eef4] bg-white p-5 shadow-[0_1px_3px_rgba(16,38,73,0.06)] sm:p-6">
          <h2 className="text-[15px] font-semibold text-[#1f2a37]">Security</h2>
          <p className="mt-1 text-[12.5px] text-[#8a97a8]">
            Session and access preferences for your admin account.
          </p>

          <div className="mt-5 space-y-3">
            <div className="rounded-lg border border-[#eef2f6] bg-[#f8fafc] px-4 py-3">
              <p className="text-[13px] font-semibold text-[#1f2a37]">
                Session duration
              </p>
              <p className="mt-1 text-[12.5px] text-[#8a97a8]">
                Access tokens currently expire after 24 hours.
              </p>
            </div>
            <div className="rounded-lg border border-[#eef2f6] bg-[#f8fafc] px-4 py-3">
              <p className="text-[13px] font-semibold text-[#1f2a37]">
                Password change
              </p>
              <p className="mt-1 text-[12.5px] text-[#8a97a8]">
                Password update API is not available yet.
              </p>
            </div>
          </div>
        </section>

        <div className="flex justify-end">
          <button
            type="submit"
            disabled={saving}
            className="inline-flex items-center justify-center gap-2 rounded-lg bg-[#ff5e16] px-5 py-2.5 text-[13.5px] font-semibold text-white shadow-[0_4px_14px_rgba(255,94,22,0.28)] transition hover:bg-[#ef5510] disabled:cursor-not-allowed disabled:opacity-70"
          >
            {saving ? "Saving..." : "Save settings"}
          </button>
        </div>
      </form>
    </div>
  );
}
