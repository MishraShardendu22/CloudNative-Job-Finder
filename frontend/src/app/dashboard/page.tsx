"use client";

import {
  Bar,
  BarChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";
import { AppShell } from "@/components/app-shell";
import { DashboardStats } from "@/components/dashboard-stats";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { useRecommendations } from "@/hooks/useRecommendations";
import { useResumes } from "@/hooks/useResumes";

export default function DashboardPage() {
  const { resumesQuery } = useResumes();
  const latestResume = resumesQuery.data?.[0] ?? null;
  const recommendationsQuery = useRecommendations(latestResume?.id ?? null);

  const recommendations = recommendationsQuery.data?.jobs ?? [];

  const chartData = recommendations.slice(0, 6).map((job) => ({
    name: job.company,
    score: Math.round(job.score),
  }));

  return (
    <AppShell
      title="Dashboard"
      description="Monitor intake velocity, recommendation quality, and next-best actions in one view."
    >
      {resumesQuery.isLoading ? (
        <div className="space-y-4">
          <Skeleton className="h-24 w-full" />
          <Skeleton className="h-64 w-full" />
        </div>
      ) : (
        <div className="space-y-6">
          <DashboardStats
            totalResumes={resumesQuery.data?.length ?? 0}
            readyResumes={
              (resumesQuery.data ?? []).filter(
                (resume) => resume.status === "ready",
              ).length
            }
            totalRecommendations={recommendations.length}
          />

          <div className="grid grid-cols-1 gap-6 xl:grid-cols-12">
            <Card className="xl:col-span-8">
              <CardHeader>
                <CardTitle>Match Quality Snapshot</CardTitle>
                <CardDescription>
                  Top recommendation scores by company, normalized out of 100.
                </CardDescription>
              </CardHeader>
              <CardContent className="h-72">
                {chartData.length === 0 ? (
                  <p className="text-sm text-muted">
                    Upload a resume to see recommendation analytics.
                  </p>
                ) : (
                  <ResponsiveContainer width="100%" height="100%">
                    <BarChart data={chartData}>
                      <XAxis
                        dataKey="name"
                        stroke="currentColor"
                        className="text-muted"
                      />
                      <YAxis
                        domain={[0, 100]}
                        stroke="currentColor"
                        className="text-muted"
                      />
                      <Tooltip />
                      <Bar dataKey="score" fill="var(--primary)" radius={8} />
                    </BarChart>
                  </ResponsiveContainer>
                )}
              </CardContent>
            </Card>

            <Card className="xl:col-span-4">
              <CardHeader>
                <CardTitle>Execution Checklist</CardTitle>
                <CardDescription>
                  Keep your recommendation quality high with a repeatable loop.
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-2 text-sm text-muted">
                <p>1. Upload a fresh resume in Resume Library.</p>
                <p>2. Validate top scoring roles in Recommendations.</p>
                <p>3. Refine profile details to improve ranking quality.</p>
              </CardContent>
            </Card>
          </div>
        </div>
      )}
    </AppShell>
  );
}
