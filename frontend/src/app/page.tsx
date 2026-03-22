import { ArrowRight, BriefcaseBusiness } from "lucide-react";
import Link from "next/link";
import { Button } from "@/components/ui/button";

export default function Home() {
  return (
    <main className="min-h-screen bg-background px-6">
      <section className="mx-auto grid w-full max-w-[1280px] gap-12 py-24 lg:grid-cols-12 lg:items-center">
        <div className="space-y-6 lg:col-span-7">
          <span className="inline-flex items-center gap-2 rounded-full border bg-card px-3 py-1 text-xs font-medium text-muted">
            <BriefcaseBusiness className="h-3.5 w-3.5" />
            Resume-driven matching engine
          </span>

          <h1 className="text-balance text-[32px] font-semibold tracking-tight text-foreground sm:text-[40px]">
            Turn every resume into targeted job recommendations.
          </h1>

          <p className="max-w-xl text-base leading-relaxed text-muted">
            Upload your resumes, analyze top-fit roles, and track opportunities
            from a single dashboard connected to your existing recommendation
            backend.
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

        <div className="rounded-[12px] border bg-card p-6 shadow-[0_1px_2px_rgba(0,0,0,0.05)] lg:col-span-5">
          <div className="space-y-4">
            <div className="rounded-lg bg-background p-4">
              <p className="text-xs uppercase tracking-wide text-muted">
                Best match
              </p>
              <p className="mt-2 text-lg font-medium">
                Senior Backend Engineer
              </p>
              <p className="text-sm text-muted">
                93% score • Remote • ACME Labs
              </p>
            </div>
            <div className="rounded-lg bg-background p-4">
              <p className="text-xs uppercase tracking-wide text-muted">
                Resume status
              </p>
              <p className="mt-2 text-lg font-medium">2 parsed, 1 processing</p>
              <p className="text-sm text-muted">
                Upload, parse, recommend in minutes
              </p>
            </div>
            <div className="rounded-lg border border-primary/20 bg-primary/5 p-4 text-foreground">
              <p className="text-xs uppercase tracking-wide text-primary">
                Insight
              </p>
              <p className="mt-2 text-sm text-foreground/85">
                Skills relevance improved by 18% after profile enrichment.
              </p>
            </div>
          </div>
        </div>
      </section>
    </main>
  );
}
