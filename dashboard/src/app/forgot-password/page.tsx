"use client";

import { useState, type FormEvent } from "react";
import Link from "next/link";
import { ArrowLeft, MailCheck, Zap } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { AuthSidePanel } from "@/components/auth/side-panel";
import { ApiError, forgotPassword } from "@/lib/api";

export default function ForgotPasswordPage() {
  const [email, setEmail] = useState("");
  const [sentTo, setSentTo] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    setError(null);
    setLoading(true);
    try {
      await forgotPassword(email.trim());
      setSentTo(email.trim());
    } catch (err) {
      if (err instanceof ApiError && err.code === "not_available") {
        setError(
          "Reset password lewat email belum tersedia. Hubungi admin untuk mengatur ulang password.",
        );
      } else if (err instanceof ApiError && err.code === "too_many_attempts") {
        setError("Terlalu banyak permintaan. Coba lagi dalam beberapa menit.");
      } else if (err instanceof ApiError) {
        setError(err.message);
      } else {
        setError("Tidak dapat menghubungi server. Periksa koneksi Anda.");
      }
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="flex min-h-screen">
      <AuthSidePanel
        eyebrow="Payment Bridge"
        title="Lupa password? Tenang, atur ulang dalam hitungan menit."
        subtitle="Kami kirim link untuk membuat password baru ke email akunmu."
      />

      <div className="flex w-full flex-col items-center justify-center px-4 py-10 lg:w-1/2">
        <div className="w-full max-w-sm">
          {sentTo ? (
            <div className="flex flex-col items-center text-center">
              <span className="mb-4 flex size-11 items-center justify-center rounded-xl bg-teal-50 text-teal-700 ring-1 ring-teal-600/20">
                <MailCheck className="size-5" />
              </span>
              <h1 className="text-xl font-semibold text-slate-900">Cek email kamu</h1>
              {/* Kalimat bersyarat, bukan "link sudah dikirim": backend sengaja
                  tidak memberi tahu apakah email ini terdaftar. */}
              <p className="mt-2 text-sm text-slate-500">
                Kalau <span className="font-medium text-slate-900">{sentTo}</span> terdaftar, link
                untuk membuat password baru sudah dikirim ke sana. Link berlaku 30 menit.
              </p>
              <p className="mt-4 text-xs text-slate-400">
                Tidak ada di kotak masuk? Cek folder spam, atau coba lagi beberapa menit lagi.
              </p>
              <Button
                variant="outline"
                className="mt-6 h-11 w-full rounded-lg"
                onClick={() => setSentTo(null)}
              >
                Kirim ulang ke email lain
              </Button>
              <Link
                href="/login"
                className="mt-4 flex items-center gap-1.5 text-sm font-medium text-slate-900 hover:underline"
              >
                <ArrowLeft className="size-4" />
                Kembali ke halaman masuk
              </Link>
            </div>
          ) : (
            <>
              <div className="flex flex-col items-center text-center">
                <span className="mb-4 flex size-11 items-center justify-center rounded-xl bg-slate-900 text-white ring-1 ring-slate-900/10">
                  <Zap className="size-5" fill="currentColor" strokeWidth={0} />
                </span>
                <h1 className="text-xl font-semibold text-slate-900">Lupa password</h1>
                <p className="mt-1 text-sm text-slate-500">
                  Masukkan email akunmu. Kami kirim link untuk membuat password baru, beserta
                  username-mu.
                </p>
              </div>

              <form onSubmit={onSubmit} className="mt-8 flex flex-col gap-4">
                {error && (
                  <Alert variant="destructive">
                    <AlertDescription>{error}</AlertDescription>
                  </Alert>
                )}

                <div className="flex flex-col gap-2">
                  <Label htmlFor="email">Email</Label>
                  <Input
                    id="email"
                    type="email"
                    autoComplete="email"
                    placeholder="nama@bisnis.com"
                    value={email}
                    onChange={(e) => setEmail(e.target.value)}
                    required
                    autoFocus
                  />
                </div>

                <Button
                  type="submit"
                  disabled={loading || !email.trim()}
                  className="mt-2 h-11 rounded-lg"
                >
                  {loading ? "Mengirim..." : "Kirim link reset"}
                </Button>

                <Link
                  href="/login"
                  className="flex items-center justify-center gap-1.5 text-sm font-medium text-slate-500 hover:text-slate-900"
                >
                  <ArrowLeft className="size-4" />
                  Kembali ke halaman masuk
                </Link>
              </form>
            </>
          )}
        </div>
      </div>
    </div>
  );
}
