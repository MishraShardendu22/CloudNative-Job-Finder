import type * as React from "react";
import { cn } from "@/lib/utils";

function Input({ className, type, ...props }: React.ComponentProps<"input">) {
  return (
    <input
      type={type}
      className={cn(
        "flex h-10 w-full rounded-lg border bg-card px-3 py-2 text-base text-foreground outline-none transition-colors duration-180 file:border-0 file:bg-transparent file:text-sm file:font-medium placeholder:text-muted focus-visible:border-primary focus-visible:ring-2 focus-visible:ring-primary/20 aria-[invalid=true]:border-danger aria-[invalid=true]:ring-2 aria-[invalid=true]:ring-danger/20",
        className,
      )}
      {...props}
    />
  );
}

export { Input };
