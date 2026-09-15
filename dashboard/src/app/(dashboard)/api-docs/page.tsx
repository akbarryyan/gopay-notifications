"use client";

import { useEffect, useState, type ReactNode } from "react";
import Link from "next/link";
import { ArrowRight, Info, TriangleAlert } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { CodeTabs } from "@/components/docs/code-tabs";

/**
 * Dokumentasi integrasi untuk server website customer. Isinya WAJIB sama
 * dengan perilaku backend -- sumbernya:
 *   backend/internal/httpapi/invoices.go, apikey_auth.go (invoice + auth)
 *   backend/internal/httpapi/webhook_send.go (payload + tanda tangan)
 *   backend/internal/store/invoice.go (nominal unik, 15 menit, idempotensi)
 *   backend/internal/store/webhook.go (5 percobaan, jeda 1/2/4/8 menit)
 *   backend/internal/store/qris_image.go (gambar QRIS di response create)
 * Kalau salah satu berubah, halaman ini ikut diubah.
 */

const SECTIONS = [
  { id: "alur", title: "Alur integrasi" },
  { id: "autentikasi", title: "Autentikasi" },
  { id: "buat-invoice", title: "Membuat invoice" },
  { id: "cek-invoice", title: "Mengecek status invoice" },
  { id: "webhook", title: "Webhook" },
  { id: "verifikasi", title: "Verifikasi tanda tangan" },
  { id: "error", title: "Kode error" },
  { id: "checklist", title: "Checklist sebelum live" },
];

export default function ApiDocsPage() {
  // Alamat API = alamat dashboard ini (satu domain, /api/* diteruskan ke
  // backend). Diambil setelah mount karena window tidak ada saat prerender.
  const [base, setBase] = useState("https://whuzpay.com");
  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect
    setBase(window.location.origin);
  }, []);

  return (
    <div className="flex flex-col gap-6">
      <div>
        <h1 className="text-2xl font-semibold">API Docs</h1>
        <p className="text-sm text-muted-foreground">
          Panduan menghubungkan website atau aplikasi kamu: buat invoice, terima kabar pembayaran
          lewat webhook, dan cek status invoice.
        </p>
      </div>

      <div className="grid gap-8 xl:grid-cols-[minmax(0,1fr)_14rem]">
        <div className="flex min-w-0 max-w-3xl flex-col gap-12">
          <Flow />
          <Auth base={base} />
          <CreateInvoice base={base} />
          <GetInvoice base={base} />
          <Webhook />
          <VerifySignature />
          <Errors />
          <Checklist />
        </div>

        <nav className="hidden xl:block" aria-label="Daftar isi">
          <div className="sticky top-6 flex flex-col gap-1 border-l border-border/60 pl-4 text-sm">
            <p className="mb-1 text-xs font-medium tracking-wide text-muted-foreground uppercase">
              Di halaman ini
            </p>
            {SECTIONS.map((s) => (
              <a
                key={s.id}
                href={`#${s.id}`}
                className="py-0.5 text-muted-foreground hover:text-foreground"
              >
                {s.title}
              </a>
            ))}
          </div>
        </nav>
      </div>
    </div>
  );
}

// --- Komponen kecil --------------------------------------------------------

function Section({ id, title, children }: { id: string; title: string; children: ReactNode }) {
  return (
    <section id={id} className="flex scroll-mt-6 flex-col gap-4">
      <h2 className="text-lg font-semibold">
        <a href={`#${id}`} className="hover:underline">
          {title}
        </a>
      </h2>
      {children}
    </section>
  );
}

function P({ children }: { children: ReactNode }) {
  return <p className="text-sm leading-relaxed text-muted-foreground">{children}</p>;
}

function C({ children }: { children: ReactNode }) {
  return (
    <code className="rounded bg-muted px-1.5 py-0.5 font-mono text-[0.8125rem] text-foreground">
      {children}
    </code>
  );
}

function Callout({ tone = "info", children }: { tone?: "info" | "warning"; children: ReactNode }) {
  const Icon = tone === "warning" ? TriangleAlert : Info;
  return (
    <div
      className={
        tone === "warning"
          ? "flex gap-3 rounded-xl border border-amber-500/30 bg-amber-500/5 p-4 text-sm text-amber-900 dark:text-amber-200"
          : "flex gap-3 rounded-xl border border-sky-500/30 bg-sky-500/5 p-4 text-sm text-sky-900 dark:text-sky-200"
      }
    >
      <Icon className="mt-0.5 size-4 shrink-0" />
      <div className="leading-relaxed">{children}</div>
    </div>
  );
}

function FieldTable({ rows }: { rows: [string, string, ReactNode][] }) {
  return (
    <div className="overflow-x-auto rounded-xl border border-border/60">
      <table className="w-full text-sm">
        <thead className="bg-muted/50 text-left text-xs text-muted-foreground">
          <tr>
            <th className="px-3 py-2 font-medium">Field</th>
            <th className="px-3 py-2 font-medium">Tipe</th>
            <th className="px-3 py-2 font-medium">Keterangan</th>
          </tr>
        </thead>
        <tbody>
          {rows.map(([field, type, desc]) => (
            <tr key={field} className="border-t border-border/60 align-top">
              <td className="px-3 py-2 font-mono text-xs whitespace-nowrap">{field}</td>
              <td className="px-3 py-2 text-xs whitespace-nowrap text-muted-foreground">{type}</td>
              <td className="px-3 py-2 text-muted-foreground">{desc}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function Endpoint({ method, path }: { method: "GET" | "POST"; path: string }) {
  return (
    <div className="flex items-center gap-2 rounded-lg border border-border/60 bg-muted/30 px-3 py-2 font-mono text-sm">
      <Badge
        className={
          method === "POST"
            ? "border-transparent bg-teal-500/15 text-teal-700 dark:text-teal-400"
            : "border-transparent bg-sky-500/15 text-sky-700 dark:text-sky-400"
        }
      >
        {method}
      </Badge>
      <span className="break-all">{path}</span>
    </div>
  );
}

// --- Isi ---------------------------------------------------------------------

function Flow() {
  const steps: [string, ReactNode][] = [
    [
      "Buat API key",
      <>
        Di halaman{" "}
        <Link href="/api-keys" className="font-medium text-foreground underline underline-offset-4">
          API Keys
        </Link>
        . Simpan di server kamu, bukan di kode frontend.
      </>,
    ],
    [
      "Daftarkan webhook",
      <>
        Di halaman{" "}
        <Link href="/webhooks" className="font-medium text-foreground underline underline-offset-4">
          Webhooks
        </Link>
        : URL server kamu yang menerima kabar pembayaran, dan simpan secret-nya.
      </>,
    ],
    [
      "Buat invoice saat pelanggan checkout",
      <>
        Server kamu memanggil <C>POST /api/v1/invoices</C> dengan nomor order dan nominal.
      </>,
    ],
    [
      "Tampilkan nominal unik",
      <>
        Minta pelanggan membayar <strong>tepat</strong> sebesar <C>unique_amount</C> ke QRIS GoPay
        Merchant kamu, dalam 15 menit.
      </>,
    ],
    [
      "Terima webhook invoice.paid",
      <>
        Begitu pembayaran masuk ke HP kamu, invoice berubah jadi <C>PAID</C> dan server kamu
        menerima webhook. Tandai order lunas.
      </>,
    ],
  ];
  return (
    <Section id="alur" title="Alur integrasi">
      <ol className="flex flex-col gap-3">
        {steps.map(([title, body], i) => (
          <li key={title} className="flex gap-3">
            <span className="flex size-6 shrink-0 items-center justify-center rounded-full bg-primary text-xs font-semibold text-primary-foreground">
              {i + 1}
            </span>
            <div className="text-sm">
              <p className="font-medium">{title}</p>
              <p className="text-muted-foreground">{body}</p>
            </div>
          </li>
        ))}
      </ol>
      <Callout>
        Semua contoh di halaman ini memakai alamat yang sama dengan dashboard yang sedang kamu buka.
        Pembayaran dibaca oleh HP di halaman{" "}
        <Link href="/devices" className="font-medium underline underline-offset-4">
          Devices
        </Link>
        , jadi invoice hanya bisa ditandai lunas selama HP itu menyala dan berstatus ONLINE.
      </Callout>
    </Section>
  );
}

function Auth({ base }: { base: string }) {
  return (
    <Section id="autentikasi" title="Autentikasi">
      <P>
        Setiap request ke API invoice membawa API key di header <C>Authorization</C>. API key
        diawali <C>sk_</C> dan hanya ditampilkan sekali saat dibuat.
      </P>
      <CodeTabs
        samples={[
          {
            label: "Header",
            code: `Authorization: Bearer sk_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx\nContent-Type: application/json`,
          },
          {
            label: "curl",
            code: `curl ${base}/api/v1/invoices/inv_xxx \\\n  -H "Authorization: Bearer $PAYMENT_BRIDGE_API_KEY"`,
          },
        ]}
      />
      <Callout tone="warning">
        API key setara kunci akun. Panggil API hanya dari <strong>server</strong> kamu, jangan dari
        browser, aplikasi mobile, atau repo publik. Kalau bocor, cabut di halaman API Keys lalu buat
        yang baru.
      </Callout>
    </Section>
  );
}

function CreateInvoice({ base }: { base: string }) {
  return (
    <Section id="buat-invoice" title="Membuat invoice">
      <Endpoint method="POST" path="/api/v1/invoices" />
      <P>
        Panggil saat pelanggan checkout. Backend menambahkan angka acak <strong>1–999</strong> ke
        nominal, supaya setiap pembayaran bisa dikenali dari jumlahnya. Nominal unik itu yang
        dibayar pelanggan.
      </P>

      <h3 className="text-sm font-semibold">Body</h3>
      <FieldTable
        rows={[
          [
            "external_ref",
            "string, wajib",
            "Nomor order di sistem kamu, mis. ORDER-1001. Unik per akun.",
          ],
          ["amount", "integer, wajib", "Nominal dalam rupiah, tanpa desimal. Harus lebih dari 0."],
        ]}
      />

      <CodeTabs
        samples={[
          {
            label: "curl",
            code: `curl -X POST ${base}/api/v1/invoices \\
  -H "Authorization: Bearer $PAYMENT_BRIDGE_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{"external_ref":"ORDER-1001","amount":150000}'`,
          },
          {
            label: "Node.js",
            code: `// Node.js 18+ (fetch bawaan)
const res = await fetch("${base}/api/v1/invoices", {
  method: "POST",
  headers: {
    Authorization: \`Bearer \${process.env.PAYMENT_BRIDGE_API_KEY}\`,
    "Content-Type": "application/json",
  },
  body: JSON.stringify({ external_ref: "ORDER-1001", amount: 150000 }),
});

const invoice = await res.json();
if (!res.ok) {
  throw new Error(\`\${invoice.error}: \${invoice.message}\`);
}

// Simpan invoice.id di order kamu, lalu tampilkan ke pelanggan:
console.log(\`Bayar tepat Rp\${invoice.unique_amount.toLocaleString("id-ID")}\`);`,
          },
          {
            label: "PHP",
            code: `<?php
$ch = curl_init("${base}/api/v1/invoices");
curl_setopt_array($ch, [
    CURLOPT_POST => true,
    CURLOPT_RETURNTRANSFER => true,
    CURLOPT_HTTPHEADER => [
        "Authorization: Bearer " . getenv("PAYMENT_BRIDGE_API_KEY"),
        "Content-Type: application/json",
    ],
    CURLOPT_POSTFIELDS => json_encode([
        "external_ref" => "ORDER-1001",
        "amount" => 150000,
    ]),
]);
$body = curl_exec($ch);
$status = curl_getinfo($ch, CURLINFO_HTTP_CODE);
curl_close($ch);

$invoice = json_decode($body, true);
if ($status >= 400) {
    throw new Exception($invoice["error"] . ": " . $invoice["message"]);
}

// Simpan $invoice["id"] di order kamu, lalu tampilkan ke pelanggan:
echo "Bayar tepat Rp" . number_format($invoice["unique_amount"], 0, ",", ".");`,
          },
        ]}
      />

      <h3 className="text-sm font-semibold">
        Respons <C>201 Created</C>
      </h3>
      <CodeTabs
        samples={[
          {
            label: "JSON",
            code: `{
  "id": "inv_3f9c2a7d1e8b4c6a9d0e5f7a2b1c3d4e",
  "external_ref": "ORDER-1001",
  "requested_amount": 150000,
  "unique_amount": 150347,
  "status": "PENDING",
  "matched_event_id": null,
  "created_at": "2026-09-14T09:30:00Z",
  "expires_at": "2026-09-14T09:45:00Z",
  "paid_at": null,
  "qris_image": "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAA..."
}`,
          },
        ]}
      />
      <InvoiceFields />

      <Callout>
        <strong>Aman diulang.</strong> Memanggil lagi dengan <C>external_ref</C> dan <C>amount</C>{" "}
        yang sama mengembalikan invoice yang sudah ada (<C>200 OK</C>, bukan invoice baru), jadi
        retry karena timeout jaringan tidak membuat tagihan ganda. <C>external_ref</C> yang sama
        dengan <C>amount</C> berbeda ditolak <C>409 external_ref_conflict</C>.
      </Callout>
      <Callout tone="warning">
        <strong>Invoice berlaku 15 menit.</strong> Setelah itu statusnya <C>EXPIRED</C> dan nominal
        unik dilepas. Kalau pelanggan mau membayar lagi, buat invoice baru dengan{" "}
        <C>external_ref</C> baru (mis. <C>ORDER-1001-2</C>), karena <C>external_ref</C> lama akan
        tetap mengembalikan invoice yang sudah kedaluwarsa.
      </Callout>
    </Section>
  );
}

function InvoiceFields() {
  return (
    <FieldTable
      rows={[
        ["id", "string", "ID invoice, diawali inv_. Dipakai untuk mengecek status."],
        ["external_ref", "string", "Nomor order yang kamu kirim."],
        ["requested_amount", "integer", "Nominal yang kamu minta."],
        [
          "unique_amount",
          "integer",
          <>
            Nominal yang <strong>harus dibayar pelanggan</strong>: requested_amount + 1..999.
          </>,
        ],
        [
          "status",
          "string",
          <>
            <C>PENDING</C> menunggu pembayaran, <C>PAID</C> lunas, <C>EXPIRED</C> lewat 15 menit
            tanpa pembayaran.
          </>,
        ],
        ["matched_event_id", "string | null", "ID notifikasi pembayaran yang mencocokkan invoice."],
        ["created_at", "string (RFC 3339)", "Waktu invoice dibuat."],
        ["expires_at", "string (RFC 3339)", "Batas waktu pembayaran."],
        ["paid_at", "string | null", "Waktu invoice ditandai lunas."],
        [
          "qris_image",
          "string",
          <>
            QRIS statis milikmu, data URI siap pakai (<C>&lt;img src=&#123;qris_image&#125;&gt;</C>).
            Selalu terisi -- kalau belum diatur, permintaan ini sudah ditolak <C>409
            qris_not_configured</C> sebelum sampai sini. Cuma ada di respons <C>POST</C>, tidak
            diulang di <C>GET /invoices/&#123;id&#125;</C>.
          </>,
        ],
      ]}
    />
  );
}

function GetInvoice({ base }: { base: string }) {
  return (
    <Section id="cek-invoice" title="Mengecek status invoice">
      <Endpoint method="GET" path="/api/v1/invoices/{id}" />
      <P>
        Mengembalikan invoice dengan bentuk yang sama seperti saat dibuat. Pakai webhook sebagai
        cara utama; endpoint ini untuk halaman &quot;cek status&quot;, rekonsiliasi, atau memastikan
        ulang setelah menerima webhook.
      </P>
      <CodeTabs
        samples={[
          {
            label: "curl",
            code: `curl ${base}/api/v1/invoices/inv_3f9c2a7d1e8b4c6a9d0e5f7a2b1c3d4e \\
  -H "Authorization: Bearer $PAYMENT_BRIDGE_API_KEY"`,
          },
          {
            label: "Node.js",
            code: `const res = await fetch(
  \`${base}/api/v1/invoices/\${encodeURIComponent(invoiceId)}\`,
  { headers: { Authorization: \`Bearer \${process.env.PAYMENT_BRIDGE_API_KEY}\` } },
);
const invoice = await res.json();
if (!res.ok) throw new Error(\`\${invoice.error}: \${invoice.message}\`);

if (invoice.status === "PAID") {
  // tandai order lunas
}`,
          },
          {
            label: "PHP",
            code: `<?php
$ch = curl_init("${base}/api/v1/invoices/" . rawurlencode($invoiceId));
curl_setopt_array($ch, [
    CURLOPT_RETURNTRANSFER => true,
    CURLOPT_HTTPHEADER => ["Authorization: Bearer " . getenv("PAYMENT_BRIDGE_API_KEY")],
]);
$invoice = json_decode(curl_exec($ch), true);
$status = curl_getinfo($ch, CURLINFO_HTTP_CODE);
curl_close($ch);

if ($status === 200 && $invoice["status"] === "PAID") {
    // tandai order lunas
}`,
          },
        ]}
      />
      <P>
        Invoice milik akun lain atau ID yang salah dijawab <C>404 not_found</C>. Jangan melakukan
        polling lebih sering dari beberapa detik sekali.
      </P>
    </Section>
  );
}

function Webhook() {
  return (
    <Section id="webhook" title="Webhook">
      <P>
        Backend mengirim <C>POST</C> ke URL yang kamu daftarkan di halaman Webhooks setiap kali
        status invoice berubah.
      </P>

      <FieldTable
        rows={[
          ["invoice.paid", "event", "Pembayaran dengan nominal unik yang cocok sudah masuk."],
          [
            "invoice.expired",
            "event",
            "15 menit lewat tanpa pembayaran. Batalkan atau lepas stok order-nya.",
          ],
          [
            "test",
            "event",
            <>
              Dikirim tombol <em>Test</em> di halaman Webhooks. Isi <C>invoice</C> <C>null</C>.
            </>,
          ],
        ]}
      />

      <h3 className="text-sm font-semibold">Request yang diterima server kamu</h3>
      <CodeTabs
        samples={[
          {
            label: "HTTP",
            code: `POST /webhooks/payment-bridge HTTP/1.1
Content-Type: application/json
X-Webhook-Event: invoice.paid
X-Webhook-Signature: 5d41402abc4b2a76b9719d911017c592a1b2c3d4e5f60718293a4b5c6d7e8f90

{
  "event": "invoice.paid",
  "invoice": {
    "id": "inv_3f9c2a7d1e8b4c6a9d0e5f7a2b1c3d4e",
    "external_ref": "ORDER-1001",
    "requested_amount": 150000,
    "unique_amount": 150347,
    "status": "PAID",
    "matched_event_id": "evt_…",
    "created_at": "2026-09-14T09:30:00Z",
    "expires_at": "2026-09-14T09:45:00Z",
    "paid_at": "2026-09-14T09:36:12Z"
  },
  "sent_at": "2026-09-14T09:36:13Z"
}`,
          },
        ]}
      />
      <P>
        Objek <C>invoice</C> sama persis dengan respons <C>GET /api/v1/invoices/{"{id}"}</C>.
      </P>

      <h3 className="text-sm font-semibold">Membalas dan percobaan ulang</h3>
      <ul className="list-disc space-y-1.5 pl-5 text-sm text-muted-foreground">
        <li>
          Balas dengan status <C>2xx</C> dalam <strong>10 detik</strong>. Selain itu, termasuk
          timeout, dianggap gagal.
        </li>
        <li>
          Pengiriman yang gagal dicoba ulang sampai <strong>5 kali</strong> total, dengan jeda 1, 2,
          4, lalu 8 menit. Riwayatnya bisa dilihat di halaman Webhooks.
        </li>
        <li>
          Proses yang berat (kirim email, update stok) kerjakan setelah membalas, atau lewat antrean
          di sistem kamu.
        </li>
      </ul>

      <Callout tone="warning">
        <p>
          <strong>Tangani webhook secara idempotent.</strong> Event yang sama bisa sampai lebih dari
          sekali, misalnya kalau server kamu sudah memproses tapi balasannya timeout. Periksa apakah
          order sudah lunas sebelum memprosesnya lagi.
        </p>
        <p className="mt-2">
          <strong>
            <C>invoice.paid</C> bisa datang setelah <C>invoice.expired</C>.
          </strong>{" "}
          Pembayaran yang masuk setelah 15 menit bisa dicocokkan manual dari halaman{" "}
          <Link href="/exceptions" className="font-medium underline underline-offset-4">
            Exceptions
          </Link>
          , lalu invoice-nya berubah jadi <C>PAID</C>. Jangan menolak <C>invoice.paid</C> hanya
          karena order sempat dibatalkan.
        </p>
      </Callout>
    </Section>
  );
}

function VerifySignature() {
  return (
    <Section id="verifikasi" title="Verifikasi tanda tangan">
      <P>
        Siapa pun bisa mengirim request ke URL webhook kamu, jadi <strong>selalu</strong> verifikasi
        tanda tangan sebelum mempercayai isinya. <C>X-Webhook-Signature</C> adalah HMAC-SHA256 dari{" "}
        <strong>body mentah</strong> request, dengan kunci secret webhook (diawali <C>whsec_</C>,
        dipakai apa adanya), dalam format hex huruf kecil.
      </P>
      <CodeTabs
        samples={[
          {
            label: "Node.js (Express)",
            code: `import express from "express";
import crypto from "node:crypto";

const app = express();

// express.raw: body HARUS mentah (Buffer). Kalau sudah di-parse jadi JSON
// lalu di-stringify ulang, byte-nya bisa berbeda dan tanda tangan tidak cocok.
app.post(
  "/webhooks/payment-bridge",
  express.raw({ type: "application/json" }),
  (req, res) => {
    const signature = req.get("X-Webhook-Signature") ?? "";
    const expected = crypto
      .createHmac("sha256", process.env.PAYMENT_BRIDGE_WEBHOOK_SECRET)
      .update(req.body)
      .digest("hex");

    const valid =
      signature.length === expected.length &&
      crypto.timingSafeEqual(Buffer.from(signature), Buffer.from(expected));
    if (!valid) return res.status(401).send("invalid signature");

    const payload = JSON.parse(req.body.toString("utf8"));

    if (payload.event === "invoice.paid") {
      // Idempotent: lewati kalau order payload.invoice.external_ref sudah lunas
      markOrderPaid(payload.invoice.external_ref, payload.invoice.id);
    } else if (payload.event === "invoice.expired") {
      markOrderExpired(payload.invoice.external_ref);
    }

    res.sendStatus(200);
  },
);`,
          },
          {
            label: "PHP",
            code: `<?php
// Baca body mentah SEBELUM framework mem-parse-nya.
$raw = file_get_contents("php://input");
$signature = $_SERVER["HTTP_X_WEBHOOK_SIGNATURE"] ?? "";
$expected = hash_hmac("sha256", $raw, getenv("PAYMENT_BRIDGE_WEBHOOK_SECRET"));

// hash_equals: perbandingan waktu-konstan, jangan pakai ===
if (!hash_equals($expected, $signature)) {
    http_response_code(401);
    exit("invalid signature");
}

$payload = json_decode($raw, true);

switch ($payload["event"]) {
    case "invoice.paid":
        // Idempotent: lewati kalau order sudah lunas
        markOrderPaid($payload["invoice"]["external_ref"], $payload["invoice"]["id"]);
        break;
    case "invoice.expired":
        markOrderExpired($payload["invoice"]["external_ref"]);
        break;
}

http_response_code(200);`,
          },
          {
            label: "curl (uji endpoint kamu)",
            code: `# Kirim webhook palsu bertanda tangan benar ke server kamu sendiri,
# untuk menguji kode verifikasi sebelum live.
SECRET="whsec_xxx"
BODY='{"event":"test","invoice":null,"sent_at":"2026-09-14T09:30:00Z"}'
SIG=$(printf '%s' "$BODY" | openssl dgst -sha256 -hmac "$SECRET" -hex | sed 's/^.* //')

curl -i -X POST https://toko-kamu.com/webhooks/payment-bridge \\
  -H "Content-Type: application/json" \\
  -H "X-Webhook-Event: test" \\
  -H "X-Webhook-Signature: $SIG" \\
  -d "$BODY"`,
          },
        ]}
      />
      <Callout>
        Laravel: ambil body mentah dengan <C>$request-&gt;getContent()</C>. Next.js App Router:
        pakai <C>await request.text()</C>, bukan <C>request.json()</C>.
      </Callout>
    </Section>
  );
}

function Errors() {
  const rows: [string, string, string][] = [
    ["400", "invalid_payload", "Body bukan JSON, external_ref kosong, atau amount ≤ 0."],
    ["401", "unauthenticated", "API key tidak ada, salah, atau sudah dicabut."],
    [
      "402",
      "account_expired / account_suspended / account_revoked",
      "Akun tidak aktif. Cek halaman License.",
    ],
    ["404", "not_found", "Invoice tidak ditemukan di akun ini."],
    ["409", "external_ref_conflict", "external_ref sudah dipakai dengan amount berbeda."],
    [
      "409",
      "qris_not_configured",
      "Belum upload gambar QRIS di Settings. Invoice tidak dibuat -- upload dulu, lalu coba lagi.",
    ],
    [
      "503",
      "allocation_full",
      "Semua nominal unik untuk amount ini sedang dipakai invoice lain yang masih PENDING. Coba lagi sebentar lagi.",
    ],
    ["500", "internal", "Kesalahan di sisi kami. Aman dicoba ulang dengan external_ref yang sama."],
  ];
  return (
    <Section id="error" title="Kode error">
      <P>Semua error memakai bentuk yang sama:</P>
      <CodeTabs
        samples={[
          {
            label: "JSON",
            code: `{
  "success": false,
  "error": "external_ref_conflict",
  "message": "external_ref sudah dipakai invoice lain dengan amount berbeda"
}`,
          },
        ]}
      />
      <P>
        Buat logika berdasarkan <C>error</C>, bukan <C>message</C>: teks <C>message</C> bisa
        berubah.
      </P>
      <div className="overflow-x-auto rounded-xl border border-border/60">
        <table className="w-full text-sm">
          <thead className="bg-muted/50 text-left text-xs text-muted-foreground">
            <tr>
              <th className="px-3 py-2 font-medium">HTTP</th>
              <th className="px-3 py-2 font-medium">error</th>
              <th className="px-3 py-2 font-medium">Arti</th>
            </tr>
          </thead>
          <tbody>
            {rows.map(([code, err, desc]) => (
              <tr key={err} className="border-t border-border/60 align-top">
                <td className="px-3 py-2 font-mono text-xs">{code}</td>
                <td className="px-3 py-2 font-mono text-xs">{err}</td>
                <td className="px-3 py-2 text-muted-foreground">{desc}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </Section>
  );
}

function Checklist() {
  const items: [string, string][] = [
    ["API key disimpan di server", "Di environment variable, bukan di kode frontend atau repo."],
    [
      "Webhook terdaftar dan lolos uji",
      "Tombol Test di halaman Webhooks dijawab 2xx oleh server kamu.",
    ],
    ["Tanda tangan diverifikasi", "Request dengan X-Webhook-Signature salah ditolak 401."],
    [
      "Webhook idempotent",
      "Mengirim invoice.paid yang sama dua kali tidak memproses order dua kali.",
    ],
    [
      "Pelanggan melihat nominal unik",
      "Halaman pembayaran menampilkan unique_amount dan batas waktu 15 menit.",
    ],
    [
      "HP bridge ONLINE",
      "Status HP di halaman Devices ONLINE, dan notifikasi Telegram/email aktif.",
    ],
    [
      "Uji dengan pembayaran sungguhan",
      "Buat invoice nominal kecil, bayar tepat unique_amount, pastikan order jadi lunas.",
    ],
  ];
  return (
    <Section id="checklist" title="Checklist sebelum live">
      <ul className="flex flex-col gap-2">
        {items.map(([title, desc]) => (
          <li key={title} className="flex gap-3 rounded-xl border border-border/60 p-3 text-sm">
            <ArrowRight className="mt-0.5 size-4 shrink-0 text-muted-foreground" />
            <div>
              <p className="font-medium">{title}</p>
              <p className="text-muted-foreground">{desc}</p>
            </div>
          </li>
        ))}
      </ul>
    </Section>
  );
}
