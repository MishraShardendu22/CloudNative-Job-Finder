import { BriefcaseBusiness, FileCheck2, Sparkles } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

type DashboardStatsProps = {
  totalResumes: number;
  readyResumes: number;
  totalRecommendations: number;
};

export function DashboardStats({
  totalResumes,
  readyResumes,
  totalRecommendations,
}: DashboardStatsProps) {
  const stats = [
    { title: "Resumes", value: totalResumes, icon: FileCheck2 },
    { title: "Ready for matching", value: readyResumes, icon: Sparkles },
    { title: "Job matches", value: totalRecommendations, icon: BriefcaseBusiness },
  ];

  return (
    <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
      {stats.map((stat) => {
        const Icon = stat.icon;
        return (
          <Card key={stat.title}>
            <CardHeader className="flex-row items-center justify-between space-y-0">
              <CardTitle className="text-sm font-medium text-zinc-500 dark:text-zinc-400">
                {stat.title}
              </CardTitle>
              <Icon className="h-4 w-4 text-zinc-500 dark:text-zinc-400" />
            </CardHeader>
            <CardContent>
              <p className="text-3xl font-semibold tracking-tight">{stat.value}</p>
            </CardContent>
          </Card>
        );
      })}
    </div>
  );
}
