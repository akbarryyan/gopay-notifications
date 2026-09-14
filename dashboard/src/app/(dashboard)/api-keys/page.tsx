"use client";

import { useState } from "react";
import { Check, Copy, KeyRound, Plus, RotateCw } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Skeleton } from "@/components/ui/skeleton";
import { Badge } from "@/components/ui/badge";
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
import toast from "react-hot-toast";
import { createAPIKey, getAPIKeys, revokeAPIKey, type AdminAPIKey } from "@/lib/api";
import { useApiData } from "@/lib/use-api-data";
import { formatDateTime } from "@/lib/format";

export default function ApiKeysPage() {
  const { data, loading, error, reload } = useApiData(getAPIKeys);
  const [creating, setCreating] = useState(false);
  const [newName, setNewName] = useState("");
  const [busy, setBusy] = useState(false);
  const [createdKey, setCreatedKey] = useState<{ name: string; key: string } | null>(null);
  const [copied, setCopied] = useState(false);
  const [pendingRevoke, setPendingRevoke] = useState<AdminAPIKey | null>(null);

  async function onCreate() {
    if (!newName.trim()) return;
    setBusy(true);
    try {
      const created = await createAPIKey(newName.trim());
      setCreatedKey({ name: created.name, key: created.key });
      setCreating(false);
      setNewName("");
      reload();
    } catch {
      toast.error("Gagal membuat API key.");
    } finally {
      setBusy(false);
    }
  }

  async function onRevoke() {
    if (!pendingRevoke) return;
    setBusy(true);
    try {
      await revokeAPIKey(pendingRevoke.id);
      toast.success(`${pendingRevoke.name} dicabut.`);
      reload();
    } catch {
      toast.error("Gagal mencabut API key.");
    } finally {
      setBusy(false);
      setPendingRevoke(null);
    }
  }

  async function copyKey() {
    if (!createdKey) return;
    try {
      await navigator.clipboard.writeText(createdKey.key);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      toast.error("Gagal menyalin — salin manual dari kotak di atas.");
    }
  }

  return (
    <div className="flex flex-col gap-6">
      <div className="flex items-start justify-between gap-4">
        <div>
          <h1 className="text-2xl font-semibold">API Keys</h1>
          <p className="text-sm text-muted-foreground">
            Dipakai server website kamu untuk memanggil <code>POST /api/v1/invoices</code>.
          </p>
        </div>
        <Button size="sm" onClick={() => setCreating(true)}>
          <Plus className="mr-1.5 size-4" />
          Buat key baru
        </Button>
      </div>

      {error && (
        <Alert variant="destructive">
          <AlertTitle>Tidak dapat memuat API keys</AlertTitle>
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
                <TableHead>Nama</TableHead>
                <TableHead>Dibuat</TableHead>
                <TableHead>Status</TableHead>
                <TableHead className="text-right">Aksi</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {data.map((k) => (
                <TableRow key={k.id}>
                  <TableCell className="font-medium">{k.name}</TableCell>
                  <TableCell>{formatDateTime(k.created_at)}</TableCell>
                  <TableCell>
                    {k.revoked_at ? (
                      <Badge variant="outline" className="text-muted-foreground">
                        Dicabut
                      </Badge>
                    ) : (
                      <Badge className="border-transparent bg-emerald-500/15 text-emerald-700 dark:text-emerald-400">
                        Aktif
                      </Badge>
                    )}
                  </TableCell>
                  <TableCell className="text-right">
                    <Button
                      size="sm"
                      variant="outline"
                      disabled={!!k.revoked_at}
                      onClick={() => setPendingRevoke(k)}
                    >
                      Cabut
                    </Button>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      ) : (
        <div className="rounded-2xl border border-dashed p-8 text-center">
          <KeyRound className="mx-auto mb-2 size-8 text-muted-foreground" />
          <p className="font-medium">Belum ada API key</p>
          <p className="mt-1 text-sm text-muted-foreground">
            Buat satu supaya server website kamu bisa membuat invoice.
          </p>
        </div>
      )}

      {/* Buat key baru */}
      <AlertDialog open={creating} onOpenChange={(open) => !open && setCreating(false)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Buat API key baru</AlertDialogTitle>
            <AlertDialogDescription>
              Beri nama supaya mudah dikenali, mis. &ldquo;Website utama&rdquo;.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <div className="flex flex-col gap-2 py-2">
            <Label htmlFor="key-name">Nama</Label>
            <Input
              id="key-name"
              value={newName}
              onChange={(e) => setNewName(e.target.value)}
              placeholder="Website utama"
              autoFocus
            />
          </div>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={busy}>Batal</AlertDialogCancel>
            <AlertDialogAction onClick={onCreate} disabled={busy || !newName.trim()}>
              Buat
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* Key mentah — tampil satu kali saja */}
      <AlertDialog open={createdKey !== null} onOpenChange={(open) => !open && setCreatedKey(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>API key &ldquo;{createdKey?.name}&rdquo; dibuat</AlertDialogTitle>
            <AlertDialogDescription>
              Salin sekarang — key ini <strong>tidak akan ditampilkan lagi</strong> setelah
              jendela ini ditutup.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <div className="flex items-center gap-2 py-2">
            <code className="min-w-0 flex-1 overflow-x-auto rounded-lg border bg-muted px-3 py-2 font-mono text-xs whitespace-nowrap">
              {createdKey?.key}
            </code>
            <Button type="button" variant="outline" size="icon" onClick={copyKey}>
              {copied ? <Check className="size-4" /> : <Copy className="size-4" />}
            </Button>
          </div>
          <AlertDialogFooter>
            <AlertDialogAction onClick={() => setCreatedKey(null)}>
              Sudah disalin, tutup
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* Konfirmasi cabut */}
      <AlertDialog open={pendingRevoke !== null} onOpenChange={(open) => !open && setPendingRevoke(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Cabut API key?</AlertDialogTitle>
            <AlertDialogDescription>
              {`"${pendingRevoke?.name}" tidak akan bisa lagi membuat invoice baru. Tidak bisa dibatalkan — buat key baru kalau perlu.`}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={busy}>Batal</AlertDialogCancel>
            <AlertDialogAction onClick={onRevoke} disabled={busy}>
              Cabut
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
