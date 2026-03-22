"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useRouter } from "next/navigation";
import { toast } from "sonner";
import { api } from "@/lib/api";
import type { AuthPayload } from "@/types/user";

function persistTokenCookie(token: string) {
  if (typeof document === "undefined") return;

  const ttlSeconds = 60 * 60 * 24;
  document.cookie = `jwt=${encodeURIComponent(token)}; Path=/; Max-Age=${ttlSeconds}; SameSite=Lax`;
}

export function useAuth() {
  const router = useRouter();
  const queryClient = useQueryClient();

  const profileQuery = useQuery({
    queryKey: ["profile"],
    queryFn: api.profile,
    retry: false,
  });

  const loginMutation = useMutation({
    mutationFn: (payload: AuthPayload) => api.login(payload),
    onSuccess: (data) => {
      if (data.token) {
        persistTokenCookie(data.token);
      }

      toast.success("Logged in successfully");
      queryClient.invalidateQueries({ queryKey: ["profile"] });
      router.refresh();
      router.push("/dashboard");
    },
    onError: (error) => {
      toast.error(error.message || "Unable to login");
    },
  });

  const signupMutation = useMutation({
    mutationFn: (payload: AuthPayload) => api.signup(payload),
    onSuccess: () => {
      toast.success("Signup successful. You can now log in.");
      router.push("/login");
    },
    onError: (error) => {
      toast.error(error.message || "Unable to signup");
    },
  });

  return {
    profileQuery,
    loginMutation,
    signupMutation,
  };
}
