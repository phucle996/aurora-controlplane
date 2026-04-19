"use client";

import Checkbox from "@/components/form/input/Checkbox";
import Input from "@/components/form/input/InputField";
import Label from "@/components/form/Label";
import Button from "@/components/ui/button/Button";
import { EyeCloseIcon, EyeIcon } from "@/icons";
import Link from "next/link";
import { useRouter } from "next/navigation";
import React, { FormEvent, useMemo, useState } from "react";
import AuthShell from "./AuthShell";
import PasswordChecklist from "./PasswordChecklist";
import { useToast } from "../ui/toast/ToastProvider";
import { isStrongPassword, parseAPIError } from "./auth-utils";

export default function SignUpForm() {
  const router = useRouter();
  const { pushToast } = useToast();
  const [showPassword, setShowPassword] = useState(false);
  const [showConfirmPassword, setShowConfirmPassword] = useState(false);
  const [fullName, setFullName] = useState("");
  const [email, setEmail] = useState("");
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [acceptedTerms, setAcceptedTerms] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState("");

  const passwordIsStrong = useMemo(() => isStrongPassword(password), [password]);
  const passwordsMatch = useMemo(
    () => password !== "" && password === confirmPassword,
    [confirmPassword, password],
  );
  const canSubmit =
    fullName.trim() !== "" &&
    email.trim() !== "" &&
    username.trim() !== "" &&
    passwordIsStrong &&
    passwordsMatch &&
    acceptedTerms &&
    !submitting;

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!canSubmit) {
      setError("Please complete all fields and satisfy the password requirements.");
      return;
    }

    setSubmitting(true);
    setError("");

    try {
      const response = await fetch("/api/v1/auth/register", {
        method: "POST",
        credentials: "include",
        headers: {
          "Content-Type": "application/json",
          "X-Skip-Auth": "1",
        },
        body: JSON.stringify({
          full_name: fullName,
          email,
          username,
          password,
          re_password: confirmPassword,
        }),
      });

      if (!response.ok) {
        setError(await parseAPIError(response));
        return;
      }

      const result = (await response.json()) as {
        message?: string;
      };
      const successMessage =
        typeof result.message === "string" && result.message.trim() !== ""
          ? result.message.trim()
          : "Sign up successful.";

      window.sessionStorage.setItem("auth:signup-success-message", successMessage);
      pushToast({ message: successMessage, kind: "success", durationMs: 2500 });
      router.push(`/signin?registered=1&username=${encodeURIComponent(username.trim())}`);
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <AuthShell
      title="Sign Up"
      description="Enter your details below to create your account."
      footer={
        <p>
          Already have an account?{" "}
          <Link
            href="/signin"
            className="font-semibold text-brand-500 transition-colors hover:text-brand-600 dark:text-brand-400"
          >
            Sign in instead
          </Link>
        </p>
      }
    >
      {error !== "" && (
        <div className="rounded-2xl border border-error-200 bg-error-50 px-4 py-3 text-sm text-error-700 dark:border-error-500/30 dark:bg-error-500/10 dark:text-error-300">
          {error}
        </div>
      )}

      <form className="space-y-5" onSubmit={handleSubmit}>
        <div>
          <Label>
            Full name <span className="text-error-500">*</span>
          </Label>
          <Input
            id="full_name"
            name="full_name"
            type="text"
            value={fullName}
            autoComplete="name"
            placeholder="Enter your full name"
            onChange={(event) => setFullName(event.target.value)}
            required
          />
        </div>

        <div>
          <Label>
            Email <span className="text-error-500">*</span>
          </Label>
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

        <div>
          <Label>
            Username <span className="text-error-500">*</span>
          </Label>
          <Input
            id="username"
            name="username"
            type="text"
            value={username}
            autoComplete="username"
            placeholder="Choose a username"
            onChange={(event) => setUsername(event.target.value)}
            hint="Use 3-32 characters: letters, numbers, dot, underscore, or hyphen."
            required
          />
        </div>

        <div>
          <Label>
            Password <span className="text-error-500">*</span>
          </Label>
          <div className="relative">
            <Input
              id="password"
              name="password"
              type={showPassword ? "text" : "password"}
              value={password}
              autoComplete="new-password"
              placeholder="Create a strong password"
              onChange={(event) => setPassword(event.target.value)}
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
            Confirm password <span className="text-error-500">*</span>
          </Label>
          <div className="relative">
            <Input
              id="confirm_password"
              name="confirm_password"
              type={showConfirmPassword ? "text" : "password"}
              value={confirmPassword}
              autoComplete="new-password"
              placeholder="Retype your password"
              onChange={(event) => setConfirmPassword(event.target.value)}
              error={confirmPassword !== "" && !passwordsMatch}
              hint={
                confirmPassword !== "" && !passwordsMatch
                  ? "Passwords do not match yet."
                  : undefined
              }
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

        <PasswordChecklist password={password} confirmPassword={confirmPassword} />

        <div className="flex items-start gap-3">
          <Checkbox
            className="mt-0.5 h-5 w-5"
            checked={acceptedTerms}
            onChange={setAcceptedTerms}
          />
          <p className="text-sm leading-6 text-gray-500 dark:text-gray-400">
            I agree to the{" "}
            <span className="font-medium text-gray-900 dark:text-white">
              Terms and Conditions
            </span>{" "}
            and the{" "}
            <span className="font-medium text-gray-900 dark:text-white">
              Privacy Policy
            </span>
            .
          </p>
        </div>

        <Button
          type="submit"
          size="sm"
          disabled={!canSubmit}
          className="w-full rounded-lg py-3.5"
        >
          {submitting ? "Creating account..." : "Sign up"}
        </Button>
      </form>
    </AuthShell>
  );
}
