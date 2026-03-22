"use client";

import { AlertTriangle } from "lucide-react";
import { AppShell } from "@/components/app-shell";
import { ResumeUpload } from "@/components/resume-upload";
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
  const { resumesQuery, uploadMutation } = useResumes();

  return (
    <AppShell
      title="Resumes"
      description="Upload resumes, track metadata, and prep for recommendations."
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
              onUpload={async (file) => uploadMutation.mutateAsync(file)}
            />
          </CardContent>
        </Card>

        <Card className="xl:col-span-8">
          <CardHeader>
            <CardTitle>Resume Library</CardTitle>
            <CardDescription>
              Latest uploads and processing status.
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
                {resumesQuery.data?.map((resume) => (
                  <div
                    key={resume.id}
                    className="rounded-lg border bg-background px-4 py-3"
                  >
                    <div className="flex items-center justify-between gap-2">
                      <p className="font-medium text-foreground">
                        {resume.filename}
                      </p>
                      <span className="rounded-full bg-primary/10 px-2 py-1 text-xs text-primary">
                        {resume.status ?? "ready"}
                      </span>
                    </div>
                    <p className="mt-1 text-xs text-muted">
                      {resume.created_at
                        ? new Date(resume.created_at).toLocaleString()
                        : "No timestamp"}
                    </p>
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
