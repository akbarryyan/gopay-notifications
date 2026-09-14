"use client";

import { useState, type FormEvent, type ReactNode } from "react";
import { Bell, Eye, EyeOff, KeyRound, RotateCw, UserRound } from "lucide-react";
import toast from "react-hot-toast";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import {
  ApiError,
  changePassword,
  getAccountProfile,
  setTelegramChatID,
  updateAccountProfile,
  type AccountProfile,
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
          <Label htmlFor="email">Email</Label>
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

/** Dipindah dari halaman License -- tempatnya memang di pengaturan akun. */
function NotificationCard({ profile, onSaved }: { profile: AccountProfile; onSaved: () => void }) {
  const [chatID, setChatID] = useState(profile.telegram_chat_id ?? "");
  const [busy, setBusy] = useState(false);
  const dirty = chatID.trim() !== (profile.telegram_chat_id ?? "");

  async function onSave() {
    setBusy(true);
    try {
      await setTelegramChatID(chatID.trim());
      toast.success(
        chatID.trim() === "" ? "Notifikasi Telegram dimatikan." : "Chat ID Telegram disimpan.",
      );
      onSaved();
    } catch (err) {
      if (err instanceof ApiError && err.code === "invalid_payload") {
        toast.error(err.message);
      } else {
        toast.error("Gagal menyimpan chat ID Telegram.");
      }
    } finally {
      setBusy(false);
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
          kembali online), dan saat password akun diganti. Tambahkan Telegram kalau mau dikabari di
          sana juga.
        </>
      }
    >
      <div className="flex flex-col gap-2 sm:max-w-sm">
        <Label htmlFor="telegram-chat-id">Telegram chat ID (opsional)</Label>
        <Input
          id="telegram-chat-id"
          value={chatID}
          onChange={(e) => setChatID(e.target.value)}
          placeholder="123456789"
          inputMode="numeric"
        />
        <p className="text-xs text-muted-foreground">
          Berupa angka, bukan username. Kirim pesan apa saja ke bot <code>@userinfobot</code> di
          Telegram untuk melihat chat ID kamu. Kosongkan untuk berhenti menerima notifikasi
          Telegram.
        </p>
        <Button size="sm" className="mt-1 self-start" disabled={busy || !dirty} onClick={onSave}>
          {busy ? "Menyimpan..." : "Simpan"}
        </Button>
      </div>
    </SettingsCard>
  );
}
