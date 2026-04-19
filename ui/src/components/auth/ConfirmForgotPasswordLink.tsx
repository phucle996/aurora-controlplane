"use client";

import { useRouter } from "next/navigation";
import { useEffect } from "react";
import AuthShell from "./AuthShell";

export default function ConfirmForgotPasswordLink() {
  const router = useRouter();

  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    const token = params.get("token")?.trim() ?? "";
    if (token === "") {
      router.replace("/reset-password/invalid");
      return;
    }

    router.replace(`/reset-password/new?token=${encodeURIComponent(token)}`);
  }, [router]);

  return (
    <AuthShell
      title="Checking reset link"
      description="Please wait while we verify your reset request."
      footer={<span className="text-sm text-gray-500 dark:text-gray-400">Validating token...</span>}
    >
      <div className="space-y-3">
        <div className="h-2 w-full overflow-hidden rounded-full bg-gray-100 dark:bg-white/10">
          <div className="auth-progress-bar h-full rounded-full bg-brand-500" />
        </div>
        <p className="text-sm leading-6 text-gray-500 dark:text-gray-400">
          We are confirming your reset token and preparing your new password
          page.
        </p>
      </div>
    </AuthShell>
  );
}
