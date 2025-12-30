"use client"

import * as React from "react"
import * as SwitchPrimitive from "@radix-ui/react-switch"

import { cn } from "@/lib/utils"

function Switch({
  className,
  ...props
}: React.ComponentProps<typeof SwitchPrimitive.Root>) {
  return (
    <SwitchPrimitive.Root
      data-slot="switch"
      className={cn(
        // Base styles
        "peer inline-flex h-[1.15rem] w-8 shrink-0 items-center rounded-full border border-transparent shadow-xs transition-all outline-none cursor-pointer",
        // Unchecked state
        "data-[state=unchecked]:bg-carbon-300 dark:data-[state=unchecked]:bg-carbon-600",
        // Checked state - using brand color
        "data-[state=checked]:bg-[var(--brand-solid)]",
        // Focus state
        "focus-visible:ring-[3px] focus-visible:ring-[var(--brand-solid)]/30 focus-visible:border-[var(--brand-solid)]",
        // Disabled state
        "disabled:cursor-not-allowed disabled:opacity-50",
        // Hover micro-animation
        "hover:scale-[1.02] active:scale-[0.98]",
        className
      )}
      {...props}
    >
      <SwitchPrimitive.Thumb
        data-slot="switch-thumb"
        className={cn(
          "bg-white dark:data-[state=unchecked]:bg-carbon-200 dark:data-[state=checked]:bg-white pointer-events-none block size-4 rounded-full ring-0 transition-transform data-[state=checked]:translate-x-[calc(100%-2px)] data-[state=unchecked]:translate-x-0"
        )}
      />
    </SwitchPrimitive.Root>
  )
}

export { Switch }

