import * as React from "react"

import { cn } from "@/lib/utils"

function Input({ className, type, ...props }: React.ComponentProps<"input">) {
  return (
    <input
      type={type}
      data-slot="input"
      className={cn(
        // Base styles
        "flex h-9 w-full min-w-0 rounded-[var(--radius-btn)] border bg-[var(--bg-main)] px-3 py-1 text-base shadow-xs transition-all outline-none md:text-sm",
        // Border
        "border-[var(--border-subtle)]",
        // Text and placeholder
        "text-[var(--text-primary)] placeholder:text-[var(--text-tertiary)]",
        // Selection
        "selection:bg-[var(--brand-solid)]/20 selection:text-[var(--text-primary)]",
        // Focus state - brand color
        "focus-visible:border-[var(--brand-solid)] focus-visible:ring-[3px] focus-visible:ring-[var(--brand-solid)]/20",
        // File input styles
        "file:text-foreground file:inline-flex file:h-7 file:border-0 file:bg-transparent file:text-sm file:font-medium",
        // Disabled
        "disabled:pointer-events-none disabled:cursor-not-allowed disabled:opacity-50",
        // Invalid state
        "aria-invalid:ring-destructive/20 dark:aria-invalid:ring-destructive/40 aria-invalid:border-destructive",
        // Dark mode adjustments
        "dark:bg-[var(--bg-card)]",
        className
      )}
      {...props}
    />
  )
}

export { Input }

