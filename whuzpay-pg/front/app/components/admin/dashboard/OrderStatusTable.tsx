import {
  IconCopy,
  IconLock,
  IconPlus,
  IconPrinter,
  IconSearch,
  IconTrash,
} from "../icons";

const rows = [
  {
    invoice: "12386",
    customer: "Charly Dues",
    from: "Brazil",
    price: "$299",
    status: "Process" as const,
  },
  {
    invoice: "12386",
    customer: "Marko",
    from: "Italy",
    price: "$2642",
    status: "Open" as const,
  },
  {
    invoice: "12386",
    customer: "Denjel Onak",
    from: "Russia",
    price: "$981",
    status: "On Hold" as const,
  },
  {
    invoice: "12386",
    customer: "Belgim Bastana",
    from: "Korea",
    price: "$369",
    status: "Process" as const,
  },
  {
    invoice: "12386",
    customer: "Sarti Onuska",
    from: "Japan",
    price: "$1240",
    status: "Open" as const,
  },
];

const statusClass: Record<(typeof rows)[number]["status"], string> = {
  Process: "bg-[#e85d3b] text-white",
  Open: "bg-[#4ca040] text-white",
  "On Hold": "bg-[#f0a04b] text-white",
};

export default function OrderStatusTable() {
  return (
    <div className="h-full rounded-xl bg-white p-5 shadow-[0_1px_3px_rgba(16,38,73,0.06)]">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h3 className="text-[15px] font-semibold text-[#1f2a37]">Order Status</h3>
          <p className="mt-0.5 text-[12px] text-[#8a97a8]">Overview of Latest Month</p>
        </div>

        <div className="flex flex-wrap items-center gap-2">
          <div className="mr-1 flex items-center gap-1.5">
            <button
              type="button"
              className="inline-flex h-8 items-center gap-1 rounded-md bg-[#ff5e16] px-2.5 text-[12px] font-semibold text-white shadow-sm"
            >
              <IconPlus className="h-3.5 w-3.5" />
              Add
            </button>
            <button
              type="button"
              className="flex h-8 w-8 items-center justify-center rounded-md bg-[#3b9eff] text-white"
              aria-label="Print"
            >
              <IconPrinter className="h-3.5 w-3.5" />
            </button>
            <button
              type="button"
              className="flex h-8 w-8 items-center justify-center rounded-md bg-[#e85d3b] text-white"
              aria-label="Delete"
            >
              <IconTrash className="h-3.5 w-3.5" />
            </button>
            <button
              type="button"
              className="flex h-8 w-8 items-center justify-center rounded-md bg-[#22c55e] text-white"
              aria-label="Copy"
            >
              <IconCopy className="h-3.5 w-3.5" />
            </button>
          </div>

          <div className="relative">
            <IconSearch className="pointer-events-none absolute left-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-[#8a97a8]" />
            <input
              type="search"
              placeholder="Search"
              className="h-8 w-[140px] rounded-md border border-[#e8eef4] bg-white pl-8 pr-3 text-[12px] text-[#1f2a37] outline-none placeholder:text-[#b0bbc8] focus:border-[#3b9eff]"
            />
          </div>
          <button
            type="button"
            className="flex h-8 w-8 items-center justify-center rounded-md border border-[#e8eef4] text-[#8a97a8]"
            aria-label="Lock"
          >
            <IconLock className="h-3.5 w-3.5" />
          </button>
        </div>
      </div>

      <div className="mt-4 overflow-x-auto">
        <table className="w-full min-w-[560px] border-collapse text-left text-[12.5px]">
          <thead>
            <tr className="border-b border-[#eef2f6] text-[11px] font-semibold uppercase tracking-wide text-[#8a97a8]">
              <th className="pb-3 pr-3 font-semibold">Invoice</th>
              <th className="pb-3 pr-3 font-semibold">Customers</th>
              <th className="pb-3 pr-3 font-semibold">From</th>
              <th className="pb-3 pr-3 font-semibold">Price</th>
              <th className="pb-3 font-semibold">Status</th>
            </tr>
          </thead>
          <tbody>
            {rows.map((row, i) => (
              <tr
                key={`${row.customer}-${i}`}
                className="border-b border-[#f3f6f9] text-[#3d4b5c] last:border-0"
              >
                <td className="py-3.5 pr-3">{row.invoice}</td>
                <td className="py-3.5 pr-3 font-medium text-[#1f2a37]">{row.customer}</td>
                <td className="py-3.5 pr-3">{row.from}</td>
                <td className="py-3.5 pr-3">{row.price}</td>
                <td className="py-3.5">
                  <span
                    className={`inline-flex min-w-[68px] items-center justify-center rounded px-2.5 py-1 text-[11px] font-semibold ${statusClass[row.status]}`}
                  >
                    {row.status}
                  </span>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
