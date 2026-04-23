"use client";

import { useEffect, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import { SMTPPageShell } from "@/components/smtp/SMTPPageShell";
import { createTemplate, listConsumerOptions } from "@/components/smtp/api";
import { useSMTPWorkspace } from "@/components/smtp/SMTPWorkspaceProvider";

type TemplateFormState = {
  name: string;
  category: string;
  trafficClass: string;
  status: string;
  consumerID: string;
  subject: string;
  fromEmail: string;
  toEmail: string;
  body: string;
};

export default function NewTemplateForm() {
  return (
    <SMTPPageShell>
      <NewTemplateFormContent />
    </SMTPPageShell>
  );
}

function NewTemplateFormContent() {
  const router = useRouter();
  const { workspace, workspaceID, isLoading: isWorkspaceLoading, error: workspaceError } = useSMTPWorkspace();
  const [form, setForm] = useState<TemplateFormState>({
    name: "",
    category: "",
    trafficClass: "transactional",
    status: "draft",
    consumerID: "",
    subject: "",
    fromEmail: "",
    toEmail: "",
    body: "Hello {$user.full_name},\n\nOpen {$meta.link} to verify your Aurora identity.\n\nIf you did not request this, you can ignore this email.",
  });
  const [consumers, setConsumers] = useState<Array<{ id: string; name: string }>>([]);
  const [fieldErrors, setFieldErrors] = useState<Partial<Record<keyof TemplateFormState, string>>>(
    {},
  );
  const [submitError, setSubmitError] = useState("");
  const [isSaving, setIsSaving] = useState(false);

  useEffect(() => {
    if (workspaceID === "") {
      setConsumers([]);
      return;
    }

    let cancelled = false;

    async function loadConsumers() {
      try {
        const result = await listConsumerOptions(workspaceID);
        if (cancelled) {
          return;
        }

        const nextConsumers = result.map((item) => ({
          id: item.id,
          name: item.label,
        }));
        setConsumers(nextConsumers);
      } catch {
        if (!cancelled) {
          setConsumers([]);
        }
      }
    }

    void loadConsumers();
    return () => {
      cancelled = true;
    };
  }, [workspaceID]);

  function updateField<K extends keyof TemplateFormState>(key: K, value: TemplateFormState[K]) {
    setForm((current) => ({ ...current, [key]: value }));
    setFieldErrors((current) => {
      if (current[key] == null) {
        return current;
      }
      const next = { ...current };
      delete next[key];
      return next;
    });
    setSubmitError("");
  }

  async function handleSubmit() {
    const nextErrors: Partial<Record<keyof TemplateFormState, string>> = {};
    const requiredFields: Array<keyof TemplateFormState> = [
      "name",
      "category",
      "status",
      "subject",
      "fromEmail",
      "toEmail",
      "body",
    ];

    for (const field of requiredFields) {
      if (form[field].trim() === "") {
        nextErrors[field] = "This field is required";
      }
    }

    if (form.status === "live" && form.consumerID.trim() === "") {
      nextErrors.consumerID = "A live template must be linked to a consumer";
    }

    if (Object.keys(nextErrors).length > 0) {
      setFieldErrors(nextErrors);
      setSubmitError("");
      return;
    }

    setIsSaving(true);
    setSubmitError("");

    try {
      if (workspaceID === "") {
        setSubmitError("Choose a workspace first.");
        return;
      }
      await createTemplate(workspaceID, {
        name: form.name.trim(),
        category: form.category.trim(),
        traffic_class: form.trafficClass.trim(),
        subject: form.subject.trim(),
        from_email: form.fromEmail.trim(),
        to_email: form.toEmail.trim(),
        status: form.status.trim(),
        consumer_id: form.consumerID.trim(),
        variables: [],
        text_body: form.body,
        html_body: form.body.replaceAll("\n", "<br/>"),
      });
      router.push(`/smtp/templates?workspace=${workspaceID}`);
    } catch {
      setSubmitError("Failed to save template.");
    } finally {
      setIsSaving(false);
    }
  }

  return (
    <div className="space-y-6">
      <div className="rounded-2xl border border-gray-200 bg-white dark:border-gray-800 dark:bg-white/[0.03]">
          <div className="flex items-start justify-between gap-4 px-6 py-5">
            <div>
              <h3 className="text-base font-medium text-gray-800 dark:text-white/90">
                Create Template
              </h3>
              <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
                Create a new SMTP template and save it directly into the SMTP schema.
              </p>
            </div>
            <div className="flex items-center gap-3">
              <button
                type="button"
                onClick={() => router.push("/smtp/templates")}
                className="inline-flex items-center rounded-xl border border-gray-200 bg-white px-4 py-3 text-sm font-semibold text-gray-700 transition hover:bg-gray-50 dark:border-gray-700 dark:bg-gray-900 dark:text-gray-200 dark:hover:bg-gray-800"
              >
                Cancel
              </button>
              <button
                type="button"
                onClick={() => void handleSubmit()}
                disabled={isSaving}
                className="inline-flex items-center rounded-xl bg-gray-900 px-4 py-3 text-sm font-semibold text-white transition hover:bg-gray-800 dark:bg-white dark:text-gray-900 dark:hover:bg-gray-200"
              >
                {isSaving ? "Saving..." : "Save Template"}
              </button>
            </div>
          </div>

          <div className="space-y-6 border-t border-gray-100 p-4 sm:p-6 dark:border-gray-800">
            {submitError !== "" ? (
              <div className="rounded-2xl border border-error-200 bg-error-50 px-4 py-3 text-sm text-error-700 dark:border-error-500/30 dark:bg-error-500/10 dark:text-error-300">
                {submitError}
              </div>
            ) : workspaceError !== "" ? (
              <div className="rounded-2xl border border-error-200 bg-error-50 px-4 py-3 text-sm text-error-700 dark:border-error-500/30 dark:bg-error-500/10 dark:text-error-300">
                {workspaceError}
              </div>
            ) : workspace == null ? (
              <div className="rounded-2xl border border-error-200 bg-error-50 px-4 py-3 text-sm text-error-700 dark:border-error-500/30 dark:bg-error-500/10 dark:text-error-300">
                No workspace is available for SMTP yet.
              </div>
            ) : null}

            <div className="rounded-2xl border border-gray-200 bg-gray-50 px-4 py-4 dark:border-gray-800 dark:bg-gray-900/40">
              <p className="text-xs font-medium tracking-[0.18em] text-gray-400 uppercase">Workspace</p>
              <p className="mt-2 text-sm font-semibold text-gray-900 dark:text-white">
                {workspace?.name ?? (isWorkspaceLoading ? "Loading..." : "No workspace selected")}
              </p>
            </div>

            <div className="grid gap-4 xl:grid-cols-[3fr_7fr]">
              <div className="space-y-4">
                <Field
                  label="Template name"
                  placeholder="Verify Email"
                  value={form.name}
                  onChange={(value) => updateField("name", value)}
                  error={fieldErrors.name}
                />
                <Field
                  label="Category"
                  placeholder="IAM"
                  value={form.category}
                  onChange={(value) => updateField("category", value)}
                  error={fieldErrors.category}
                />
                <SelectField
                  label="Traffic class"
                  value={form.trafficClass}
                  onChange={(value) => updateField("trafficClass", value)}
                  options={[
                    { value: "critical_auth", label: "Critical Auth" },
                    { value: "transactional", label: "Transactional" },
                    { value: "bulk_marketing", label: "Bulk Marketing" },
                  ]}
                />
                <SelectField
                  label="Status"
                  value={form.status}
                  onChange={(value) => updateField("status", value)}
                  options={[
                    { value: "draft", label: "Draft" },
                    { value: "review", label: "Review" },
                    { value: "live", label: "Live" },
                  ]}
                  error={fieldErrors.status}
                />
                <SelectField
                  label="Consumer"
                  value={form.consumerID}
                  onChange={(value) => updateField("consumerID", value)}
                  options={[
                    { value: "", label: "None" },
                    ...consumers.map((consumer) => ({
                      value: consumer.id,
                      label: consumer.name,
                    })),
                  ]}
                />
              </div>

              <div className="space-y-4 rounded-2xl border border-gray-200 bg-gray-50 p-5 dark:border-gray-800 dark:bg-gray-900/40">
                <Field
                  label="Subject"
                  placeholder="Verify your Aurora email address"
                  value={form.subject}
                  onChange={(value) => updateField("subject", value)}
                  error={fieldErrors.subject}
                />
                <Field
                  label="From"
                  placeholder="identity@aurora.local"
                  value={form.fromEmail}
                  onChange={(value) => updateField("fromEmail", value)}
                  error={fieldErrors.fromEmail}
                />
                <Field
                  label="To"
                  placeholder="{$user.email}"
                  value={form.toEmail}
                  onChange={(value) => updateField("toEmail", value)}
                  error={fieldErrors.toEmail}
                />
                <TextAreaField
                  label="Body"
                  value={form.body}
                  onChange={(value) => updateField("body", value)}
                  placeholder=""
                  rows={14}
                  error={fieldErrors.body}
                />
              </div>
            </div>
          </div>
      </div>
    </div>
  );
}

function Field({
  label,
  placeholder,
  value,
  onChange,
  error,
}: {
  label: string;
  placeholder: string;
  value: string;
  onChange: (value: string) => void;
  error?: string;
}) {
  return (
    <div className="space-y-2">
      <span className="text-xs font-medium tracking-[0.12em] text-gray-400 uppercase">
        {label}
      </span>
      <input
        value={value}
        onChange={(event) => onChange(event.target.value)}
        placeholder={placeholder}
        className={`w-full rounded-2xl border bg-white px-4 py-3 text-sm text-gray-800 outline-none placeholder:text-gray-400 dark:bg-gray-900/40 dark:text-white dark:placeholder:text-gray-500 ${
          error
            ? "border-error-400 dark:border-error-500"
            : "border-gray-200 dark:border-gray-700"
        }`}
      />
      {error ? <p className="text-xs text-error-600 dark:text-error-300">{error}</p> : null}
    </div>
  );
}

function SelectField({
  label,
  value,
  onChange,
  options,
  error,
}: {
  label: string;
  value: string;
  onChange: (value: string) => void;
  options: Array<{ value: string; label: string }>;
  error?: string;
}) {
  const [isOpen, setIsOpen] = useState(false);
  const rootRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (rootRef.current == null) {
        return;
      }
      if (!rootRef.current.contains(event.target as Node)) {
        setIsOpen(false);
      }
    }

    document.addEventListener("mousedown", handleClickOutside);
    return () => {
      document.removeEventListener("mousedown", handleClickOutside);
    };
  }, []);

  const selectedOption =
    options.find((option) => option.value === value) ?? options[0] ?? { value: "", label: "-" };

  return (
    <div ref={rootRef} className="space-y-2">
      <span className="text-xs font-medium tracking-[0.12em] text-gray-400 uppercase">
        {label}
      </span>
      <div className="relative">
        <button
          type="button"
          onClick={() => setIsOpen((current) => !current)}
          aria-haspopup="listbox"
          aria-expanded={isOpen}
          className={`flex w-full items-center justify-between rounded-2xl border bg-white px-4 py-3 text-left text-sm text-gray-800 outline-none transition dark:bg-gray-900/40 dark:text-white ${
            error
              ? "border-error-400 dark:border-error-500"
              : "border-gray-200 hover:border-gray-300 dark:border-gray-700 dark:hover:border-gray-600"
          }`}
        >
          <span className="truncate">{selectedOption.label}</span>
          <span
            className={`ml-3 inline-flex h-5 w-5 items-center justify-center text-gray-400 transition dark:text-gray-500 ${
              isOpen ? "rotate-180" : ""
            }`}
          >
            <svg viewBox="0 0 20 20" fill="currentColor" className="h-4 w-4">
              <path
                fillRule="evenodd"
                d="M5.23 7.21a.75.75 0 0 1 1.06.02L10 11.168l3.71-3.938a.75.75 0 1 1 1.08 1.04l-4.25 4.5a.75.75 0 0 1-1.08 0l-4.25-4.5a.75.75 0 0 1 .02-1.06Z"
                clipRule="evenodd"
              />
            </svg>
          </span>
        </button>
        <input type="hidden" value={value} readOnly />
        {isOpen ? (
          <div className="absolute top-[calc(100%+0.5rem)] left-0 z-30 w-full overflow-hidden rounded-2xl border border-gray-200 bg-white shadow-2xl dark:border-gray-700 dark:bg-gray-950">
            <div className="max-h-60 overflow-y-auto py-2">
              {options.map((option) => {
                const active = option.value === value;
                return (
                  <button
                    key={`${label}-${option.value}`}
                    type="button"
                    onClick={() => {
                      onChange(option.value);
                      setIsOpen(false);
                    }}
                    className={`flex w-full items-center justify-between px-4 py-3 text-left text-sm transition ${
                      active
                        ? "bg-gray-900 text-white dark:bg-white dark:text-gray-900"
                        : "text-gray-700 hover:bg-gray-50 dark:text-gray-200 dark:hover:bg-white/5"
                    }`}
                  >
                    <span>{option.label}</span>
                    {active ? (
                      <span className="text-xs font-semibold uppercase opacity-80">Selected</span>
                    ) : null}
                  </button>
                );
              })}
            </div>
          </div>
        ) : null}
      </div>
      {error ? <p className="text-xs text-error-600 dark:text-error-300">{error}</p> : null}
    </div>
  );
}

function TextAreaField({
  label,
  value,
  onChange,
  placeholder,
  rows,
  readOnly = false,
  error,
}: {
  label: string;
  value: string;
  onChange: (value: string) => void;
  placeholder: string;
  rows: number;
  readOnly?: boolean;
  error?: string;
}) {
  return (
    <div className="space-y-2">
      <span className="text-xs font-medium tracking-[0.12em] text-gray-400 uppercase">
        {label}
      </span>
      <textarea
        rows={rows}
        value={value}
        onChange={(event) => onChange(event.target.value)}
        placeholder={placeholder}
        readOnly={readOnly}
        className={`w-full rounded-2xl border bg-white px-4 py-3 text-sm text-gray-800 outline-none placeholder:text-gray-400 dark:bg-gray-900/40 dark:text-white dark:placeholder:text-gray-500 ${
          error
            ? "border-error-400 dark:border-error-500"
            : "border-gray-200 dark:border-gray-700"
        }`}
      />
      {error ? <p className="text-xs text-error-600 dark:text-error-300">{error}</p> : null}
    </div>
  );
}
