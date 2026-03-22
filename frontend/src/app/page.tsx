import { ArrowRight, BriefcaseBusiness } from "lucide-react";
import Link from "next/link";
import { Button } from "@/components/ui/button";

export default function Home() {
  return (
    <main className="relative flex min-h-screen items-center overflow-hidden bg-[radial-gradient(circle_at_top,_rgba(24,24,27,0.15),_transparent_40%),linear-gradient(to_bottom,_#fafafa,_#f1f5f9)] px-4 dark:bg-[radial-gradient(circle_at_top,_rgba(255,255,255,0.1),_transparent_35%),linear-gradient(to_bottom,_#09090b,_#111827)]">
      <section className="mx-auto grid w-full max-w-6xl gap-12 py-20 lg:grid-cols-2 lg:items-center">
        <div className="space-y-6">
          <span className="inline-flex items-center gap-2 rounded-full border border-zinc-300 bg-white/70 px-3 py-1 text-xs font-medium text-zinc-700 dark:border-zinc-700 dark:bg-zinc-900/70 dark:text-zinc-200">
            <BriefcaseBusiness className="h-3.5 w-3.5" />
            Resume-driven matching engine
          </span>

          <h1 className="text-balance text-4xl font-semibold tracking-tight text-zinc-900 sm:text-5xl dark:text-zinc-100">
            Turn every resume into targeted job recommendations.
          </h1>

          <p className="max-w-xl text-base leading-relaxed text-zinc-600 sm:text-lg dark:text-zinc-300">
            Upload your resumes, analyze top-fit roles, and track opportunities from a
            single dashboard connected to your existing recommendation backend.
          </p>

          <div className="flex flex-wrap gap-3">
            <Button asChild size="lg">
              <Link href="/signup">
                Create account
                <ArrowRight className="h-4 w-4" />
              </Link>
            </Button>
            <Button asChild variant="outline" size="lg">
              <Link href="/login">Login</Link>
            </Button>
          </div>
        </div>

        <div className="rounded-3xl border border-zinc-200 bg-white/80 p-6 shadow-xl shadow-zinc-200/60 backdrop-blur dark:border-zinc-800 dark:bg-zinc-900/70 dark:shadow-black/30">
          <div className="space-y-4">
            <div className="rounded-2xl bg-zinc-100 p-4 dark:bg-zinc-800">
              <p className="text-xs uppercase tracking-wide text-zinc-500">Best match</p>
              <p className="mt-2 text-lg font-medium">Senior Backend Engineer</p>
              <p className="text-sm text-zinc-500">93% score • Remote • ACME Labs</p>
            </div>
            <div className="rounded-2xl bg-zinc-100 p-4 dark:bg-zinc-800">
              <p className="text-xs uppercase tracking-wide text-zinc-500">Resume status</p>
              <p className="mt-2 text-lg font-medium">2 parsed, 1 processing</p>
              <p className="text-sm text-zinc-500">Upload, parse, recommend in minutes</p>
            </div>
            <div className="rounded-2xl bg-zinc-900 p-4 text-zinc-100 dark:bg-zinc-100 dark:text-zinc-900">
              <p className="text-xs uppercase tracking-wide opacity-80">Insight</p>
              <p className="mt-2 text-sm">
                Skills relevance improved by 18% after profile enrichment.
              </p>
            </div>
          </div>
        </div>
      </section>
    </main>
  );
}
