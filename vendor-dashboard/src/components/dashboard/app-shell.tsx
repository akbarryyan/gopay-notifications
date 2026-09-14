"use client";

import { useEffect, useState, type ReactNode } from "react";
import { useRouter } from "next/navigation";
import { LogOut, Menu, PanelLeftClose, PanelLeftOpen, ShieldCheck } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Sheet, SheetContent, SheetTrigger, SheetTitle } from "@/components/ui/sheet";
import toast from "react-hot-toast";
import { logout } from "@/lib/api";
import { cn } from "@/lib/utils";
import { SidebarNav } from "./sidebar-nav";

const COLLAPSE_STORAGE_KEY = "vc.sidebarCollapsed";

function LogoMark({ collapsed }: { collapsed?: boolean }) {
  return (
    <div className="flex items-center gap-2.5">
      <span className="flex size-8 shrink-0 items-center justify-center rounded-xl bg-primary text-primary-foreground">
        <ShieldCheck className="size-4.5" />
      </span>
      {!collapsed && (
        <span className="text-sm font-semibold tracking-tight text-sidebar-foreground">
          Vendor Console
        </span>
      )}
    </div>
  );
}

// Layout sidebar collapsible ini sengaja disamakan strukturnya dengan
// AppShell di dashboard/ (Customer Dashboard) -- token warna (--sidebar-*),
// lebar, perilaku collapse/localStorage, dan drawer mobile semuanya pola
// yang sama, cuma LogoMark dan daftar nav yang beda konten. Vendor Console
// tidak punya kotak pencarian "Segera" seperti Customer Dashboard karena
// tidak ada rencana fitur pencarian di sini.
export function AppShell({ children }: { children: ReactNode }) {
  const router = useRouter();
  const [mobileOpen, setMobileOpen] = useState(false);
  const [loggingOut, setLoggingOut] = useState(false);
  const [collapsed, setCollapsed] = useState(false);

  // Kenyamanan per-browser saja (bukan state yang perlu dibagi atau tahan
  // lama) — wajar dipertahankan lewat localStorage, dibungkus try/catch
  // karena bisa gagal di mode privat.
  useEffect(() => {
    // localStorage tidak ada di server, jadi nilai sungguhannya baru bisa
    // dibaca setelah mount — react-hooks/set-state-in-effect menandai ini,
    // tapi tidak ada cara membaca localStorage lebih awal dari ini.
    try {
      // eslint-disable-next-line react-hooks/set-state-in-effect
      setCollapsed(window.localStorage.getItem(COLLAPSE_STORAGE_KEY) === "1");
    } catch {
      // Diamkan — default terbuka sudah benar.
    }
  }, []);

  function toggleCollapsed() {
    setCollapsed((prev) => {
      const next = !prev;
      try {
        window.localStorage.setItem(COLLAPSE_STORAGE_KEY, next ? "1" : "0");
      } catch {
        // Diamkan — state tetap berubah untuk sesi ini walau tidak tersimpan.
      }
      return next;
    });
  }

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
    <div className="flex min-h-screen bg-background">
      {/* Sidebar tetap, hanya pada layar cukup lebar. Lebarnya bisa
          di-collapse jadi rel ikon lewat tombol di sebelah LogoMark. */}
      <aside
        className={cn(
          "hidden shrink-0 border-r border-sidebar-border bg-sidebar transition-[width] duration-200 md:block",
          collapsed ? "w-21" : "w-64",
        )}
      >
        <div
          className={cn(
            "flex h-16 items-center border-b border-sidebar-border",
            collapsed ? "justify-center gap-1 px-2" : "justify-between px-4",
          )}
        >
          <LogoMark collapsed={collapsed} />
          <Button
            variant="ghost"
            size="icon"
            className="size-8 shrink-0 rounded-lg text-sidebar-foreground/60 hover:bg-white/10 hover:text-sidebar-foreground"
            onClick={toggleCollapsed}
            title={collapsed ? "Buka sidebar" : "Ciutkan sidebar"}
          >
            {collapsed ? <PanelLeftOpen className="size-4" /> : <PanelLeftClose className="size-4" />}
          </Button>
        </div>
        <SidebarNav collapsed={collapsed} />
      </aside>

      <div className="flex min-w-0 flex-1 flex-col">
        <header className="flex h-16 shrink-0 items-center justify-between gap-3 border-b bg-card/60 px-4 backdrop-blur md:px-6">
          <Sheet open={mobileOpen} onOpenChange={setMobileOpen}>
            {/* base-ui (bukan Radix) memakai prop `render`, bukan `asChild`,
                untuk mengganti elemen yang dirender Trigger. */}
            <SheetTrigger render={<Button variant="ghost" size="icon" className="md:hidden" />}>
              <Menu className="size-5" />
            </SheetTrigger>
            <SheetContent
              side="left"
              className="w-64 border-sidebar-border bg-sidebar p-0 text-sidebar-foreground"
            >
              <SheetTitle className="sr-only">Navigasi</SheetTitle>
              <div className="flex h-16 items-center border-b border-sidebar-border px-4">
                <LogoMark />
              </div>
              <SidebarNav onNavigate={() => setMobileOpen(false)} />
            </SheetContent>
          </Sheet>

          <div className="flex-1" />

          <Button
            variant="ghost"
            size="sm"
            className="rounded-xl"
            onClick={() => {
              onLogout().catch(() => toast.error("Gagal keluar, coba lagi."));
            }}
            disabled={loggingOut}
          >
            <LogOut className="mr-1.5 size-4" />
            Keluar
          </Button>
        </header>

        <main className="flex-1 overflow-x-hidden p-4 md:p-8">{children}</main>
      </div>
    </div>
  );
}
