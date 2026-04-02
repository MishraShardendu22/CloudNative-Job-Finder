"use client";

import {
  type ColumnDef,
  flexRender,
  getCoreRowModel,
  getPaginationRowModel,
  useReactTable,
} from "@tanstack/react-table";
import { ExternalLink } from "lucide-react";
import Link from "next/link";
import { useEffect, useMemo, useRef } from "react";
import { Button } from "@/components/ui/button";
import { api } from "@/lib/api";
import type { JobInteractionType, JobRecommendation } from "@/types/job";

type JobTableProps = {
  data: JobRecommendation[];
  resumeId: string | null;
};

export function JobTable({ data, resumeId }: JobTableProps) {
  const trackedImpressions = useRef<Set<string>>(new Set());

  const trackInteraction = (
    jobId: string,
    interactionType: JobInteractionType,
  ) => {
    if (!resumeId) return;

    void api.trackInteraction({
      resume_id: resumeId,
      job_id: jobId,
      interaction_type: interactionType,
      source: "job-table",
    });
  };

  useEffect(() => {
    if (!resumeId) return;

    for (const job of data) {
      if (trackedImpressions.current.has(job.id)) continue;
      trackedImpressions.current.add(job.id);
      trackInteraction(job.id, "impression");
    }
  }, [data, resumeId]);

  const columns = useMemo<ColumnDef<JobRecommendation>[]>(
    () => [
      {
        accessorKey: "title",
        header: "Job Title",
      },
      {
        accessorKey: "company",
        header: "Company",
      },
      {
        accessorKey: "location",
        header: "Location",
      },
      {
        accessorKey: "score",
        header: "Match Score",
        cell: ({ row }) => `${Math.round(row.original.score)}/100`,
      },
      {
        accessorKey: "apply_url",
        header: "Apply",
        cell: ({ row }) => (
          <Link
            href={row.original.apply_url}
            target="_blank"
            rel="noreferrer"
            className="inline-flex items-center gap-1 text-sm text-primary hover:text-primary-hover"
            onClick={(event) => {
              event.stopPropagation();
              trackInteraction(row.original.id, "apply");
            }}
          >
            Link
            <ExternalLink className="h-3 w-3" />
          </Link>
        ),
      },
    ],
    [resumeId],
  );

  const table = useReactTable({
    data,
    columns,
    getCoreRowModel: getCoreRowModel(),
    getPaginationRowModel: getPaginationRowModel(),
  });

  return (
    <div className="space-y-4">
      <div className="overflow-hidden rounded-2xl border border-border/80 bg-background/70">
        <div className="overflow-x-auto">
          <table className="w-full min-w-[760px] border-collapse text-sm">
            <thead className="bg-card/80 text-left">
              {table.getHeaderGroups().map((headerGroup) => (
                <tr key={headerGroup.id}>
                  {headerGroup.headers.map((header) => (
                    <th
                      key={header.id}
                      className="px-5 py-3 text-xs font-semibold uppercase tracking-[0.12em] text-muted"
                    >
                      {header.isPlaceholder
                        ? null
                        : flexRender(
                            header.column.columnDef.header,
                            header.getContext(),
                          )}
                    </th>
                  ))}
                </tr>
              ))}
            </thead>
            <tbody>
              {table.getRowModel().rows.length === 0 ? (
                <tr>
                  <td
                    colSpan={columns.length}
                    className="px-4 py-8 text-center text-muted"
                  >
                    No recommendations found.
                  </td>
                </tr>
              ) : (
                table.getRowModel().rows.map((row) => (
                  <tr
                    key={row.id}
                    onClick={() => trackInteraction(row.original.id, "click")}
                    className="border-t border-border/80 transition-colors duration-180 hover:bg-card/70"
                  >
                    {row.getVisibleCells().map((cell) => (
                      <td
                        key={cell.id}
                        className="px-5 py-3 text-foreground/90"
                      >
                        {flexRender(
                          cell.column.columnDef.cell,
                          cell.getContext(),
                        )}
                      </td>
                    ))}
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </div>

      <div className="flex items-center justify-end gap-2">
        <Button
          variant="outline"
          size="sm"
          onClick={() => table.previousPage()}
          disabled={!table.getCanPreviousPage()}
        >
          Previous
        </Button>
        <Button
          variant="outline"
          size="sm"
          onClick={() => table.nextPage()}
          disabled={!table.getCanNextPage()}
        >
          Next
        </Button>
      </div>
    </div>
  );
}
