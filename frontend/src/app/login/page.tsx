"use client";

import Link from "next/link";
import { AuthForm } from "@/components/auth-form";
import { useAuth } from "@/hooks/useAuth";

export default function LoginPage() {
  const { loginMutation } = useAuth();

  return (
    <main className="flex min-h-screen items-center justify-center bg-background px-4 py-12">
      <div className="w-full max-w-md space-y-4">
        <AuthForm
          mode="login"
          onSubmit={async (values) => loginMutation.mutateAsync(values)}
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
