import { ExternalLink, MapPin } from "lucide-react";
import Link from "next/link";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import type { JobRecommendation } from "@/types/job";

export function JobCard({ job }: { job: JobRecommendation }) {
  const score = Math.round(job.score);

  return (
    <Card className="group">
      <CardHeader className="space-y-2">
        <div className="flex items-center justify-between gap-4">
          <CardTitle className="text-base leading-snug">{job.title}</CardTitle>
          <div className="rounded-lg border border-primary/40 bg-primary/10 px-2.5 py-1.5 text-right">
            <p className="text-[10px] font-semibold uppercase tracking-[0.12em] text-primary">
              Match Score
            </p>
            <p className="text-sm font-bold text-primary">{score}/100</p>
          </div>
        </div>
        <p className="text-sm text-muted">{job.company}</p>
      </CardHeader>
      <CardContent className="space-y-4">
        <p className="flex items-center gap-1 text-sm text-muted">
          <MapPin className="h-4 w-4" />
          {job.location}
        </p>
        {job.summary ? (
          <p className="text-sm leading-relaxed text-foreground/85">
            {job.summary}
          </p>
        ) : null}
        <Button asChild size="sm" className="group-hover:translate-y-[-1px]">
          <Link href={job.apply_url} target="_blank" rel="noreferrer">
            Apply
            <ExternalLink className="h-4 w-4" />
          </Link>
        </Button>
      </CardContent>
    </Card>
  );
}
