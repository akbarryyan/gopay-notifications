"use client";

import { useState, type FormEvent } from "react";
import { useRouter } from "next/navigation";
import { Zap } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { ApiError, login } from "@/lib/api";

export default function LoginPage() {
  const router = useRouter();
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
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
      router.push("/");
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
    <div className="flex min-h-screen items-center justify-center bg-secondary/40 px-4">
      <Card className="w-full max-w-sm rounded-2xl border-none py-8 shadow-sm ring-1 ring-border/60">
        <CardHeader className="items-center text-center">
          <span className="mb-2 flex size-11 items-center justify-center rounded-xl bg-primary text-primary-foreground">
            <Zap className="size-5" fill="currentColor" strokeWidth={0} />
          </span>
          <CardTitle className="text-xl">Payment Bridge</CardTitle>
          <CardDescription>Masuk ke dashboard admin instalasi ini.</CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={onSubmit} className="flex flex-col gap-4">
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
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                required
                autoFocus
              />
            </div>

            <div className="flex flex-col gap-2">
              <Label htmlFor="password">Password</Label>
              <Input
                id="password"
                type="password"
                autoComplete="current-password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                required
              />
            </div>

            <Button type="submit" disabled={loading} className="mt-2">
              {loading ? "Memeriksa..." : "Masuk"}
            </Button>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}
