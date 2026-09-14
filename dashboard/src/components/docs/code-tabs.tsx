"use client";

import { useState } from "react";
import { Check, Copy } from "lucide-react";
import toast from "react-hot-toast";
import { cn } from "@/lib/utils";

export interface CodeSample {
  label: string;
  code: string;
}

/**
 * Blok kode dengan tab bahasa (curl/Node/PHP) dan tombol salin. Satu sampel
 * saja berarti tanpa tab, cuma judul kecil.
 */
export function CodeTabs({ samples, className }: { samples: CodeSample[]; className?: string }) {
  const [active, setActive] = useState(0);
  const [copied, setCopied] = useState(false);
  const current = samples[Math.min(active, samples.length - 1)];

  async function onCopy() {
    try {
      await navigator.clipboard.writeText(current.code);
      setCopied(true);
      setTimeout(() => setCopied(false), 1500);
    } catch {
      toast.error("Gagal menyalin, salin manual dari kotak kode.");
    }
  }

  return (
    <div className={cn("overflow-hidden rounded-xl bg-slate-950 ring-1 ring-slate-800", className)}>
      <div className="flex items-center justify-between gap-2 border-b border-slate-800 px-2">
        <div className="flex min-w-0 overflow-x-auto" role="tablist">
          {samples.map((s, i) => (
            <button
              key={s.label}
              type="button"
              role="tab"
              aria-selected={i === active}
              onClick={() => setActive(i)}
              className={cn(
                "shrink-0 border-b-2 px-3 py-2 text-xs font-medium transition-colors",
                i === active
                  ? "border-teal-400 text-slate-100"
                  : "border-transparent text-slate-400 hover:text-slate-200",
              )}
            >
              {s.label}
            </button>
          ))}
        </div>
        <button
          type="button"
          onClick={onCopy}
          className="flex shrink-0 items-center gap-1 rounded-md px-2 py-1 text-xs text-slate-400 hover:bg-slate-800 hover:text-slate-100"
          aria-label="Salin kode"
        >
          {copied ? <Check className="size-3.5" /> : <Copy className="size-3.5" />}
          {copied ? "Tersalin" : "Salin"}
        </button>
      </div>
      <pre className="overflow-x-auto p-4 text-[13px] leading-relaxed text-slate-100">
        <code>{current.code}</code>
      </pre>
    </div>
  );
}
