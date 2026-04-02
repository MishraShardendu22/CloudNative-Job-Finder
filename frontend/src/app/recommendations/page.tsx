"use client";

import { ChevronDown, FileText } from "lucide-react";
import { useMemo, useState } from "react";
import { AppShell } from "@/components/app-shell";
import { JobCard } from "@/components/job-card";
import { JobTable } from "@/components/job-table";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import { useRecommendations } from "@/hooks/useRecommendations";
import { useResumes } from "@/hooks/useResumes";

export default function RecommendationsPage() {
  const { resumesQuery } = useResumes();
  const [resumeId, setResumeId] = useState<string | null>(null);

  const selectedResumeId = useMemo(() => {
    return resumeId ?? resumesQuery.data?.[0]?.id ?? null;
  }, [resumeId, resumesQuery.data]);

  const recommendationsQuery = useRecommendations(selectedResumeId);
  const jobs = recommendationsQuery.data?.jobs ?? [];

  return (
    <AppShell
      title="Recommendations"
      description="Evaluate ranked opportunities for each resume with standardized match scoring."
    >
      <div className="space-y-6">
        <Card className="relative overflow-hidden">
          <div className="pointer-events-none absolute -right-16 -top-16 h-36 w-36 rounded-full bg-primary/10 blur-3xl" />
          <CardHeader className="pb-3">
            <CardTitle className="flex items-center gap-2">
              <FileText className="h-5 w-5 text-primary" />
              Select Resume
            </CardTitle>
            <CardDescription>
              Choose which resume to analyze and rank against active roles.
            </CardDescription>
          </CardHeader>
          <CardContent className="max-w-lg space-y-2">
            <Label
              htmlFor="resume-select"
              className="text-xs uppercase tracking-[0.14em] text-muted"
            >
              Resume
            </Label>
            <div className="relative">
              <select
                id="resume-select"
                className="h-12 w-full appearance-none rounded-xl border border-border bg-background pl-3 pr-10 text-sm font-medium text-foreground outline-none transition-colors duration-180 focus-visible:border-primary focus-visible:ring-2 focus-visible:ring-primary/20"
                value={selectedResumeId ?? ""}
                onChange={(event) => setResumeId(event.target.value || null)}
              >
                {(resumesQuery.data?.length ?? 0) === 0 ? (
                  <option value="" className="text-foreground">
                    No resumes available
                  </option>
                ) : null}
                {(resumesQuery.data ?? []).map((resume) => (
                  <option
                    key={resume.id}
                    value={resume.id}
                    className="text-foreground"
                  >
                    {resume.filename || `Resume ${resume.id.slice(0, 8)}`}
                  </option>
                ))}
              </select>
              <ChevronDown className="pointer-events-none absolute right-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted" />
            </div>
            <p className="text-xs text-muted">
              {jobs.length > 0
                ? `${jobs.length} ranked roles available for this resume.`
                : "No ranked roles yet for this resume."}
            </p>
          </CardContent>
        </Card>

        {recommendationsQuery.isLoading ? (
          <div className="space-y-3">
            <Skeleton className="h-28 w-full" />
            <Skeleton className="h-28 w-full" />
          </div>
        ) : recommendationsQuery.isError ? (
          <Card>
            <CardContent className="pt-6 text-sm text-danger">
              Failed to fetch recommendations. Try again in a moment.
            </CardContent>
          </Card>
        ) : jobs.length === 0 ? (
          <Card>
            <CardContent className="pt-6 text-sm text-muted">
              No recommendations available for this resume yet.
            </CardContent>
          </Card>
        ) : (
          <>
            <Card>
              <CardHeader className="pb-3">
                <CardTitle className="text-lg">Ranked Opportunities</CardTitle>
                <CardDescription>
                  Sorted recommendations with normalized match scores.
                </CardDescription>
              </CardHeader>
              <CardContent>
                <JobTable data={jobs} resumeId={selectedResumeId} />
              </CardContent>
            </Card>

            <div className="space-y-3">
              <p className="text-xs font-semibold uppercase tracking-[0.14em] text-muted">
                Top Matches
              </p>
              <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
                {jobs.slice(0, 6).map((job) => (
                  <JobCard
                    key={job.id}
                    job={job}
                    resumeId={selectedResumeId}
                  />
                ))}
              </div>
            </div>
          </>
        )}
      </div>
    </AppShell>
  );
}
