import * as React from "react"

import { cn } from "@/lib/utils"

function Textarea({ className, ...props }: React.ComponentProps<"textarea">) {
  return (
    <textarea
      data-slot="textarea"
      className={cn(
        // Base
        "flex field-sizing-content min-h-[80px] w-full rounded-xl border border-border bg-input/50 px-3 py-2.5 text-sm transition-colors outline-none",
        // Placeholder
        "placeholder:text-muted-foreground/50",
        // Focus
        "focus-visible:border-primary/60 focus-visible:ring-2 focus-visible:ring-primary/20 focus-visible:bg-background",
        // Disabled
        "disabled:cursor-not-allowed disabled:opacity-50 disabled:bg-input/30",
        // Invalid
        "aria-invalid:border-destructive aria-invalid:ring-2 aria-invalid:ring-destructive/20",
        className
      )}
      {...props}
    />
  )
}

export { Textarea }
