"use client";

import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import AuthShell from "./AuthShell";
import { parseAPIError } from "./auth-utils";

export default function VerifyEmailLink() {
  const router = useRouter();
  const [error, setError] = useState("");

  useEffect(() => {
    let cancelled = false;

    async function verify() {
      const params = new URLSearchParams(window.location.search);
      const token = params.get("token")?.trim() ?? "";

      if (token === "") {
        if (!cancelled) {
          setError("Verification link is invalid.");
        }
        return;
      }

      const response = await fetch(`/api/v1/auth/activate?token=${encodeURIComponent(token)}`, {
        method: "GET",
        credentials: "include",
        headers: {
          "X-Skip-Auth": "1",
        },
      });

      if (!response.ok) {
        if (!cancelled) {
          setError(await parseAPIError(response));
        }
        return;
      }

      if (!cancelled) {
        router.replace("/signin?verified=1");
      }
    }

    void verify();
    return () => {
      cancelled = true;
    };
  }, [router]);

  return (
    <AuthShell
      title={error === "" ? "Verifying Email" : "Verification Failed"}
      description={
        error === ""
          ? "Please wait while we confirm your email address."
          : "The verification link could not be completed."
      }
      footer={
        <span className="text-sm text-gray-500 dark:text-gray-400">
          {error === "" ? "Checking token..." : "You can request a fresh verification email by signing in again."}
        </span>
      }
    >
      {error === "" ? (
        <div className="space-y-3">
          <div className="h-2 w-full overflow-hidden rounded-full bg-gray-100 dark:bg-white/10">
            <div className="auth-progress-bar h-full rounded-full bg-brand-500" />
          </div>
          <p className="text-sm leading-6 text-gray-500 dark:text-gray-400">
            We are activating your account and will take you back to sign in in a moment.
          </p>
        </div>
      ) : (
        <p className="text-sm font-medium text-error-600 dark:text-error-400">{error}</p>
      )}
    </AuthShell>
  );
}
