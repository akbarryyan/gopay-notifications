"use client";

import { ChevronDownIcon } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { cn } from "@/lib/utils";

export type FilterOption = {
  value: string;
  label: string;
};

type FilterDropdownProps = {
  label?: string;
  value: string;
  options: FilterOption[];
  onChange: (value: string) => void;
  className?: string;
  triggerClassName?: string;
};

export default function FilterDropdown({
  label,
  value,
  options,
  onChange,
  className,
  triggerClassName,
}: FilterDropdownProps) {
  const selected = options.find((o) => o.value === value) ?? options[0];

  return (
    <div className={cn("flex min-w-[140px] flex-col gap-1.5", className)}>
      {label ? (
        <span className="text-[11px] font-semibold uppercase tracking-wide text-[#8a97a8]">
          {label}
        </span>
      ) : null}
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button
            type="button"
            variant="outline"
            className={cn(
              "h-9 w-full justify-between border-[#e8eef4] bg-[#f8fafc] px-3 text-[13px] font-medium text-[#1f2a37] shadow-none hover:bg-white hover:text-[#1f2a37]",
              triggerClassName,
            )}
          >
            <span className="truncate capitalize">{selected?.label ?? "All"}</span>
            <ChevronDownIcon className="size-4 opacity-60" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent
          align="start"
          className="min-w-[var(--radix-dropdown-menu-trigger-width)] border border-[#e8eef4] bg-white p-1 text-[#1f2a37] shadow-[0_12px_30px_rgba(16,38,73,0.12)]"
        >
          <DropdownMenuRadioGroup value={value} onValueChange={onChange}>
            {options.map((option) => (
              <DropdownMenuRadioItem
                key={option.value || "all"}
                value={option.value}
                className="cursor-pointer rounded-md px-2 py-1.5 text-[13px] capitalize focus:bg-[#f4f7fb] focus:text-[#1f2a37]"
              >
                {option.label}
              </DropdownMenuRadioItem>
            ))}
          </DropdownMenuRadioGroup>
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  );
}
