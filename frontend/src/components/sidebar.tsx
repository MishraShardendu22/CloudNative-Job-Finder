"use client";

import { FileText, LayoutDashboard, Sparkles, UserRound } from "lucide-react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { cn } from "@/lib/utils";

const items = [
  { href: "/dashboard", label: "Dashboard", icon: LayoutDashboard },
  { href: "/resumes", label: "Resumes", icon: FileText },
  { href: "/recommendations", label: "Recommendations", icon: Sparkles },
  { href: "/profile", label: "Profile", icon: UserRound },
];

export function Sidebar() {
  const pathname = usePathname();

  return (
    <aside className="hidden w-[248px] shrink-0 border-r border-border/80 bg-background/85 px-4 py-6 lg:block">
      <div className="space-y-1">
        {items.map((item) => {
          const Icon = item.icon;
          const active = pathname === item.href;

          return (
            <Link
              key={item.href}
              href={item.href}
              className={cn(
                "flex items-center gap-3 rounded-xl px-3 py-2.5 text-sm font-medium text-muted transition-colors duration-180 hover:bg-card hover:text-foreground",
                active &&
                  "bg-primary/12 text-primary shadow-[inset_0_0_0_1px_rgba(249,115,22,0.35)]",
              )}
            >
              <Icon className="h-4 w-4" />
              <span>{item.label}</span>
            </Link>
          );
        })}
      </div>
    </aside>
  );
}
