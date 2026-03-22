"use client";

import Link from "next/link";
import { AuthForm } from "@/components/auth-form";
import { useAuth } from "@/hooks/useAuth";

export default function SignupPage() {
  const { signupMutation } = useAuth();

  return (
    <main className="flex min-h-screen items-center justify-center bg-background px-4 py-12">
      <div className="w-full max-w-md space-y-4">
        <AuthForm
          mode="signup"
          onSubmit={async (values) => signupMutation.mutateAsync(values)}
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
