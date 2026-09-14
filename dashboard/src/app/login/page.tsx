"use client";

import { useState, type FormEvent } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { Eye, EyeOff, Zap } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { AuthSidePanel } from "@/components/auth/side-panel";
import { ApiError, login } from "@/lib/api";

export default function LoginPage() {
  const router = useRouter();
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [showPassword, setShowPassword] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    setError(null);
    setLoading(true);
    try {
      await login(username, password);
      // Cookie sesi sudah diterapkan browser sebelum baris ini jalan (respons
      // fetch sudah selesai), jadi navigasi client-side biasa sudah cukup —
      // proxy.ts akan melihat cookie yang benar pada request berikutnya.
      router.push("/overview");
    } catch (err) {
      if (err instanceof ApiError) {
        if (err.code === "too_many_attempts") {
          setError("Terlalu banyak percobaan login. Coba lagi dalam beberapa menit.");
        } else {
          setError("Username atau password salah.");
        }
      } else {
        setError("Tidak dapat menghubungi server. Periksa koneksi Anda.");
      }
      setLoading(false);
    }
  }

  return (
    <div className="flex min-h-screen">
      <AuthSidePanel
        eyebrow="Payment Bridge"
        title="Bersama Payment Bridge, notifikasi pembayaran jadi otomatis dan akurat."
        subtitle="Payment Bridge adalah solusi lengkap otomasi notifikasi pembayaran untuk bisnismu."
      />

      <div className="flex w-full flex-col items-center justify-center px-4 py-10 lg:w-1/2">
        <div className="w-full max-w-sm">
          <div className="flex flex-col items-center text-center">
            <span className="mb-4 flex size-11 items-center justify-center rounded-xl bg-slate-900 text-white ring-1 ring-slate-900/10">
              <Zap className="size-5" fill="currentColor" strokeWidth={0} />
            </span>
            <h1 className="text-xl font-semibold text-slate-900">Masuk ke Payment Bridge</h1>
            <p className="mt-1 text-sm text-slate-500">
              Masuk untuk kelola notifikasi pembayaran bisnismu dengan mudah.
            </p>
          </div>

          <form onSubmit={onSubmit} className="mt-8 flex flex-col gap-4">
            {error && (
              <Alert variant="destructive">
                <AlertDescription>{error}</AlertDescription>
              </Alert>
            )}

            <div className="flex flex-col gap-2">
              <Label htmlFor="username">Username</Label>
              <Input
                id="username"
                autoComplete="username"
                placeholder="Masukkan username"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                required
                autoFocus
              />
            </div>

            <div className="flex flex-col gap-2">
              <div className="flex items-center justify-between">
                <Label htmlFor="password">Password</Label>
                <Link
                  href="/forgot-password"
                  className="text-xs font-medium text-slate-500 hover:text-slate-900 hover:underline"
                >
                  Lupa password?
                </Link>
              </div>
              <div className="relative">
                <Input
                  id="password"
                  type={showPassword ? "text" : "password"}
                  autoComplete="current-password"
                  placeholder="Masukkan password"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  required
                  className="pr-10"
                />
                <button
                  type="button"
                  onClick={() => setShowPassword((v) => !v)}
                  aria-label={showPassword ? "Sembunyikan password" : "Tampilkan password"}
                  className="absolute inset-y-0 right-0 flex w-9 items-center justify-center text-slate-400 hover:text-slate-600"
                >
                  {showPassword ? <EyeOff className="size-4" /> : <Eye className="size-4" />}
                </button>
              </div>
            </div>

            <Button type="submit" disabled={loading} className="mt-2 h-11 rounded-lg">
              {loading ? "Memeriksa..." : "Masuk"}
            </Button>

            <p className="text-center text-sm text-slate-500">
              Belum punya akun?{" "}
              <Link href="/register" className="font-medium text-slate-900 hover:underline">
                Daftar Sekarang
              </Link>
            </p>
          </form>
        </div>
      </div>
    </div>
  );
}
