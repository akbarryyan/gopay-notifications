"use client";

import { useCallback, useEffect, useRef, useState, type SVGProps } from "react";
import { usePathname, useRouter } from "next/navigation";
import toast from "react-hot-toast";
import {
  clearAdminSession,
  getAdminProfile,
  type AdminProfile,
} from "@/lib/admin-auth";
import { IconBell, IconChat, IconMenu, IconUser } from "./icons";
import NotificationsDrawer from "./NotificationsDrawer";
import SignOutConfirmModal from "./SignOutConfirmModal";

type AdminHeaderProps = {
  sidebarCollapsed: boolean;
  onMenuClick: () => void;
};

function getInitials(profile: AdminProfile | null): string {
  if (!profile?.name?.trim()) return "AD";
  const parts = profile.name.trim().split(/\s+/).filter(Boolean);
  if (parts.length === 1) return parts[0].slice(0, 2).toUpperCase();
  return `${parts[0][0] ?? ""}${parts[1][0] ?? ""}`.toUpperCase();
}

function IconSettings(props: SVGProps<SVGSVGElement>) {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={1.75} {...props}>
      <circle cx="12" cy="12" r="3" />
      <path
        d="M19.4 15a1.7 1.7 0 0 0 .3 1.8l.1.1a2 2 0 1 1-2.8 2.8l-.1-.1a1.7 1.7 0 0 0-1.8-.3 1.7 1.7 0 0 0-1 1.5V21a2 2 0 1 1-4 0v-.1a1.7 1.7 0 0 0-1-1.5 1.7 1.7 0 0 0-1.8.3l-.1.1a2 2 0 1 1-2.8-2.8l.1-.1a1.7 1.7 0 0 0 .3-1.8 1.7 1.7 0 0 0-1.5-1H3a2 2 0 1 1 0-4h.1a1.7 1.7 0 0 0 1.5-1 1.7 1.7 0 0 0-.3-1.8l-.1-.1a2 2 0 1 1 2.8-2.8l.1.1a1.7 1.7 0 0 0 1.8.3H9a1.7 1.7 0 0 0 1-1.5V3a2 2 0 1 1 4 0v.1a1.7 1.7 0 0 0 1 1.5 1.7 1.7 0 0 0 1.8-.3l.1-.1a2 2 0 1 1 2.8 2.8l-.1.1a1.7 1.7 0 0 0-.3 1.8V9c.3.6.9 1 1.6 1H21a2 2 0 1 1 0 4h-.1a1.7 1.7 0 0 0-1.5 1Z"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  );
}

function IconLogout(props: SVGProps<SVGSVGElement>) {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={1.75} {...props}>
      <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4" strokeLinecap="round" strokeLinejoin="round" />
      <path d="M16 17l5-5-5-5" strokeLinecap="round" strokeLinejoin="round" />
      <path d="M21 12H9" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  );
}

export default function AdminHeader({
  sidebarCollapsed,
  onMenuClick,
}: AdminHeaderProps) {
  const router = useRouter();
  const pathname = usePathname();
  const [menuOpen, setMenuOpen] = useState(false);
  const [signOutOpen, setSignOutOpen] = useState(false);
  const [signingOut, setSigningOut] = useState(false);
  const [notificationsOpen, setNotificationsOpen] = useState(false);
  const [notificationCount, setNotificationCount] = useState(0);
  const [profile, setProfile] = useState<AdminProfile | null>(null);
  const menuRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    // Reads localStorage — see AdminAuthGate.tsx for why this must stay an
    // effect rather than a render-time computation (SSR hydration safety).
    // eslint-disable-next-line react-hooks/set-state-in-effect
    setProfile(getAdminProfile());
  }, [pathname]);

  useEffect(() => {
    if (!menuOpen) return;

    function handlePointerDown(event: MouseEvent) {
      if (!menuRef.current?.contains(event.target as Node)) {
        setMenuOpen(false);
      }
    }

    function handleKeyDown(event: KeyboardEvent) {
      if (event.key === "Escape") setMenuOpen(false);
    }

    document.addEventListener("mousedown", handlePointerDown);
    document.addEventListener("keydown", handleKeyDown);
    return () => {
      document.removeEventListener("mousedown", handlePointerDown);
      document.removeEventListener("keydown", handleKeyDown);
    };
  }, [menuOpen]);

  const handleNotificationCount = useCallback((count: number) => {
    setNotificationCount(count);
  }, []);

  function openSignOutModal() {
    setMenuOpen(false);
    setSignOutOpen(true);
  }

  function handleConfirmSignOut() {
    setSigningOut(true);
    window.setTimeout(() => {
      clearAdminSession();
      setSigningOut(false);
      setSignOutOpen(false);
      toast.success("Signed out successfully.");
      router.replace("/admin/login");
    }, 700);
  }

  const initials = getInitials(profile);
  const displayName = profile?.name ?? "Admin";
  const displayEmail = profile?.email ?? "admin@pg-aggregator.local";

  return (
    <>
      <header className="sticky top-0 z-30 flex h-16 shrink-0 items-center justify-between border-b border-[#e8eef4] bg-white px-4 sm:px-6">
        <button
          type="button"
          onClick={onMenuClick}
          className="flex h-9 w-9 items-center justify-center rounded-md text-[#6b7c93] transition hover:bg-[#f4f7fb] hover:text-[#06163a]"
          aria-label={sidebarCollapsed ? "Expand sidebar" : "Collapse sidebar"}
          aria-expanded={!sidebarCollapsed}
        >
          <IconMenu className="h-5 w-5" />
        </button>

        <div className="flex items-center gap-1 sm:gap-2">
          <button
            type="button"
            onClick={() => setNotificationsOpen(true)}
            className="relative flex h-9 w-9 items-center justify-center rounded-full text-[#6b7c93] transition hover:bg-[#f4f7fb] hover:text-[#06163a]"
            aria-label={
              notificationCount > 0
                ? `Notifications (${notificationCount})`
                : "Notifications"
            }
          >
            <IconBell className="h-4.5 w-4.5" />
            {notificationCount > 0 && (
              <span
                className="pointer-events-none absolute top-0.5 right-0.5 flex h-3.5 min-w-3.5 translate-x-0.5 -translate-y-0.5 items-center justify-center rounded-full bg-[#ff5e16] px-0.5 text-[8px] leading-none font-bold text-white ring-2 ring-white"
                aria-hidden
              >
                {notificationCount > 9 ? "9+" : notificationCount}
              </span>
            )}
          </button>

          <button
            type="button"
            className="relative flex h-9 w-9 items-center justify-center rounded-full text-[#6b7c93] transition hover:bg-[#f4f7fb] hover:text-[#06163a]"
            aria-label="Messages"
          >
            <IconChat className="h-4.5 w-4.5" />
            <span className="absolute right-1.5 top-1.5 h-2 w-2 rounded-full bg-[#3b82f6] ring-2 ring-white" />
          </button>

          <div className="relative ml-1" ref={menuRef}>
            <button
              type="button"
              onClick={() => setMenuOpen((open) => !open)}
              className={`flex h-9 w-9 items-center justify-center overflow-hidden rounded-full ring-2 transition ${
                menuOpen
                  ? "ring-[#3b9eff]/50 ring-offset-2 ring-offset-white"
                  : "ring-[#e8eef4] hover:ring-[#cfd9e6]"
              }`}
              aria-label="Account menu"
              aria-haspopup="menu"
              aria-expanded={menuOpen}
            >
              <span className="flex h-full w-full items-center justify-center bg-linear-to-br from-[#7dd3fc] to-[#3b82f6] text-[11px] font-semibold text-white">
                {initials}
              </span>
            </button>

            <div
              role="menu"
              aria-hidden={!menuOpen}
              className={`absolute right-0 z-50 mt-2 w-60 origin-top-right rounded-xl border border-[#e8eef4] bg-white shadow-[0_12px_40px_rgba(16,38,73,0.12)] transition-[opacity,transform,visibility] duration-200 ease-out ${
                menuOpen
                  ? "visible translate-y-0 scale-100 opacity-100"
                  : "invisible pointer-events-none -translate-y-1 scale-95 opacity-0"
              }`}
            >
              <div className="border-b border-[#eef2f6] px-4 py-3">
                <p className="truncate text-[13px] font-semibold text-[#1f2a37]">
                  {displayName}
                </p>
                <p className="mt-0.5 truncate text-[12px] text-[#8a97a8]">
                  {displayEmail}
                </p>
              </div>

              <div className="p-1.5">
                <button
                  type="button"
                  role="menuitem"
                  onClick={() => {
                    setMenuOpen(false);
                    router.push("/admin/profile");
                  }}
                  className="flex w-full items-center gap-2.5 rounded-lg px-3 py-2.5 text-left text-[13px] text-[#3d4b5c] transition hover:bg-[#f4f7fb] hover:text-[#06163a]"
                >
                  <IconUser className="h-4 w-4 shrink-0 text-[#8a97a8]" />
                  My Profile
                </button>

                <button
                  type="button"
                  role="menuitem"
                  onClick={() => {
                    setMenuOpen(false);
                    router.push("/admin/settings");
                  }}
                  className="flex w-full items-center gap-2.5 rounded-lg px-3 py-2.5 text-left text-[13px] text-[#3d4b5c] transition hover:bg-[#f4f7fb] hover:text-[#06163a]"
                >
                  <IconSettings className="h-4 w-4 shrink-0 text-[#8a97a8]" />
                  Settings
                </button>
              </div>

              <div className="border-t border-[#eef2f6] p-1.5">
                <button
                  type="button"
                  role="menuitem"
                  onClick={openSignOutModal}
                  className="flex w-full items-center gap-2.5 rounded-lg px-3 py-2.5 text-left text-[13px] font-medium text-[#e85d3b] transition hover:bg-[#fff5f2]"
                >
                  <IconLogout className="h-4 w-4 shrink-0" />
                  Sign out
                </button>
              </div>
            </div>
          </div>
        </div>
      </header>

      <NotificationsDrawer
        open={notificationsOpen}
        onOpenChange={setNotificationsOpen}
        onCountChange={handleNotificationCount}
      />

      <SignOutConfirmModal
        open={signOutOpen}
        loading={signingOut}
        onClose={() => {
          if (!signingOut) setSignOutOpen(false);
        }}
        onConfirm={handleConfirmSignOut}
      />
    </>
  );
}
