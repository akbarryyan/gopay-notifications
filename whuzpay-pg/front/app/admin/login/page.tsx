"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, useState, type FormEvent } from "react";
import toast from "react-hot-toast";
import {
  consumeAdminAuthNotice,
  consumeAdminRedirectPath,
  getAdminToken,
  loginAdmin,
  saveAdminSession,
} from "@/lib/admin-auth";

function EyeIcon({ open }: { open: boolean }) {
  if (open) {
    return (
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={1.75} className="h-5 w-5">
        <path d="M2 12s3.6-7 10-7 10 7 10 7-3.6 7-10 7-10-7-10-7Z" strokeLinecap="round" strokeLinejoin="round" />
        <circle cx="12" cy="12" r="3" />
      </svg>
    );
  }
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={1.75} className="h-5 w-5">
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

function LectorLogo({ className = "" }: { className?: string }) {
  return (
    <div className={`flex items-center gap-2.5 ${className}`}>
      <span className="flex h-9 w-9 items-center justify-center rounded-md bg-[#ff5e16]">
        <svg viewBox="0 0 24 24" className="h-4.5 w-4.5 text-white" fill="currentColor">
          <path d="M4 12c3-6 6-8 8-8s5 2 8 8c-3 6-6 8-8 8s-5-2-8-8Z" />
          <path
            d="M9 12c1.2-2.2 2.3-3 3-3s1.8.8 3 3c-1.2 2.2-2.3 3-3 3s-1.8-.8-3-3Z"
            fill="#ffffff"
          />
        </svg>
      </span>
      <span className="text-[24px] font-semibold tracking-tight text-[#06163a]">
        Lector<span className="text-[#ff5e16]">.</span>
      </span>
    </div>
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

const LOGIN_MIN_DELAY_MS = 2000;

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

export default function AdminLoginPage() {
  const router = useRouter();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [showPassword, setShowPassword] = useState(false);
  const [remember, setRemember] = useState(false);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    // If already signed in, go straight to admin panel
    if (getAdminToken()) {
      consumeAdminAuthNotice();
      router.replace(consumeAdminRedirectPath("/admin"));
      return;
    }

    const notice = consumeAdminAuthNotice();
    if (notice === "required") {
      toast.error("Please sign in to access the admin panel.");
    } else if (notice === "expired") {
      toast.error("Your session has expired. Please sign in again.");
    }
  }, [router]);

  async function handleSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();

    if (!email.trim() || !password) {
      toast.error("Email dan password wajib diisi.");
      return;
    }

    setLoading(true);
    try {
      const data = await withMinDelay(
        loginAdmin(email.trim(), password),
        LOGIN_MIN_DELAY_MS,
      );
      saveAdminSession(data, remember);
      toast.success(`Welcome back, ${data.admin.name}`);
      router.push(consumeAdminRedirectPath("/admin"));
    } catch (err) {
      const message =
        err instanceof Error ? err.message : "Gagal masuk. Coba lagi.";
      toast.error(message);
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="admin-root relative flex min-h-screen items-center justify-center overflow-hidden bg-white px-4 py-10">
      <div
        className="pointer-events-none absolute -left-24 -top-24 h-72 w-72 rounded-full bg-[#ff5e16]/20 blur-3xl"
        aria-hidden
      />
      <div
        className="pointer-events-none absolute -bottom-32 -right-16 h-80 w-80 rounded-full bg-[#3b9eff]/20 blur-3xl"
        aria-hidden
      />
      <div
        className="pointer-events-none absolute inset-0 opacity-[0.045]"
        style={{
          backgroundImage:
            "linear-gradient(rgba(6,22,58,0.12) 1px, transparent 1px), linear-gradient(90deg, rgba(6,22,58,0.12) 1px, transparent 1px)",
          backgroundSize: "48px 48px",
        }}
        aria-hidden
      />

      <div className="relative w-full max-w-110">
        <div className="mb-8 flex flex-col items-center text-center">
          <LectorLogo />
          <p className="mt-3 text-[13px] text-[#8a97a8]">Admin Panel</p>
        </div>

        <div className="rounded-2xl border border-[#e8eef4] bg-white p-7 shadow-[0_12px_40px_rgba(16,38,73,0.08)] sm:p-8">
          <div className="mb-6">
            <h1 className="text-[22px] font-semibold tracking-tight text-[#1f2a37]">
              Welcome back
            </h1>
            <p className="mt-1.5 text-[13px] text-[#8a97a8]">
              Sign in to manage merchants, payments, and platform settings.
            </p>
          </div>

          <form onSubmit={handleSubmit} className="space-y-4" noValidate>
            <div>
              <label htmlFor="admin-email" className="mb-1.5 block text-[13px] font-medium text-[#3d4b5c]">
                Email
              </label>
              <input
                id="admin-email"
                type="email"
                autoComplete="username"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                placeholder="admin@pg-aggregator.local"
                className="block w-full rounded-lg border border-[#e8eef4] bg-[#f8fafc] px-3.5 py-2.5 text-[13.5px] text-[#1f2a37] outline-none transition placeholder:text-[#b0bbc8] focus:border-[#3b9eff] focus:bg-white focus:ring-2 focus:ring-[#3b9eff]/15"
              />
            </div>

            <div>
              <div className="mb-1.5 flex items-center justify-between">
                <label htmlFor="admin-password" className="block text-[13px] font-medium text-[#3d4b5c]">
                  Password
                </label>
                <a href="#" className="text-[12.5px] font-medium text-[#ff5e16] hover:text-[#ef5510]">
                  Forgot password?
                </a>
              </div>
              <div className="relative">
                <input
                  id="admin-password"
                  type={showPassword ? "text" : "password"}
                  autoComplete="current-password"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  placeholder="••••••••"
                  className="block w-full rounded-lg border border-[#e8eef4] bg-[#f8fafc] px-3.5 py-2.5 pr-11 text-[13.5px] text-[#1f2a37] outline-none transition placeholder:text-[#b0bbc8] focus:border-[#3b9eff] focus:bg-white focus:ring-2 focus:ring-[#3b9eff]/15"
                />
                <button
                  type="button"
                  onClick={() => setShowPassword((v) => !v)}
                  className="absolute inset-y-0 right-0 flex w-11 items-center justify-center text-[#8a97a8] hover:text-[#3d4b5c]"
                  aria-label={showPassword ? "Hide password" : "Show password"}
                >
                  <EyeIcon open={showPassword} />
                </button>
              </div>
            </div>

            <label className="flex cursor-pointer items-center gap-2.5 pt-0.5">
              <input
                type="checkbox"
                checked={remember}
                onChange={(e) => setRemember(e.target.checked)}
                className="h-4 w-4 rounded border-[#d0d8e2] text-[#ff5e16] accent-[#ff5e16] focus:ring-[#ff5e16]/30"
              />
              <span className="text-[13px] text-[#6b7c93]">Remember me</span>
            </label>

            <button
              type="submit"
              disabled={loading}
              aria-busy={loading}
              className="mt-1 inline-flex w-full items-center justify-center gap-2 rounded-lg bg-[#ff5e16] px-4 py-3 text-[13.5px] font-semibold text-white shadow-[0_4px_14px_rgba(255,94,22,0.35)] transition hover:bg-[#ef5510] disabled:cursor-not-allowed disabled:opacity-80"
            >
              {loading ? (
                <>
                  <ButtonSpinner />
                  <span>Signing in...</span>
                </>
              ) : (
                "Sign in"
              )}
            </button>
          </form>

          <div className="mt-6 border-t border-[#eef2f6] pt-5 text-center">
            <p className="text-[12.5px] text-[#8a97a8]">
              Protected area for platform administrators only.
            </p>
          </div>
        </div>

        <p className="mt-6 text-center text-[12.5px] text-[#8a97a8]">
          <Link href="/" className="transition hover:text-[#06163a]">
            ← Back to website
          </Link>
          <span className="mx-2 text-[#c5ced9]">·</span>
          <Link href="/login" className="transition hover:text-[#06163a]">
            Merchant login
          </Link>
        </p>
      </div>
    </div>
  );
}
