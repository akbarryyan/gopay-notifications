import type { ReactNode } from "react";

export function PageHeader({
  title,
  description,
  actions,
}: {
  title: string;
  description?: string;
  actions?: ReactNode;
}) {
  return (
    <div className="mb-5 flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
      <div>
        <h1 className="text-[20px] font-semibold tracking-tight text-[#1f2a37]">
          {title}
        </h1>
        {description && (
          <p className="mt-1 text-[13px] text-[#8a97a8]">{description}</p>
        )}
      </div>
      {actions}
    </div>
  );
}

export function Card({
  children,
  className = "",
}: {
  children: ReactNode;
  className?: string;
}) {
  return (
    <div
      className={`rounded-xl border border-[#e8eef4] bg-white shadow-[0_1px_3px_rgba(16,38,73,0.06)] ${className}`}
    >
      {children}
    </div>
  );
}

export function StatusBadge({ status }: { status: string }) {
  const styles: Record<string, string> = {
    pending: "bg-[#fff7e6] text-[#c27a00]",
    paid: "bg-[#e8f8ee] text-[#2f9e5a]",
    expired: "bg-[#eef2f6] text-[#6b7c93]",
    failed: "bg-[#fff1ed] text-[#e85d3b]",
    cancelled: "bg-[#f3f0ff] text-[#7c3aed]",
  };
  const className = styles[status] ?? "bg-[#eef2f6] text-[#6b7c93]";

  return (
    <span
      className={`inline-flex rounded-full px-2.5 py-0.5 text-[11px] font-semibold capitalize ${className}`}
    >
      {status}
    </span>
  );
}

export function formatIDR(amount: number, currency = "IDR") {
  try {
    return new Intl.NumberFormat("id-ID", {
      style: "currency",
      currency,
      maximumFractionDigits: 0,
    }).format(amount);
  } catch {
    return `${currency} ${amount.toLocaleString("id-ID")}`;
  }
}

export function formatDateTime(value?: string | null) {
  if (!value) return "—";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "—";
  return date.toLocaleString("en-GB", {
    day: "2-digit",
    month: "short",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

export function EmptyState({ message }: { message: string }) {
  return (
    <div className="px-5 py-12 text-center text-[13px] text-[#8a97a8]">
      {message}
    </div>
  );
}

export function LoadingBlock({ label = "Loading..." }: { label?: string }) {
  return (
    <div className="px-5 py-12 text-center text-[13px] text-[#8a97a8]">
      {label}
    </div>
  );
}

export function TableShell({ children }: { children: ReactNode }) {
  return <div className="overflow-x-auto">{children}</div>;
}
