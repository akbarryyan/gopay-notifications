"use client";

import { useState, type FormEvent } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { Zap } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { ApiError, signup } from "@/lib/api";

export default function RegisterPage() {
  const router = useRouter();
  const [businessName, setBusinessName] = useState("");
  const [email, setEmail] = useState("");
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
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
    <div className="flex min-h-screen items-center justify-center bg-secondary/40 px-4 py-10">
      <Card className="w-full max-w-sm rounded-2xl border-none py-8 shadow-sm ring-1 ring-border/60">
        <CardHeader className="items-center text-center">
          <span className="mb-2 flex size-11 items-center justify-center rounded-xl bg-primary text-primary-foreground">
            <Zap className="size-5" fill="currentColor" strokeWidth={0} />
          </span>
          <CardTitle className="text-xl">Daftar Gratis 3 Hari</CardTitle>
          <CardDescription>Tanpa kartu kredit, langsung aktif.</CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={onSubmit} className="flex flex-col gap-4">
            {error && (
              <Alert variant="destructive">
                <AlertDescription>{error}</AlertDescription>
              </Alert>
            )}
            <div className="flex flex-col gap-1.5">
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
            <div className="flex flex-col gap-1.5">
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
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="username">Username</Label>
              <Input
                id="username"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                placeholder="toko_contoh"
                required
              />
            </div>
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="password">Password</Label>
              <Input
                id="password"
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                minLength={8}
                required
              />
            </div>
            <Button type="submit" disabled={loading} className="mt-2">
              {loading ? "Memproses..." : "Daftar"}
            </Button>
            <p className="text-center text-sm text-muted-foreground">
              Sudah punya akun?{" "}
              <Link href="/login" className="font-medium text-foreground hover:underline">
                Masuk
              </Link>
            </p>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}
