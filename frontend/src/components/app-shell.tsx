import { Navbar } from "@/components/navbar";
import { Sidebar } from "@/components/sidebar";

export function AppShell({
  title,
  description,
  children,
}: {
  title: string;
  description: string;
  children: React.ReactNode;
}) {
  return (
    <div className="min-h-screen bg-background">
      <Navbar />
      <div className="mx-auto flex w-full max-w-[1360px]">
        <Sidebar />
        <main className="min-h-[calc(100vh-4rem)] flex-1 px-6 py-8 lg:px-9">
          <header className="mb-9 space-y-2 border-b border-border/70 pb-6">
            <h1 className="text-[34px] font-bold tracking-tight text-foreground">
              {title}
            </h1>
            <p className="max-w-2xl text-sm leading-relaxed text-muted">
              {description}
            </p>
          </header>
          {children}
        </main>
      </div>
    </div>
  );
}
