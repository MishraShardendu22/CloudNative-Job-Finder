"use client";

import { BriefcaseBusiness } from "lucide-react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { cn } from "@/lib/utils";

const links = [
  { href: "/dashboard", label: "Dashboard" },
  { href: "/resumes", label: "Resumes" },
  { href: "/recommendations", label: "Recommendations" },
  { href: "/profile", label: "Profile" },
];

export function Navbar() {
  const pathname = usePathname();

  return (
    <header className="sticky top-0 z-50 border-b border-border/80 bg-background/90 backdrop-blur-xl">
      <div className="mx-auto flex h-16 w-full max-w-[1280px] items-center justify-between px-6">
        <Link href="/dashboard" className="flex items-center gap-3">
          <span className="rounded-xl border border-primary/40 bg-primary/15 p-2.5 text-primary shadow-[0_0_0_1px_rgba(249,115,22,0.15)]">
            <BriefcaseBusiness className="h-4 w-4" />
          </span>
          <span className="text-sm font-semibold uppercase tracking-[0.18em] text-foreground">
            Job Finder
          </span>
        </Link>

        <nav className="hidden items-center gap-1 lg:flex">
          {links.map((link) => (
            <Link
              key={link.href}
              href={link.href}
              className={cn(
                "rounded-lg px-3 py-2 text-sm font-medium text-muted transition-all duration-180 hover:bg-card hover:text-foreground",
                pathname === link.href &&
                  "bg-primary/10 text-primary shadow-[inset_0_0_0_1px_rgba(249,115,22,0.35)]",
              )}
            >
              {link.label}
            </Link>
          ))}
        </nav>

        <div className="w-20" />
      </div>

      <div className="border-t border-border/80 lg:hidden">
        <nav className="mx-auto flex h-12 w-full max-w-[1280px] items-center gap-1 overflow-x-auto px-6">
          {links.map((link) => (
            <Link
              key={link.href}
              href={link.href}
              className={cn(
                "whitespace-nowrap rounded-lg px-3 py-1.5 text-sm font-medium text-muted",
                pathname === link.href && "bg-primary/10 text-primary",
              )}
            >
              {link.label}
            </Link>
          ))}
        </nav>
      </div>
    </header>
  );
}
