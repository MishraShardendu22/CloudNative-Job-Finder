import { ExternalLink, MapPin } from "lucide-react";
import Link from "next/link";
import type { JobRecommendation } from "@/types/job";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

export function JobCard({ job }: { job: JobRecommendation }) {
  return (
    <Card>
      <CardHeader className="space-y-2">
        <div className="flex items-center justify-between gap-4">
          <CardTitle className="text-base">{job.title}</CardTitle>
          <span className="rounded-full bg-zinc-100 px-2 py-1 text-xs font-medium dark:bg-zinc-800">
            {(job.score * 100).toFixed(0)}% match
          </span>
        </div>
        <p className="text-sm text-zinc-500 dark:text-zinc-400">{job.company}</p>
      </CardHeader>
      <CardContent className="space-y-4">
        <p className="flex items-center gap-1 text-sm text-zinc-500 dark:text-zinc-400">
          <MapPin className="h-4 w-4" />
          {job.location}
        </p>
        {job.summary ? <p className="text-sm text-zinc-600 dark:text-zinc-300">{job.summary}</p> : null}
        <Button asChild size="sm">
          <Link href={job.apply_url} target="_blank" rel="noreferrer">
            Apply
            <ExternalLink className="h-4 w-4" />
          </Link>
        </Button>
      </CardContent>
    </Card>
  );
}
