"use client";

import { useEffect, useRef, useState } from "react";
import { Check, ChevronDown, Search } from "lucide-react";
import { cn } from "@/lib/utils";

export interface FilterOption {
  value: string;
  label: string;
}

/**
 * Dropdown filter dengan kotak pencarian di dalamnya, dipakai untuk daftar
 * pilihan pendek (sumber pembayaran, status, dsb). Sengaja tidak dibangun di
 * atas DropdownMenu (base-ui Menu) — Menu punya navigasi keyboard dan
 * type-ahead sendiri yang akan berebut fokus dengan input teks di dalamnya.
 * Implementasi manual: state buka/tutup sendiri, klik-di-luar menutup,
 * Escape menutup.
 */
export function FilterDropdown({
  allLabel,
  value,
  options,
  onChange,
  searchPlaceholder = "Cari...",
  className,
}: {
  allLabel: string;
  value: string;
  options: FilterOption[];
  onChange: (value: string) => void;
  searchPlaceholder?: string;
  className?: string;
}) {
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState("");
  const containerRef = useRef<HTMLDivElement>(null);
  const searchRef = useRef<HTMLInputElement>(null);

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
    searchRef.current?.focus();
    return () => {
      document.removeEventListener("mousedown", onPointerDown);
      document.removeEventListener("keydown", onKeyDown);
    };
  }, [open]);

  const selectedLabel = options.find((o) => o.value === value)?.label ?? allLabel;
  const filtered =
    query.trim() === ""
      ? options
      : options.filter((o) => o.label.toLowerCase().includes(query.trim().toLowerCase()));

  function select(next: string) {
    onChange(next);
    setOpen(false);
    setQuery("");
  }

  return (
    <div ref={containerRef} className={cn("relative", className)}>
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        className="flex h-9 w-full items-center justify-between gap-2 rounded-lg border bg-background px-3 text-sm text-foreground/80 transition-colors hover:bg-secondary/60"
      >
        <span className="truncate">{selectedLabel}</span>
        <ChevronDown className="size-4 shrink-0 text-muted-foreground" />
      </button>

      {open && (
        <div className="absolute top-full left-0 z-30 mt-1.5 w-56 overflow-hidden rounded-xl border bg-popover text-popover-foreground shadow-md ring-1 ring-foreground/10">
          <div className="flex items-center gap-2 border-b px-2.5 py-2">
            <Search className="size-4 shrink-0 text-muted-foreground" />
            <input
              ref={searchRef}
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder={searchPlaceholder}
              className="w-full bg-transparent text-sm outline-none placeholder:text-muted-foreground"
            />
          </div>
          <div className="max-h-64 overflow-y-auto p-1">
            <button
              type="button"
              onClick={() => select("")}
              className={cn(
                "flex w-full items-center gap-2 rounded-md px-2.5 py-1.5 text-left text-sm transition-colors hover:bg-secondary",
                value === "" && "bg-primary/10 text-primary",
              )}
            >
              <Check className={cn("size-3.5 shrink-0", value !== "" && "invisible")} />
              {allLabel}
            </button>
            {filtered.map((o) => (
              <button
                key={o.value}
                type="button"
                onClick={() => select(o.value)}
                className={cn(
                  "flex w-full items-center gap-2 rounded-md px-2.5 py-1.5 text-left text-sm transition-colors hover:bg-secondary",
                  value === o.value && "bg-primary/10 text-primary",
                )}
              >
                <Check className={cn("size-3.5 shrink-0", value !== o.value && "invisible")} />
                <span className="truncate">{o.label}</span>
              </button>
            ))}
            {filtered.length === 0 && (
              <p className="px-2.5 py-3 text-center text-xs text-muted-foreground">
                Tidak ada yang cocok.
              </p>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
