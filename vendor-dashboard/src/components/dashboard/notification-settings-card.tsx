"use client";

import { useEffect, useState, type FormEvent } from "react";
import { Eye, EyeOff, Mail, Send } from "lucide-react";
import toast from "react-hot-toast";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { Separator } from "@/components/ui/separator";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import {
  ApiError,
  getNotificationSettings,
  saveNotificationSettings,
  sendTestNotification,
  type NotificationSettings,
  type SaveNotificationSettingsInput,
} from "@/lib/api";

/**
 * "keep" = biarkan yang tersimpan (field tidak dikirim), "replace" = ganti
 * dengan isi input, "clear" = hapus. Kredensial lama tidak pernah dikirim
 * backend, jadi form tidak bisa menampilkannya -- hanya bisa diganti/dihapus.
 */
type SecretMode = "keep" | "replace" | "clear";

interface SecretState {
  mode: SecretMode;
  value: string;
}

const KEEP: SecretState = { mode: "keep", value: "" };

function secretPayload(s: SecretState, isSet: boolean): string | undefined {
  if (s.mode === "clear") return "";
  if (s.mode === "replace") return s.value;
  // Belum pernah tersimpan: input langsung tampil, kosong berarti tidak
  // mengubah apa pun.
  return isSet || s.value === "" ? undefined : s.value;
}

function formatDateTime(iso: string): string {
  return new Date(iso).toLocaleString("id-ID", {
    dateStyle: "medium",
    timeStyle: "short",
  });
}

/**
 * Pengaturan pengiriman pengingat kedaluwarsa ke customer (SMTP + bot
 * Telegram). Tersimpan di database, kredensial terenkripsi -- perubahan
 * berlaku di putaran pengingat berikutnya tanpa restart backend.
 */
export function NotificationSettingsCard() {
  const [saved, setSaved] = useState<NotificationSettings | null>(null);
  const [loadError, setLoadError] = useState(false);

  const [host, setHost] = useState("");
  const [port, setPort] = useState("587");
  const [username, setUsername] = useState("");
  const [from, setFrom] = useState("");
  const [password, setPassword] = useState<SecretState>(KEEP);
  const [botToken, setBotToken] = useState<SecretState>(KEEP);
  const [showSecrets, setShowSecrets] = useState(false);

  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  function applySaved(s: NotificationSettings) {
    setSaved(s);
    setHost(s.smtp_host);
    setPort(String(s.smtp_port));
    setUsername(s.smtp_username);
    setFrom(s.smtp_from);
    setPassword(KEEP);
    setBotToken(KEEP);
  }

  useEffect(() => {
    getNotificationSettings()
      .then(applySaved)
      .catch(() => setLoadError(true));
  }, []);

  const dirty =
    saved !== null &&
    (host.trim() !== saved.smtp_host ||
      Number(port) !== saved.smtp_port ||
      username.trim() !== saved.smtp_username ||
      from.trim() !== saved.smtp_from ||
      secretPayload(password, saved.smtp_password_set) !== undefined ||
      secretPayload(botToken, saved.telegram_bot_token_set) !== undefined);

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    if (!saved) return;
    setError(null);

    const portNumber = Number(port);
    if (!Number.isInteger(portNumber) || portNumber < 1 || portNumber > 65535) {
      setError("Port SMTP harus angka 1 sampai 65535.");
      return;
    }
    if (host.trim() && !from.trim()) {
      setError("Alamat pengirim wajib diisi bila host SMTP diisi.");
      return;
    }

    const input: SaveNotificationSettingsInput = {
      smtp_host: host.trim(),
      smtp_port: portNumber,
      smtp_username: username.trim(),
      smtp_from: from.trim(),
    };
    const pw = secretPayload(password, saved.smtp_password_set);
    if (pw !== undefined) input.smtp_password = pw;
    const token = secretPayload(botToken, saved.telegram_bot_token_set);
    if (token !== undefined) input.telegram_bot_token = token;

    setSaving(true);
    try {
      applySaved(await saveNotificationSettings(input));
      toast.success("Pengaturan notifikasi disimpan.");
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Gagal menyimpan pengaturan. Coba lagi.");
    } finally {
      setSaving(false);
    }
  }

  if (loadError) {
    return (
      <Card className="rounded-2xl border-none shadow-sm ring-1 ring-border/60">
        <CardContent>
          <Alert variant="destructive">
            <AlertDescription>
              Gagal memuat pengaturan notifikasi. Muat ulang halaman.
            </AlertDescription>
          </Alert>
        </CardContent>
      </Card>
    );
  }

  if (!saved) {
    return <Skeleton className="h-96 rounded-2xl" />;
  }

  return (
    <Card className="rounded-2xl border-none shadow-sm ring-1 ring-border/60">
      <CardHeader>
        <div className="flex flex-wrap items-center gap-2">
          <CardTitle className="text-base">Notifikasi Pengingat Kedaluwarsa</CardTitle>
          {saved.email_configured ? (
            <Badge className="border-transparent bg-emerald-500/15 text-emerald-700 dark:text-emerald-400">
              Aktif
            </Badge>
          ) : (
            <Badge className="border-transparent bg-amber-500/15 text-amber-700 dark:text-amber-400">
              Belum aktif
            </Badge>
          )}
        </div>
        <CardDescription>
          Email dikirim ke customer yang masa aktifnya tinggal 7 hari atau kurang, plus Telegram
          bila customer mengisi chat id-nya. Pengingat baru berjalan setelah SMTP diisi.
          {saved.updated_at && (
            <>
              {" "}
              Terakhir diubah {formatDateTime(saved.updated_at)}
              {saved.updated_by && <> oleh {saved.updated_by}</>}.
            </>
          )}
        </CardDescription>
      </CardHeader>

      <CardContent className="flex flex-col gap-6">
        <form onSubmit={onSubmit} className="flex flex-col gap-6">
          {error && (
            <Alert variant="destructive">
              <AlertDescription>{error}</AlertDescription>
            </Alert>
          )}

          <section className="flex flex-col gap-4">
            <div className="flex items-center justify-between gap-2">
              <h3 className="text-sm font-medium">SMTP (email)</h3>
              <button
                type="button"
                onClick={() => setShowSecrets((v) => !v)}
                className="flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground"
              >
                {showSecrets ? <EyeOff className="size-3.5" /> : <Eye className="size-3.5" />}
                {showSecrets ? "Sembunyikan" : "Tampilkan"} isian rahasia
              </button>
            </div>

            <div className="grid gap-4 sm:grid-cols-[1fr_8rem]">
              <div className="flex flex-col gap-2">
                <Label htmlFor="smtp-host">Host</Label>
                <Input
                  id="smtp-host"
                  placeholder="smtp.gmail.com"
                  value={host}
                  onChange={(e) => setHost(e.target.value)}
                  autoComplete="off"
                />
              </div>
              <div className="flex flex-col gap-2">
                <Label htmlFor="smtp-port">Port</Label>
                <Input
                  id="smtp-port"
                  type="number"
                  inputMode="numeric"
                  min={1}
                  max={65535}
                  value={port}
                  onChange={(e) => setPort(e.target.value)}
                />
              </div>
            </div>
            <p className="-mt-2 text-xs text-muted-foreground">
              Port 465 memakai TLS langsung; 587 atau 25 memakai STARTTLS.
            </p>

            <div className="grid gap-4 sm:grid-cols-2">
              <div className="flex flex-col gap-2">
                <Label htmlFor="smtp-username">Username</Label>
                <Input
                  id="smtp-username"
                  placeholder="no-reply@domain.com"
                  value={username}
                  onChange={(e) => setUsername(e.target.value)}
                  autoComplete="off"
                />
              </div>
              <SecretField
                id="smtp-password"
                label="Password"
                isSet={saved.smtp_password_set}
                state={password}
                onChange={setPassword}
                show={showSecrets}
              />
            </div>

            <div className="flex flex-col gap-2">
              <Label htmlFor="smtp-from">Alamat pengirim</Label>
              <Input
                id="smtp-from"
                placeholder="Nama Bisnis <no-reply@domain.com>"
                value={from}
                onChange={(e) => setFrom(e.target.value)}
                autoComplete="off"
              />
            </div>
          </section>

          <Separator />

          <section className="flex flex-col gap-4">
            <div>
              <h3 className="text-sm font-medium">Bot Telegram (opsional)</h3>
              <p className="text-xs text-muted-foreground">
                Token dari @BotFather. Kosong berarti pengingat hanya lewat email.
              </p>
            </div>
            <SecretField
              id="telegram-bot-token"
              label="Token bot"
              placeholder="123456789:ABC-DEF..."
              isSet={saved.telegram_bot_token_set}
              state={botToken}
              onChange={setBotToken}
              show={showSecrets}
            />
          </section>

          <div className="flex flex-wrap items-center justify-end gap-2">
            {dirty && (
              <Button
                type="button"
                variant="ghost"
                onClick={() => applySaved(saved)}
                disabled={saving}
              >
                Batalkan perubahan
              </Button>
            )}
            <Button type="submit" disabled={saving || !dirty}>
              {saving ? "Menyimpan..." : "Simpan Pengaturan"}
            </Button>
          </div>
        </form>

        <Separator />

        <TestSection saved={saved} dirty={dirty} />
      </CardContent>
    </Card>
  );
}

function SecretField({
  id,
  label,
  placeholder,
  isSet,
  state,
  onChange,
  show,
}: {
  id: string;
  label: string;
  placeholder?: string;
  isSet: boolean;
  state: SecretState;
  onChange: (s: SecretState) => void;
  show: boolean;
}) {
  if (isSet && state.mode === "keep") {
    return (
      <div className="flex flex-col gap-2">
        <Label htmlFor={id}>{label}</Label>
        <div className="flex h-8 items-center justify-between gap-2 rounded-lg border border-input px-2.5">
          <span className="text-sm text-muted-foreground">•••••••• tersimpan</span>
          <div className="flex gap-1">
            <Button
              id={id}
              type="button"
              variant="ghost"
              size="xs"
              onClick={() => onChange({ mode: "replace", value: "" })}
            >
              Ganti
            </Button>
            <Button
              type="button"
              variant="ghost"
              size="xs"
              className="text-destructive hover:text-destructive"
              onClick={() => onChange({ mode: "clear", value: "" })}
            >
              Hapus
            </Button>
          </div>
        </div>
      </div>
    );
  }

  if (state.mode === "clear") {
    return (
      <div className="flex flex-col gap-2">
        <Label htmlFor={id}>{label}</Label>
        <div className="flex h-8 items-center justify-between gap-2 rounded-lg border border-destructive/40 bg-destructive/5 px-2.5">
          <span className="text-sm text-destructive">Akan dihapus saat disimpan</span>
          <Button id={id} type="button" variant="ghost" size="xs" onClick={() => onChange(KEEP)}>
            Batal
          </Button>
        </div>
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-2">
      <Label htmlFor={id}>{label}</Label>
      <div className="flex gap-2">
        <Input
          id={id}
          type={show ? "text" : "password"}
          placeholder={placeholder}
          autoComplete="new-password"
          value={state.value}
          onChange={(e) =>
            onChange({
              mode: isSet ? "replace" : "keep",
              value: e.target.value,
            })
          }
          autoFocus={isSet}
        />
        {isSet && (
          <Button type="button" variant="ghost" onClick={() => onChange(KEEP)}>
            Batal
          </Button>
        )}
      </div>
    </div>
  );
}

/**
 * Pesan uji memakai pengaturan yang SUDAH tersimpan, bukan isi form --
 * yang diuji harus persis konfigurasi yang nanti dipakai pengingat.
 */
function TestSection({ saved, dirty }: { saved: NotificationSettings; dirty: boolean }) {
  const [email, setEmail] = useState("");
  const [chatID, setChatID] = useState("");
  const [busy, setBusy] = useState<"email" | "telegram" | null>(null);

  async function runTest(channel: "email" | "telegram", to: string) {
    setBusy(channel);
    try {
      await sendTestNotification(channel, to.trim());
      toast.success(
        channel === "email"
          ? `Email uji terkirim ke ${to.trim()}.`
          : "Pesan uji Telegram terkirim.",
      );
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "Gagal mengirim pesan uji.");
    } finally {
      setBusy(null);
    }
  }

  return (
    <section className="flex flex-col gap-4">
      <div>
        <h3 className="text-sm font-medium">Kirim pesan uji</h3>
        <p className="text-xs text-muted-foreground">
          Memakai pengaturan yang sudah tersimpan.
          {dirty && " Simpan perubahan dulu supaya ikut teruji."}
        </p>
      </div>

      <div className="grid gap-4 sm:grid-cols-2">
        <form
          className="flex flex-col gap-2"
          onSubmit={(e) => {
            e.preventDefault();
            runTest("email", email);
          }}
        >
          <Label htmlFor="test-email">Email tujuan</Label>
          <div className="flex gap-2">
            <Input
              id="test-email"
              type="email"
              placeholder="kamu@domain.com"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              disabled={!saved.email_configured}
            />
            <Button
              type="submit"
              variant="outline"
              disabled={!saved.email_configured || !email.trim() || busy !== null}
            >
              <Mail className="size-4" />
              {busy === "email" ? "Mengirim..." : "Uji"}
            </Button>
          </div>
          {!saved.email_configured && (
            <p className="text-xs text-muted-foreground">Isi dan simpan SMTP dulu.</p>
          )}
        </form>

        <form
          className="flex flex-col gap-2"
          onSubmit={(e) => {
            e.preventDefault();
            runTest("telegram", chatID);
          }}
        >
          <Label htmlFor="test-chat-id">Chat id Telegram</Label>
          <div className="flex gap-2">
            <Input
              id="test-chat-id"
              inputMode="numeric"
              placeholder="123456789"
              value={chatID}
              onChange={(e) => setChatID(e.target.value)}
              disabled={!saved.telegram_bot_token_set}
            />
            <Button
              type="submit"
              variant="outline"
              disabled={!saved.telegram_bot_token_set || !chatID.trim() || busy !== null}
            >
              <Send className="size-4" />
              {busy === "telegram" ? "Mengirim..." : "Uji"}
            </Button>
          </div>
          {!saved.telegram_bot_token_set ? (
            <p className="text-xs text-muted-foreground">Simpan token bot dulu.</p>
          ) : (
            <p className="text-xs text-muted-foreground">
              Kirim /start ke bot lebih dulu, Telegram menolak bot memulai obrolan.
            </p>
          )}
        </form>
      </div>
    </section>
  );
}
