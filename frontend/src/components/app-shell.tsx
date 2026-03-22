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
    <div className="min-h-screen bg-[radial-gradient(circle_at_top,_rgba(113,113,122,0.15),_transparent_40%),linear-gradient(to_bottom,_#fafafa,_#f4f4f5)] dark:bg-[radial-gradient(circle_at_top,_rgba(161,161,170,0.22),_transparent_38%),linear-gradient(to_bottom,_#09090b,_#0f172a)]">
      <Navbar />
      <div className="mx-auto flex w-full max-w-7xl">
        <Sidebar />
        <main className="min-h-[calc(100vh-4rem)] flex-1 p-4 sm:p-6 lg:p-8">
          <header className="mb-6 space-y-1">
            <h1 className="text-2xl font-semibold tracking-tight text-zinc-900 dark:text-zinc-100">
              {title}
            </h1>
            <p className="text-sm text-zinc-500 dark:text-zinc-400">{description}</p>
          </header>
          {children}
        </main>
      </div>
    </div>
  );
}
