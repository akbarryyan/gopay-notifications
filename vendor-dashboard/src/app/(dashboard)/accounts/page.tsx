"use client";

import { useState } from "react";
import Link from "next/link";
import { Plus, RotateCw, Users, Check, Copy } from "lucide-react";
import { Button, buttonVariants } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { toast } from "sonner";
import { createAccount, getAccounts, type AccountPlan } from "@/lib/api";
import { useApiData } from "@/lib/use-api-data";
import { formatDateOnly } from "@/lib/format";
import { STATUS_BADGE } from "@/lib/account-status";

const PLANS: AccountPlan[] = ["Starter", "Business", "Enterprise"];

export default function AccountsPage() {
  const { data, loading, error, reload } = useApiData(getAccounts);
  const [creating, setCreating] = useState(false);
  const [businessName, setBusinessName] = useState("");
  const [email, setEmail] = useState("");
  const [username, setUsername] = useState("");
  const [plan, setPlan] = useState<AccountPlan>("Business");
  const [expiresAt, setExpiresAt] = useState("");
  const [busy, setBusy] = useState(false);
  const [reveal, setReveal] = useState<{ username: string; password: string } | null>(null);
  const [copied, setCopied] = useState(false);

  function resetForm() {
    setBusinessName("");
    setEmail("");
    setUsername("");
    setPlan("Business");
    setExpiresAt("");
  }

  async function onCreate() {
    if (!businessName.trim() || !email.trim() || !username.trim() || !expiresAt) return;
    setBusy(true);
    try {
      const res = await createAccount({
        business_name: businessName.trim(),
        email: email.trim(),
        username: username.trim(),
        plan,
        expires_at: expiresAt,
      });
      toast.success(`Akun "${businessName.trim()}" dibuat.`);
      setCreating(false);
      setReveal({ username: res.account.username, password: res.initial_password });
      resetForm();
      reload();
    } catch {
      toast.error("Gagal membuat akun.");
    } finally {
      setBusy(false);
    }
  }

  async function copyPassword() {
    if (!reveal) return;
    await navigator.clipboard.writeText(reveal.password);
    setCopied(true);
    setTimeout(() => setCopied(false), 1500);
  }

  return (
    <div className="flex flex-col gap-6">
      <div className="flex items-start justify-between gap-4">
        <div>
          <h1 className="text-2xl font-semibold">Accounts</h1>
          <p className="text-sm text-muted-foreground">
            Seluruh customer Payment Bridge (hosted).
          </p>
        </div>
        <Button size="sm" onClick={() => setCreating(true)}>
          <Plus className="mr-1.5 size-4" />
          Buat akun
        </Button>
      </div>

      {error && (
        <Alert variant="destructive">
          <AlertTitle>Tidak dapat memuat akun</AlertTitle>
          <AlertDescription className="flex items-center justify-between gap-4">
            <span>{error}</span>
            <Button size="sm" variant="outline" onClick={reload}>
              <RotateCw className="mr-1.5 size-3.5" />
              Coba lagi
            </Button>
          </AlertDescription>
        </Alert>
      )}

      {loading ? (
        <div className="flex flex-col gap-2">
          {Array.from({ length: 3 }).map((_, i) => (
            <Skeleton key={i} className="h-14" />
          ))}
        </div>
      ) : data && data.length > 0 ? (
        <div className="overflow-x-auto rounded-xl border border-border/60 shadow-sm">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Nama Bisnis</TableHead>
                <TableHead>Username</TableHead>
                <TableHead>Plan</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Kedaluwarsa</TableHead>
                <TableHead className="text-right">Detail</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {data.map((acc) => (
                <TableRow key={acc.id}>
                  <TableCell className="font-medium">{acc.business_name}</TableCell>
                  <TableCell className="font-mono text-xs">{acc.username}</TableCell>
                  <TableCell>{acc.plan}</TableCell>
                  <TableCell>
                    <Badge className={STATUS_BADGE[acc.status]}>{acc.status}</Badge>
                  </TableCell>
                  <TableCell>{formatDateOnly(acc.expires_at)}</TableCell>
                  <TableCell className="text-right">
                    <Link
                      href={`/accounts/${acc.id}`}
                      className={buttonVariants({ size: "sm", variant: "outline" })}
                    >
                      Kelola
                    </Link>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      ) : (
        <div className="rounded-2xl border border-dashed p-8 text-center">
          <Users className="mx-auto mb-2 size-8 text-muted-foreground" />
          <p className="font-medium">Belum ada akun</p>
        </div>
      )}

      <AlertDialog open={creating} onOpenChange={(open) => !open && setCreating(false)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Buat akun baru</AlertDialogTitle>
            <AlertDialogDescription>
              Password awal akan ditampilkan sekali setelah dibuat — catat sebelum
              menutup dialog itu.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <div className="flex flex-col gap-3 py-2">
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="business-name">Nama bisnis</Label>
              <Input
                id="business-name"
                value={businessName}
                onChange={(e) => setBusinessName(e.target.value)}
                placeholder="Toko Contoh"
                autoFocus
              />
            </div>
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="email">Email</Label>
              <Input
                id="email"
                type="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                placeholder="toko@contoh.test"
              />
            </div>
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="username">Username</Label>
              <Input
                id="username"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                placeholder="toko_contoh"
              />
            </div>
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="plan">Plan</Label>
              <select
                id="plan"
                value={plan}
                onChange={(e) => setPlan(e.target.value as AccountPlan)}
                className="h-9 rounded-md border border-input bg-transparent px-3 text-sm"
              >
                {PLANS.map((p) => (
                  <option key={p} value={p}>
                    {p}
                  </option>
                ))}
              </select>
            </div>
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="expires-at">Kedaluwarsa</Label>
              <Input
                id="expires-at"
                type="date"
                value={expiresAt}
                onChange={(e) => setExpiresAt(e.target.value)}
              />
            </div>
          </div>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={busy}>Batal</AlertDialogCancel>
            <AlertDialogAction
              onClick={onCreate}
              disabled={busy || !businessName.trim() || !email.trim() || !username.trim() || !expiresAt}
            >
              Buat
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <AlertDialog open={reveal !== null} onOpenChange={(open) => !open && setReveal(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Akun berhasil dibuat</AlertDialogTitle>
            <AlertDialogDescription>
              Password ini cuma tampil sekarang — kirim ke customer lewat kanal
              vendor sendiri, tidak bisa dilihat lagi setelah dialog ini ditutup.
            </AlertDialogDescription>
          </AlertDialogHeader>
          {reveal && (
            <div className="flex flex-col gap-2 py-2">
              <div className="text-sm">
                <span className="text-muted-foreground">Username: </span>
                <span className="font-mono">{reveal.username}</span>
              </div>
              <div className="flex items-center gap-2 rounded-md border bg-muted/40 px-3 py-2 font-mono text-sm">
                <span className="flex-1 select-all break-all">{reveal.password}</span>
                <Button size="sm" variant="ghost" onClick={copyPassword}>
                  {copied ? <Check className="size-4" /> : <Copy className="size-4" />}
                </Button>
              </div>
            </div>
          )}
          <AlertDialogFooter>
            <AlertDialogAction onClick={() => setReveal(null)}>Selesai</AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
