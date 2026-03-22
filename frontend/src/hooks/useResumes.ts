"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { api } from "@/lib/api";

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

  return {
    resumesQuery,
    uploadMutation,
  };
}
