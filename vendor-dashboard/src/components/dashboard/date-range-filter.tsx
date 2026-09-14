"use client";

import { useEffect, useRef, useState } from "react";
import { CalendarRange } from "lucide-react";
import { Button } from "@/components/ui/button";

/**
 * Kontrol rentang tanggal ringkas: tombol yang membuka panel berisi dua
 * <input type="date"> (Dari / Sampai). Bukan calendar picker penuh — cukup
 * untuk memfilter berdasarkan tanggal terima event tanpa membangun komponen
 * kalender sendiri.
 */
export function DateRangeFilter({
  from,
  to,
  onChange,
}: {
  from: string;
  to: string;
  onChange: (next: { from: string; to: string }) => void;
}) {
  const [open, setOpen] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open) return;
    function onPointerDown(e: MouseEvent) {
      if (containerRef.current && !containerRef.current.contains(e.target as Node)) {
        setOpen(false);
      }
    }
    function onKeyDown(e: KeyboardEvent) {
      if (e.key === "Escape") setOpen(false);
    }
    document.addEventListener("mousedown", onPointerDown);
    document.addEventListener("keydown", onKeyDown);
    return () => {
      document.removeEventListener("mousedown", onPointerDown);
      document.removeEventListener("keydown", onKeyDown);
    };
  }, [open]);

  const label = from || to ? `${from || "…"} – ${to || "…"}` : "Pilih rentang tanggal";

  return (
    <div ref={containerRef} className="relative">
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        className="flex h-9 items-center gap-2 rounded-lg border bg-background px-3 text-sm text-foreground/80 transition-colors hover:bg-secondary/60"
      >
        <CalendarRange className="size-4 shrink-0 text-muted-foreground" />
        <span className="truncate">{label}</span>
      </button>

      {open && (
        <div className="absolute top-full left-0 z-30 mt-1.5 w-64 rounded-xl border bg-popover p-3 text-popover-foreground shadow-md ring-1 ring-foreground/10">
          <div className="flex flex-col gap-2">
            <label className="flex flex-col gap-1 text-xs text-muted-foreground">
              Dari
              <input
                type="date"
                value={from}
                max={to || undefined}
                onChange={(e) => onChange({ from: e.target.value, to })}
                className="rounded-md border bg-background px-2 py-1.5 text-sm text-foreground"
              />
            </label>
            <label className="flex flex-col gap-1 text-xs text-muted-foreground">
              Sampai
              <input
                type="date"
                value={to}
                min={from || undefined}
                onChange={(e) => onChange({ from, to: e.target.value })}
                className="rounded-md border bg-background px-2 py-1.5 text-sm text-foreground"
              />
            </label>
          </div>
          {(from || to) && (
            <Button
              type="button"
              variant="ghost"
              size="sm"
              className="mt-2 w-full"
              onClick={() => onChange({ from: "", to: "" })}
            >
              Bersihkan
            </Button>
          )}
        </div>
      )}
    </div>
  );
}
