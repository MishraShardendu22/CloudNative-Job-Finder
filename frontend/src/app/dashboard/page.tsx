"use client";

import { Bar, BarChart, ResponsiveContainer, Tooltip, XAxis, YAxis } from "recharts";
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
    score: Math.round(job.score * 100),
  }));

  return (
    <AppShell
      title="Dashboard"
      description="Track resumes, recommendation quality, and quick actions."
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
            readyResumes={(resumesQuery.data ?? []).filter((resume) => resume.status === "ready").length}
            totalRecommendations={recommendations.length}
          />

          <div className="grid gap-6 lg:grid-cols-3">
            <Card className="lg:col-span-2">
              <CardHeader>
                <CardTitle>Match quality snapshot</CardTitle>
                <CardDescription>Top recommendation scores by company.</CardDescription>
              </CardHeader>
              <CardContent className="h-72">
                {chartData.length === 0 ? (
                  <p className="text-sm text-zinc-500 dark:text-zinc-400">
                    Upload a resume to see recommendation analytics.
                  </p>
                ) : (
                  <ResponsiveContainer width="100%" height="100%">
                    <BarChart data={chartData}>
                      <XAxis dataKey="name" stroke="currentColor" />
                      <YAxis stroke="currentColor" />
                      <Tooltip />
                      <Bar dataKey="score" fill="currentColor" radius={8} />
                    </BarChart>
                  </ResponsiveContainer>
                )}
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle>Quick actions</CardTitle>
                <CardDescription>Move faster through your job search flow.</CardDescription>
              </CardHeader>
              <CardContent className="space-y-2 text-sm text-zinc-600 dark:text-zinc-300">
                <p>1. Upload a fresh resume on the resumes page.</p>
                <p>2. Open recommendations to review best-fit roles.</p>
                <p>3. Update your profile to improve relevance.</p>
              </CardContent>
            </Card>
          </div>
        </div>
      )}
    </AppShell>
  );
}
