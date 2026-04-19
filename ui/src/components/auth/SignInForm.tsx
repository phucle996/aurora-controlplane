"use client";

import Input from "@/components/form/input/InputField";
import Label from "@/components/form/Label";
import Button from "@/components/ui/button/Button";
import { EyeCloseIcon, EyeIcon } from "@/icons";
import Link from "next/link";
import { useRouter } from "next/navigation";
import React, { FormEvent, useEffect, useState } from "react";
import AuthShell from "./AuthShell";
import { startSession } from "./auth-session";
import { parseAPIError } from "./auth-utils";
import { useToast } from "../ui/toast/ToastProvider";

type APIResponse<T> = {
  data?: T;
  message?: string;
  error?: string;
};

type LoginResponseData = {
  mfa_required?: boolean;
  challenge_id?: string;
  available_methods?: string[];
  access_token?: string;
};

export default function SignInForm() {
  const router = useRouter();
  const { pushToast } = useToast();
  const [showPassword, setShowPassword] = useState(false);
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState("");
  const [isVerified, setIsVerified] = useState(false);
  const [step, setStep] = useState<"password" | "mfa">("password");
  const [mfaMethods, setMFAMethods] = useState<string[]>([]);
  const [selectedMFAMethod, setSelectedMFAMethod] = useState("");
  const [mfaCode, setMFACode] = useState("");
  const [mfaChallengeID, setMFAChallengeID] = useState("");

  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    const usernameFromQuery = params.get("username")?.trim() ?? "";

    if (usernameFromQuery !== "") {
      setUsername((current) => (current === "" ? usernameFromQuery : current));
    }
    setIsVerified(params.get("verified") === "1");

    const signupMessage = window.sessionStorage.getItem("auth:signup-success-message")?.trim() ?? "";
    if (params.get("registered") === "1" && signupMessage !== "") {
      pushToast({ message: signupMessage, kind: "success" });
      window.sessionStorage.removeItem("auth:signup-success-message");
    }
  }, []);

  function resetMFAStep() {
    setStep("password");
    setMFAMethods([]);
    setSelectedMFAMethod("");
    setMFACode("");
    setMFAChallengeID("");
  }

  function formatMFAMethodLabel(method: string) {
    switch (method.toLowerCase()) {
      case "totp":
      case "authenticator":
        return "Authenticator App";
      case "recovery_code":
        return "Recovery Code";
      default:
        return method.replaceAll("_", " ").replace(/\b\w/g, (char) => char.toUpperCase());
    }
  }

  function completeSignIn(accessToken: string) {
    startSession({
      accessToken,
      username: username.trim().toLowerCase(),
    });
    router.replace("/");
  }

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setSubmitting(true);
    setError("");

    try {
      if (step === "mfa") {
        const response = await fetch("/api/v1/auth/mfa/verify", {
          method: "POST",
          credentials: "include",
          headers: {
            "Content-Type": "application/json",
            "X-Skip-Auth": "1",
          },
          body: JSON.stringify({
            challenge_id: mfaChallengeID,
            method: selectedMFAMethod,
            code: mfaCode,
          }),
        });

        if (!response.ok) {
          setError(await parseAPIError(response));
          return;
        }

        const result = (await response.json()) as APIResponse<LoginResponseData>;
        const accessToken = result.data?.access_token?.trim() ?? "";
        if (accessToken === "") {
          setError("Signed in, but the session token was missing.");
          return;
        }

        completeSignIn(accessToken);
        return;
      }

      const response = await fetch("/api/v1/auth/login", {
        method: "POST",
        credentials: "include",
        headers: {
          "Content-Type": "application/json",
          "X-Skip-Auth": "1",
        },
        body: JSON.stringify({
          username,
          password,
        }),
      });

      if (!response.ok) {
        setError(await parseAPIError(response));
        return;
      }

      const result = (await response.json()) as APIResponse<LoginResponseData>;
      const data = result.data ?? {};
      if (response.status === 202 && data.mfa_required) {
        const methods = Array.isArray(data.available_methods)
          ? data.available_methods.filter((method) => typeof method === "string" && method.trim() !== "")
          : [];
        const challengeID = data.challenge_id?.trim() ?? "";
        if (challengeID === "") {
          setError("MFA challenge could not be started. Please try again.");
          return;
        }
        setMFAMethods(methods);
        setSelectedMFAMethod(methods[0] ?? "totp");
        setMFAChallengeID(challengeID);
        setMFACode("");
        setStep("mfa");
        return;
      }

      if (response.status === 202) {
        pushToast({
          kind: "success",
          message:
            result.message?.trim() || "Your account is pending activation. We sent a new verification email.",
        });
        setPassword("");
        resetMFAStep();
        return;
      }

      const accessToken = data.access_token?.trim() ?? "";
      if (accessToken === "") {
        setError("Signed in, but the session token was missing.");
        return;
      }

      completeSignIn(accessToken);
    } catch (err) {
      if (err instanceof Error && err.message.trim() !== "") {
        setError(err.message);
      } else {
        setError("Something went wrong. Please try again.");
      }
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <AuthShell
      title="Sign In"
      description="Enter your username and password to sign in!"
      footer={
        <p>
          Don&apos;t have an account?{" "}
          <Link
            href="/signup"
            className="font-semibold text-brand-500 transition-colors hover:text-brand-600 dark:text-brand-400"
          >
            Create one now
          </Link>
        </p>
      }
    >
      {isVerified && (
        <div className="rounded-2xl border border-success-200 bg-success-50 px-4 py-3 text-sm text-success-700 dark:border-success-500/30 dark:bg-success-500/10 dark:text-success-300">
          Email verified. You can sign in now.
        </div>
      )}

      {error !== "" && (
        <p className="text-sm font-medium text-error-600 dark:text-error-400" role="alert">
          {error}
        </p>
      )}

      <form className="space-y-5" onSubmit={handleSubmit}>
        {step === "password" ? (
          <>
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
                placeholder="Enter your username"
                onChange={(event) => setUsername(event.target.value)}
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
                  autoComplete="current-password"
                  placeholder="Enter your password"
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

            <div className="flex justify-end">
              <Link
                href="/reset-password"
                className="text-sm text-brand-500 transition-colors hover:text-brand-600 dark:text-brand-400"
              >
                Forgot password?
              </Link>
            </div>

            <Button
              type="submit"
              size="sm"
              disabled={submitting || username.trim() === "" || password === ""}
              className="w-full rounded-lg py-3.5"
            >
              {submitting ? "Signing in..." : "Sign in"}
            </Button>
          </>
        ) : (
          <>
            <div className="rounded-2xl border border-brand-200 bg-brand-50 px-4 py-3 text-sm text-brand-700 dark:border-brand-500/30 dark:bg-brand-500/10 dark:text-brand-300">
              Two-factor authentication is enabled for this account. Use your authenticator app or a
              recovery code to finish signing in.
            </div>

            <div>
              <Label>
                Verification Method <span className="text-error-500">*</span>
              </Label>
              <select
                value={selectedMFAMethod}
                onChange={(event) => setSelectedMFAMethod(event.target.value)}
                className="h-11 w-full rounded-lg border border-gray-300 bg-transparent px-4 py-2.5 text-sm text-gray-800 shadow-theme-xs outline-none transition placeholder:text-gray-400 focus:border-brand-300 focus:ring-3 focus:ring-brand-500/10 dark:border-gray-700 dark:bg-gray-900 dark:text-white/90 dark:placeholder:text-white/30 dark:focus:border-brand-800"
              >
                {mfaMethods.map((method) => (
                  <option key={method} value={method}>
                    {formatMFAMethodLabel(method)}
                  </option>
                ))}
              </select>
            </div>

            <div>
              <Label>
                Authentication Code <span className="text-error-500">*</span>
              </Label>
              <Input
                id="mfa-code"
                name="mfa-code"
                type="text"
                value={mfaCode}
                autoComplete="one-time-code"
                placeholder={
                  selectedMFAMethod === "recovery_code"
                    ? "Enter one of your recovery codes"
                    : "Enter your 2FA code"
                }
                onChange={(event) => setMFACode(event.target.value)}
                required
              />
            </div>

            <div className="flex gap-3">
              <Button
                type="button"
                variant="outline"
                size="sm"
                disabled={submitting}
                className="w-full rounded-lg py-3.5"
                onClick={resetMFAStep}
              >
                Use another account
              </Button>
              <Button
                type="submit"
                size="sm"
                disabled={
                  submitting ||
                  mfaCode.trim() === "" ||
                  selectedMFAMethod.trim() === "" ||
                  mfaChallengeID.trim() === ""
                }
                className="w-full rounded-lg py-3.5"
              >
                {submitting ? "Verifying..." : "Verify and sign in"}
              </Button>
            </div>
          </>
        )}
      </form>
    </AuthShell>
  );
}
