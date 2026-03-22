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
    { title: "Resumes", value: totalResumes },
    { title: "Ready for matching", value: readyResumes },
    { title: "Job matches", value: totalRecommendations },
  ];

  return (
    <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
      {stats.map((stat) => {
        return (
          <Card key={stat.title}>
            <CardHeader>
              <CardTitle className="text-sm font-medium text-muted">
                {stat.title}
              </CardTitle>
            </CardHeader>
            <CardContent>
              <p className="text-[32px] font-semibold tracking-tight text-foreground">
                {stat.value}
              </p>
            </CardContent>
          </Card>
        );
      })}
    </div>
  );
}
