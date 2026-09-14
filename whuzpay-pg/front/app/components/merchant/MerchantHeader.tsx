"use client";

import { useCallback, useEffect, useRef, useState, type SVGProps } from "react";
import { usePathname, useRouter } from "next/navigation";
import toast from "react-hot-toast";
import {
  clearMerchantSession,
  getMerchantProfile,
  type MerchantProfile,
} from "@/lib/merchant-auth";
import { useMerchantEnvironment } from "@/lib/use-merchant-environment";
import { fetchMerchantNotifications } from "@/lib/merchant-api";
import { IconBell, IconMenu, IconUser } from "../admin/icons";
import SignOutConfirmModal from "../admin/SignOutConfirmModal";
import MerchantNotificationsDrawer from "./MerchantNotificationsDrawer";

type Props = {
  sidebarCollapsed: boolean;
  onMenuClick: () => void;
};

function getInitials(profile: MerchantProfile | null): string {
  if (!profile?.name?.trim()) return "M";
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

export default function MerchantHeader({ onMenuClick }: Props) {
  const router = useRouter();
  const pathname = usePathname();
  const { environment, setEnvironment } = useMerchantEnvironment();
  const [menuOpen, setMenuOpen] = useState(false);
  const [signOutOpen, setSignOutOpen] = useState(false);
  const [signingOut, setSigningOut] = useState(false);
  const [notificationsOpen, setNotificationsOpen] = useState(false);
  const [notificationCount, setNotificationCount] = useState(0);
  const [profile, setProfile] = useState<MerchantProfile | null>(null);
  const menuRef = useRef<HTMLDivElement>(null);

  const handleNotificationCount = useCallback((count: number) => {
    setNotificationCount(count);
  }, []);

  useEffect(() => {
    // Reads localStorage — see MerchantAuthGate.tsx for why this must stay
    // an effect rather than a render-time computation (SSR hydration
    // safety).
    // eslint-disable-next-line react-hooks/set-state-in-effect
    setProfile(getMerchantProfile());
  }, [pathname]);

  useEffect(() => {
    let cancelled = false;
    async function loadBadge() {
      try {
        const data = await fetchMerchantNotifications(environment, 20);
        if (!cancelled) setNotificationCount(data.attention_count ?? 0);
      } catch {
        /* ignore badge errors */
      }
    }
    void loadBadge();
    return () => {
      cancelled = true;
    };
  }, [environment, pathname]);

  useEffect(() => {
    if (!menuOpen) return;
    function handlePointerDown(event: MouseEvent) {
      if (!menuRef.current?.contains(event.target as Node)) setMenuOpen(false);
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

  function handleConfirmSignOut() {
    setSigningOut(true);
    window.setTimeout(() => {
      clearMerchantSession();
      setSigningOut(false);
      setSignOutOpen(false);
      toast.success("Signed out successfully.");
      router.replace("/login");
    }, 700);
  }

  const initials = getInitials(profile);
  const displayName = profile?.name ?? "Merchant";
  const displayEmail = profile?.email ?? "merchant@example.com";

  return (
    <>
      <header className="sticky top-0 z-30 flex h-16 shrink-0 items-center justify-between border-b border-[#e8eef4] bg-white px-4 sm:px-6">
        <button
          type="button"
          onClick={onMenuClick}
          className="flex h-9 w-9 items-center justify-center rounded-md text-[#6b7c93] transition hover:bg-[#f4f7fb] hover:text-[#06163a]"
          aria-label="Toggle sidebar"
        >
          <IconMenu className="h-5 w-5" />
        </button>

        <div className="flex items-center gap-2 sm:gap-3">
          <button
            type="button"
            onClick={() => setNotificationsOpen(true)}
            className="relative flex h-9 w-9 items-center justify-center rounded-md text-[#6b7c93] transition hover:bg-[#f4f7fb] hover:text-[#06163a]"
            aria-label="Notifications"
          >
            <IconBell className="h-5 w-5" />
            {notificationCount > 0 && (
              <span className="absolute top-1 right-1 flex h-4 min-w-4 items-center justify-center rounded-full bg-[#e85d3b] px-1 text-[9px] font-bold text-white">
                {notificationCount > 9 ? "9+" : notificationCount}
              </span>
            )}
          </button>

          <div
            className="inline-flex rounded-full border border-[#e8eef4] bg-[#f8fafc] p-0.5"
            role="group"
            aria-label="Environment"
          >
            <button
              type="button"
              onClick={() => setEnvironment("sandbox")}
              className={`rounded-full px-3 py-1.5 text-[12px] font-semibold transition ${
                environment === "sandbox"
                  ? "bg-[#fff7e6] text-[#c27a00] shadow-sm"
                  : "text-[#8a97a8] hover:text-[#3d4b5c]"
              }`}
            >
              Sandbox
            </button>
            <button
              type="button"
              onClick={() => setEnvironment("production")}
              className={`rounded-full px-3 py-1.5 text-[12px] font-semibold transition ${
                environment === "production"
                  ? "bg-[#e8f8ee] text-[#2f9e5a] shadow-sm"
                  : "text-[#8a97a8] hover:text-[#3d4b5c]"
              }`}
            >
              Production
            </button>
          </div>

        <div className="relative" ref={menuRef}>
          <button
            type="button"
            onClick={() => setMenuOpen((v) => !v)}
            className="flex items-center gap-2.5 rounded-full py-1 pr-2 pl-1 transition hover:bg-[#f4f7fb]"
          >
            <span className="flex h-9 w-9 items-center justify-center rounded-full bg-[#06163a] text-[12px] font-semibold text-white">
              {initials}
            </span>
            <span className="hidden text-left sm:block">
              <span className="block text-[13px] font-semibold text-[#1f2a37]">
                {displayName}
              </span>
              <span className="block text-[11px] text-[#8a97a8]">
                {displayEmail}
              </span>
            </span>
          </button>

          {menuOpen && (
            <div className="absolute top-full right-0 z-50 mt-2 w-56 overflow-hidden rounded-xl border border-[#e8eef4] bg-white py-1 shadow-lg">
              <button
                type="button"
                className="flex w-full items-center gap-2.5 px-4 py-2.5 text-left text-[13px] text-[#3d4b5c] hover:bg-[#f4f7fb]"
                onClick={() => {
                  setMenuOpen(false);
                  router.push("/dashboard/profile");
                }}
              >
                <IconUser className="h-4 w-4" />
                Profile
              </button>
              <button
                type="button"
                className="flex w-full items-center gap-2.5 px-4 py-2.5 text-left text-[13px] text-[#3d4b5c] hover:bg-[#f4f7fb]"
                onClick={() => {
                  setMenuOpen(false);
                  router.push("/dashboard/settings");
                }}
              >
                <IconSettings className="h-4 w-4" />
                Settings
              </button>
              <div className="my-1 border-t border-[#eef2f6]" />
              <button
                type="button"
                className="flex w-full items-center gap-2.5 px-4 py-2.5 text-left text-[13px] text-[#e85d3b] hover:bg-[#fff5f2]"
                onClick={() => {
                  setMenuOpen(false);
                  setSignOutOpen(true);
                }}
              >
                <IconLogout className="h-4 w-4" />
                Sign out
              </button>
            </div>
          )}
        </div>
        </div>
      </header>

      <MerchantNotificationsDrawer
        open={notificationsOpen}
        onOpenChange={setNotificationsOpen}
        environment={environment}
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
