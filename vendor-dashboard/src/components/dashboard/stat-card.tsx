import type { ReactNode } from "react";
import Link from "next/link";
import { Card, CardContent } from "@/components/ui/card";
import { cn } from "@/lib/utils";

export function StatCard({
  title,
  value,
  subtitle,
  icon,
  href,
}: {
  title: string;
  value: ReactNode;
  subtitle?: ReactNode;
  icon?: ReactNode;
  /** Bila diisi, seluruh kartu jadi link -- dipakai mis. "Kedaluwarsa" di
   * Dashboard yang menuju Accounts dengan filter status sudah terpasang. */
  href?: string;
}) {
  const card = (
    <Card
      className={cn(
        "gap-0 rounded border-none py-0 shadow-sm ring-1 ring-border/60",
        href && "transition-shadow hover:shadow-md hover:ring-primary/30",
      )}
    >
      <CardContent className="flex flex-col gap-3 p-5">
        <div className="flex items-center justify-between">
          <p className="text-sm font-medium text-muted-foreground">{title}</p>
          {icon && (
            <span className="flex size-8 items-center justify-center rounded-lg bg-primary/10 text-primary">
              {icon}
            </span>
          )}
        </div>
        <div className="text-2xl font-semibold tracking-tight">{value}</div>
        {subtitle && <p className="text-xs text-muted-foreground">{subtitle}</p>}
      </CardContent>
    </Card>
  );

  if (!href) return card;
  return (
    <Link href={href} className="block">
      {card}
    </Link>
  );
}
