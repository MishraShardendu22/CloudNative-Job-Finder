"use client";

import { UploadCloud } from "lucide-react";
import { useEffect, useMemo, useRef, useState } from "react";
import { Button } from "@/components/ui/button";
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
      return "border-zinc-900 bg-zinc-50 dark:border-zinc-100 dark:bg-zinc-900";
    }
    return "border-zinc-300 dark:border-zinc-700";
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
      <div
        className={`rounded-2xl border-2 border-dashed p-8 text-center transition ${borderClass}`}
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
        <input
          ref={inputRef}
          type="file"
          className="hidden"
          accept=".pdf,.doc,.docx"
          onChange={(event) => {
            void handleFiles(event.target.files);
          }}
        />

        <div className="mx-auto mb-3 flex h-11 w-11 items-center justify-center rounded-full bg-zinc-100 dark:bg-zinc-800">
          <UploadCloud className="h-5 w-5" />
        </div>

        <p className="text-sm text-zinc-700 dark:text-zinc-300">
          Drag and drop your resume, or click below to choose a file.
        </p>

        <Button
          className="mt-4"
          variant="secondary"
          disabled={uploading}
          onClick={() => inputRef.current?.click()}
        >
          {uploading ? "Uploading..." : "Select Resume"}
        </Button>
      </div>

      {uploading || progress > 0 ? <Progress value={progress} /> : null}
    </div>
  );
}
