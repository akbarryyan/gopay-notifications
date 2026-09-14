"use client";

import { useEffect, useState, type FormEvent } from "react";
import toast from "react-hot-toast";
import {
  changeAdminPassword,
  fetchAdminMe,
  getAdminProfile,
  updateAdminProfile,
  updateStoredAdminProfile,
  type AdminProfile,
} from "@/lib/admin-auth";

function formatDate(value?: string | null) {
  if (!value) return "—";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "—";
  return date.toLocaleString("en-GB", {
    day: "2-digit",
    month: "short",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

function getInitials(profile: AdminProfile | null): string {
  if (!profile?.name?.trim()) return "AD";
  const parts = profile.name.trim().split(/\s+/).filter(Boolean);
  if (parts.length === 1) return parts[0].slice(0, 2).toUpperCase();
  return `${parts[0][0] ?? ""}${parts[1][0] ?? ""}`.toUpperCase();
}

function EyeIcon({ open }: { open: boolean }) {
  if (open) {
    return (
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={1.75} className="h-4.5 w-4.5">
        <path d="M2 12s3.6-7 10-7 10 7 10 7-3.6 7-10 7-10-7-10-7Z" strokeLinecap="round" strokeLinejoin="round" />
        <circle cx="12" cy="12" r="3" />
      </svg>
    );
  }
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={1.75} className="h-4.5 w-4.5">
      <path d="M3 3l18 18" strokeLinecap="round" />
      <path
        d="M10.6 5.2A10.6 10.6 0 0 1 12 5c6.4 0 10 7 10 7a16.9 16.9 0 0 1-3.9 4.6M6.7 6.7C4 8.5 2 12 2 12s3.6 7 10 7a9.9 9.9 0 0 0 4.1-.9"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
      <path d="M9.9 9.9a3 3 0 0 0 4.2 4.2" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  );
}

function ButtonSpinner() {
  return (
    <svg
      className="h-4 w-4 animate-spin"
      viewBox="0 0 24 24"
      fill="none"
      aria-hidden
    >
      <circle
        className="opacity-25"
        cx="12"
        cy="12"
        r="10"
        stroke="currentColor"
        strokeWidth="3"
      />
      <path
        className="opacity-90"
        fill="currentColor"
        d="M4 12a8 8 0 0 1 8-8v3a5 5 0 0 0-5 5H4Z"
      />
    </svg>
  );
}

/** Keep button loader visible long enough for smooth UX even when API is fast. */
const ACTION_MIN_DELAY_MS = 2000;

function sleep(ms: number) {
  return new Promise<void>((resolve) => {
    window.setTimeout(resolve, ms);
  });
}

async function withMinDelay<T>(promise: Promise<T>, minMs: number): Promise<T> {
  const started = Date.now();
  try {
    const result = await promise;
    const remaining = minMs - (Date.now() - started);
    if (remaining > 0) await sleep(remaining);
    return result;
  } catch (err) {
    const remaining = minMs - (Date.now() - started);
    if (remaining > 0) await sleep(remaining);
    throw err;
  }
}

export default function AdminProfilePage() {
  const [profile, setProfile] = useState<AdminProfile | null>(null);
  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [saving, setSaving] = useState(false);

  const [currentPassword, setCurrentPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [showCurrentPassword, setShowCurrentPassword] = useState(false);
  const [showNewPassword, setShowNewPassword] = useState(false);
  const [changingPassword, setChangingPassword] = useState(false);

  useEffect(() => {
    let cancelled = false;

    async function loadProfile() {
      // Prefer fresh data from backend; fall back to cached session profile.
      try {
        const remote = await fetchAdminMe();
        if (cancelled) return;
        setProfile(remote);
        setName(remote.name ?? "");
        setEmail(remote.email ?? "");
        updateStoredAdminProfile(remote);
      } catch {
        if (cancelled) return;
        const cached = getAdminProfile();
        setProfile(cached);
        setName(cached?.name ?? "");
        setEmail(cached?.email ?? "");
      }
    }

    void loadProfile();
    return () => {
      cancelled = true;
    };
  }, []);

  async function handleSaveProfile(e: FormEvent) {
    e.preventDefault();
    if (!name.trim()) {
      toast.error("Name is required.");
      return;
    }
    if (!email.trim()) {
      toast.error("Email is required.");
      return;
    }

    setSaving(true);
    try {
      const updated = await withMinDelay(
        updateAdminProfile(name.trim(), email.trim()),
        ACTION_MIN_DELAY_MS,
      );
      setProfile(updated);
      setName(updated.name);
      setEmail(updated.email);
      updateStoredAdminProfile(updated);
      toast.success("Profile updated successfully.");
    } catch (err) {
      const message =
        err instanceof Error ? err.message : "Failed to update profile.";
      toast.error(message);
    } finally {
      setSaving(false);
    }
  }

  async function handleChangePassword(e: FormEvent) {
    e.preventDefault();

    if (!currentPassword || !newPassword) {
      toast.error("Current password and new password are required.");
      return;
    }

    if (newPassword.length < 8) {
      toast.error("New password must be at least 8 characters.");
      return;
    }

    if (currentPassword === newPassword) {
      toast.error("New password must be different from current password.");
      return;
    }

    setChangingPassword(true);
    try {
      await withMinDelay(
        changeAdminPassword(currentPassword, newPassword),
        ACTION_MIN_DELAY_MS,
      );
      setCurrentPassword("");
      setNewPassword("");
      toast.success("Password changed successfully.");
    } catch (err) {
      const message =
        err instanceof Error ? err.message : "Failed to change password.";
      toast.error(message);
    } finally {
      setChangingPassword(false);
    }
  }

  return (
    <div className="w-full space-y-5">
      <div>
        <h1 className="text-[20px] font-semibold tracking-tight text-[#1f2a37]">
          My Profile
        </h1>
        <p className="mt-1 text-[13px] text-[#8a97a8]">
          View and manage your admin account details.
        </p>
      </div>

      <div className="rounded-xl border border-[#e8eef4] bg-white p-5 shadow-[0_1px_3px_rgba(16,38,73,0.06)] sm:p-6">
        <div className="flex flex-col gap-4 sm:flex-row sm:items-center">
          <span className="flex h-16 w-16 shrink-0 items-center justify-center rounded-full bg-linear-to-br from-[#7dd3fc] to-[#3b82f6] text-[18px] font-semibold text-white shadow-sm">
            {getInitials(profile)}
          </span>
          <div className="min-w-0">
            <p className="truncate text-[16px] font-semibold text-[#1f2a37]">
              {profile?.name ?? "Admin"}
            </p>
            <p className="mt-0.5 truncate text-[13px] text-[#8a97a8]">
              {profile?.email ?? "—"}
            </p>
            <span
              className={`mt-2 inline-flex rounded-full px-2.5 py-0.5 text-[11px] font-semibold ${
                profile?.is_active !== false
                  ? "bg-[#e8f8ee] text-[#2f9e5a]"
                  : "bg-[#fff1ed] text-[#e85d3b]"
              }`}
            >
              {profile?.is_active !== false ? "Active" : "Inactive"}
            </span>
          </div>
        </div>
      </div>

      <form
        onSubmit={handleSaveProfile}
        className="rounded-xl border border-[#e8eef4] bg-white p-5 shadow-[0_1px_3px_rgba(16,38,73,0.06)] sm:p-6"
      >
        <h2 className="text-[15px] font-semibold text-[#1f2a37]">
          Account information
        </h2>
        <p className="mt-1 text-[12.5px] text-[#8a97a8]">
          These details are used for your admin panel identity.
        </p>

        <div className="mt-5 grid grid-cols-1 gap-4 sm:grid-cols-2">
          <div>
            <label
              htmlFor="profile-name"
              className="mb-1.5 block text-[13px] font-medium text-[#3d4b5c]"
            >
              Full name
            </label>
            <input
              id="profile-name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              className="block w-full rounded-lg border border-[#e8eef4] bg-[#f8fafc] px-3.5 py-2.5 text-[13.5px] text-[#1f2a37] outline-none transition focus:border-[#3b9eff] focus:bg-white focus:ring-2 focus:ring-[#3b9eff]/15"
            />
          </div>

          <div>
            <label
              htmlFor="profile-email"
              className="mb-1.5 block text-[13px] font-medium text-[#3d4b5c]"
            >
              Email
            </label>
            <input
              id="profile-email"
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              className="block w-full rounded-lg border border-[#e8eef4] bg-[#f8fafc] px-3.5 py-2.5 text-[13.5px] text-[#1f2a37] outline-none transition focus:border-[#3b9eff] focus:bg-white focus:ring-2 focus:ring-[#3b9eff]/15"
            />
          </div>
        </div>

        <div className="mt-5 grid grid-cols-1 gap-4 border-t border-[#eef2f6] pt-5 sm:grid-cols-2">
          <div className="rounded-lg bg-[#f8fafc] px-4 py-3">
            <p className="text-[11px] font-semibold uppercase tracking-wide text-[#8a97a8]">
              Last login
            </p>
            <p className="mt-1 text-[13.5px] font-medium text-[#1f2a37]">
              {formatDate(profile?.last_login_at)}
            </p>
          </div>
          <div className="rounded-lg bg-[#f8fafc] px-4 py-3">
            <p className="text-[11px] font-semibold uppercase tracking-wide text-[#8a97a8]">
              Account created
            </p>
            <p className="mt-1 text-[13.5px] font-medium text-[#1f2a37]">
              {formatDate(profile?.created_at)}
            </p>
          </div>
        </div>

        <div className="mt-6 flex justify-end">
          <button
            type="submit"
            disabled={saving}
            aria-busy={saving}
            className="inline-flex min-w-37 items-center justify-center gap-2 rounded-full bg-[#ff5e16] px-5 py-2.5 text-[13.5px] font-semibold text-white shadow-[0_4px_14px_rgba(255,94,22,0.28)] transition-all duration-200 hover:bg-[#ef5510] disabled:cursor-not-allowed disabled:opacity-85"
          >
            {saving ? (
              <>
                <ButtonSpinner />
                <span>Saving...</span>
              </>
            ) : (
              "Save changes"
            )}
          </button>
        </div>
      </form>

      <form
        onSubmit={handleChangePassword}
        className="rounded-xl border border-[#e8eef4] bg-white p-5 shadow-[0_1px_3px_rgba(16,38,73,0.06)] sm:p-6"
      >
        <h2 className="text-[15px] font-semibold text-[#1f2a37]">
          Change password
        </h2>
        <p className="mt-1 text-[12.5px] text-[#8a97a8]">
          Use a strong password with at least 8 characters.
        </p>

        <div className="mt-5 grid grid-cols-1 gap-4 sm:grid-cols-2">
          <div>
            <label
              htmlFor="current-password"
              className="mb-1.5 block text-[13px] font-medium text-[#3d4b5c]"
            >
              Current password
            </label>
            <div className="relative">
              <input
                id="current-password"
                type={showCurrentPassword ? "text" : "password"}
                autoComplete="current-password"
                value={currentPassword}
                onChange={(e) => setCurrentPassword(e.target.value)}
                placeholder="••••••••"
                className="block w-full rounded-lg border border-[#e8eef4] bg-[#f8fafc] px-3.5 py-2.5 pr-11 text-[13.5px] text-[#1f2a37] outline-none transition placeholder:text-[#b0bbc8] focus:border-[#3b9eff] focus:bg-white focus:ring-2 focus:ring-[#3b9eff]/15"
              />
              <button
                type="button"
                onClick={() => setShowCurrentPassword((v) => !v)}
                className="absolute inset-y-0 right-0 flex w-11 items-center justify-center text-[#8a97a8] transition hover:text-[#3d4b5c]"
                aria-label={showCurrentPassword ? "Hide current password" : "Show current password"}
              >
                <EyeIcon open={showCurrentPassword} />
              </button>
            </div>
          </div>

          <div>
            <label
              htmlFor="new-password"
              className="mb-1.5 block text-[13px] font-medium text-[#3d4b5c]"
            >
              New password
            </label>
            <div className="relative">
              <input
                id="new-password"
                type={showNewPassword ? "text" : "password"}
                autoComplete="new-password"
                value={newPassword}
                onChange={(e) => setNewPassword(e.target.value)}
                placeholder="••••••••"
                className="block w-full rounded-lg border border-[#e8eef4] bg-[#f8fafc] px-3.5 py-2.5 pr-11 text-[13.5px] text-[#1f2a37] outline-none transition placeholder:text-[#b0bbc8] focus:border-[#3b9eff] focus:bg-white focus:ring-2 focus:ring-[#3b9eff]/15"
              />
              <button
                type="button"
                onClick={() => setShowNewPassword((v) => !v)}
                className="absolute inset-y-0 right-0 flex w-11 items-center justify-center text-[#8a97a8] transition hover:text-[#3d4b5c]"
                aria-label={showNewPassword ? "Hide new password" : "Show new password"}
              >
                <EyeIcon open={showNewPassword} />
              </button>
            </div>
          </div>
        </div>

        <div className="mt-6 flex justify-end">
          <button
            type="submit"
            disabled={changingPassword}
            aria-busy={changingPassword}
            className="inline-flex min-w-42 items-center justify-center gap-2 rounded-full bg-[#06163a] px-5 py-2.5 text-[13.5px] font-semibold text-white shadow-[0_4px_14px_rgba(6,22,58,0.22)] transition-all duration-200 hover:bg-[#0a1f4d] disabled:cursor-not-allowed disabled:opacity-85"
          >
            {changingPassword ? (
              <>
                <ButtonSpinner />
                <span>Updating...</span>
              </>
            ) : (
              "Update password"
            )}
          </button>
        </div>
      </form>
    </div>
  );
}
