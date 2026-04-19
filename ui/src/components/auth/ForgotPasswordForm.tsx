"use client";

import Input from "@/components/form/input/InputField";
import Label from "@/components/form/Label";
import Button from "@/components/ui/button/Button";
import Link from "next/link";
import { useRouter } from "next/navigation";
import React, { FormEvent, useEffect, useState } from "react";
import AuthShell from "./AuthShell";
import { parseAPIError } from "./auth-utils";

export default function ForgotPasswordForm() {
  const router = useRouter();
  const [email, setEmail] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState(false);

  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    const token = params.get("token")?.trim() ?? "";
    if (token !== "") {
      router.replace(`/reset-password/new?token=${encodeURIComponent(token)}`);
    }
  }, [router]);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setSubmitting(true);
    setError("");
    setSuccess(false);

    try {
      const response = await fetch("/api/v1/auth/forgot-password", {
        method: "POST",
        credentials: "include",
        headers: {
          "Content-Type": "application/json",
          "X-Skip-Auth": "1",
        },
        body: JSON.stringify({
          email,
        }),
      });

      if (!response.ok) {
        setError(await parseAPIError(response));
        return;
      }

      setSuccess(true);
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <AuthShell
      title="Forgot password?"
      description="No worries, we'll send you reset instructions."
      footer={
        <Link
          href="/signin"
          className="inline-flex items-center gap-2 text-sm text-gray-600 transition-colors hover:text-gray-900 dark:text-gray-400 dark:hover:text-white"
        >
          <span aria-hidden="true">&larr;</span>
          <span>Back to log in</span>
        </Link>
      }
    >
      {success && (
        <div className="text-sm leading-6 text-success-700 dark:text-success-300">
          If the email exists in our system, please check your inbox for reset instructions.
        </div>
      )}

      {error !== "" && (
        <div className="text-sm leading-6 text-error-700 dark:text-error-300">
          {error}
        </div>
      )}

      <form className="space-y-5" onSubmit={handleSubmit}>
        <div>
          <Label>Email</Label>
          <Input
            id="email"
            name="email"
            type="email"
            value={email}
            autoComplete="email"
            placeholder="Enter your email"
            onChange={(event) => setEmail(event.target.value)}
            required
          />
        </div>

        <Button
          type="submit"
          size="sm"
          disabled={submitting || email.trim() === ""}
          className="w-full rounded-lg py-3.5"
        >
          {submitting ? "Sending reset instructions..." : "Reset password"}
        </Button>
      </form>
    </AuthShell>
  );
}
