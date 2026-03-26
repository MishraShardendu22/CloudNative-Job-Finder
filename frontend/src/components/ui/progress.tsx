import { cn } from "@/lib/utils";

export function Progress({ value }: { value: number }) {
  return (
    <div className="h-2 w-full rounded-full bg-black/60">
      <div
        className={cn(
          "h-full rounded-full bg-primary shadow-[0_0_12px_rgba(249,115,22,0.6)] transition-all duration-180",
        )}
        style={{ width: `${Math.max(0, Math.min(100, value))}%` }}
      />
    </div>
  );
}
