"use client";

import Link from "next/link";
import { AuthForm } from "@/components/auth-form";
import { useAuth } from "@/hooks/useAuth";

export default function SignupPage() {
  const { signupMutation } = useAuth();

  return (
    <main className="relative flex min-h-screen items-center justify-center overflow-hidden px-4 py-12">
      <div className="pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_top_right,_rgba(39,39,42,0.12),_transparent_40%),radial-gradient(circle_at_bottom_left,_rgba(39,39,42,0.1),_transparent_45%)] dark:bg-[radial-gradient(circle_at_top_right,_rgba(250,250,250,0.08),_transparent_36%),radial-gradient(circle_at_bottom_left,_rgba(250,250,250,0.06),_transparent_40%)]" />
      <div className="relative z-10 w-full max-w-md space-y-4">
        <AuthForm mode="signup" onSubmit={async (values) => signupMutation.mutateAsync(values)} />
        <p className="text-center text-sm text-zinc-600 dark:text-zinc-300">
          Already have an account?{" "}
          <Link href="/login" className="font-medium underline">
            Login
          </Link>
        </p>
      </div>
    </main>
  );
}
