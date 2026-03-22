"use client";

import { AlertTriangle } from "lucide-react";
import { AppShell } from "@/components/app-shell";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { useAuth } from "@/hooks/useAuth";

export default function ProfilePage() {
  const { profileQuery } = useAuth();

  return (
    <AppShell
      title="Profile"
      description="Manage account details used by your recommendation engine."
    >
      <Card className="max-w-2xl">
        <CardHeader>
          <CardTitle>Your account</CardTitle>
        </CardHeader>
        <CardContent>
          {profileQuery.isLoading ? (
            <div className="space-y-3">
              <Skeleton className="h-6 w-52" />
              <Skeleton className="h-6 w-64" />
            </div>
          ) : profileQuery.isError ? (
            <div className="flex items-center gap-2 text-sm text-danger">
              <AlertTriangle className="h-4 w-4" />
              Could not fetch profile.
            </div>
          ) : (
            <div className="space-y-4 text-sm">
              <div className="space-y-1">
                <p className="text-xs font-medium uppercase tracking-wide text-muted">
                  Name
                </p>
                <p className="text-foreground">
                  {profileQuery.data?.name ?? "No display name set"}
                </p>
              </div>
              <div className="space-y-1">
                <p className="text-xs font-medium uppercase tracking-wide text-muted">
                  Email
                </p>
                <p className="text-foreground">
                  {profileQuery.data?.email ?? "No email found"}
                </p>
              </div>
            </div>
          )}
        </CardContent>
      </Card>
    </AppShell>
  );
}
