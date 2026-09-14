const bars = [30, 55, 40, 70, 50, 65, 45, 80];

const methods = ["VA BCA", "QRIS", "GoPay", "OVO"];

/**
 * Stylised merchant dashboard, drawn with plain markup rather than a
 * screenshot so it stays crisp at every size and ships no image request.
 *
 * The phone is absolutely positioned against a relative parent that always
 * reserves room for it (the wrapper's bottom padding), so it can overlap
 * the tablet without ever escaping the section on narrow viewports.
 */
export default function DashboardMockup() {
  return (
    <div className="relative mx-auto w-full max-w-lg pb-16 pl-8 sm:pb-20 sm:pl-12">
      {/* Tablet */}
      <div className="ml-auto w-[92%] overflow-hidden rounded-2xl border-4 border-slate-800 bg-white shadow-2xl">
        <div className="flex items-center justify-between border-b border-slate-100 px-4 py-3">
          <span className="text-xs font-bold italic text-brand-navy">whuzpay</span>
          <span className="text-[10px] text-slate-400">Dashboard</span>
        </div>
        <div className="p-4">
          <div className="mb-3 flex items-center justify-between rounded-lg bg-slate-50 px-3 py-2">
            <span className="text-[10px] font-medium text-slate-500">Saldo Pending</span>
            <span className="text-xs font-bold text-slate-700">Rp 30.223.000</span>
          </div>
          <div className="grid grid-cols-3 gap-2">
            {[
              { label: "Sukses", value: "30.223.000", tone: "bg-teal-500" },
              { label: "Gagal", value: "0", tone: "bg-rose-400" },
              { label: "Pending", value: "0", tone: "bg-cyan-500" },
            ].map((stat) => (
              <div key={stat.label} className={`rounded-lg ${stat.tone} p-2 text-white`}>
                <p className="text-[9px] opacity-80">{stat.label}</p>
                <p className="text-xs font-bold">{stat.value}</p>
              </div>
            ))}
          </div>
          <div className="mt-4 rounded-lg border border-slate-100 p-3">
            <p className="mb-2 text-[10px] font-medium text-slate-400">Transaksi 7 hari terakhir</p>
            <div className="flex h-20 items-end gap-1.5">
              {bars.map((height, i) => (
                <span
                  key={i}
                  style={{ height: `${height}%` }}
                  className="w-full rounded-sm bg-brand-navy-light/70"
                />
              ))}
            </div>
          </div>
        </div>
      </div>

      {/* Phone */}
      <div className="absolute bottom-0 left-0 w-32 overflow-hidden rounded-2xl border-4 border-slate-800 bg-white shadow-2xl sm:w-40">
        <div className="border-b border-slate-100 px-3 py-2 text-center">
          <span className="text-[9px] font-bold italic text-brand-navy">whuzpay</span>
        </div>
        <div className="flex flex-col gap-1.5 p-3">
          {methods.map((method) => (
            <div
              key={method}
              className="flex items-center justify-between rounded-md bg-slate-50 px-2 py-1.5"
            >
              <span className="text-[9px] font-medium text-slate-500">{method}</span>
              <span className="h-1.5 w-1.5 rounded-full bg-teal-500" />
            </div>
          ))}
          <div className="mt-1 rounded-md bg-brand-yellow px-2 py-1.5 text-center text-[9px] font-bold text-brand-navy-dark">
            BAYAR
          </div>
        </div>
      </div>
    </div>
  );
}
