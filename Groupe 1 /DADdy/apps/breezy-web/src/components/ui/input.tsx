import * as React from "react"
import { Input as InputPrimitive } from "@base-ui/react/input"

import { cn } from "@/lib/utils"

function Input({ className, type, ...props }: React.ComponentProps<"input">) {
  return (
    <InputPrimitive
      type={type}
      data-slot="input"
      className={cn(
        // Base
        "h-10 w-full min-w-0 rounded-xl border border-border bg-input/50 px-3 py-2 text-sm transition-colors outline-none",
        // Placeholder
        "placeholder:text-muted-foreground/50",
        // Focus
        "focus-visible:border-primary/60 focus-visible:ring-2 focus-visible:ring-primary/20 focus-visible:bg-background",
        // File input
        "file:inline-flex file:h-6 file:border-0 file:bg-transparent file:text-sm file:font-medium file:text-foreground",
        // Disabled
        "disabled:pointer-events-none disabled:cursor-not-allowed disabled:opacity-50 disabled:bg-input/30",
        // Invalid
        "aria-invalid:border-destructive aria-invalid:ring-2 aria-invalid:ring-destructive/20",
        className
      )}
      {...props}
    />
  )
}

export { Input }
