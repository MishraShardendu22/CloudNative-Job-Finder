"use client";

import { UploadCloud } from "lucide-react";
import { useEffect, useMemo, useRef, useState } from "react";
import { Progress } from "@/components/ui/progress";

type ResumeUploadProps = {
  onUpload: (file: File) => Promise<unknown>;
  uploading: boolean;
};

export function ResumeUpload({ onUpload, uploading }: ResumeUploadProps) {
  const inputRef = useRef<HTMLInputElement | null>(null);
  const [dragging, setDragging] = useState(false);
  const [progress, setProgress] = useState(0);

  useEffect(() => {
    if (!uploading) {
      setProgress(0);
      return;
    }

    const timer = setInterval(() => {
      setProgress((prev) => {
        if (prev >= 93) return prev;
        return prev + 7;
      });
    }, 180);

    return () => clearInterval(timer);
  }, [uploading]);

  const borderClass = useMemo(() => {
    if (dragging) {
      return "border-primary bg-primary/5";
    }
    return "border";
  }, [dragging]);

  const handleFiles = async (files: FileList | null) => {
    const file = files?.[0];
    if (!file) return;

    try {
      await onUpload(file);
      setProgress(100);
    } finally {
      setTimeout(() => setProgress(0), 400);
    }
  };

  return (
    <div className="space-y-3">
      <input
        ref={inputRef}
        type="file"
        className="hidden"
        accept=".pdf,.doc,.docx"
        onChange={(event) => {
          void handleFiles(event.target.files);
        }}
      />

      <button
        type="button"
        className={`w-full rounded-2xl border-2 border-dashed p-8 text-center transition-colors duration-180 ${borderClass}`}
        disabled={uploading}
        onClick={() => inputRef.current?.click()}
        onDragOver={(event) => {
          event.preventDefault();
          setDragging(true);
        }}
        onDragLeave={() => setDragging(false)}
        onDrop={(event) => {
          event.preventDefault();
          setDragging(false);
          void handleFiles(event.dataTransfer.files);
        }}
      >
        <div className="mx-auto mb-3 flex h-11 w-11 items-center justify-center rounded-full border border-border bg-background text-muted">
          <UploadCloud className="h-5 w-5" />
        </div>

        <p className="text-sm text-muted">
          Drag and drop your resume, or click below to choose a file.
        </p>

        <span className="mt-4 inline-flex h-10 items-center rounded-lg border bg-card px-4 text-sm font-medium text-foreground">
          {uploading ? "Uploading..." : "Select Resume"}
        </span>
      </button>

      {uploading || progress > 0 ? <Progress value={progress} /> : null}
    </div>
  );
}
