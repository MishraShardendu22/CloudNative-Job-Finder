"use client";

import Link from "next/link";
import { AuthForm } from "@/components/auth-form";
import { useAuth } from "@/hooks/useAuth";

export default function SignupPage() {
  const { signupMutation } = useAuth();

  return (
    <main className="relative flex min-h-screen items-center justify-center overflow-hidden bg-background px-4 py-12">
      <div className="pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_20%_0%,rgba(249,115,22,0.15),transparent_35%)]" />
      <div className="relative w-full max-w-md space-y-4">
        <AuthForm
          mode="signup"
          onSubmit={async (values) => {
            try {
              await signupMutation.mutateAsync(values);
            } catch {
              // Errors are surfaced via mutation onError toast.
            }
          }}
        />
        <p className="text-center text-sm text-muted">
          Already have an account?{" "}
          <Link
            href="/login"
            className="font-medium text-primary hover:text-primary-hover"
          >
            Login
          </Link>
        </p>
      </div>
    </main>
  );
}
