"use client";

import Link from "next/link";
import { AuthForm } from "@/components/auth-form";
import { useAuth } from "@/hooks/useAuth";

export default function LoginPage() {
  const { loginMutation } = useAuth();

  return (
    <main className="relative flex min-h-screen items-center justify-center overflow-hidden bg-background px-4 py-12">
      <div className="pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_80%_0%,rgba(249,115,22,0.15),transparent_35%)]" />
      <div className="relative w-full max-w-md space-y-4">
        <AuthForm
          mode="login"
          onSubmit={async (values) => {
            try {
              await loginMutation.mutateAsync(values);
            } catch {
              // Errors are surfaced via mutation onError toast.
            }
          }}
        />
        <p className="text-center text-sm text-muted">
          No account yet?{" "}
          <Link
            href="/signup"
            className="font-medium text-primary hover:text-primary-hover"
          >
            Signup
          </Link>
        </p>
      </div>
    </main>
  );
}
