"use client";

import { useEffect, useMemo, useState } from "react";
import QRCode from "qrcode";
import { useToast } from "@/components/ui/toast/ToastProvider";
import Button from "@/components/ui/button/Button";
import Input from "@/components/form/input/InputField";
import Label from "@/components/form/Label";

type MFAMethod = {
  id?: string;
  mfa_type?: string;
  device_name?: string;
  is_enabled?: boolean;
};

type MethodsResponse = {
  data?: MFAMethod[];
  message?: string;
  error?: string;
};

type SetupResponse = {
  data?: {
    setting_id?: string;
    provisioning_uri?: string;
  };
  message?: string;
  error?: string;
};

function isTOTPEnabled(methods: MFAMethod[]) {
  return methods.some((method) => method.mfa_type?.toLowerCase() === "totp");
}

export function TwoFactorTab() {
  const [methods, setMethods] = useState<MFAMethod[]>([]);
  const [loading, setLoading] = useState(true);
  const [busyAction, setBusyAction] = useState("");
  const [otpAuthURL, setOTPAuthURL] = useState("");
  const [qrCodeURL, setQRCodeURL] = useState("");
  const [verificationCode, setVerificationCode] = useState("");
  const [recoveryCodes, setRecoveryCodes] = useState<string[]>([]);
  const [totpSettingID, setTotpSettingID] = useState("");
  const { pushToast } = useToast();

  const enabled = useMemo(() => isTOTPEnabled(methods), [methods]);

  const parsedSecret = useMemo(() => {
    if (!otpAuthURL) return "";
    try {
      const url = new URL(otpAuthURL);
      return url.searchParams.get("secret") || otpAuthURL;
    } catch {
      return otpAuthURL;
    }
  }, [otpAuthURL]);

  useEffect(() => {
    void loadMethods();
  }, []);

  useEffect(() => {
    let cancelled = false;

    async function buildQRCode() {
      if (otpAuthURL.trim() === "") {
        setQRCodeURL("");
        return;
      }

      try {
        const dataURL = await QRCode.toDataURL(otpAuthURL, {
          width: 280,
          margin: 1,
        });
        if (!cancelled) {
          setQRCodeURL(dataURL);
        }
      } catch {
        if (!cancelled) {
          setQRCodeURL("");
        }
      }
    }

    void buildQRCode();
    return () => {
      cancelled = true;
    };
  }, [otpAuthURL]);

  async function loadMethods() {
    setLoading(true);

    try {
      const response = await fetch("/api/v1/me/mfa", {
        method: "GET",
        credentials: "include",
        cache: "no-store",
      });

      if (!response.ok) {
        throw new Error("Failed to load your 2FA methods.");
      }

      const payload = (await response.json()) as MethodsResponse;
      const nextMethods = Array.isArray(payload.data) ? payload.data : [];
      setMethods(nextMethods);
      const totpMethod = nextMethods.find((method) => method.mfa_type?.toLowerCase() === "totp");
      setTotpSettingID(totpMethod?.id?.trim() ?? "");
    } catch (err) {
      pushToast({
        kind: "error",
        message: err instanceof Error ? err.message : "Failed to load your 2FA methods.",
      });
    } finally {
      setLoading(false);
    }
  }

  async function handleBeginSetup() {
    setBusyAction("setup");

    try {
      const response = await fetch("/api/v1/me/mfa/totp/enroll", {
        method: "POST",
        credentials: "include",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          device_name: "Authenticator App",
        }),
      });

      const payload = (await response.json()) as SetupResponse;

      if (!response.ok) {
        throw new Error(payload.message || payload.error || "Failed to prepare 2FA setup.");
      }

      setTotpSettingID(payload.data?.setting_id?.trim() ?? "");
      setOTPAuthURL(payload.data?.provisioning_uri?.trim() ?? "");
      pushToast({
        kind: "success",
        message: payload.message || "Authenticator setup is ready.",
      });
    } catch (err) {
      pushToast({
        kind: "error",
        message: err instanceof Error ? err.message : "Failed to prepare 2FA setup.",
      });
    } finally {
      setBusyAction("");
    }
  }

  async function handleVerifySetup() {
    if (verificationCode.trim() === "") {
      pushToast({
        kind: "error",
        message: "Enter the 6-digit code from your authenticator app.",
      });
      return;
    }
    if (totpSettingID.trim() === "") {
      pushToast({
        kind: "error",
        message: "Start TOTP setup again before confirming the code.",
      });
      return;
    }

    setBusyAction("verify");

    try {
      const response = await fetch("/api/v1/me/mfa/totp/confirm", {
        method: "POST",
        credentials: "include",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          setting_id: totpSettingID,
          code: verificationCode.trim(),
        }),
      });

      const payload = (await response.json()) as {
        message?: string;
        error?: string;
      };

      if (!response.ok) {
        throw new Error(payload.message || payload.error || "Failed to verify the 2FA code.");
      }

      await loadMethods();
      setVerificationCode("");
      await handleGenerateRecoveryCodes();
      pushToast({
        kind: "success",
        message: payload.message || "2FA enabled.",
      });
    } catch (err) {
      pushToast({
        kind: "error",
        message: err instanceof Error ? err.message : "Failed to verify the 2FA code.",
      });
    } finally {
      setBusyAction("");
    }
  }

  async function handleDisable() {
    if (totpSettingID.trim() === "") {
      pushToast({
        kind: "error",
        message: "No TOTP authenticator is active right now.",
      });
      return;
    }

    setBusyAction("disable");

    try {
      const response = await fetch(`/api/v1/me/mfa/${encodeURIComponent(totpSettingID)}/disable`, {
        method: "PATCH",
        credentials: "include",
        headers: {
          "Content-Type": "application/json",
        },
      });

      const payload = (await response.json()) as {
        message?: string;
        error?: string;
      };

      if (!response.ok) {
        throw new Error(payload.message || payload.error || "Failed to disable 2FA.");
      }

      setMethods([]);
      setTotpSettingID("");
      setOTPAuthURL("");
      setQRCodeURL("");
      setVerificationCode("");
      setRecoveryCodes([]);
      pushToast({
        kind: "success",
        message: payload.message || "2FA disabled.",
      });
    } catch (err) {
      pushToast({
        kind: "error",
        message: err instanceof Error ? err.message : "Failed to disable 2FA.",
      });
    } finally {
      setBusyAction("");
    }
  }

  async function handleCopySecret() {
    if (parsedSecret.trim() === "") {
      return;
    }
    try {
      await navigator.clipboard.writeText(parsedSecret);
      pushToast({
        kind: "success",
        message: "Setup Key copied.",
      });
    } catch {
      pushToast({
        kind: "error",
        message: "Could not copy the setup key.",
      });
    }
  }

  async function handleGenerateRecoveryCodes() {
    const response = await fetch("/api/v1/me/mfa/recovery-codes", {
      method: "POST",
      credentials: "include",
      headers: {
        "Content-Type": "application/json",
      },
      body: "{}",
    });

    const payload = (await response.json()) as {
      recovery_codes?: string[];
      message?: string;
      error?: string;
      warning?: string;
    };

    if (!response.ok) {
      throw new Error(payload.message || payload.error || "Failed to generate recovery codes.");
    }

    const codes = Array.isArray(payload.recovery_codes) ? payload.recovery_codes : [];
    setRecoveryCodes(codes);
    pushToast({
      kind: "success",
      message: payload.warning || payload.message || "Recovery codes generated.",
    });
  }

  function handleDownloadRecoveryCodes() {
    if (recoveryCodes.length === 0) {
      return;
    }

    const body = [
      "Aurora 2FA Recovery Codes",
      "",
      "Keep these codes in a safe place.",
      "Each code can be used once to sign in if you lose access to your authenticator app.",
      "",
      ...recoveryCodes.map((code, index) => `${index + 1}. ${code}`),
      "",
    ].join("\n");

    const blob = new Blob([body], { type: "text/plain;charset=utf-8" });
    const url = URL.createObjectURL(blob);
    const anchor = document.createElement("a");
    anchor.href = url;
    anchor.download = "aurora-2fa-recovery-codes.txt";
    document.body.appendChild(anchor);
    anchor.click();
    document.body.removeChild(anchor);
    URL.revokeObjectURL(url);
  }

  return (
    <div className="space-y-6">
      <section className="overflow-hidden rounded-3xl border border-gray-200 bg-white dark:border-gray-800 dark:bg-white/[0.03]">
        <div className="flex flex-col gap-6 border-b border-gray-200 px-6 py-6 dark:border-gray-800 lg:flex-row lg:items-start lg:justify-between">
          <div className="space-y-3">
            <div className="flex items-center gap-3">
              <span
                className={`inline-flex items-center rounded-full px-3 py-1 text-xs font-semibold ${
                  enabled
                    ? "bg-emerald-500/12 text-emerald-600 dark:text-emerald-400"
                    : "bg-amber-500/12 text-amber-600 dark:text-amber-400"
                }`}
              >
                <span
                  className={`mr-2 h-2 w-2 rounded-full ${
                    enabled ? "bg-emerald-500" : "bg-amber-500"
                  }`}
                />
                {enabled ? "Enabled" : "Not enabled"}
              </span>
              <span className="text-xs uppercase tracking-[0.28em] text-gray-400 dark:text-gray-500">
                Security
              </span>
            </div>
            <div>
              <h3 className="text-2xl font-semibold text-gray-900 dark:text-white">
                Two-Factor Authentication
              </h3>
              <p className="mt-2 max-w-2xl text-sm leading-6 text-gray-500 dark:text-gray-400">
                Protect your account with a time-based one-time password from Google Authenticator,
                1Password, Authy, or any compatible TOTP app.
              </p>
            </div>
          </div>

          <div className="flex flex-wrap items-center gap-3">
            {!enabled && (
              <Button
                onClick={handleBeginSetup}
                disabled={loading || (busyAction !== "" && busyAction !== "setup")}
                className="rounded-xl px-5"
              >
                {busyAction === "setup" ? "Preparing..." : "Enable 2FA"}
              </Button>
            )}
            {enabled && (
              <Button
                variant="outline"
                onClick={handleDisable}
                disabled={busyAction !== "" && busyAction !== "disable"}
                className="rounded-xl border-rose-200 text-rose-600 hover:bg-rose-50 hover:text-rose-700 dark:border-rose-500/30 dark:bg-transparent dark:text-rose-300 dark:hover:bg-rose-500/10"
              >
                {busyAction === "disable" ? "Disabling..." : "Disable 2FA"}
              </Button>
            )}
          </div>
        </div>

        <div className="space-y-6 px-6 py-6">
          {enabled ? (
            <div className="space-y-6">
              <div className="grid gap-6 xl:grid-cols-[minmax(0,1.2fr)_360px]">
                <div className="rounded-2xl border border-emerald-200 bg-emerald-50/80 px-5 py-5 dark:border-emerald-500/20 dark:bg-emerald-500/10">
                  <h4 className="text-lg font-semibold text-gray-900 dark:text-white">
                    Authenticator protection is active
                  </h4>
                  <p className="mt-2 text-sm leading-6 text-gray-600 dark:text-gray-300">
                    Your account now requires a 6-digit code after entering your password. Keep your
                    authenticator app available before signing out on other devices.
                  </p>
                </div>
                <div className="rounded-2xl border border-gray-200 bg-white p-5 dark:border-gray-800 dark:bg-gray-900/60">
                  <p className="text-xs uppercase tracking-[0.28em] text-gray-400 dark:text-gray-500">
                    Active Method
                  </p>
                  <p className="mt-3 text-lg font-semibold text-gray-900 dark:text-white">
                    TOTP Authenticator
                  </p>
                  <p className="mt-2 text-sm leading-6 text-gray-500 dark:text-gray-400">
                    Compatible with Google Authenticator, 1Password, Authy, Bitwarden, and similar
                    apps.
                  </p>
                </div>
              </div>

              {recoveryCodes.length > 0 && (
                <div className="rounded-2xl border border-gray-200 bg-white p-5 dark:border-gray-800 dark:bg-gray-900/60">
                  <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
                    <div>
                      <p className="text-xs uppercase tracking-[0.28em] text-gray-400 dark:text-gray-500">
                        Recovery Codes
                      </p>
                      <h4 className="mt-2 text-lg font-semibold text-gray-900 dark:text-white">
                        Save these backup codes now
                      </h4>
                      <p className="mt-2 max-w-2xl text-sm leading-6 text-gray-500 dark:text-gray-400">
                        Each recovery code works once. Download them now and keep them somewhere safe.
                      </p>
                    </div>
                    <Button onClick={handleDownloadRecoveryCodes} className="rounded-xl px-5">
                      Download Codes
                    </Button>
                  </div>

                  <div className="mt-5 grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
                    {recoveryCodes.map((code) => (
                      <div
                        key={code}
                        className="rounded-2xl border border-gray-200 bg-gray-50 px-4 py-3 font-mono text-sm font-semibold tracking-[0.16em] text-gray-900 dark:border-gray-800 dark:bg-gray-950/40 dark:text-white"
                      >
                        {code}
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </div>
          ) : (
            <div className="grid gap-6 xl:grid-cols-[minmax(0,1.25fr)_380px]">
              <div className="rounded-2xl border border-gray-200 bg-white p-5 dark:border-gray-800 dark:bg-gray-900/60">
                <div className="mb-5">
                  <p className="text-xs uppercase tracking-[0.28em] text-gray-400 dark:text-gray-500">
                    Step 1
                  </p>
                  <h4 className="mt-2 text-lg font-semibold text-gray-900 dark:text-white">
                    Scan this QR code with your authenticator app
                  </h4>
                  <p className="mt-2 text-sm leading-6 text-gray-500 dark:text-gray-400">
                    If you prefer, you can copy the setup key below and enter it into a
                    compatible authenticator app manually.
                  </p>
                </div>

                {qrCodeURL !== "" ? (
                  <div className="flex flex-col gap-5 xl:flex-row xl:items-start">
                    <div className="shrink-0 rounded-2xl border border-gray-200 bg-white p-4 shadow-theme-xs dark:border-gray-800 dark:bg-white">
                      <img src={qrCodeURL} alt="TOTP QR Code" className="h-64 w-64 rounded-xl object-contain" />
                    </div>
                    <div className="flex-1 space-y-4">
                      <div className="rounded-2xl border border-gray-200 bg-gray-50 px-4 py-4 dark:border-gray-800 dark:bg-gray-950/40">
                        <p className="text-xs uppercase tracking-[0.24em] text-gray-400 dark:text-gray-500">
                          Setup Key
                        </p>
                        <div className="mt-3">
                          <button
                            type="button"
                            onClick={handleCopySecret}
                            disabled={parsedSecret.trim() === ""}
                            className="block w-full break-all rounded-xl bg-white px-4 py-3 text-left font-mono text-sm font-semibold text-gray-900 transition hover:bg-gray-50 disabled:cursor-default dark:bg-gray-900 dark:text-white dark:hover:bg-gray-800"
                            title="Click to copy"
                          >
                            {parsedSecret || "Setup Key will appear here after setup starts."}
                          </button>
                        </div>
                      </div>

                      <div className="rounded-2xl border border-gray-200 bg-gray-50 px-4 py-4 dark:border-gray-800 dark:bg-gray-950/40">
                        <p className="text-xs uppercase tracking-[0.24em] text-gray-400 dark:text-gray-500">
                          Step 2
                        </p>
                        <h5 className="mt-2 text-base font-semibold text-gray-900 dark:text-white">
                          Enter the 6-digit code to activate 2FA
                        </h5>
                        <div className="mt-4 flex flex-col gap-4 sm:flex-row sm:items-end">
                          <div className="flex-1">
                            <Label>Verification Code</Label>
                            <Input
                              type="text"
                              value={verificationCode}
                              onChange={(event) => setVerificationCode(event.target.value)}
                              placeholder="123456"
                              autoComplete="one-time-code"
                            />
                          </div>
                          <Button
                            onClick={handleVerifySetup}
                            disabled={busyAction !== "" && busyAction !== "verify"}
                            className="rounded-xl sm:min-w-[180px]"
                          >
                            {busyAction === "verify" ? "Verifying..." : "Enable 2FA"}
                          </Button>
                        </div>
                      </div>
                    </div>
                  </div>
                ) : (
                  <div className="rounded-2xl border border-dashed border-gray-300 px-6 py-12 text-center text-sm text-gray-500 dark:border-gray-700 dark:text-gray-400">
                    Press “Enable 2FA” to generate a QR code and provisioning URI.
                  </div>
                )}
              </div>

              <div className="rounded-2xl border border-gray-200 bg-white p-5 dark:border-gray-800 dark:bg-gray-900/60">
                <p className="text-xs uppercase tracking-[0.28em] text-gray-400 dark:text-gray-500">
                  What you need
                </p>
                <ul className="mt-4 space-y-3 text-sm leading-6 text-gray-600 dark:text-gray-300">
                  <li>Any TOTP-compatible authenticator app on your phone or desktop.</li>
                  <li>The QR code or provisioning URI shown here.</li>
                  <li>A fresh 6-digit code to verify before 2FA becomes active.</li>
                </ul>
                <div className="mt-6 rounded-2xl border border-amber-200 bg-amber-50 px-4 py-4 text-sm text-amber-800 dark:border-amber-500/20 dark:bg-amber-500/10 dark:text-amber-200">
                  Keep your authenticator app accessible. If you lose it before adding recovery
                  codes, signing back in will be harder.
                </div>
              </div>
            </div>
          )}
        </div>
      </section>
    </div>
  );
}
