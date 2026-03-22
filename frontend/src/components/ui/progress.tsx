import { cn } from "@/lib/utils";

export function Progress({ value }: { value: number }) {
  return (
    <div className="h-2 w-full rounded-full bg-background">
      <div
        className={cn(
          "h-full rounded-full bg-primary transition-all duration-180",
        )}
        style={{ width: `${Math.max(0, Math.min(100, value))}%` }}
      />
    </div>
  );
}
