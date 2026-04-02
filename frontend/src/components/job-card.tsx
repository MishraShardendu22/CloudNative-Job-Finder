"use client";

import { ExternalLink, MapPin } from "lucide-react";
import Link from "next/link";
import { useEffect, useRef } from "react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { api } from "@/lib/api";
import type { JobInteractionType, JobRecommendation } from "@/types/job";

type JobCardProps = {
  job: JobRecommendation;
  resumeId: string | null;
};

export function JobCard({ job, resumeId }: JobCardProps) {
  const score = Math.round(job.score);
  const cardRef = useRef<HTMLDivElement | null>(null);
  const hasTrackedImpression = useRef(false);

  const trackInteraction = (interactionType: JobInteractionType) => {
    if (!resumeId) return;

    void api.trackInteraction({
      resume_id: resumeId,
      job_id: job.id,
      interaction_type: interactionType,
      source: "job-card",
    });
  };

  useEffect(() => {
    if (!resumeId || hasTrackedImpression.current || !cardRef.current) return;

    const observer = new IntersectionObserver(
      (entries) => {
        for (const entry of entries) {
          if (!entry.isIntersecting || hasTrackedImpression.current) continue;
          hasTrackedImpression.current = true;
          trackInteraction("impression");
          observer.disconnect();
          break;
        }
      },
      { threshold: 0.6 },
    );

    observer.observe(cardRef.current);
    return () => observer.disconnect();
  }, [job.id, resumeId]);

  return (
    <Card
      ref={cardRef}
      onClick={() => trackInteraction("click")}
      className="group h-full border-border/90 bg-card/95 transition-all duration-180 hover:border-primary/35 hover:shadow-[0_20px_45px_rgba(0,0,0,0.45)]"
    >
      <CardHeader className="space-y-3 pb-3">
        <div className="flex items-start justify-between gap-3">
          <CardTitle className="line-clamp-3 text-[22px] leading-tight">
            {job.title}
          </CardTitle>
          <div className="shrink-0 rounded-xl border border-primary/35 bg-primary/8 px-3 py-2 text-right">
            <p className="text-[10px] font-semibold uppercase tracking-[0.14em] text-primary">
              Match Score
            </p>
            <p className="text-[30px] font-bold leading-none text-primary">
              {score}
            </p>
            <p className="mt-1 text-[10px] uppercase tracking-[0.12em] text-muted">
              out of 100
            </p>
          </div>
        </div>
        <p className="text-sm font-medium text-muted">{job.company}</p>
      </CardHeader>
      <CardContent className="flex flex-1 flex-col gap-4">
        <p className="flex items-center gap-1 text-sm text-muted">
          <MapPin className="h-4 w-4" />
          {job.location}
        </p>
        {job.summary ? (
          <p className="line-clamp-3 text-sm leading-relaxed text-foreground/85">
            {job.summary}
          </p>
        ) : null}
        <Button
          asChild
          size="sm"
          className="mt-auto w-fit group-hover:translate-y-[-1px]"
        >
          <Link
            href={job.apply_url}
            target="_blank"
            rel="noreferrer"
            onClick={(event) => {
              event.stopPropagation();
              trackInteraction("apply");
            }}
          >
            Apply
            <ExternalLink className="h-4 w-4" />
          </Link>
        </Button>
      </CardContent>
    </Card>
  );
}
