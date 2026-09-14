"use client";

import { useEffect, useState, type FormEvent } from "react";
import Link from "next/link";
import { CheckCircle2, Eye, EyeOff, KeyRound, TriangleAlert } from "lucide-react";
import { Button, buttonVariants } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { AuthSidePanel } from "@/components/auth/side-panel";
import { ApiError, resetPassword } from "@/lib/api";

type Phase = "loading" | "form" | "missing" | "invalid" | "done";

/**
 * Token dibaca dari FRAGMENT (#token=...), bukan query string -- fragment
 * tidak pernah dikirim browser ke server, jadi tidak tercatat di log akses
 * maupun header Referer. Setelah dibaca langsung dihapus dari address bar
 * supaya tidak tertinggal di riwayat browser atau terlihat saat layar
 * dibagikan.
 */
export default function ResetPasswordPage() {
  const [token, setToken] = useState<string | null>(null);
  const [phase, setPhase] = useState<Phase>("loading");
  const [password, setPassword] = useState("");
  const [confirm, setConfirm] = useState("");
  const [show, setShow] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    const value = new URLSearchParams(window.location.hash.slice(1)).get("token");
    if (window.location.hash) {
      window.history.replaceState(null, "", window.location.pathname);
    }
    // eslint-disable-next-line react-hooks/set-state-in-effect
    setToken(value);
    setPhase(value ? "form" : "missing");
  }, []);

  const tooShort = password.length > 0 && password.length < 8;
  const mismatch = confirm.length > 0 && password !== confirm;

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    if (!token) return;
    setError(null);
    if (password.length < 8) {
      setError("Password minimal 8 karakter.");
      return;
    }
    if (password !== confirm) {
      setError("Konfirmasi password tidak sama.");
      return;
    }
    setBusy(true);
    try {
      await resetPassword(token, password);
      setPhase("done");
    } catch (err) {
      if (err instanceof ApiError && err.code === "invalid_token") {
        setPhase("invalid");
      } else if (err instanceof ApiError) {
        setError(err.message);
      } else {
        setError("Tidak dapat menghubungi server. Periksa koneksi Anda.");
      }
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="flex min-h-screen">
      <AuthSidePanel
        eyebrow="Payment Bridge"
        title="Buat password baru untuk akunmu."
        subtitle="Setelah password diganti, semua perangkat yang sedang login otomatis dikeluarkan."
      />

      <div className="flex w-full flex-col items-center justify-center px-4 py-10 lg:w-1/2">
        <div className="w-full max-w-sm">
          {phase === "loading" ? null : phase === "done" ? (
            <StatusBlock
              icon={<CheckCircle2 className="size-5" />}
              tone="success"
              title="Password berhasil diganti"
              body="Silakan masuk memakai password baru. Semua sesi login sebelumnya sudah dikeluarkan."
              action={
                <Link href="/login" className={cn(buttonVariants(), "h-11 w-full rounded-lg")}>
                  Masuk sekarang
                </Link>
              }
            />
          ) : phase === "missing" || phase === "invalid" ? (
            <StatusBlock
              icon={<TriangleAlert className="size-5" />}
              tone="warning"
              title={phase === "missing" ? "Link reset tidak lengkap" : "Link reset tidak berlaku"}
              body={
                phase === "missing"
                  ? "Buka link langsung dari email reset password. Pastikan link tersalin utuh."
                  : "Link ini sudah kedaluwarsa atau sudah pernah dipakai. Minta link baru untuk melanjutkan."
              }
              action={
                <Link
                  href="/forgot-password"
                  className={cn(buttonVariants(), "h-11 w-full rounded-lg")}
                >
                  Minta link baru
                </Link>
              }
            />
          ) : (
            <>
              <div className="flex flex-col items-center text-center">
                <span className="mb-4 flex size-11 items-center justify-center rounded-xl bg-slate-900 text-white ring-1 ring-slate-900/10">
                  <KeyRound className="size-5" />
                </span>
                <h1 className="text-xl font-semibold text-slate-900">Buat password baru</h1>
                <p className="mt-1 text-sm text-slate-500">Minimal 8 karakter.</p>
              </div>

              <form onSubmit={onSubmit} className="mt-8 flex flex-col gap-4">
                {error && (
                  <Alert variant="destructive">
                    <AlertDescription>{error}</AlertDescription>
                  </Alert>
                )}

                <div className="flex flex-col gap-2">
                  <Label htmlFor="new-password">Password baru</Label>
                  <div className="relative">
                    <Input
                      id="new-password"
                      type={show ? "text" : "password"}
                      autoComplete="new-password"
                      value={password}
                      onChange={(e) => setPassword(e.target.value)}
                      required
                      autoFocus
                      className="pr-10"
                    />
                    <button
                      type="button"
                      onClick={() => setShow((v) => !v)}
                      aria-label={show ? "Sembunyikan password" : "Tampilkan password"}
                      className="absolute inset-y-0 right-0 flex w-9 items-center justify-center text-slate-400 hover:text-slate-600"
                    >
                      {show ? <EyeOff className="size-4" /> : <Eye className="size-4" />}
                    </button>
                  </div>
                  {tooShort && <p className="text-xs text-destructive">Minimal 8 karakter.</p>}
                </div>

                <div className="flex flex-col gap-2">
                  <Label htmlFor="confirm-password">Ulangi password baru</Label>
                  <Input
                    id="confirm-password"
                    type={show ? "text" : "password"}
                    autoComplete="new-password"
                    value={confirm}
                    onChange={(e) => setConfirm(e.target.value)}
                    required
                  />
                  {mismatch && (
                    <p className="text-xs text-destructive">Tidak sama dengan password baru.</p>
                  )}
                </div>

                <Button
                  type="submit"
                  disabled={busy || !password || !confirm || tooShort || mismatch}
                  className="mt-2 h-11 rounded-lg"
                >
                  {busy ? "Menyimpan..." : "Simpan password baru"}
                </Button>
              </form>
            </>
          )}
        </div>
      </div>
    </div>
  );
}

function StatusBlock({
  icon,
  tone,
  title,
  body,
  action,
}: {
  icon: React.ReactNode;
  tone: "success" | "warning";
  title: string;
  body: string;
  action: React.ReactNode;
}) {
  return (
    <div className="flex flex-col items-center text-center">
      <span
        className={
          tone === "success"
            ? "mb-4 flex size-11 items-center justify-center rounded-xl bg-teal-50 text-teal-700 ring-1 ring-teal-600/20"
            : "mb-4 flex size-11 items-center justify-center rounded-xl bg-amber-50 text-amber-700 ring-1 ring-amber-600/20"
        }
      >
        {icon}
      </span>
      <h1 className="text-xl font-semibold text-slate-900">{title}</h1>
      <p className="mt-2 text-sm text-slate-500">{body}</p>
      <div className="mt-6 w-full">{action}</div>
    </div>
  );
}
