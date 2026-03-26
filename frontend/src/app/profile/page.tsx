"use client";

import { AlertTriangle } from "lucide-react";
import { useEffect, useState } from "react";
import { AppShell } from "@/components/app-shell";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import { useAuth } from "@/hooks/useAuth";

export default function ProfilePage() {
  const { profileQuery, updateProfileMutation } = useAuth();
  const [email, setEmail] = useState("");

  useEffect(() => {
    if (profileQuery.data?.email) {
      setEmail(profileQuery.data.email);
    }
  }, [profileQuery.data?.email]);

  const normalizedInitialEmail =
    profileQuery.data?.email?.trim().toLowerCase() ?? "";
  const normalizedDraftEmail = email.trim().toLowerCase();
  const hasPendingProfileChanges =
    normalizedDraftEmail.length > 0 &&
    normalizedDraftEmail !== normalizedInitialEmail;

  return (
    <AppShell
      title="Profile"
      description="Maintain account settings and keep your profile information up to date."
    >
      <Card className="max-w-3xl">
        <CardHeader>
          <CardTitle>Account Settings</CardTitle>
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
            <div className="space-y-6 text-sm">
              <div className="rounded-xl border border-border/80 bg-background px-4 py-3">
                <p className="text-xs font-medium uppercase tracking-wide text-muted">
                  User ID
                </p>
                <p className="mt-1 break-all text-foreground">
                  {profileQuery.data?.id ?? "Unavailable"}
                </p>
              </div>

              <form
                className="space-y-4 rounded-xl border border-border/80 bg-background px-4 py-4"
                onSubmit={async (event) => {
                  event.preventDefault();

                  try {
                    await updateProfileMutation.mutateAsync({
                      email: normalizedDraftEmail,
                    });
                  } catch {
                    // Errors are surfaced via mutation onError toast.
                  }
                }}
              >
                <div className="space-y-2">
                  <Label htmlFor="email">Account Email</Label>
                  <Input
                    id="email"
                    type="email"
                    value={email}
                    onChange={(event) => setEmail(event.target.value)}
                    placeholder="you@example.com"
                  />
                  <p className="text-xs text-muted">
                    Used for account access and notifications.
                  </p>
                </div>

                <div className="flex items-center justify-between gap-3">
                  <p className="text-xs text-muted">
                    {hasPendingProfileChanges
                      ? "You have unsaved changes"
                      : "Your profile is up to date"}
                  </p>

                  <div className="flex items-center gap-2">
                    <Button
                      type="button"
                      variant="outline"
                      disabled={
                        updateProfileMutation.isPending ||
                        !hasPendingProfileChanges
                      }
                      onClick={() => setEmail(profileQuery.data?.email ?? "")}
                    >
                      Reset
                    </Button>
                    <Button
                      type="submit"
                      disabled={
                        updateProfileMutation.isPending ||
                        !hasPendingProfileChanges
                      }
                    >
                      {updateProfileMutation.isPending
                        ? "Saving..."
                        : "Save Changes"}
                    </Button>
                  </div>
                </div>
              </form>
            </div>
          )}
        </CardContent>
      </Card>
    </AppShell>
  );
}
