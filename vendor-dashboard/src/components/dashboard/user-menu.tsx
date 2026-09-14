"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { ChevronDown, LogOut, UserRound } from "lucide-react";
import toast from "react-hot-toast";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
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
import { getVendorMe, logout } from "@/lib/api";

/**
 * Avatar + menu akun di header: ke halaman Profile, atau keluar lewat modal
 * konfirmasi (pola AlertDialog yang sama dengan suspend/revoke account).
 *
 * Kalau GET /vendor/me gagal, avatar tetap tampil dengan inisial "V" --
 * menu keluar tidak boleh ikut hilang gara-gara nama gagal dimuat. Sesi yang
 * sungguh tidak valid sudah ditangani halaman lewat useApiData.
 */
export function UserMenu() {
  const router = useRouter();
  const [username, setUsername] = useState<string | null>(null);
  const [confirmOpen, setConfirmOpen] = useState(false);
  const [loggingOut, setLoggingOut] = useState(false);

  useEffect(() => {
    getVendorMe()
      .then((me) => setUsername(me.username))
      .catch(() => setUsername(null));
  }, []);

  async function onLogout() {
    setLoggingOut(true);
    try {
      await logout();
    } catch {
      toast.error("Gagal menghubungi server, sesi di browser ini tetap diakhiri.");
    } finally {
      // Navigasi tetap terjadi walau permintaan logout gagal -- cookie sesi
      // yang salah tidak boleh mengunci orang di dalam dashboard.
      router.push("/login");
    }
  }

  const initial = (username?.[0] ?? "V").toUpperCase();

  return (
    <>
      <DropdownMenu>
        <DropdownMenuTrigger
          className="flex items-center gap-2 rounded-xl py-1 pr-2 pl-1 outline-none hover:bg-muted focus-visible:ring-2 focus-visible:ring-ring data-popup-open:bg-muted"
          aria-label="Menu akun"
        >
          <Avatar>
            <AvatarFallback className="bg-primary text-xs font-semibold text-primary-foreground">
              {initial}
            </AvatarFallback>
          </Avatar>
          <span className="hidden max-w-32 truncate text-sm font-medium sm:inline">
            {username ?? "Vendor"}
          </span>
          <ChevronDown className="size-4 text-muted-foreground" />
        </DropdownMenuTrigger>

        <DropdownMenuContent align="end" sideOffset={8} className="w-56">
          <div className="flex items-center gap-2.5 px-1.5 py-2">
            <Avatar>
              <AvatarFallback className="bg-primary text-xs font-semibold text-primary-foreground">
                {initial}
              </AvatarFallback>
            </Avatar>
            <div className="min-w-0">
              <p className="truncate text-sm font-medium">{username ?? "Vendor"}</p>
              <p className="text-xs text-muted-foreground">Vendor Console</p>
            </div>
          </div>
          <DropdownMenuSeparator />
          <DropdownMenuItem onClick={() => router.push("/profile")}>
            <UserRound />
            Profile
          </DropdownMenuItem>
          <DropdownMenuSeparator />
          <DropdownMenuItem variant="destructive" onClick={() => setConfirmOpen(true)}>
            <LogOut />
            Keluar
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>

      <AlertDialog open={confirmOpen} onOpenChange={(open) => !loggingOut && setConfirmOpen(open)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Keluar dari Vendor Console?</AlertDialogTitle>
            <AlertDialogDescription>
              Sesi di browser ini akan diakhiri. Kamu perlu login lagi untuk mengelola account.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={loggingOut}>Batal</AlertDialogCancel>
            <AlertDialogAction onClick={onLogout} disabled={loggingOut}>
              {loggingOut ? "Keluar..." : "Keluar"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  );
}
