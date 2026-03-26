"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { ApiError, api } from "@/lib/api";

export function useResumes() {
  const queryClient = useQueryClient();

  const resumesQuery = useQuery({
    queryKey: ["resumes"],
    queryFn: api.resumes,
  });

  const uploadMutation = useMutation({
    mutationFn: api.uploadResume,
    onSuccess: () => {
      toast.success("Resume uploaded successfully");
      queryClient.invalidateQueries({ queryKey: ["resumes"] });
    },
    onError: (error) => {
      toast.error(error.message || "Resume upload failed");
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (resumeId: string) => api.deleteResume(resumeId),
    onSuccess: () => {
      toast.success("Resume removed");
      queryClient.invalidateQueries({ queryKey: ["resumes"] });
      queryClient.invalidateQueries({ queryKey: ["recommendations"] });
    },
    onError: (error) => {
      if (
        error instanceof ApiError &&
        (error.status === 404 || error.status === 405)
      ) {
        toast.error(
          "Resume delete API is unavailable. Restart backend services.",
        );
        return;
      }
      toast.error(error.message || "Unable to remove resume");
    },
  });

  return {
    resumesQuery,
    uploadMutation,
    deleteMutation,
  };
}
