"use client";

import { useState, type ReactNode } from "react";
import toast from "react-hot-toast";
import { CopyIcon } from "lucide-react";
import { MERCHANT_API_URL } from "@/lib/merchant-auth";
import { Card, PageHeader, StatusBadge } from "@/app/components/admin/ui";

const NAV = [
  { id: "auth", label: "Authentication" },
  { id: "create-payment", label: "Create Payment" },
  { id: "get-payment", label: "Get Payment" },
  { id: "get-status", label: "Get Payment Status" },
  { id: "lifecycle", label: "Status Lifecycle" },
  { id: "webhooks", label: "Webhooks" },
  { id: "errors", label: "Error Format" },
];

function scrollToSection(id: string) {
  const el = document.getElementById(id);
  if (!el) return;
  el.scrollIntoView({ behavior: "smooth", block: "start" });
  window.history.replaceState(null, "", `#${id}`);
}

function CodeBlock({ code, language = "bash" }: { code: string; language?: string }) {
  const [copied, setCopied] = useState(false);

  async function handleCopy() {
    try {
      await navigator.clipboard.writeText(code);
      setCopied(true);
      window.setTimeout(() => setCopied(false), 1500);
    } catch {
      toast.error("Failed to copy.");
    }
  }

  return (
    <div className="relative overflow-hidden rounded-xl bg-[#0b1526]">
      <div className="flex items-center justify-between border-b border-white/10 px-4 py-2">
        <span className="text-[11px] font-medium tracking-wide text-white/40 uppercase">
          {language}
        </span>
        <button
          type="button"
          onClick={() => void handleCopy()}
          className="flex items-center gap-1.5 rounded-md px-2 py-1 text-[11.5px] text-white/60 hover:bg-white/10 hover:text-white"
        >
          <CopyIcon className="size-3.5" />
          {copied ? "Copied" : "Copy"}
        </button>
      </div>
      <pre className="overflow-x-auto px-4 py-3.5 text-[12.5px] leading-relaxed text-[#d6e2f0]">
        <code>{code}</code>
      </pre>
    </div>
  );
}

function MethodBadge({ method }: { method: "GET" | "POST" }) {
  const style =
    method === "GET"
      ? "bg-[#e6f4ff] text-[#1a73c9]"
      : "bg-[#e8f8ee] text-[#2f9e5a]";
  return (
    <span
      className={`inline-flex shrink-0 items-center rounded-md px-2 py-1 text-[11px] font-bold tracking-wide ${style}`}
    >
      {method}
    </span>
  );
}

function Endpoint({ method, path }: { method: "GET" | "POST"; path: string }) {
  return (
    <div className="flex items-center gap-2.5 rounded-lg border border-[#e8eef4] bg-[#f8fafc] px-3.5 py-2.5">
      <MethodBadge method={method} />
      <code className="text-[13px] text-[#1f2a37]">{path}</code>
    </div>
  );
}

function Section({
  id,
  title,
  description,
  children,
}: {
  id: string;
  title: string;
  description?: string;
  children: ReactNode;
}) {
  return (
    <div id={id} className="scroll-mt-20">
      <Card className="p-5 sm:p-6">
        <h2 className="text-[16px] font-semibold tracking-tight text-[#1f2a37]">
          {title}
        </h2>
        {description && (
          <p className="mt-1.5 text-[13px] leading-relaxed text-[#6b7c93]">
            {description}
          </p>
        )}
        <div className="mt-4 space-y-4">{children}</div>
      </Card>
    </div>
  );
}

function ParamTable({
  rows,
}: {
  rows: { field: string; type: string; required: boolean; description: string }[];
}) {
  return (
    <div className="overflow-x-auto rounded-lg border border-[#eef2f6]">
      <table className="w-full min-w-125 text-left text-[12.5px]">
        <thead>
          <tr className="border-b border-[#eef2f6] bg-[#f8fafc] text-[11px] font-semibold tracking-wide text-[#8a97a8] uppercase">
            <th className="px-3.5 py-2.5">Field</th>
            <th className="px-3.5 py-2.5">Type</th>
            <th className="px-3.5 py-2.5">Required</th>
            <th className="px-3.5 py-2.5">Description</th>
          </tr>
        </thead>
        <tbody>
          {rows.map((r) => (
            <tr key={r.field} className="border-b border-[#f3f6f9] last:border-0">
              <td className="px-3.5 py-2.5 font-mono text-[12px] text-[#1f2a37]">
                {r.field}
              </td>
              <td className="px-3.5 py-2.5 text-[#6b7c93]">{r.type}</td>
              <td className="px-3.5 py-2.5">
                {r.required ? (
                  <span className="text-[11.5px] font-semibold text-[#e85d3b]">
                    required
                  </span>
                ) : (
                  <span className="text-[11.5px] text-[#8a97a8]">optional</span>
                )}
              </td>
              <td className="px-3.5 py-2.5 text-[#3d4b5c]">{r.description}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function StatusCodeTable({
  rows,
}: {
  rows: { code: string; label: string; description: string }[];
}) {
  return (
    <div className="overflow-x-auto rounded-lg border border-[#eef2f6]">
      <table className="w-full min-w-100 text-left text-[12.5px]">
        <thead>
          <tr className="border-b border-[#eef2f6] bg-[#f8fafc] text-[11px] font-semibold tracking-wide text-[#8a97a8] uppercase">
            <th className="px-3.5 py-2.5">Code</th>
            <th className="px-3.5 py-2.5">Description</th>
          </tr>
        </thead>
        <tbody>
          {rows.map((r) => (
            <tr key={r.code} className="border-b border-[#f3f6f9] last:border-0">
              <td className="px-3.5 py-2.5 whitespace-nowrap">
                <span className="font-mono text-[12px] font-semibold text-[#1f2a37]">
                  {r.code}
                </span>{" "}
                <span className="text-[#8a97a8]">{r.label}</span>
              </td>
              <td className="px-3.5 py-2.5 text-[#3d4b5c]">{r.description}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

export default function ApiDocsPage() {
  return (
    <div className="w-full space-y-5">
      <PageHeader
        title="API Documentation"
        description="Integrate QRIS payments directly from your own backend. This page covers what's actually implemented today — status: under active development, endpoints may change."
      />

      {/* Quick nav */}
      <Card className="flex flex-wrap gap-1.5 p-3">
        {NAV.map((item) => (
          <a
            key={item.id}
            href={`#${item.id}`}
            onClick={(e) => {
              e.preventDefault();
              scrollToSection(item.id);
            }}
            className="rounded-full px-3 py-1.5 text-[12.5px] font-medium text-[#3d4b5c] transition-colors duration-150 hover:bg-[#f4f7fb]"
          >
            {item.label}
          </a>
        ))}
      </Card>

      <Section
        id="auth"
        title="Authentication"
        description="Every request to the endpoints below must include your API key. Create or rotate keys from Dashboard → API keys — the secret is shown once, at creation."
      >
        <p className="text-[13px] text-[#3d4b5c]">
          Send it as either header:
        </p>
        <CodeBlock
          language="http"
          code={`X-API-Key: YOUR_API_KEY\n\n# or\nAuthorization: Bearer YOUR_API_KEY`}
        />
        <p className="text-[13px] text-[#3d4b5c]">
          Keys are scoped per environment. A <strong>sandbox</strong> key
          never reaches the real payment provider — payments are created by
          an in-process mock, useful for integration testing without moving
          real money. A <strong>production</strong> key creates real QRIS
          payments. Base URL for this environment:
        </p>
        <CodeBlock language="text" code={MERCHANT_API_URL} />
      </Section>

      <Section
        id="create-payment"
        title="Create Payment"
        description="Creates a QRIS payment and returns a QR code for your customer to scan. merchant_id and environment are forced from your API key — you don't send them."
      >
        <Endpoint method="POST" path="/api/v1/payments" />

        <ParamTable
          rows={[
            { field: "amount", type: "integer", required: true, description: "Amount in IDR (whole rupiah, no decimals). Provider may add a small unique suffix for payment matching." },
            { field: "description", type: "string", required: true, description: "Shown to the customer on the checkout page." },
            { field: "currency", type: "string", required: false, description: 'Defaults to "IDR" — the only supported currency right now.' },
            { field: "payment_method", type: "string", required: false, description: 'Defaults to "qris" — the only supported method right now.' },
            { field: "customer_name", type: "string", required: false, description: "Optional, shown on the checkout page." },
            { field: "customer_email", type: "string", required: false, description: "Optional." },
            { field: "callback_url", type: "string", required: false, description: "Per-payment webhook override. Falls back to the webhook URL set in Dashboard → Settings if omitted." },
            { field: "expires_in_minutes", type: "integer", required: false, description: "Defaults to 30. Max 1440 (24h)." },
            { field: "use_custom_merchant_name", type: "boolean", required: false, description: "Requests a custom merchant name on the QR, where the provider supports it. Currently only takes effect on Cashi if that feature is enabled on your Cashi account — otherwise silently ignored." },
          ]}
        />

        <CodeBlock
          code={`curl -X POST ${MERCHANT_API_URL}/api/v1/payments \\
  -H "X-API-Key: YOUR_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{
  "amount": 50000,
  "description": "Order #1234",
  "customer_name": "Budi Santoso",
  "expires_in_minutes": 30
}'`}
        />

        <p className="text-[13px] font-medium text-[#3d4b5c]">Example response (201)</p>
        <CodeBlock
          language="json"
          code={`{
  "id": "3a1b2c3d-...",
  "reference": "PAY-1784900000-abc12345",
  "merchant_id": "11111111-...",
  "amount": 50042,
  "currency": "IDR",
  "payment_method": "qris",
  "provider_name": "cashi",
  "provider_reference": "INV-9921",
  "status": "pending",
  "description": "Order #1234",
  "customer_name": "Budi Santoso",
  "customer_email": null,
  "qris_data": "data:image/png;base64,...",
  "checkout_url": "https://your-frontend.com/pay/PAY-1784900000-abc12345",
  "environment": "production",
  "expires_at": "2026-07-25T13:20:09Z",
  "paid_at": null,
  "created_at": "2026-07-25T13:10:08Z",
  "updated_at": "2026-07-25T13:10:08Z"
}`}
        />
        <p className="text-[12.5px] text-[#8a97a8]">
          Note: <code className="font-mono">amount</code> in the response is
          the final amount the customer must pay — it may differ slightly
          from what you sent (provider-added unique suffix for payment
          matching). Always display the response amount, not your request
          amount.
        </p>
      </Section>

      <Section
        id="get-payment"
        title="Get Payment"
        description="Fetch full payment details by id. Returns the same shape as Create Payment. Returns 404 if the payment doesn't belong to your merchant account."
      >
        <Endpoint method="GET" path="/api/v1/payments/:id" />
        <CodeBlock
          code={`curl ${MERCHANT_API_URL}/api/v1/payments/3a1b2c3d-... \\
  -H "X-API-Key: YOUR_API_KEY"`}
        />
      </Section>

      <Section
        id="get-status"
        title="Get Payment Status"
        description="Lighter-weight endpoint for polling — use this instead of Get Payment if you only need the status."
      >
        <Endpoint method="GET" path="/api/v1/payments/:id/status" />
        <CodeBlock
          code={`curl ${MERCHANT_API_URL}/api/v1/payments/3a1b2c3d-.../status \\
  -H "X-API-Key: YOUR_API_KEY"`}
        />
        <p className="text-[13px] font-medium text-[#3d4b5c]">Example response (200)</p>
        <CodeBlock
          language="json"
          code={`{
  "id": "3a1b2c3d-...",
  "reference": "PAY-1784900000-abc12345",
  "status": "paid",
  "amount": 50042,
  "currency": "IDR",
  "paid_at": "2026-07-25T13:15:40Z",
  "expires_at": "2026-07-25T13:20:09Z"
}`}
        />
        <p className="text-[12.5px] text-[#8a97a8]">
          Polling is fine for UX (e.g. a checkout page), but don&apos;t rely
          on it as your only source of truth — use webhooks below for
          authoritative status changes, since a customer can close their
          browser before a poll ever confirms payment.
        </p>
      </Section>

      <Section
        id="lifecycle"
        title="Status Lifecycle"
        description="A payment moves through at most one of these transitions — once it reaches a terminal status, it never changes again."
      >
        <div className="grid gap-2.5 sm:grid-cols-2">
          {[
            { s: "pending", d: "Created, waiting for the customer to pay before expires_at." },
            { s: "paid", d: "Confirmed paid by the provider. Terminal." },
            { s: "expired", d: "Not paid before expires_at. Terminal." },
            { s: "failed", d: "Provider or internal error prevented completion. Terminal." },
            { s: "cancelled", d: "Cancelled before payment. Terminal." },
          ].map((row) => (
            <div
              key={row.s}
              className="flex items-start gap-3 rounded-lg border border-[#eef2f6] px-3.5 py-3"
            >
              <StatusBadge status={row.s} />
              <p className="text-[12.5px] text-[#6b7c93]">{row.d}</p>
            </div>
          ))}
        </div>
      </Section>

      <Section
        id="webhooks"
        title="Webhooks"
        description="Set your webhook URL in Dashboard → Settings (or override per-payment with callback_url). We POST here whenever a payment's status changes to a terminal state."
      >
        <p className="text-[13px] font-medium text-[#3d4b5c]">Request headers</p>
        <CodeBlock
          language="http"
          code={`Content-Type: application/json\nUser-Agent: pg-aggregator-callback/1.0\nX-PG-Event: payment.paid\nX-PG-Signature: <hmac-sha256-hex>`}
        />

        <p className="text-[13px] font-medium text-[#3d4b5c]">Example payload</p>
        <CodeBlock
          language="json"
          code={`{
  "event": "payment.paid",
  "payment_id": "3a1b2c3d-...",
  "reference": "PAY-1784900000-abc12345",
  "status": "paid",
  "amount": 50042,
  "currency": "IDR",
  "merchant_id": "11111111-...",
  "provider": "cashi",
  "provider_reference": "INV-9921",
  "paid_at": "2026-07-25T13:15:40Z",
  "occurred_at": "2026-07-25T13:15:41Z"
}`}
        />

        <p className="text-[13px] text-[#3d4b5c]">
          <code className="font-mono">event</code> is one of{" "}
          <code className="font-mono">payment.paid</code>,{" "}
          <code className="font-mono">payment.expired</code>,{" "}
          <code className="font-mono">payment.failed</code>,{" "}
          <code className="font-mono">payment.cancelled</code>.
        </p>

        <p className="text-[13px] text-[#3d4b5c]">
          Respond with any <code className="font-mono">2xx</code> status to
          acknowledge. Failed deliveries (non-2xx, timeout, connection
          refused) retry automatically — 5 attempts total, 5 minutes apart —
          visible any time under{" "}
          <code className="font-mono">Dashboard → Webhook Logs</code>.
        </p>

        <div className="rounded-lg border border-[#ffe1b3] bg-[#fff8ec] px-3.5 py-3 text-[12.5px] text-[#8a5a1f]">
          <strong>Verify the signature before trusting the payload.</strong>{" "}
          Anyone who finds your webhook URL can otherwise POST a fake{" "}
          <code className="font-mono">payment.paid</code> event and trick
          your system into shipping an order that was never paid for.
        </div>

        <p className="text-[13px] font-medium text-[#3d4b5c]">
          How to verify
        </p>
        <ol className="list-decimal space-y-1 pl-5 text-[13px] text-[#3d4b5c]">
          <li>Get your signing secret from Dashboard → Settings.</li>
          <li>Take the raw request body exactly as received (don&apos;t re-serialize it — key order/whitespace must match what we signed).</li>
          <li>Compute HMAC-SHA256 of that raw body using your secret.</li>
          <li>Compare the hex digest to the <code className="font-mono">X-PG-Signature</code> header using a constant-time comparison.</li>
          <li>If it doesn&apos;t match, reject with 401 and don&apos;t touch your database.</li>
        </ol>

        <p className="text-[13px] font-medium text-[#3d4b5c]">Example (Node.js)</p>
        <CodeBlock
          language="javascript"
          code={`const crypto = require('crypto');

app.post('/webhooks/payments', (req, res) => {
  const signature = req.headers['x-pg-signature'];
  if (!signature) return res.status(401).send('Missing signature');

  // req.rawBody must be the exact bytes received — see your framework's
  // raw-body middleware (e.g. express.raw()) if you're not already keeping it.
  const expected = crypto
    .createHmac('sha256', process.env.PG_WEBHOOK_SECRET)
    .update(req.rawBody)
    .digest('hex');

  const ok =
    signature.length === expected.length &&
    crypto.timingSafeEqual(Buffer.from(signature), Buffer.from(expected));

  if (!ok) return res.status(401).send('Invalid signature');

  const { event, status } = req.body;
  if (event === 'payment.paid' && status === 'paid') {
    // mark the order paid in your own database
  }

  res.status(200).send('OK');
});`}
        />

        <p className="text-[12.5px] text-[#8a97a8]">
          Regenerating your secret (also in Settings) invalidates the old one
          immediately — update your verification code before you rotate it,
          not after.
        </p>
      </Section>

      <Section
        id="errors"
        title="Error Format"
        description="Non-2xx responses share this shape."
      >
        <CodeBlock
          language="json"
          code={`{
  "error": "Bad Request",
  "message": "amount must be greater than 0"
}`}
        />
        <StatusCodeTable
          rows={[
            { code: "400", label: "Bad Request", description: "Invalid request body or validation failure (bad amount, missing description, etc)." },
            { code: "401", label: "Unauthorized", description: "Missing, invalid, or revoked API key." },
            { code: "403", label: "Forbidden", description: "Merchant account is inactive." },
            { code: "404", label: "Not Found", description: "Payment doesn't exist, or belongs to a different merchant." },
            { code: "429", label: "Too Many Requests", description: "Rate limit exceeded — back off and retry." },
            { code: "502", label: "Bad Gateway", description: "The payment provider (e.g. Cashi) returned an error." },
            { code: "503", label: "Service Unavailable", description: "No healthy provider available for this payment method right now." },
          ]}
        />
      </Section>
    </div>
  );
}
