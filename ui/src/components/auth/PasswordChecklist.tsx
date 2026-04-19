"use client";

import { CheckLineIcon, CloseLineIcon } from "@/icons";
import { getPasswordChecklist } from "./auth-utils";

type PasswordChecklistProps = {
  password: string;
  confirmPassword?: string;
};

type RequirementItemProps = {
  label: string;
  met: boolean;
};

function RequirementItem({ label, met }: RequirementItemProps) {
  return (
    <li
      className={`flex items-center gap-2 text-sm transition-colors ${
        met ? "text-success-600 dark:text-success-400" : "text-gray-500 dark:text-gray-400"
      }`}
    >
      <span
        className={`flex h-5 w-5 items-center justify-center rounded-full border ${
          met
            ? "border-success-200 bg-success-50 dark:border-success-500/40 dark:bg-success-500/10"
            : "border-gray-200 bg-white dark:border-gray-700 dark:bg-gray-900"
        }`}
      >
        {met ? <CheckLineIcon /> : <CloseLineIcon />}
      </span>
      <span>{label}</span>
    </li>
  );
}

export default function PasswordChecklist({
  password,
  confirmPassword = "",
}: PasswordChecklistProps) {
  const checklist = getPasswordChecklist(password);
  const passwordsMatch = confirmPassword.length > 0 && password === confirmPassword;

  return (
    <div className="border-t border-gray-200 pt-4 dark:border-gray-800">
      <p className="mb-3 text-xs font-semibold uppercase tracking-[0.22em] text-gray-500 dark:text-gray-400">
        Password requirements
      </p>
      <ul className="grid gap-2 sm:grid-cols-2">
        <RequirementItem label="At least 8 characters" met={checklist.minLength} />
        <RequirementItem label="One lowercase letter" met={checklist.lowercase} />
        <RequirementItem label="One uppercase letter" met={checklist.uppercase} />
        <RequirementItem label="One number" met={checklist.number} />
        <RequirementItem label="One special character" met={checklist.special} />
        <RequirementItem label="Passwords match" met={passwordsMatch} />
      </ul>
    </div>
  );
}
