"use client";

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
      description="Fetch and evaluate role matches for each resume."
    >
      <div className="space-y-6">
        <Card>
          <CardHeader>
            <CardTitle>Select Resume</CardTitle>
            <CardDescription>Choose which resume to analyze.</CardDescription>
          </CardHeader>
          <CardContent className="max-w-md space-y-2">
            <Label htmlFor="resume-select">Resume</Label>
            <select
              id="resume-select"
              className="h-10 w-full rounded-xl border border-zinc-300 bg-white px-3 text-sm text-zinc-900 dark:border-zinc-700 dark:bg-zinc-900 dark:text-zinc-100"
              value={selectedResumeId ?? ""}
              onChange={(event) => setResumeId(event.target.value || null)}
            >
              {(resumesQuery.data?.length ?? 0) === 0 ? (
                <option value="" className="text-zinc-900 dark:text-zinc-100">
                  No resumes available
                </option>
              ) : null}
              {(resumesQuery.data ?? []).map((resume) => (
                <option key={resume.id} value={resume.id} className="text-zinc-900 dark:text-zinc-100">
                  {resume.filename || `Resume ${resume.id.slice(0, 8)}`}
                </option>
              ))}
            </select>
          </CardContent>
        </Card>

        {recommendationsQuery.isLoading ? (
          <div className="space-y-3">
            <Skeleton className="h-28 w-full" />
            <Skeleton className="h-28 w-full" />
          </div>
        ) : recommendationsQuery.isError ? (
          <Card>
            <CardContent className="pt-6 text-sm text-red-600 dark:text-red-300">
              Failed to fetch recommendations. Try again in a moment.
            </CardContent>
          </Card>
        ) : jobs.length === 0 ? (
          <Card>
            <CardContent className="pt-6 text-sm text-zinc-500 dark:text-zinc-400">
              No recommendations available for this resume yet.
            </CardContent>
          </Card>
        ) : (
          <>
            <JobTable data={jobs} />
            <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
              {jobs.slice(0, 6).map((job) => (
                <JobCard key={job.id} job={job} />
              ))}
            </div>
          </>
        )}
      </div>
    </AppShell>
  );
}
