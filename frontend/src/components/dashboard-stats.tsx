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
    { title: "Resume Submissions", value: totalResumes },
    { title: "Ready for matching", value: readyResumes },
    { title: "Job matches", value: totalRecommendations },
  ];

  return (
    <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
      {stats.map((stat) => {
        return (
          <Card key={stat.title} className="relative overflow-hidden">
            <div className="absolute -right-8 -top-8 h-20 w-20 rounded-full bg-primary/10 blur-2xl" />
            <CardHeader className="pb-3">
              <CardTitle className="text-xs font-semibold uppercase tracking-[0.14em] text-muted">
                {stat.title}
              </CardTitle>
            </CardHeader>
            <CardContent>
              <p className="text-[38px] font-bold tracking-tight text-foreground">
                {stat.value}
              </p>
            </CardContent>
          </Card>
        );
      })}
    </div>
  );
}
