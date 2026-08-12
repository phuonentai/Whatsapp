import * as React from "react";
import { cva, type VariantProps } from "class-variance-authority";
import type { LucideIcon } from "lucide-react";

import { cn } from "@/lib/utils";

/**
 * StatusChip — semantic status chip (design language, settings-redesign).
 *
 * Always renders icon + text (a11y rule: never color-only). Tones map to the
 * Verifika design language: emerald = active/connected, amber = pending/
 * threshold, red = error/disconnected, gray = neutral, blue = info.
 */
const statusChipVariants = cva(
  "inline-flex items-center gap-1.5 rounded-full border px-2.5 py-0.5 text-xs font-medium",
  {
    variants: {
      tone: {
        emerald: "border-emerald-200 bg-emerald-50 text-emerald-800",
        amber: "border-amber-200 bg-amber-50 text-amber-800",
        red: "border-red-200 bg-red-50 text-red-700",
        gray: "border-slate-200 bg-slate-100 text-slate-600",
        blue: "border-blue-200 bg-blue-50 text-blue-700",
      },
    },
    defaultVariants: {
      tone: "gray",
    },
  }
);

export interface StatusChipProps
  extends React.HTMLAttributes<HTMLSpanElement>,
    VariantProps<typeof statusChipVariants> {
  icon?: LucideIcon;
}

function StatusChip({
  className,
  tone,
  icon: Icon,
  children,
  ...props
}: StatusChipProps) {
  return (
    <span className={cn(statusChipVariants({ tone }), className)} {...props}>
      {Icon ? <Icon className="h-3.5 w-3.5 shrink-0" aria-hidden="true" /> : null}
      <span>{children}</span>
    </span>
  );
}

StatusChip.displayName = "StatusChip";

export { StatusChip, statusChipVariants };
