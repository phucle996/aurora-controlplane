"use client";

import Input from "@/components/form/input/InputField";
import Label from "@/components/form/Label";
import Button from "@/components/ui/button/Button";
import { EyeCloseIcon, EyeIcon } from "@/icons";
import Link from "next/link";
import { useRouter } from "next/navigation";
import React, { FormEvent, useEffect, useMemo, useState } from "react";
import AuthShell from "./AuthShell";
import PasswordChecklist from "./PasswordChecklist";
import {
  isForgotPasswordTokenError,
  isStrongPassword,
  parseAPIError,
} from "./auth-utils";

export default function NewPasswordForm() {
  const router = useRouter();
  const [token, setToken] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [showPassword, setShowPassword] = useState(false);
  const [showConfirmPassword, setShowConfirmPassword] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState(false);

  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    const rawToken = params.get("token")?.trim() ?? "";

    if (rawToken === "") {
      router.replace("/reset-password/invalid");
      return;
    }

    setToken(rawToken);
  }, [router]);

  const passwordIsStrong = useMemo(
    () => isStrongPassword(newPassword),
    [newPassword],
  );
  const passwordsMatch = useMemo(
    () => newPassword !== "" && newPassword === confirmPassword,
    [confirmPassword, newPassword],
  );

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!passwordIsStrong || !passwordsMatch || token === "") {
      setError("Please provide a strong password and make sure both fields match.");
      return;
    }

    setSubmitting(true);
    setError("");

    try {
      const response = await fetch("/api/v1/auth/reset-password", {
        method: "POST",
        credentials: "include",
        headers: {
          "Content-Type": "application/json",
          "X-Skip-Auth": "1",
        },
        body: JSON.stringify({
          token,
          new_password: newPassword,
          re_password: confirmPassword,
        }),
      });

      if (!response.ok) {
        const apiError = await parseAPIError(response);
        if (isForgotPasswordTokenError(apiError)) {
          router.replace("/reset-password/invalid");
          return;
        }

        setError(apiError);
        return;
      }

      setSuccess(true);
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <AuthShell
      title="Set new password"
      description="Enter your new password below and confirm it to finish resetting your account."
      footer={
        <Link
          href="/signin"
          className="inline-flex items-center gap-2 text-sm text-gray-600 transition-colors hover:text-gray-900 dark:text-gray-400 dark:hover:text-white"
        >
          <span aria-hidden="true">&larr;</span>
          <span>Back to sign in</span>
        </Link>
      }
    >
      {success && (
        <div className="space-y-4">
          <div className="rounded-2xl border border-success-200 bg-success-50 px-4 py-3 text-sm text-success-700 dark:border-success-500/30 dark:bg-success-500/10 dark:text-success-300">
            Your password has been reset successfully.
          </div>
          <Link href="/signin" className="block">
            <Button className="w-full rounded-lg py-3.5" size="sm">
              Go to sign in
            </Button>
          </Link>
        </div>
      )}

      {!success && (
        <>
          {error !== "" && (
            <div className="rounded-2xl border border-error-200 bg-error-50 px-4 py-3 text-sm text-error-700 dark:border-error-500/30 dark:bg-error-500/10 dark:text-error-300">
              {error}
            </div>
          )}

          <form className="space-y-5" onSubmit={handleSubmit}>
            <div>
              <Label>
                New password <span className="text-error-500">*</span>
              </Label>
              <div className="relative">
                <Input
                  id="new_password"
                  name="new_password"
                  type={showPassword ? "text" : "password"}
                  value={newPassword}
                  autoComplete="new-password"
                  placeholder="Enter your new password"
                  onChange={(event) => setNewPassword(event.target.value)}
                  required
                />
                <button
                  type="button"
                  onClick={() => setShowPassword((value) => !value)}
                  className="absolute right-4 top-1/2 z-30 -translate-y-1/2 text-gray-500 transition-colors hover:text-gray-700 dark:text-gray-400 dark:hover:text-white"
                  aria-label={showPassword ? "Hide password" : "Show password"}
                >
                  {showPassword ? (
                    <EyeIcon className="fill-current" />
                  ) : (
                    <EyeCloseIcon className="fill-current" />
                  )}
                </button>
              </div>
            </div>

            <div>
              <Label>
                Re-enter new password <span className="text-error-500">*</span>
              </Label>
              <div className="relative">
                <Input
                  id="confirm_password"
                  name="confirm_password"
                  type={showConfirmPassword ? "text" : "password"}
                  value={confirmPassword}
                  autoComplete="new-password"
                  placeholder="Re-enter your new password"
                  onChange={(event) => setConfirmPassword(event.target.value)}
                  error={confirmPassword !== "" && !passwordsMatch}
                  required
                />
                <button
                  type="button"
                  onClick={() => setShowConfirmPassword((value) => !value)}
                  className="absolute right-4 top-1/2 z-30 -translate-y-1/2 text-gray-500 transition-colors hover:text-gray-700 dark:text-gray-400 dark:hover:text-white"
                  aria-label={showConfirmPassword ? "Hide password" : "Show password"}
                >
                  {showConfirmPassword ? (
                    <EyeIcon className="fill-current" />
                  ) : (
                    <EyeCloseIcon className="fill-current" />
                  )}
                </button>
              </div>
            </div>

            <PasswordChecklist
              password={newPassword}
              confirmPassword={confirmPassword}
            />

            <Button
              type="submit"
              size="sm"
              disabled={submitting || !passwordIsStrong || !passwordsMatch || token === ""}
              className="w-full rounded-lg py-3.5"
            >
              {submitting ? "Saving new password..." : "Save new password"}
            </Button>
          </form>
        </>
      )}
    </AuthShell>
  );
}
