"use client";

import * as React from "react";
import { format, parse, isValid } from "date-fns";
import { CalendarIcon } from "lucide-react";

import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { Calendar } from "@/components/ui/calendar";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";

export type DatePickerProps = {
  /** YYYY-MM-DD string (API-friendly). Empty string = no selection. */
  value?: string;
  onChange?: (value: string) => void;
  placeholder?: string;
  label?: string;
  disabled?: boolean;
  /** Minimum selectable date as YYYY-MM-DD */
  min?: string;
  /** Maximum selectable date as YYYY-MM-DD */
  max?: string;
  className?: string;
  triggerClassName?: string;
};

function parseYmd(value?: string): Date | undefined {
  if (!value) return undefined;
  const d = parse(value, "yyyy-MM-dd", new Date());
  return isValid(d) ? d : undefined;
}

function toYmd(date: Date): string {
  return format(date, "yyyy-MM-dd");
}

export function DatePicker({
  value = "",
  onChange,
  placeholder = "Pick a date",
  label,
  disabled,
  min,
  max,
  className,
  triggerClassName,
}: DatePickerProps) {
  const [open, setOpen] = React.useState(false);
  const selected = parseYmd(value);
  const minDate = parseYmd(min);
  const maxDate = parseYmd(max);

  return (
    <div className={cn("flex min-w-[160px] flex-col gap-1.5", className)}>
      {label ? (
        <span className="text-[11px] font-semibold uppercase tracking-wide text-[#8a97a8]">
          {label}
        </span>
      ) : null}
      <Popover open={open} onOpenChange={setOpen}>
        <PopoverTrigger asChild>
          <Button
            type="button"
            variant="outline"
            disabled={disabled}
            data-empty={!selected}
            className={cn(
              "h-9 w-full justify-start rounded-full border-[#e8eef4] bg-[#f8fafc] px-3 text-left text-[13px] font-normal text-[#1f2a37] shadow-none hover:bg-white data-[empty=true]:text-[#8a97a8]",
              triggerClassName,
            )}
          >
            <CalendarIcon className="size-4 text-[#8a97a8]" />
            {selected ? format(selected, "dd MMM yyyy") : placeholder}
          </Button>
        </PopoverTrigger>
        <PopoverContent
          align="start"
          className="w-auto border border-[#e8eef4] bg-white p-0 shadow-lg"
        >
          <Calendar
            mode="single"
            selected={selected}
            defaultMonth={selected}
            disabled={(date) => {
              if (minDate && date < minDate) return true;
              if (maxDate && date > maxDate) return true;
              return false;
            }}
            onSelect={(date) => {
              onChange?.(date ? toYmd(date) : "");
              setOpen(false);
            }}
          />
          {value ? (
            <div className="border-t border-[#eef2f6] p-2">
              <Button
                type="button"
                variant="ghost"
                size="sm"
                className="h-8 w-full text-[12.5px] text-[#6b7c93]"
                onClick={() => {
                  onChange?.("");
                  setOpen(false);
                }}
              >
                Clear
              </Button>
            </div>
          ) : null}
        </PopoverContent>
      </Popover>
    </div>
  );
}
