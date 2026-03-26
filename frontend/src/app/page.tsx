import { ArrowRight, BriefcaseBusiness } from "lucide-react";
import Link from "next/link";
import { Button } from "@/components/ui/button";

export default function Home() {
  return (
    <main className="min-h-screen bg-background px-6">
      <section className="mx-auto grid w-full max-w-[1320px] gap-14 py-24 lg:grid-cols-12 lg:items-center">
        <div className="space-y-8 lg:col-span-7">
          <span className="inline-flex items-center gap-2 rounded-full border border-primary/40 bg-primary/10 px-4 py-1.5 text-xs font-semibold uppercase tracking-[0.16em] text-primary">
            <BriefcaseBusiness className="h-3.5 w-3.5" />
            Resume-driven matching engine
          </span>

          <h1 className="text-balance text-[36px] font-bold tracking-tight text-foreground sm:text-[54px] sm:leading-[1.05]">
            Precision Job Matching for High-Intent Candidates
          </h1>

          <p className="max-w-xl text-base leading-relaxed text-muted sm:text-lg">
            Centralize resume intake, ranking, and opportunity review in one
            focused command center built for consistent, high-quality outbound.
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

        <div className="rounded-3xl border border-border/90 bg-card/95 p-6 shadow-[0_24px_70px_rgba(0,0,0,0.5)] lg:col-span-5">
          <div className="space-y-4">
            <div className="rounded-2xl border border-border/90 bg-background p-4">
              <p className="text-xs font-semibold uppercase tracking-[0.14em] text-muted">
                Best match
              </p>
              <p className="mt-2 text-lg font-semibold text-foreground">
                Senior Backend Engineer
              </p>
              <p className="text-sm text-muted">
                93/100 score • Remote • ACME Labs
              </p>
            </div>
            <div className="rounded-2xl border border-border/90 bg-background p-4">
              <p className="text-xs font-semibold uppercase tracking-[0.14em] text-muted">
                Resume Pipeline
              </p>
              <p className="mt-2 text-lg font-semibold text-foreground">
                2 indexed, 1 in processing
              </p>
              <p className="text-sm text-muted">
                Upload, parse, and rank in minutes
              </p>
            </div>
            <div className="rounded-2xl border border-primary/30 bg-primary/10 p-4 text-foreground">
              <p className="text-xs font-semibold uppercase tracking-[0.14em] text-primary">
                Insight
              </p>
              <p className="mt-2 text-sm text-foreground/90">
                Skills relevance improved by 18% after profile enrichment.
              </p>
            </div>
          </div>
        </div>
      </section>
    </main>
  );
}
