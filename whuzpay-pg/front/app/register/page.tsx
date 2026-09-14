"use client";

import { useState, type FormEvent } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import toast from "react-hot-toast";
import { registerMerchant } from "@/lib/merchant-auth";

function EyeIcon({ open }: { open: boolean }) {
  if (open) {
    return (
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2} className="h-5 w-5">
        <path d="M2 12s3.6-7 10-7 10 7 10 7-3.6 7-10 7-10-7-10-7Z" strokeLinecap="round" strokeLinejoin="round" />
        <circle cx="12" cy="12" r="3" />
      </svg>
    );
  }
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2} className="h-5 w-5">
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

export default function RegisterPage() {
  const router = useRouter();
  const [name, setName] = useState("");
  const [businessName, setBusinessName] = useState("");
  const [email, setEmail] = useState("");
  const [phone, setPhone] = useState("");
  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [showPassword, setShowPassword] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function handleSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    setError(null);

    if (!name.trim() || !businessName.trim() || !email.trim() || !password) {
      setError("Nama, nama bisnis, email, dan password wajib diisi.");
      return;
    }
    if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email.trim())) {
      setError("Format email tidak valid.");
      return;
    }
    if (password.length < 8) {
      setError("Password minimal 8 karakter.");
      return;
    }
    if (password !== confirmPassword) {
      setError("Konfirmasi password tidak cocok.");
      return;
    }

    setLoading(true);
    try {
      await registerMerchant({
        name: name.trim(),
        business_name: businessName.trim(),
        email: email.trim(),
        phone: phone.trim() || undefined,
        password,
      });
      toast.success("Pendaftaran berhasil. Silakan masuk.");
      router.replace("/login");
    } catch (err) {
      setError(
        err instanceof Error
          ? err.message
          : "Tidak bisa terhubung ke server. Coba lagi beberapa saat lagi.",
      );
      setLoading(false);
    }
  }

  return (
    <main className="flex min-h-screen">
      <div className="relative hidden w-1/2 flex-col justify-between overflow-hidden bg-linear-to-br from-brand-navy-light via-brand-navy to-brand-navy-dark p-12 lg:flex">
        <Link href="/" className="text-2xl font-bold lowercase italic tracking-tight text-white">
          whuzpay
        </Link>

        <div className="max-w-md">
          <h1 className="text-2xl font-extrabold capitalize leading-[1.2] tracking-[-0.04em] text-white sm:text-3xl">
            Mulai Terima Pembayaran QRIS Hari Ini.
          </h1>
          <p className="mt-4 text-base leading-relaxed text-white/80">
            Daftarkan bisnis Anda dan kelola semua transaksi dalam satu dashboard.
          </p>
        </div>

        <p className="text-sm text-white/50">© {new Date().getFullYear()} whuzpay. All rights reserved.</p>
      </div>

      <div className="flex w-full flex-col justify-center bg-white px-6 py-16 sm:px-12 lg:w-1/2 lg:px-20">
        <div className="mx-auto w-full max-w-sm">
          <Link href="/" className="mb-10 inline-block text-2xl font-bold lowercase italic tracking-tight text-brand-navy lg:hidden">
            whuzpay
          </Link>

          <h2 className="text-2xl font-extrabold leading-[1.02] tracking-[-0.03em] text-brand-navy">
            Daftarkan bisnis Anda
          </h2>
          <p className="mt-2 text-sm text-slate-500">
            Sudah punya akun?{" "}
            <Link href="/login" className="font-medium text-brand-navy hover:text-brand-navy-light">
              Masuk di sini
            </Link>
          </p>

          <form onSubmit={handleSubmit} className="mt-8 space-y-5" noValidate>
            {error && (
              <div
                role="alert"
                className="rounded-md border border-rose-200 bg-rose-50 px-4 py-3 text-sm text-rose-600"
              >
                {error}
              </div>
            )}

            <div>
              <label htmlFor="name" className="block text-sm font-medium text-slate-700">
                Nama Lengkap
              </label>
              <input
                id="name"
                type="text"
                autoComplete="name"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="Budi Santoso"
                className="mt-1.5 block w-full rounded-md border border-slate-200 px-3.5 py-2.5 text-sm text-slate-900 placeholder:text-slate-400 focus:border-brand-navy-light focus:outline-none focus:ring-2 focus:ring-brand-navy-light/20"
              />
            </div>

            <div>
              <label htmlFor="businessName" className="block text-sm font-medium text-slate-700">
                Nama Bisnis
              </label>
              <input
                id="businessName"
                type="text"
                autoComplete="organization"
                value={businessName}
                onChange={(e) => setBusinessName(e.target.value)}
                placeholder="Toko Budi Jaya"
                className="mt-1.5 block w-full rounded-md border border-slate-200 px-3.5 py-2.5 text-sm text-slate-900 placeholder:text-slate-400 focus:border-brand-navy-light focus:outline-none focus:ring-2 focus:ring-brand-navy-light/20"
              />
            </div>

            <div>
              <label htmlFor="email" className="block text-sm font-medium text-slate-700">
                Email
              </label>
              <input
                id="email"
                type="email"
                autoComplete="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                placeholder="nama@bisnis.com"
                className="mt-1.5 block w-full rounded-md border border-slate-200 px-3.5 py-2.5 text-sm text-slate-900 placeholder:text-slate-400 focus:border-brand-navy-light focus:outline-none focus:ring-2 focus:ring-brand-navy-light/20"
              />
            </div>

            <div>
              <label htmlFor="phone" className="block text-sm font-medium text-slate-700">
                No. Telepon <span className="text-slate-400">(opsional)</span>
              </label>
              <input
                id="phone"
                type="tel"
                autoComplete="tel"
                value={phone}
                onChange={(e) => setPhone(e.target.value)}
                placeholder="08123456789"
                className="mt-1.5 block w-full rounded-md border border-slate-200 px-3.5 py-2.5 text-sm text-slate-900 placeholder:text-slate-400 focus:border-brand-navy-light focus:outline-none focus:ring-2 focus:ring-brand-navy-light/20"
              />
            </div>

            <div>
              <label htmlFor="password" className="block text-sm font-medium text-slate-700">
                Password
              </label>
              <div className="relative mt-1.5">
                <input
                  id="password"
                  type={showPassword ? "text" : "password"}
                  autoComplete="new-password"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  placeholder="Minimal 8 karakter"
                  maxLength={72}
                  className="block w-full rounded-md border border-slate-200 px-3.5 py-2.5 pr-11 text-sm text-slate-900 placeholder:text-slate-400 focus:border-brand-navy-light focus:outline-none focus:ring-2 focus:ring-brand-navy-light/20"
                />
                <button
                  type="button"
                  onClick={() => setShowPassword((v) => !v)}
                  className="absolute inset-y-0 right-0 flex w-11 items-center justify-center text-slate-400 hover:text-slate-600"
                  aria-label={showPassword ? "Sembunyikan password" : "Tampilkan password"}
                >
                  <EyeIcon open={showPassword} />
                </button>
              </div>
            </div>

            <div>
              <label htmlFor="confirmPassword" className="block text-sm font-medium text-slate-700">
                Konfirmasi Password
              </label>
              <input
                id="confirmPassword"
                type={showPassword ? "text" : "password"}
                autoComplete="new-password"
                value={confirmPassword}
                onChange={(e) => setConfirmPassword(e.target.value)}
                placeholder="Ulangi password"
                maxLength={72}
                className="mt-1.5 block w-full rounded-md border border-slate-200 px-3.5 py-2.5 text-sm text-slate-900 placeholder:text-slate-400 focus:border-brand-navy-light focus:outline-none focus:ring-2 focus:ring-brand-navy-light/20"
              />
            </div>

            <button
              type="submit"
              disabled={loading}
              className="w-full rounded-md bg-brand-yellow px-6 py-3 text-sm font-bold uppercase tracking-wide text-brand-navy-dark shadow-sm transition-colors hover:bg-brand-yellow-dark disabled:cursor-not-allowed disabled:opacity-60"
            >
              {loading ? "Memproses..." : "Daftar"}
            </button>
          </form>

          <p className="mt-8 text-center text-sm text-slate-500">
            <Link href="/" className="font-medium text-brand-navy hover:text-brand-navy-light">
              ← Kembali ke beranda
            </Link>
          </p>
        </div>
      </div>
    </main>
  );
}
