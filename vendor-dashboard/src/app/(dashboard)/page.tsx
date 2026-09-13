"use client";

import { useState } from "react";
import Link from "next/link";
import { Plus, RotateCw, Users } from "lucide-react";
import { Button, buttonVariants } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
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
import { createCustomer, getCustomers } from "@/lib/api";
import { useApiData } from "@/lib/use-api-data";
import { formatDateTime } from "@/lib/format";

export default function CustomersPage() {
  const { data, loading, error, reload } = useApiData(getCustomers);
  const [creating, setCreating] = useState(false);
  const [name, setName] = useState("");
  const [busy, setBusy] = useState(false);

  async function onCreate() {
    if (!name.trim()) return;
    setBusy(true);
    try {
      await createCustomer(name.trim());
      toast.success(`Customer "${name.trim()}" dibuat.`);
      setCreating(false);
      setName("");
      reload();
    } catch {
      toast.error("Gagal membuat customer.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="flex flex-col gap-6">
      <div className="flex items-start justify-between gap-4">
        <div>
          <h1 className="text-2xl font-semibold">Customers</h1>
          <p className="text-sm text-muted-foreground">
            Seluruh customer self-hosted Payment Bridge.
          </p>
        </div>
        <Button size="sm" onClick={() => setCreating(true)}>
          <Plus className="mr-1.5 size-4" />
          Buat customer
        </Button>
      </div>

      {error && (
        <Alert variant="destructive">
          <AlertTitle>Tidak dapat memuat customer</AlertTitle>
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
                <TableHead className="text-right">Detail</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {data.map((c) => (
                <TableRow key={c.id}>
                  <TableCell className="font-medium">{c.name}</TableCell>
                  <TableCell>{formatDateTime(c.created_at)}</TableCell>
                  <TableCell className="text-right">
                    <Link
                      href={`/customers/${c.id}`}
                      className={buttonVariants({ size: "sm", variant: "outline" })}
                    >
                      Lihat lisensi
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
          <p className="font-medium">Belum ada customer</p>
        </div>
      )}

      <AlertDialog open={creating} onOpenChange={(open) => !open && setCreating(false)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Buat customer baru</AlertDialogTitle>
            <AlertDialogDescription>Nama customer, mis. &ldquo;Toko Contoh&rdquo;.</AlertDialogDescription>
          </AlertDialogHeader>
          <div className="flex flex-col gap-2 py-2">
            <Label htmlFor="customer-name">Nama</Label>
            <Input
              id="customer-name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              autoFocus
            />
          </div>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={busy}>Batal</AlertDialogCancel>
            <AlertDialogAction onClick={onCreate} disabled={busy || !name.trim()}>
              Buat
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
