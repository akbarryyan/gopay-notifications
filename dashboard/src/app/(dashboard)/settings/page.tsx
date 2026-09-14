"use client";

import { useEffect, useState, type FormEvent, type ReactNode } from "react";
import {
  Bell,
  Eye,
  EyeOff,
  KeyRound,
  Loader2,
  MailWarning,
  RotateCw,
  Send,
  Unlink,
  UserRound,
} from "lucide-react";
import toast from "react-hot-toast";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button, buttonVariants } from "@/components/ui/button";
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
import { cn } from "@/lib/utils";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import {
  ApiError,
  changePassword,
  createTelegramLink,
  getAccountProfile,
  resendVerificationEmail,
  setTelegramChatID,
  updateAccountProfile,
  type AccountProfile,
  type TelegramLink,
} from "@/lib/api";
import { useApiData } from "@/lib/use-api-data";

export default function SettingsPage() {
  const { data, loading, error, reload } = useApiData(getAccountProfile);

  return (
    <div className="flex flex-col gap-6">
      <div>
        <h1 className="text-2xl font-semibold">Settings</h1>
        <p className="text-sm text-muted-foreground">
          Profil bisnis, keamanan akun, dan tujuan notifikasi.
        </p>
      </div>

      {error && (
        <Alert variant="destructive">
          <AlertTitle>Tidak dapat memuat pengaturan akun</AlertTitle>
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
        <div className="grid gap-6 lg:grid-cols-2">
          <Skeleton className="h-80 rounded-2xl" />
          <Skeleton className="h-80 rounded-2xl" />
        </div>
      ) : data ? (
        <div className="flex flex-col gap-6">
          {!data.email_verified && <VerifyEmailBanner email={data.email} />}
          <div className="grid items-start gap-6 lg:grid-cols-2">
            {/* key: form diinisialisasi ulang dari data terbaru setelah disimpan. */}
            <ProfileCard
              key={`${data.business_name}|${data.email}`}
              profile={data}
              onSaved={reload}
            />
            <PasswordCard />
            <div className="lg:col-span-2">
              <NotificationCard key={data.telegram_chat_id ?? ""} profile={data} onSaved={reload} />
            </div>
          </div>
        </div>
      ) : null}
    </div>
  );
}

function SettingsCard({
  icon,
  title,
  description,
  children,
}: {
  icon: ReactNode;
  title: string;
  description: ReactNode;
  children: ReactNode;
}) {
  return (
    <section className="rounded-2xl border border-border/60 p-6 shadow-sm">
      <div className="flex items-center gap-2">
        {icon}
        <h2 className="text-base font-semibold">{title}</h2>
      </div>
      <p className="mt-1 text-sm text-muted-foreground">{description}</p>
      <div className="mt-5">{children}</div>
    </section>
  );
}

function ProfileCard({ profile, onSaved }: { profile: AccountProfile; onSaved: () => void }) {
  const [businessName, setBusinessName] = useState(profile.business_name);
  const [email, setEmail] = useState(profile.email);
  const [currentPassword, setCurrentPassword] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const emailChanged = email.trim().toLowerCase() !== profile.email.toLowerCase();
  const dirty = businessName.trim() !== profile.business_name || email.trim() !== profile.email;

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    setError(null);
    setBusy(true);
    try {
      await updateAccountProfile({
        business_name: businessName.trim(),
        email: email.trim(),
        current_password: emailChanged ? currentPassword : undefined,
      });
      toast.success("Profil disimpan.");
      onSaved();
    } catch (err) {
      if (err instanceof ApiError && err.code === "invalid_credentials") {
        setError("Password saat ini salah.");
      } else if (err instanceof ApiError && err.code === "email_taken") {
        setError("Email ini sudah dipakai akun lain.");
      } else if (err instanceof ApiError && err.code === "too_many_attempts") {
        setError("Terlalu banyak percobaan. Coba lagi dalam beberapa menit.");
      } else if (err instanceof ApiError) {
        setError(err.message);
      } else {
        setError("Gagal menyimpan profil. Coba lagi.");
      }
    } finally {
      setBusy(false);
    }
  }

  return (
    <SettingsCard
      icon={<UserRound className="size-4 text-muted-foreground" />}
      title="Profil"
      description="Nama bisnis tampil di email dan notifikasi. Email dipakai untuk notifikasi dan reset password."
    >
      <form onSubmit={onSubmit} className="flex flex-col gap-4">
        {error && (
          <Alert variant="destructive">
            <AlertDescription>{error}</AlertDescription>
          </Alert>
        )}

        <div className="flex flex-col gap-2">
          <Label htmlFor="username">Username</Label>
          <Input id="username" value={profile.username} disabled />
          <p className="text-xs text-muted-foreground">Dipakai untuk masuk, tidak bisa diubah.</p>
        </div>

        <div className="flex flex-col gap-2">
          <Label htmlFor="business-name">Nama bisnis</Label>
          <Input
            id="business-name"
            value={businessName}
            onChange={(e) => setBusinessName(e.target.value)}
            required
          />
        </div>

        <div className="flex flex-col gap-2">
          <div className="flex items-center gap-2">
            <Label htmlFor="email">Email</Label>
            {profile.email_verified && !emailChanged && (
              <Badge className="border-transparent bg-emerald-500/15 text-emerald-700 dark:text-emerald-400">
                Terverifikasi
              </Badge>
            )}
          </div>
          <Input
            id="email"
            type="email"
            autoComplete="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            required
          />
        </div>

        {emailChanged && (
          <div className="flex flex-col gap-2 rounded-xl bg-muted/50 p-3">
            <Label htmlFor="profile-current-password">Password saat ini</Label>
            <Input
              id="profile-current-password"
              type="password"
              autoComplete="current-password"
              value={currentPassword}
              onChange={(e) => setCurrentPassword(e.target.value)}
              required
            />
            <p className="text-xs text-muted-foreground">
              Wajib untuk mengganti email, karena link reset password dikirim ke alamat ini.
            </p>
          </div>
        )}

        <Button
          type="submit"
          className="self-start"
          disabled={
            busy ||
            !dirty ||
            !businessName.trim() ||
            !email.trim() ||
            (emailChanged && !currentPassword)
          }
        >
          {busy ? "Menyimpan..." : "Simpan profil"}
        </Button>
      </form>
    </SettingsCard>
  );
}

function PasswordCard() {
  const [currentPassword, setCurrentPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [confirm, setConfirm] = useState("");
  const [show, setShow] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const tooShort = newPassword.length > 0 && newPassword.length < 8;
  const mismatch = confirm.length > 0 && newPassword !== confirm;

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    setError(null);
    setBusy(true);
    try {
      await changePassword(currentPassword, newPassword);
      toast.success("Password diganti. Perangkat lain yang sedang login sudah dikeluarkan.");
      setCurrentPassword("");
      setNewPassword("");
      setConfirm("");
    } catch (err) {
      if (err instanceof ApiError && err.code === "invalid_credentials") {
        setError("Password saat ini salah.");
      } else if (err instanceof ApiError && err.code === "too_many_attempts") {
        setError("Terlalu banyak percobaan. Coba lagi dalam beberapa menit.");
      } else if (err instanceof ApiError) {
        setError(err.message);
      } else {
        setError("Gagal mengganti password. Coba lagi.");
      }
    } finally {
      setBusy(false);
    }
  }

  return (
    <SettingsCard
      icon={<KeyRound className="size-4 text-muted-foreground" />}
      title="Ganti password"
      description="Setelah diganti, semua perangkat lain yang sedang login otomatis dikeluarkan."
    >
      <form onSubmit={onSubmit} className="flex flex-col gap-4">
        {error && (
          <Alert variant="destructive">
            <AlertDescription>{error}</AlertDescription>
          </Alert>
        )}

        <div className="flex flex-col gap-2">
          <Label htmlFor="current-password">Password saat ini</Label>
          <div className="relative">
            <Input
              id="current-password"
              type={show ? "text" : "password"}
              autoComplete="current-password"
              value={currentPassword}
              onChange={(e) => setCurrentPassword(e.target.value)}
              required
              className="pr-10"
            />
            <button
              type="button"
              onClick={() => setShow((v) => !v)}
              aria-label={show ? "Sembunyikan password" : "Tampilkan password"}
              className="absolute inset-y-0 right-0 flex w-9 items-center justify-center text-muted-foreground hover:text-foreground"
            >
              {show ? <EyeOff className="size-4" /> : <Eye className="size-4" />}
            </button>
          </div>
        </div>

        <div className="flex flex-col gap-2">
          <Label htmlFor="new-password">Password baru</Label>
          <Input
            id="new-password"
            type={show ? "text" : "password"}
            autoComplete="new-password"
            value={newPassword}
            onChange={(e) => setNewPassword(e.target.value)}
            required
          />
          {tooShort && <p className="text-xs text-destructive">Minimal 8 karakter.</p>}
        </div>

        <div className="flex flex-col gap-2">
          <Label htmlFor="confirm-password">Ulangi password baru</Label>
          <Input
            id="confirm-password"
            type={show ? "text" : "password"}
            autoComplete="new-password"
            value={confirm}
            onChange={(e) => setConfirm(e.target.value)}
            required
          />
          {mismatch && <p className="text-xs text-destructive">Tidak sama dengan password baru.</p>}
        </div>

        <Button
          type="submit"
          className="self-start"
          disabled={busy || !currentPassword || !newPassword || !confirm || tooShort || mismatch}
        >
          {busy ? "Menyimpan..." : "Ganti password"}
        </Button>
      </form>
    </SettingsCard>
  );
}

/**
 * Telegram dihubungkan lewat deep link, bukan chat id yang diketik: bot
 * Telegram tidak bisa mengirim pesan duluan ke orang yang belum pernah
 * menekan Start di bot itu. Setelah link dibuka, halaman ini menunggu
 * (polling GET /admin/account) sampai backend menyimpan chat id-nya.
 */
function NotificationCard({ profile, onSaved }: { profile: AccountProfile; onSaved: () => void }) {
  const [link, setLink] = useState<TelegramLink | null>(null);
  const [creating, setCreating] = useState(false);
  const [disconnecting, setDisconnecting] = useState(false);
  const [confirmDisconnect, setConfirmDisconnect] = useState(false);
  const connected = profile.telegram_chat_id !== null;

  // Selama link masih berlaku, cek tiap 3 detik apakah customer sudah
  // menekan Start. Berhenti sendiri saat kedaluwarsa atau kartu dilepas.
  useEffect(() => {
    if (!link) return;
    const expiresAt = new Date(link.expires_at).getTime();
    const timer = setInterval(async () => {
      if (Date.now() > expiresAt) {
        clearInterval(timer);
        setLink(null);
        toast.error("Link Telegram kedaluwarsa. Buat link baru untuk mencoba lagi.");
        return;
      }
      try {
        const latest = await getAccountProfile();
        if (latest.telegram_chat_id) {
          clearInterval(timer);
          toast.success("Telegram terhubung.");
          onSaved();
        }
      } catch {
        // Diamkan -- dicoba lagi di detik berikutnya.
      }
    }, 3000);
    return () => clearInterval(timer);
  }, [link, onSaved]);

  async function onCreateLink() {
    setCreating(true);
    try {
      setLink(await createTelegramLink());
    } catch (err) {
      if (err instanceof ApiError && err.code === "not_available") {
        toast.error("Notifikasi Telegram belum tersedia.");
      } else if (err instanceof ApiError && err.code === "telegram_unreachable") {
        toast.error("Tidak dapat menghubungi Telegram. Coba lagi beberapa saat lagi.");
      } else {
        toast.error("Gagal membuat link Telegram.");
      }
    } finally {
      setCreating(false);
    }
  }

  async function onDisconnect() {
    setDisconnecting(true);
    try {
      await setTelegramChatID("");
      toast.success("Telegram diputuskan.");
      setConfirmDisconnect(false);
      onSaved();
    } catch {
      toast.error("Gagal memutuskan Telegram.");
    } finally {
      setDisconnecting(false);
    }
  }

  return (
    <SettingsCard
      icon={<Bell className="size-4 text-muted-foreground" />}
      title="Notifikasi"
      description={
        <>
          Kami mengabari <span className="font-medium text-foreground">{profile.email}</span> saat
          masa aktif tinggal 7 hari, saat HP berhenti mengirim kabar lebih dari 45 menit (serta saat
          kembali online), dan saat password akun diganti. Hubungkan Telegram kalau mau dikabari di
          sana juga.
        </>
      }
    >
      <div className="flex flex-col gap-3 rounded-xl border border-border/60 p-4 sm:max-w-lg">
        <div className="flex items-center gap-2">
          <Send className="size-4 text-muted-foreground" />
          <span className="text-sm font-medium">Telegram</span>
          {connected ? (
            <Badge className="border-transparent bg-emerald-500/15 text-emerald-700 dark:text-emerald-400">
              Terhubung
            </Badge>
          ) : (
            <Badge variant="outline">Belum terhubung</Badge>
          )}
        </div>

        {connected ? (
          <div className="flex flex-wrap items-center justify-between gap-3">
            <p className="text-sm text-muted-foreground">
              Notifikasi juga dikirim ke Telegram kamu.
            </p>
            <Button size="sm" variant="outline" onClick={() => setConfirmDisconnect(true)}>
              <Unlink className="mr-1.5 size-3.5" />
              Putuskan
            </Button>
          </div>
        ) : !profile.telegram_available ? (
          <p className="text-sm text-muted-foreground">
            Notifikasi Telegram belum tersedia untuk saat ini. Notifikasi tetap dikirim lewat email.
          </p>
        ) : link ? (
          <div className="flex flex-col gap-3">
            <ol className="list-decimal space-y-1 pl-5 text-sm text-muted-foreground">
              <li>
                Buka bot <span className="font-medium text-foreground">@{link.bot_username}</span>{" "}
                di Telegram lewat tombol di bawah.
              </li>
              <li>
                Tekan <span className="font-medium text-foreground">Start</span> (atau{" "}
                <span className="font-medium text-foreground">Mulai</span>).
              </li>
              <li>Kembali ke halaman ini, statusnya berubah sendiri.</li>
            </ol>
            <div className="flex flex-wrap items-center gap-2">
              <a
                href={link.url}
                target="_blank"
                rel="noopener noreferrer"
                className={cn(buttonVariants({ size: "sm" }))}
              >
                <Send className="mr-1.5 size-3.5" />
                Buka Telegram
              </a>
              <Button size="sm" variant="ghost" onClick={() => setLink(null)}>
                Batal
              </Button>
            </div>
            <p className="flex items-center gap-2 text-xs text-muted-foreground">
              <Loader2 className="size-3.5 animate-spin" />
              Menunggu kamu menekan Start… Link berlaku sampai{" "}
              {new Date(link.expires_at).toLocaleTimeString("id-ID", {
                hour: "2-digit",
                minute: "2-digit",
              })}
              .
            </p>
          </div>
        ) : (
          <div className="flex flex-wrap items-center justify-between gap-3">
            <p className="text-sm text-muted-foreground">
              Terima notifikasi yang sama langsung di Telegram.
            </p>
            <Button size="sm" onClick={onCreateLink} disabled={creating}>
              <Send className="mr-1.5 size-3.5" />
              {creating ? "Menyiapkan..." : "Hubungkan Telegram"}
            </Button>
          </div>
        )}
      </div>

      <AlertDialog
        open={confirmDisconnect}
        onOpenChange={(open) => !disconnecting && setConfirmDisconnect(open)}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Putuskan Telegram?</AlertDialogTitle>
            <AlertDialogDescription>
              Notifikasi berhenti dikirim ke Telegram dan tetap dikirim lewat email. Kamu bisa
              menghubungkannya lagi kapan saja.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={disconnecting}>Batal</AlertDialogCancel>
            <AlertDialogAction onClick={onDisconnect} disabled={disconnecting}>
              {disconnecting ? "Memutuskan..." : "Putuskan"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </SettingsCard>
  );
}

/** Pengingat, bukan gerbang -- account tetap bisa dipakai penuh sambil ini tampil. */
function VerifyEmailBanner({ email }: { email: string }) {
  const [sending, setSending] = useState(false);
  const [sent, setSent] = useState(false);

  async function onResend() {
    setSending(true);
    try {
      const res = await resendVerificationEmail();
      if (res.already_verified) {
        toast.success("Email sudah terverifikasi.");
      } else {
        setSent(true);
        toast.success("Email verifikasi dikirim.");
      }
    } catch (err) {
      if (err instanceof ApiError && err.code === "not_available") {
        toast.error("Verifikasi email belum tersedia untuk saat ini.");
      } else if (err instanceof ApiError && err.code === "too_many_attempts") {
        toast.error("Tunggu sebentar sebelum meminta email verifikasi lagi.");
      } else {
        toast.error("Gagal mengirim email verifikasi.");
      }
    } finally {
      setSending(false);
    }
  }

  return (
    <div className="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-amber-500/30 bg-amber-500/5 p-4">
      <div className="flex items-start gap-3">
        <MailWarning className="mt-0.5 size-4 shrink-0 text-amber-600 dark:text-amber-400" />
        <div className="text-sm">
          <p className="font-medium text-amber-900 dark:text-amber-200">Email belum diverifikasi</p>
          <p className="text-amber-800/80 dark:text-amber-300/80">
            Pastikan <span className="font-medium">{email}</span> benar supaya pengingat dan link
            reset password sampai ke tempat yang tepat.
          </p>
        </div>
      </div>
      <Button size="sm" variant="outline" onClick={onResend} disabled={sending || sent}>
        {sending ? "Mengirim..." : sent ? "Terkirim" : "Kirim email verifikasi"}
      </Button>
    </div>
  );
}
