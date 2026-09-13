"use client";

import { useState, type FormEvent } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { Eye, EyeOff, Zap } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { AuthSidePanel } from "@/components/auth/side-panel";
import { ApiError, signup } from "@/lib/api";

export default function RegisterPage() {
  const router = useRouter();
  const [businessName, setBusinessName] = useState("");
  const [email, setEmail] = useState("");
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
      await signup({
        business_name: businessName,
        email,
        username,
        password,
      });
      // Cookie sesi sudah diterapkan browser sebelum baris ini jalan
      // (signup() melakukan auto-login di respons yang sama).
      router.push("/overview");
    } catch (err) {
      if (err instanceof ApiError) {
        if (err.code === "email_taken") {
          setError("Email ini sudah dipakai akun lain.");
        } else if (err.code === "username_taken") {
          setError("Username ini sudah dipakai akun lain.");
        } else if (err.code === "too_many_attempts") {
          setError("Terlalu banyak percobaan. Coba lagi dalam beberapa menit.");
        } else {
          setError(err.message || "Pendaftaran gagal, periksa kembali isian kamu.");
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
        title="Mulai otomasi notifikasi pembayaran bisnismu, gratis 3 hari."
        subtitle="Daftar tanpa kartu kredit, langsung aktif dan siap dipakai."
      />

      <div className="flex w-full flex-col items-center justify-center px-4 py-10 lg:w-1/2">
        <div className="w-full max-w-sm">
          <div className="flex flex-col items-center text-center">
            <span className="mb-4 flex size-11 items-center justify-center rounded-xl bg-slate-900 text-white ring-1 ring-slate-900/10">
              <Zap className="size-5" fill="currentColor" strokeWidth={0} />
            </span>
            <h1 className="text-xl font-semibold text-slate-900">Daftar Gratis 3 Hari</h1>
            <p className="mt-1 text-sm text-slate-500">Tanpa kartu kredit, langsung aktif.</p>
          </div>

          <form onSubmit={onSubmit} className="mt-8 flex flex-col gap-4">
            {error && (
              <Alert variant="destructive">
                <AlertDescription>{error}</AlertDescription>
              </Alert>
            )}

            <div className="flex flex-col gap-2">
              <Label htmlFor="business-name">Nama bisnis</Label>
              <Input
                id="business-name"
                value={businessName}
                onChange={(e) => setBusinessName(e.target.value)}
                placeholder="Toko Contoh"
                autoFocus
                required
              />
            </div>

            <div className="flex flex-col gap-2">
              <Label htmlFor="email">Email</Label>
              <Input
                id="email"
                type="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                placeholder="kamu@contoh.test"
                required
              />
            </div>

            <div className="flex flex-col gap-2">
              <Label htmlFor="username">Username</Label>
              <Input
                id="username"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                placeholder="toko_contoh"
                required
              />
            </div>

            <div className="flex flex-col gap-2">
              <Label htmlFor="password">Password</Label>
              <div className="relative">
                <Input
                  id="password"
                  type={showPassword ? "text" : "password"}
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  placeholder="Minimal 8 karakter"
                  minLength={8}
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
              {loading ? "Memproses..." : "Daftar"}
            </Button>

            <p className="text-center text-sm text-slate-500">
              Sudah punya akun?{" "}
              <Link href="/login" className="font-medium text-slate-900 hover:underline">
                Masuk
              </Link>
            </p>
          </form>
        </div>
      </div>
    </div>
  );
}
