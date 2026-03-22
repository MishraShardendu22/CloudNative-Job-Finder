"use client";

import { useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api";

export function useRecommendations(resumeId: string | null) {
  return useQuery({
    queryKey: ["recommendations", resumeId],
    queryFn: () => api.recommendations(resumeId as string),
    enabled: Boolean(resumeId),
  });
}
