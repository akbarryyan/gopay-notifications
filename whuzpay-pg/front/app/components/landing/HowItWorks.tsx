import Section from "./Section";
import CodeBlock from "./CodeBlock";

const steps = [
  {
    title: "Buat API key",
    body: "Ambil kunci sandbox dari dashboard. Tidak perlu kartu kredit, tidak perlu menunggu persetujuan.",
  },
  {
    title: "Panggil satu endpoint",
    body: "Kirim nominal dan metode pembayaran, terima QRIS beserta tautan checkout siap pakai.",
  },
  {
    title: "Terima webhook",
    body: "Saat pembayaran lunas, WhuzPay mengirim notifikasi bertanda tangan ke server Anda dan status berubah menjadi paid.",
  },
];

const snippet = `# Membuat pembayaran QRIS
curl -X POST https://api.whuzpay.co.id/api/v1/payments \\
  -H "X-API-Key: $WHUZPAY_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{
    "amount": 150000,
    "payment_method": "qris",
    "description": "Pesanan #1042",
    "expires_in_minutes": 30
  }'`;

const badges = [
  "Webhook bertanda tangan HMAC",
  "Sandbox tanpa kartu kredit",
  "Status ternormalisasi lintas provider",
];

export default function HowItWorks() {
  return (
    <Section
      id="integrasi"
      background="navy-dark"
      eyebrow="Untuk developer"
      title="Tiga langkah dari nol sampai transaksi pertama"
      subtitle="Satu endpoint untuk membuat pembayaran, satu webhook untuk mengetahui hasilnya. Sisanya kami yang urus."
      containerClassName="max-w-6xl"
    >
      <div className="grid gap-12 lg:grid-cols-2 lg:items-center lg:gap-16">
        <ol className="flex flex-col gap-8">
          {steps.map((step, i) => (
            <li key={step.title} className="flex gap-5">
              <span className="flex h-10 w-10 shrink-0 items-center justify-center rounded-full border border-brand-yellow/40 bg-brand-yellow/10 text-sm font-bold text-brand-yellow">
                {i + 1}
              </span>
              <div>
                <h3 className="text-lg font-bold leading-snug text-white">
                  {step.title}
                </h3>
                <p className="mt-2 text-base leading-relaxed text-white/70">
                  {step.body}
                </p>
              </div>
            </li>
          ))}
        </ol>

        <div>
          <CodeBlock title="POST /api/v1/payments" code={snippet} />
          <div className="mt-5 flex flex-wrap gap-2">
            {badges.map((badge) => (
              <span
                key={badge}
                className="rounded-full border border-white/15 bg-white/5 px-3 py-1.5 text-xs font-medium text-white/70"
              >
                {badge}
              </span>
            ))}
          </div>
        </div>
      </div>
    </Section>
  );
}
