export default function EarningsOverview() {
  return (
    <div className="flex h-full flex-col justify-between rounded-xl bg-white p-5 shadow-[0_1px_3px_rgba(16,38,73,0.06)]">
      <div>
        <h2 className="text-[15px] font-semibold text-[#1f2a37]">Dashboard</h2>
        <p className="mt-0.5 text-[12px] text-[#8a97a8]">Overview of Latest Month</p>

        <div className="mt-6">
          <p className="text-[28px] font-bold leading-none tracking-tight text-[#1f2a37]">
            $3468.96
          </p>
          <p className="mt-1.5 text-[12px] text-[#8a97a8]">Current Month Earnings</p>
        </div>

        <div className="mt-5">
          <p className="text-[28px] font-bold leading-none tracking-tight text-[#1f2a37]">82</p>
          <p className="mt-1.5 text-[12px] text-[#8a97a8]">Current Month Sales</p>
        </div>
      </div>

      <button
        type="button"
        className="mt-6 inline-flex w-fit items-center rounded-md bg-[#ff5e16] px-4 py-2.5 text-[12px] font-semibold text-white shadow-[0_2px_8px_rgba(255,94,22,0.35)] transition hover:bg-[#ef5510]"
      >
        Last Month Summary
      </button>
    </div>
  );
}
