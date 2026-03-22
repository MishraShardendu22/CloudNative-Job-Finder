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
      <div className="grid gap-6 lg:grid-cols-3">
        <Card className="lg:col-span-1">
          <CardHeader>
            <CardTitle>Upload Resume</CardTitle>
            <CardDescription>PDF and DOC formats are supported.</CardDescription>
          </CardHeader>
          <CardContent>
            <ResumeUpload
              uploading={uploadMutation.isPending}
              onUpload={async (file) => uploadMutation.mutateAsync(file)}
            />
          </CardContent>
        </Card>

        <Card className="lg:col-span-2">
          <CardHeader>
            <CardTitle>Resume Library</CardTitle>
            <CardDescription>Latest uploads and processing status.</CardDescription>
          </CardHeader>
          <CardContent>
            {resumesQuery.isLoading ? (
              <div className="space-y-3">
                <Skeleton className="h-14 w-full" />
                <Skeleton className="h-14 w-full" />
                <Skeleton className="h-14 w-full" />
              </div>
            ) : resumesQuery.isError ? (
              <div className="flex items-center gap-2 rounded-xl border border-red-200 bg-red-50 p-3 text-sm text-red-700 dark:border-red-900 dark:bg-red-950/40 dark:text-red-300">
                <AlertTriangle className="h-4 w-4" />
                Failed to load resumes. Please retry shortly.
              </div>
            ) : (resumesQuery.data?.length ?? 0) === 0 ? (
              <p className="text-sm text-zinc-500 dark:text-zinc-400">
                No resumes uploaded yet. Add your first resume to get started.
              </p>
            ) : (
              <div className="space-y-2">
                {resumesQuery.data?.map((resume) => (
                  <div
                    key={resume.id}
                    className="rounded-xl border border-zinc-200 bg-zinc-50/70 px-4 py-3 dark:border-zinc-800 dark:bg-zinc-900/50"
                  >
                    <div className="flex items-center justify-between gap-2">
                      <p className="font-medium text-zinc-900 dark:text-zinc-100">{resume.filename}</p>
                      <span className="rounded-full bg-zinc-200 px-2 py-1 text-xs text-zinc-700 dark:bg-zinc-800 dark:text-zinc-200">
                        {resume.status ?? "ready"}
                      </span>
                    </div>
                    <p className="mt-1 text-xs text-zinc-500 dark:text-zinc-400">
                      {resume.created_at ? new Date(resume.created_at).toLocaleString() : "No timestamp"}
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
