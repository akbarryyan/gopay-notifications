"use client";

import { useState } from "react";
import {
  ArrowDown,
  ArrowUp,
  Check,
  ExternalLink,
  Package,
  Pencil,
  Plus,
  RotateCw,
  Trash2,
  X,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Checkbox } from "@/components/ui/checkbox";
import { Badge } from "@/components/ui/badge";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Skeleton } from "@/components/ui/skeleton";
import { Card, CardContent } from "@/components/ui/card";
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
import {
  ApiError,
  createPlan,
  deletePlan,
  getPlans,
  movePlan,
  updatePlan,
  type Plan,
  type PlanInput,
} from "@/lib/api";
import { useApiData } from "@/lib/use-api-data";

const EMPTY_FORM: PlanInput = {
  name: "",
  max_devices: 1,
  unlimited: false,
  price_label: "",
  price_period: "",
  description: "",
  features: [],
  highlighted: false,
  visible: true,
};

function formFromPlan(p: Plan): PlanInput {
  return {
    name: p.name,
    max_devices: p.max_devices < 0 ? 1 : p.max_devices,
    unlimited: p.max_devices < 0,
    price_label: p.price_label,
    price_period: p.price_period,
    description: p.description,
    features: p.features,
    highlighted: p.highlighted,
    visible: p.visible,
  };
}

export default function PlansPage() {
  const { data, loading, error, reload } = useApiData(getPlans);
  const [editing, setEditing] = useState<Plan | "new" | null>(null);
  const [pendingDelete, setPendingDelete] = useState<Plan | null>(null);
  const [busyId, setBusyId] = useState<string | null>(null);

  async function onMove(plan: Plan, direction: "up" | "down") {
    setBusyId(plan.id);
    try {
      await movePlan(plan.id, direction);
      reload();
    } catch {
      toast.error("Gagal memindahkan urutan plan.");
    } finally {
      setBusyId(null);
    }
  }

  async function onDelete() {
    if (!pendingDelete) return;
    setBusyId(pendingDelete.id);
    try {
      await deletePlan(pendingDelete.id);
      toast.success(`Plan "${pendingDelete.name}" dihapus.`);
      setPendingDelete(null);
      reload();
    } catch {
      toast.error("Gagal menghapus plan.");
    } finally {
      setBusyId(null);
    }
  }

  return (
    <div className="flex flex-col gap-6">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <h1 className="text-2xl font-semibold">Plans</h1>
          <p className="text-sm text-muted-foreground">
            Paket yang bisa dipilih saat membuat/mengubah account, sekaligus data yang tampil di
            section{" "}
            <a
              href="/#harga"
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex items-center gap-1 font-medium text-foreground underline underline-offset-4"
            >
              Harga
              <ExternalLink className="size-3" />
            </a>{" "}
            landing page publik.
          </p>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" size="sm" onClick={reload} disabled={loading}>
            <RotateCw className="mr-1.5 size-3.5" />
            Muat ulang
          </Button>
          <Button size="sm" onClick={() => setEditing("new")}>
            <Plus className="mr-1.5 size-4" />
            Tambah Plan
          </Button>
        </div>
      </div>

      {error && (
        <Alert variant="destructive">
          <AlertTitle>Tidak dapat memuat plans</AlertTitle>
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
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {Array.from({ length: 3 }).map((_, i) => (
            <Skeleton key={i} className="h-64 rounded-2xl" />
          ))}
        </div>
      ) : data && data.length > 0 ? (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {data.map((plan, i) => (
            <Card key={plan.id} className="rounded-2xl border-none shadow-sm ring-1 ring-border/60">
              <CardContent className="flex flex-col gap-3 pt-6">
                <div className="flex items-start justify-between gap-2">
                  <div>
                    <div className="flex items-center gap-2">
                      <p className="font-semibold">{plan.name}</p>
                      {plan.highlighted && (
                        <Badge className="border-transparent bg-teal-500/15 text-teal-700 dark:text-teal-400">
                          Direkomendasikan
                        </Badge>
                      )}
                      {!plan.visible && <Badge variant="outline">Disembunyikan</Badge>}
                    </div>
                    <p className="mt-0.5 text-sm text-muted-foreground">
                      {plan.max_devices < 0 ? "Device tanpa batas" : `${plan.max_devices} device`}
                    </p>
                  </div>
                  <div className="flex flex-col gap-1">
                    <Button
                      variant="ghost"
                      size="icon"
                      className="size-6"
                      disabled={i === 0 || busyId === plan.id}
                      onClick={() => onMove(plan, "up")}
                      aria-label="Naikkan urutan"
                    >
                      <ArrowUp className="size-3.5" />
                    </Button>
                    <Button
                      variant="ghost"
                      size="icon"
                      className="size-6"
                      disabled={i === data.length - 1 || busyId === plan.id}
                      onClick={() => onMove(plan, "down")}
                      aria-label="Turunkan urutan"
                    >
                      <ArrowDown className="size-3.5" />
                    </Button>
                  </div>
                </div>

                {plan.price_label ? (
                  <p className="text-2xl font-semibold">
                    {plan.price_label}
                    {plan.price_period && (
                      <span className="text-sm font-normal text-muted-foreground">
                        {" "}
                        {plan.price_period}
                      </span>
                    )}
                  </p>
                ) : (
                  <p className="text-sm text-muted-foreground italic">
                    Harga belum diisi -- tidak tampil di landing page.
                  </p>
                )}

                {plan.description && (
                  <p className="text-sm text-muted-foreground">{plan.description}</p>
                )}

                {plan.features.length > 0 && (
                  <ul className="flex flex-col gap-1.5 text-sm">
                    {plan.features.map((f) => (
                      <li key={f} className="flex items-center gap-2">
                        <Check className="size-3.5 shrink-0 text-teal-700 dark:text-teal-400" />
                        {f}
                      </li>
                    ))}
                  </ul>
                )}

                <div className="mt-2 flex gap-2">
                  <Button
                    size="sm"
                    variant="outline"
                    className="flex-1"
                    onClick={() => setEditing(plan)}
                  >
                    <Pencil className="mr-1.5 size-3.5" />
                    Edit
                  </Button>
                  <Button
                    size="sm"
                    variant="outline"
                    className="text-destructive hover:text-destructive"
                    onClick={() => setPendingDelete(plan)}
                  >
                    <Trash2 className="size-3.5" />
                  </Button>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      ) : (
        <div className="rounded-2xl border border-dashed p-8 text-center">
          <Package className="mx-auto mb-2 size-6 text-muted-foreground" />
          <p className="font-medium">Belum ada plan</p>
          <p className="mt-1 text-sm text-muted-foreground">
            Tambah plan pertama supaya vendor bisa memilihnya saat membuat account.
          </p>
        </div>
      )}

      {editing && (
        <PlanFormDialog
          initial={editing === "new" ? EMPTY_FORM : formFromPlan(editing)}
          planId={editing === "new" ? null : editing.id}
          onClose={() => setEditing(null)}
          onSaved={() => {
            setEditing(null);
            reload();
          }}
        />
      )}

      <AlertDialog
        open={pendingDelete !== null}
        onOpenChange={(open) => !open && setPendingDelete(null)}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Hapus plan &ldquo;{pendingDelete?.name}&rdquo;?</AlertDialogTitle>
            <AlertDialogDescription>
              Account yang sudah memakai plan ini TIDAK terpengaruh -- kuota device mereka sudah
              tersimpan sendiri sejak dibuat. Plan ini cuma tidak akan muncul lagi di daftar pilihan
              dan landing page.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={busyId === pendingDelete?.id}>Batal</AlertDialogCancel>
            <AlertDialogAction onClick={onDelete} disabled={busyId === pendingDelete?.id}>
              Hapus
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}

function PlanFormDialog({
  initial,
  planId,
  onClose,
  onSaved,
}: {
  initial: PlanInput;
  planId: string | null;
  onClose: () => void;
  onSaved: () => void;
}) {
  const [form, setForm] = useState(initial);
  const [featureDraft, setFeatureDraft] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  function addFeature() {
    const value = featureDraft.trim();
    if (!value) return;
    setForm((f) => ({ ...f, features: [...f.features, value] }));
    setFeatureDraft("");
  }

  function removeFeature(index: number) {
    setForm((f) => ({ ...f, features: f.features.filter((_, i) => i !== index) }));
  }

  async function onSubmit() {
    setError(null);
    if (!form.name.trim()) {
      setError("Nama plan wajib diisi.");
      return;
    }
    if (!form.unlimited && form.max_devices < 1) {
      setError("Jumlah device harus lebih dari 0, atau centang Unlimited.");
      return;
    }
    setBusy(true);
    try {
      if (planId) {
        await updatePlan(planId, form);
        toast.success(`Plan "${form.name}" diperbarui.`);
      } else {
        await createPlan(form);
        toast.success(`Plan "${form.name}" dibuat.`);
      }
      onSaved();
    } catch (err) {
      if (err instanceof ApiError && err.code === "name_taken") {
        setError("Nama plan ini sudah dipakai plan lain.");
      } else if (err instanceof ApiError) {
        setError(err.message);
      } else {
        setError("Gagal menyimpan plan.");
      }
    } finally {
      setBusy(false);
    }
  }

  return (
    <AlertDialog open onOpenChange={(open) => !open && !busy && onClose()}>
      <AlertDialogContent className="max-w-lg">
        <AlertDialogHeader>
          <AlertDialogTitle>{planId ? "Edit Plan" : "Tambah Plan"}</AlertDialogTitle>
          <AlertDialogDescription>
            Nama dan jumlah device menentukan kuota account yang memakai plan ini. Sisanya (harga,
            deskripsi, fitur) cuma teks yang tampil di section Harga landing page.
          </AlertDialogDescription>
        </AlertDialogHeader>

        <div className="flex max-h-[60vh] flex-col gap-4 overflow-y-auto py-1 pr-1">
          {error && (
            <Alert variant="destructive">
              <AlertDescription>{error}</AlertDescription>
            </Alert>
          )}

          <div className="flex flex-col gap-2">
            <Label htmlFor="plan-name">Nama plan</Label>
            <Input
              id="plan-name"
              value={form.name}
              onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))}
              placeholder="Starter"
              autoFocus
            />
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div className="flex flex-col gap-2">
              <Label htmlFor="plan-max-devices">Jumlah device</Label>
              <Input
                id="plan-max-devices"
                type="number"
                min={1}
                value={form.max_devices}
                disabled={form.unlimited}
                onChange={(e) => setForm((f) => ({ ...f, max_devices: Number(e.target.value) }))}
              />
            </div>
            <label className="flex items-center gap-2 self-end pb-2 text-sm">
              <Checkbox
                checked={form.unlimited}
                onCheckedChange={(checked) =>
                  setForm((f) => ({ ...f, unlimited: checked === true }))
                }
              />
              Unlimited
            </label>
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div className="flex flex-col gap-2">
              <Label htmlFor="plan-price-label">Harga</Label>
              <Input
                id="plan-price-label"
                value={form.price_label}
                onChange={(e) => setForm((f) => ({ ...f, price_label: e.target.value }))}
                placeholder="Rp 149.000"
              />
            </div>
            <div className="flex flex-col gap-2">
              <Label htmlFor="plan-price-period">Periode</Label>
              <Input
                id="plan-price-period"
                value={form.price_period}
                onChange={(e) => setForm((f) => ({ ...f, price_period: e.target.value }))}
                placeholder="/bulan"
                disabled={!form.price_label.trim()}
              />
            </div>
          </div>
          <p className="-mt-2 text-xs text-muted-foreground">
            Kosongkan Harga bila belum ditentukan -- section Harga landing page tidak akan
            menampilkan baris harga sama sekali untuk plan ini, bukan angka 0.
          </p>

          <div className="flex flex-col gap-2">
            <Label htmlFor="plan-description">Deskripsi singkat</Label>
            <Input
              id="plan-description"
              value={form.description}
              onChange={(e) => setForm((f) => ({ ...f, description: e.target.value }))}
              placeholder="Untuk usaha kecil yang baru mulai."
            />
          </div>

          <div className="flex flex-col gap-2">
            <Label htmlFor="plan-feature-draft">Daftar fitur</Label>
            <div className="flex gap-2">
              <Input
                id="plan-feature-draft"
                value={featureDraft}
                onChange={(e) => setFeatureDraft(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === "Enter") {
                    e.preventDefault();
                    addFeature();
                  }
                }}
                placeholder="Webhook & retry"
              />
              <Button type="button" variant="outline" onClick={addFeature}>
                Tambah
              </Button>
            </div>
            {form.features.length > 0 && (
              <ul className="flex flex-col gap-1.5">
                {form.features.map((f, i) => (
                  <li
                    key={`${f}-${i}`}
                    className="flex items-center justify-between gap-2 rounded-lg bg-muted/50 px-3 py-1.5 text-sm"
                  >
                    {f}
                    <button
                      type="button"
                      onClick={() => removeFeature(i)}
                      aria-label={`Hapus fitur ${f}`}
                      className="text-muted-foreground hover:text-destructive"
                    >
                      <X className="size-3.5" />
                    </button>
                  </li>
                ))}
              </ul>
            )}
          </div>

          <div className="flex flex-col gap-2">
            <label className="flex items-center gap-2 text-sm">
              <Checkbox
                checked={form.highlighted}
                onCheckedChange={(checked) =>
                  setForm((f) => ({ ...f, highlighted: checked === true }))
                }
              />
              Tandai &ldquo;Direkomendasikan&rdquo; di landing page
            </label>
            <label className="flex items-center gap-2 text-sm">
              <Checkbox
                checked={form.visible}
                onCheckedChange={(checked) => setForm((f) => ({ ...f, visible: checked === true }))}
              />
              Tampilkan di section Harga landing page
            </label>
            {!form.visible && (
              <p className="text-xs text-muted-foreground">
                Tetap bisa dipilih saat membuat/mengubah account, cuma disembunyikan dari publik.
              </p>
            )}
          </div>
        </div>

        <AlertDialogFooter>
          <AlertDialogCancel disabled={busy} onClick={onClose}>
            Batal
          </AlertDialogCancel>
          <AlertDialogAction onClick={onSubmit} disabled={busy}>
            {busy ? "Menyimpan..." : "Simpan"}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
