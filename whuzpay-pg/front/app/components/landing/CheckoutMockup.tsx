/**
 * Stylised public checkout page (`/pay/{reference}`) as the payer sees it.
 *
 * The QR is a deterministic pattern drawn in SVG, not a real QRIS payload —
 * it exists to show the page layout, so it must never look scannable enough
 * for someone to try.
 */
function FakeQr() {
  const cells = Array.from({ length: 144 }, (_, i) => {
    // Deterministic pseudo-noise: no Math.random, so server and client
    // markup match and the pattern stays stable between renders.
    const x = i % 12;
    const y = Math.floor(i / 12);
    return (x * 7 + y * 13 + x * y) % 3 === 0;
  });

  return (
    <svg viewBox="0 0 12 12" className="h-full w-full" role="presentation">
      <rect width="12" height="12" fill="white" />
      {cells.map((filled, i) =>
        filled ? (
          <rect
            key={i}
            x={i % 12}
            y={Math.floor(i / 12)}
            width="1"
            height="1"
            fill="#0e2a52"
          />
        ) : null,
      )}
      {/* Finder patterns, the three corner squares every QR has */}
      {[
        [0, 0],
        [9, 0],
        [0, 9],
      ].map(([x, y]) => (
        <g key={`${x}-${y}`}>
          <rect x={x} y={y} width="3" height="3" fill="white" />
          <rect x={x} y={y} width="3" height="3" fill="none" stroke="#0e2a52" strokeWidth="0.6" />
          <rect x={x + 1} y={y + 1} width="1" height="1" fill="#0e2a52" />
        </g>
      ))}
    </svg>
  );
}

export default function CheckoutMockup() {
  return (
    <div className="mx-auto w-full max-w-xs overflow-hidden rounded-3xl border-4 border-slate-800 bg-white shadow-2xl">
      <div className="bg-brand-navy px-5 py-4 text-center">
        <p className="text-xs font-bold italic text-white">whuzpay</p>
        <p className="mt-1 text-[10px] text-white/60">Toko Serba Ada</p>
      </div>

      <div className="px-5 py-5 text-center">
        <p className="text-[10px] uppercase tracking-wider text-slate-400">
          Total Pembayaran
        </p>
        <p className="mt-1 text-2xl font-extrabold tracking-tight text-brand-navy">
          Rp 150.047
        </p>

        <div className="mx-auto mt-4 h-36 w-36 rounded-lg border border-slate-200 p-2">
          <FakeQr />
        </div>

        <p className="mt-3 text-[11px] text-slate-500">
          Pindai dengan aplikasi apa pun yang mendukung QRIS
        </p>

        <div className="mt-4 flex items-center justify-center gap-2 rounded-lg bg-amber-50 px-3 py-2">
          <span className="h-1.5 w-1.5 animate-pulse rounded-full bg-amber-500" />
          <span className="text-[11px] font-semibold text-amber-700">
            Menunggu pembayaran · 09:42
          </span>
        </div>

        <p className="mt-3 text-[10px] text-slate-400">
          Status diperbarui otomatis
        </p>
      </div>
    </div>
  );
}
