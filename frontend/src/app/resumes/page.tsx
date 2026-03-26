"use client";

import { AlertTriangle, Trash2 } from "lucide-react";
import { AppShell } from "@/components/app-shell";
import { ResumeUpload } from "@/components/resume-upload";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { useResumes } from "@/hooks/useResumes";

export default function ResumesPage() {
  const { resumesQuery, uploadMutation, deleteMutation } = useResumes();

  return (
    <AppShell
      title="Resumes"
      description="Manage resume intake, processing status, and readiness for matching workflows."
    >
      <div className="grid grid-cols-1 gap-6 xl:grid-cols-12">
        <Card className="xl:col-span-4">
          <CardHeader>
            <CardTitle>Upload Resume</CardTitle>
            <CardDescription>
              PDF and DOC formats are supported.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <ResumeUpload
              uploading={uploadMutation.isPending}
              onUpload={async (file) => {
                try {
                  await uploadMutation.mutateAsync(file);
                } catch {
                  // Errors are surfaced via mutation onError toast.
                }
              }}
            />
          </CardContent>
        </Card>

        <Card className="xl:col-span-8">
          <CardHeader>
            <CardTitle>Resume Library</CardTitle>
            <CardDescription>
              Latest submissions and parsing status.
            </CardDescription>
          </CardHeader>
          <CardContent>
            {resumesQuery.isLoading ? (
              <div className="space-y-3">
                <Skeleton className="h-14 w-full" />
                <Skeleton className="h-14 w-full" />
                <Skeleton className="h-14 w-full" />
              </div>
            ) : resumesQuery.isError ? (
              <div className="flex items-center gap-2 rounded-lg border border-danger/20 bg-danger/10 p-3 text-sm text-danger">
                <AlertTriangle className="h-4 w-4" />
                Failed to load resumes. Please retry shortly.
              </div>
            ) : (resumesQuery.data?.length ?? 0) === 0 ? (
              <p className="text-sm text-muted">
                No resumes uploaded yet. Add your first resume to get started.
              </p>
            ) : (
              <div className="space-y-2">
                <div className="hidden grid-cols-[1.7fr_0.7fr_0.9fr_0.35fr] gap-3 px-3 py-1 text-[11px] font-semibold uppercase tracking-[0.12em] text-muted md:grid">
                  <p>Resume</p>
                  <p>Status</p>
                  <p>Uploaded</p>
                  <p className="text-right">Actions</p>
                </div>

                {resumesQuery.data?.map((resume) => (
                  <div
                    key={resume.id}
                    className="rounded-xl border border-border/80 bg-background px-4 py-3"
                  >
                    <div className="grid gap-3 md:grid-cols-[1.7fr_0.7fr_0.9fr_0.35fr] md:items-center">
                      <p className="break-words font-medium text-foreground">
                        {resume.filename}
                      </p>

                      <div className="md:justify-self-start">
                        <span className="rounded-full bg-primary/10 px-2 py-1 text-xs text-primary">
                          {resume.status ?? "ready"}
                        </span>
                      </div>

                      <p className="text-xs text-muted md:text-sm">
                        {resume.created_at
                          ? new Date(resume.created_at).toLocaleString()
                          : "No timestamp"}
                      </p>

                      <div className="flex justify-end">
                        <Button
                          type="button"
                          size="icon"
                          variant="ghost"
                          aria-label={`Remove ${resume.filename}`}
                          disabled={deleteMutation.isPending}
                          onClick={() => {
                            const shouldDelete = window.confirm(
                              "Remove this resume from your library?",
                            );

                            if (!shouldDelete) return;

                            deleteMutation.mutate(resume.id);
                          }}
                        >
                          <Trash2 className="h-4 w-4 text-danger" />
                        </Button>
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </CardContent>
        </Card>
      </div>
    </AppShell>
  );
}
