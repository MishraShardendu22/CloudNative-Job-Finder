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
      <div className="mx-auto flex w-full max-w-[1280px]">
        <Sidebar />
        <main className="min-h-[calc(100vh-4rem)] flex-1 p-6">
          <header className="mb-8 space-y-2">
            <h1 className="text-[32px] font-semibold tracking-tight text-foreground">
              {title}
            </h1>
            <p className="max-w-2xl text-sm text-muted">{description}</p>
          </header>
          {children}
        </main>
      </div>
    </div>
  );
}
