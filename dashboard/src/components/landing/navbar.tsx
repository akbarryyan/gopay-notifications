"use client";

import { useState } from "react";
import Link from "next/link";
import { Menu, Zap } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Sheet, SheetContent, SheetTitle, SheetTrigger } from "@/components/ui/sheet";

const NAV_LINKS = [
  { label: "Fitur", href: "#fitur" },
  { label: "Cara Kerja", href: "#cara-kerja" },
  { label: "Harga", href: "#harga" },
  { label: "FAQ", href: "#faq" },
];

export function LandingNavbar() {
  const [mobileOpen, setMobileOpen] = useState(false);

  return (
    <header className="sticky top-0 z-40 border-b border-slate-100 bg-white/90 backdrop-blur-md">
      <div className="mx-auto flex h-16 max-w-6xl items-center justify-between px-4 sm:px-6 lg:px-8">
        <Link href="/" className="flex items-center gap-2 font-semibold tracking-tight text-slate-900">
          <span className="flex size-8 items-center justify-center rounded-lg bg-slate-900 text-white">
            <Zap className="size-4" fill="currentColor" strokeWidth={0} />
          </span>
          Payment Bridge
        </Link>

        <nav className="hidden items-center gap-1 md:flex">
          {NAV_LINKS.map((link) => (
            <a
              key={link.href}
              href={link.href}
              className="rounded-full px-3.5 py-2 text-sm font-medium text-slate-500 transition-colors duration-200 hover:bg-slate-50 hover:text-slate-900"
            >
              {link.label}
            </a>
          ))}
        </nav>

        <div className="hidden items-center gap-2 md:flex">
          <Link
            href="/login"
            className="rounded-full px-4 py-2 text-sm font-medium text-slate-600 transition-colors duration-200 hover:text-slate-900"
          >
            Masuk
          </Link>
          <Link
            href="/register"
            className="rounded-full bg-teal-500 px-4 py-2 text-sm font-semibold text-white shadow-sm shadow-teal-500/30 transition-all duration-200 hover:-translate-y-0.5 hover:bg-teal-600 hover:shadow-md hover:shadow-teal-500/40 active:translate-y-0 active:scale-[0.97]"
          >
            Daftar
          </Link>
        </div>

        <Sheet open={mobileOpen} onOpenChange={setMobileOpen}>
          {/* base-ui (bukan Radix) memakai prop `render`, bukan `asChild`. */}
          <SheetTrigger render={<Button variant="ghost" size="icon" className="md:hidden" />}>
            <Menu className="size-5" />
          </SheetTrigger>
          <SheetContent side="right" className="w-72 p-0">
            <SheetTitle className="sr-only">Navigasi</SheetTitle>
            <div className="flex h-16 items-center gap-2 border-b border-slate-100 px-4 font-semibold text-slate-900">
              <span className="flex size-8 items-center justify-center rounded-lg bg-slate-900 text-white">
                <Zap className="size-4" fill="currentColor" strokeWidth={0} />
              </span>
              Payment Bridge
            </div>
            <nav className="flex flex-col gap-1 p-4">
              {NAV_LINKS.map((link) => (
                <a
                  key={link.href}
                  href={link.href}
                  onClick={() => setMobileOpen(false)}
                  className="rounded-lg px-3 py-2 text-sm font-medium text-slate-500 transition-colors duration-200 hover:bg-slate-50 hover:text-slate-900"
                >
                  {link.label}
                </a>
              ))}
              <div className="mt-4 flex flex-col gap-2 border-t border-slate-100 pt-4">
                <Link
                  href="/login"
                  onClick={() => setMobileOpen(false)}
                  className="rounded-full border border-slate-200 px-4 py-2 text-center text-sm font-medium text-slate-700 transition-colors duration-200 hover:bg-slate-50"
                >
                  Masuk
                </Link>
                <Link
                  href="/register"
                  onClick={() => setMobileOpen(false)}
                  className="rounded-full bg-teal-500 px-4 py-2 text-center text-sm font-semibold text-white transition-colors duration-200 hover:bg-teal-600"
                >
                  Daftar
                </Link>
              </div>
            </nav>
          </SheetContent>
        </Sheet>
      </div>
    </header>
  );
}
