"use client";

import { useState, type ReactNode } from "react";
import { useRouter } from "next/navigation";
import { Menu, LogOut } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Sheet, SheetContent, SheetTrigger, SheetTitle } from "@/components/ui/sheet";
import { toast } from "sonner";
import { logout } from "@/lib/api";
import { SidebarNav } from "./sidebar-nav";

export function AppShell({ children }: { children: ReactNode }) {
  const router = useRouter();
  const [mobileOpen, setMobileOpen] = useState(false);
  const [loggingOut, setLoggingOut] = useState(false);

  async function onLogout() {
    setLoggingOut(true);
    try {
      await logout();
    } finally {
      // Navigasi tetap terjadi walau permintaan logout gagal — cookie sesi
      // yang salah tidak boleh mengunci orang di dalam dashboard.
      router.push("/login");
    }
  }

  return (
    <div className="flex min-h-screen">
      {/* Sidebar tetap, hanya pada layar cukup lebar. */}
      <aside className="hidden w-60 shrink-0 border-r bg-sidebar md:block">
        <div className="flex h-14 items-center border-b px-4">
          <span className="text-sm font-semibold">Payment Bridge</span>
        </div>
        <SidebarNav />
      </aside>

      <div className="flex min-w-0 flex-1 flex-col">
        <header className="flex h-14 shrink-0 items-center gap-3 border-b px-4">
          <Sheet open={mobileOpen} onOpenChange={setMobileOpen}>
            {/* base-ui (bukan Radix) memakai prop `render`, bukan `asChild`,
                untuk mengganti elemen yang dirender Trigger. */}
            <SheetTrigger render={<Button variant="ghost" size="icon" className="md:hidden" />}>
              <Menu className="size-5" />
            </SheetTrigger>
            <SheetContent side="left" className="w-64 p-0">
              <SheetTitle className="sr-only">Navigasi</SheetTitle>
              <div className="flex h-14 items-center border-b px-4">
                <span className="text-sm font-semibold">Payment Bridge</span>
              </div>
              <SidebarNav onNavigate={() => setMobileOpen(false)} />
            </SheetContent>
          </Sheet>

          <div className="flex-1" />

          <Button
            variant="ghost"
            size="sm"
            onClick={() => {
              onLogout().catch(() => toast.error("Gagal keluar, coba lagi."));
            }}
            disabled={loggingOut}
          >
            <LogOut className="mr-1.5 size-4" />
            Keluar
          </Button>
        </header>

        <main className="flex-1 overflow-x-hidden p-4 md:p-6">{children}</main>
      </div>
    </div>
  );
}
