"use client";

import { BriefcaseBusiness, Moon, Sun } from "lucide-react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { useTheme } from "@/components/theme-provider";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

const links = [
  { href: "/dashboard", label: "Dashboard" },
  { href: "/resumes", label: "Resumes" },
  { href: "/recommendations", label: "Recommendations" },
  { href: "/profile", label: "Profile" },
];

export function Navbar() {
  const pathname = usePathname();
  const { setTheme, resolvedTheme } = useTheme();

  return (
    <header className="sticky top-0 z-50 border-b bg-background/95 backdrop-blur">
      <div className="mx-auto flex h-16 w-full max-w-[1280px] items-center justify-between px-6">
        <Link href="/dashboard" className="flex items-center gap-2.5">
          <span className="rounded-lg bg-primary p-2 text-white">
            <BriefcaseBusiness className="h-4 w-4" />
          </span>
          <span className="text-sm font-semibold tracking-wide text-foreground">
            Job Finder
          </span>
        </Link>

        <nav className="hidden items-center gap-1 lg:flex">
          {links.map((link) => (
            <Link
              key={link.href}
              href={link.href}
              className={cn(
                "rounded-lg px-3 py-2 text-sm font-medium text-muted hover:bg-card hover:text-foreground",
                pathname === link.href && "bg-primary/10 text-primary",
              )}
            >
              {link.label}
            </Link>
          ))}
        </nav>

        <Button
          variant="outline"
          size="icon"
          aria-label="Toggle theme"
          onClick={() => setTheme(resolvedTheme === "dark" ? "light" : "dark")}
        >
          {resolvedTheme === "dark" ? (
            <Sun className="h-4 w-4" />
          ) : (
            <Moon className="h-4 w-4" />
          )}
        </Button>
      </div>

      <div className="border-t lg:hidden">
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
