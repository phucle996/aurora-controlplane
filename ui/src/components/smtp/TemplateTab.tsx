"use client";

import type { ReactNode } from "react";
import { useEffect, useMemo, useState } from "react";
import { useRouter } from "next/navigation";
import { parseAPIError } from "@/components/auth/auth-utils";
import ComponentCard from "@/components/common/ComponentCard";
import type { TemplateItem } from "@/components/smtp/types";

type TemplateListResponse = {
  items?: Array<{
    id: string;
    name: string;
    category: string;
    traffic_class: string;
    subject: string;
    from_email: string;
    to_email: string;
    status: string;
    variables: string[];
    consumer_id: string;
    consumer_name: string;
    text_body: string;
    html_body: string;
    active_version: number;
    runtime_version: number;
    created_at: string;
    updated_at: string;
  }>;
};

type TemplateSummaryItem = NonNullable<TemplateListResponse["items"]>[number];

type TemplateDetailResponse = {
  id: string;
  name: string;
  category: string;
  traffic_class: string;
  subject: string;
  from_email: string;
  to_email: string;
  status: string;
  variables: string[];
  consumer_id: string;
  consumer_name: string;
  text_body: string;
  html_body: string;
  active_version: number;
  runtime_version: number;
  created_at: string;
  updated_at: string;
};

type ConsumerListResponse = {
  items?: Array<{
    id: string;
    name: string;
  }>;
};

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
  variablesText: string;
};

const trafficClassOptions = [
  { value: "critical_auth", label: "Critical Auth" },
  { value: "transactional", label: "Transactional" },
  { value: "bulk_marketing", label: "Bulk Marketing" },
];

const statusOptions = [
  { value: "draft", label: "Draft" },
  { value: "review", label: "Review" },
  { value: "live", label: "Live" },
];

export function TemplateTab({
  search,
  onSearchChange,
}: {
  search: string;
  onSearchChange: (value: string) => void;
}) {
  const router = useRouter();
  const [templates, setTemplates] = useState<TemplateItem[]>([]);
  const [selectedTemplateId, setSelectedTemplateId] = useState("");
  const [selectedTemplate, setSelectedTemplate] = useState<TemplateItem | null>(null);
  const [consumerOptions, setConsumerOptions] = useState<Array<{ id: string; name: string }>>([]);
  const [form, setForm] = useState<TemplateFormState>(emptyTemplateForm());
  const [isEditing, setIsEditing] = useState(false);
  const [isSaving, setIsSaving] = useState(false);
  const [isDeleting, setIsDeleting] = useState(false);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState("");
  const [isDetailLoading, setIsDetailLoading] = useState(false);
  const [detailError, setDetailError] = useState("");

  useEffect(() => {
    let cancelled = false;

    async function loadData() {
      setIsLoading(true);
      setError("");

      try {
        const [templateResponse, consumerResponse] = await Promise.all([
          fetch("/api/v1/smtp/templates", {
            method: "GET",
            cache: "no-store",
          }),
          fetch("/api/v1/smtp/consumers", {
            method: "GET",
            cache: "no-store",
          }),
        ]);

        if (!templateResponse.ok) {
          throw new Error("Failed to load SMTP templates");
        }
        if (!consumerResponse.ok) {
          throw new Error("Failed to load SMTP consumers");
        }

        const templateResult = (await templateResponse.json()) as TemplateListResponse;
        const consumerResult = (await consumerResponse.json()) as ConsumerListResponse;
        if (cancelled) {
          return;
        }

        const nextTemplates = (templateResult.items ?? []).map(mapTemplateResponse);
        setTemplates(nextTemplates);
        setConsumerOptions(consumerResult.items ?? []);

        if (nextTemplates.length === 0) {
          setSelectedTemplateId("");
          setSelectedTemplate(null);
          setForm(emptyTemplateForm());
          return;
        }

        const hasSelected = nextTemplates.some((template) => template.id === selectedTemplateId);
        if (!hasSelected) {
          setSelectedTemplateId(nextTemplates[0].id);
        }
      } catch (err) {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : "Failed to load SMTP templates");
        }
      } finally {
        if (!cancelled) {
          setIsLoading(false);
        }
      }
    }

    void loadData();
    return () => {
      cancelled = true;
    };
  }, []);

  useEffect(() => {
    if (selectedTemplateId === "") {
      setSelectedTemplate(null);
      setForm(emptyTemplateForm());
      setDetailError("");
      setIsEditing(false);
      return;
    }

    let cancelled = false;

    async function loadDetail() {
      setIsDetailLoading(true);
      setDetailError("");

      try {
        const detailResponse = await fetch(`/api/v1/smtp/templates/${selectedTemplateId}`, {
          method: "GET",
          cache: "no-store",
        });
        if (!detailResponse.ok) {
          throw new Error("Failed to load SMTP template detail");
        }

        const result = (await detailResponse.json()) as TemplateDetailResponse;
        if (cancelled) {
          return;
        }

        const next = mapTemplateResponse(result);
        setSelectedTemplate(next);
        setForm(templateToForm(next));
        setIsEditing(false);
      } catch (err) {
        if (!cancelled) {
          setDetailError(
            err instanceof Error ? err.message : "Failed to load SMTP template detail",
          );
        }
      } finally {
        if (!cancelled) {
          setIsDetailLoading(false);
        }
      }
    }

    void loadDetail();
    return () => {
      cancelled = true;
    };
  }, [selectedTemplateId]);

  const keyword = search.trim().toLowerCase();
  const filteredTemplates = useMemo(
    () =>
      templates.filter((template) => {
        if (keyword === "") {
          return true;
        }

        return (
          template.name.toLowerCase().includes(keyword) ||
          template.category.toLowerCase().includes(keyword) ||
          template.subject.toLowerCase().includes(keyword) ||
          template.status.toLowerCase().includes(keyword) ||
          template.variables.some((variable) => variable.toLowerCase().includes(keyword))
        );
      }),
    [templates, keyword],
  );

  useEffect(() => {
    if (filteredTemplates.length === 0) {
      if (selectedTemplateId !== "") {
        setSelectedTemplateId("");
      }
      return;
    }
    const hasSelected = filteredTemplates.some((template) => template.id === selectedTemplateId);
    if (!hasSelected) {
      setSelectedTemplateId(filteredTemplates[0].id);
    }
  }, [filteredTemplates, selectedTemplateId]);

  function updateForm<K extends keyof TemplateFormState>(key: K, value: TemplateFormState[K]) {
    setForm((current) => ({ ...current, [key]: value }));
    setDetailError("");
  }

  async function handleSave() {
    if (selectedTemplate == null) {
      return;
    }

    setIsSaving(true);
    setDetailError("");

    try {
      const response = await fetch(`/api/v1/smtp/templates/${selectedTemplate.id}`, {
        method: "PUT",
        headers: {
          "Content-Type": "application/json",
        },
        credentials: "include",
        body: JSON.stringify({
          name: form.name.trim(),
          category: form.category.trim(),
          traffic_class: form.trafficClass.trim(),
          subject: form.subject.trim(),
          from_email: form.fromEmail.trim(),
          to_email: form.toEmail.trim(),
          status: form.status.trim(),
          consumer_id: form.consumerID.trim(),
          body: form.body,
          variables: splitVariables(form.variablesText),
        }),
      });

      if (!response.ok) {
        throw new Error(await parseAPIError(response));
      }

      const result = (await response.json()) as TemplateDetailResponse;
      const next = mapTemplateResponse(result);
      setSelectedTemplate(next);
      setTemplates((current) =>
        current.map((item) => (item.id === next.id ? next : item)),
      );
      setForm(templateToForm(next));
      setIsEditing(false);
    } catch (err) {
      setDetailError(err instanceof Error ? err.message : "Failed to save SMTP template");
    } finally {
      setIsSaving(false);
    }
  }

  async function handleDelete() {
    if (selectedTemplate == null) {
      return;
    }
    if (!window.confirm(`Delete template "${selectedTemplate.name}"?`)) {
      return;
    }

    setIsDeleting(true);
    setDetailError("");

    try {
      const response = await fetch(`/api/v1/smtp/templates/${selectedTemplate.id}`, {
        method: "DELETE",
        credentials: "include",
      });
      if (!response.ok) {
        throw new Error(await parseAPIError(response));
      }

      const deletedID = selectedTemplate.id;
      const nextTemplates = templates.filter((item) => item.id !== deletedID);
      setTemplates(nextTemplates);
      setSelectedTemplate(null);
      setForm(emptyTemplateForm());
      setSelectedTemplateId(nextTemplates[0]?.id ?? "");
      setIsEditing(false);
    } catch (err) {
      setDetailError(err instanceof Error ? err.message : "Failed to delete SMTP template");
    } finally {
      setIsDeleting(false);
    }
  }

  return (
    <div className="space-y-6">
      <ComponentCard
        title="Template Library"
        desc="Transactional templates are loaded from the SMTP schema and edited from one unified form."
      >
        <div className="grid gap-4 xl:grid-cols-[3fr_7fr]">
          <div className="space-y-3">
            <div className="flex items-center gap-3">
              <div className="max-w-[320px] flex-1 rounded-2xl border border-gray-200 bg-gray-50 px-4 py-3 dark:border-gray-800 dark:bg-gray-900/40">
                <input
                  type="text"
                  value={search}
                  onChange={(event) => onSearchChange(event.target.value)}
                  placeholder="Search templates by name, tag, variable, or status"
                  className="w-full bg-transparent text-sm text-gray-800 outline-none placeholder:text-gray-400 dark:text-white dark:placeholder:text-gray-500"
                />
              </div>
              <button
                type="button"
                onClick={() => router.push("/smtp/templates/new")}
                className="inline-flex items-center rounded-xl bg-gray-900 px-4 py-3 text-sm font-semibold text-white transition hover:bg-gray-800 dark:bg-white dark:text-gray-900 dark:hover:bg-gray-200"
              >
                Add Template
              </button>
            </div>

            {isLoading ? (
              <div className="rounded-2xl border border-gray-200 bg-gray-50 px-5 py-5 dark:border-gray-800 dark:bg-gray-900/40">
                <p className="text-sm text-gray-500 dark:text-gray-400">Loading templates...</p>
              </div>
            ) : error !== "" ? (
              <div className="rounded-2xl border border-error-200 bg-error-50 px-5 py-5 dark:border-error-500/30 dark:bg-error-500/10">
                <p className="text-sm text-error-700 dark:text-error-300">{error}</p>
              </div>
            ) : filteredTemplates.length === 0 ? (
              <div className="rounded-2xl border border-gray-200 bg-gray-50 px-5 py-5 dark:border-gray-800 dark:bg-gray-900/40">
                <p className="text-sm text-gray-500 dark:text-gray-400">
                  No template matches your search.
                </p>
              </div>
            ) : (
              filteredTemplates.map((template) => {
                const active = selectedTemplateId === template.id;
                return (
                  <div
                    key={template.id}
                    className={`w-full rounded-2xl border px-4 py-4 transition ${
                      active
                        ? "border-gray-900 bg-white shadow-theme-sm dark:border-white dark:bg-gray-900"
                        : "border-gray-200 bg-gray-50 dark:border-gray-800 dark:bg-gray-900/40"
                    }`}
                  >
                    <button
                      type="button"
                      onClick={() => setSelectedTemplateId(template.id)}
                      className="w-full text-left"
                    >
                      <div className="flex items-start justify-between gap-3">
                        <div className="min-w-0">
                          <h3 className="text-sm font-semibold text-gray-900 dark:text-white">
                            {template.name}
                          </h3>
                          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
                            {template.category}
                          </p>
                        </div>
                        <span className="rounded-full bg-white px-2.5 py-1 text-[11px] font-semibold text-gray-700 shadow-theme-xs dark:bg-white/5 dark:text-gray-300">
                          {template.status}
                        </span>
                      </div>
                    </button>

                    {active ? (
                      <div className="mt-4 flex flex-wrap gap-2 border-t border-gray-200 pt-4 dark:border-gray-800">
                        <button
                          type="button"
                          onClick={() => {
                            setForm(templateToForm(template));
                            setIsEditing(true);
                            setDetailError("");
                          }}
                          className="inline-flex items-center rounded-xl border border-gray-200 bg-white px-3 py-2 text-xs font-semibold text-gray-700 transition hover:bg-gray-50 dark:border-gray-700 dark:bg-gray-900 dark:text-gray-200 dark:hover:bg-gray-800"
                        >
                          Edit
                        </button>
                        <button
                          type="button"
                          onClick={() => void handleDelete()}
                          disabled={isDeleting}
                          className="inline-flex items-center rounded-xl border border-error-200 bg-error-50 px-3 py-2 text-xs font-semibold text-error-700 transition hover:bg-error-100 disabled:cursor-not-allowed disabled:opacity-60 dark:border-error-500/30 dark:bg-error-500/10 dark:text-error-300 dark:hover:bg-error-500/20"
                        >
                          {isDeleting ? "Deleting..." : "Delete"}
                        </button>
                      </div>
                    ) : null}
                  </div>
                );
              })
            )}
          </div>

          <div className="rounded-2xl border border-gray-200 bg-gray-50 p-5 dark:border-gray-800 dark:bg-gray-900/40">
            {isLoading ? (
              <p className="text-sm text-gray-500 dark:text-gray-400">
                Loading template detail...
              </p>
            ) : detailError !== "" ? (
              <p className="text-sm text-error-700 dark:text-error-300">{detailError}</p>
            ) : selectedTemplate != null ? (
              <div className="space-y-5">
                <div className="flex flex-wrap items-start justify-between gap-3">
                  <div>
                    <p className="text-xs font-medium tracking-[0.2em] text-gray-400 uppercase">
                      Template Detail
                    </p>
                    <h3 className="mt-2 text-xl font-semibold text-gray-900 dark:text-white">
                      {selectedTemplate.name}
                    </h3>
                    <p className="mt-2 text-sm text-gray-500 dark:text-gray-400">
                      {isEditing
                        ? "Update the fields below, then save once at the bottom."
                        : "Select Edit from the chosen template card to change this template."}
                    </p>
                  </div>
                  <div className="flex flex-wrap gap-2">
                    <TemplateTag>{selectedTemplate.category}</TemplateTag>
                    <TemplateTag>{selectedTemplate.trafficClass}</TemplateTag>
                    <TemplateTag>{selectedTemplate.status}</TemplateTag>
                  </div>
                </div>

                <div className="grid gap-4 xl:grid-cols-2">
                  <Field
                    label="Template Name"
                    value={form.name}
                    onChange={(value) => updateForm("name", value)}
                    readOnly={!isEditing}
                  />
                  <Field
                    label="Category"
                    value={form.category}
                    onChange={(value) => updateForm("category", value)}
                    readOnly={!isEditing}
                  />
                  <SelectField
                    label="Traffic Class"
                    value={form.trafficClass}
                    onChange={(value) => updateForm("trafficClass", value)}
                    options={trafficClassOptions}
                    disabled={!isEditing}
                  />
                  <SelectField
                    label="Status"
                    value={form.status}
                    onChange={(value) => updateForm("status", value)}
                    options={statusOptions}
                    disabled={!isEditing}
                  />
                  <Field
                    label="Subject"
                    value={form.subject}
                    onChange={(value) => updateForm("subject", value)}
                    readOnly={!isEditing}
                  />
                  <Field
                    label="From"
                    value={form.fromEmail}
                    onChange={(value) => updateForm("fromEmail", value)}
                    readOnly={!isEditing}
                  />
                  <Field
                    label="To"
                    value={form.toEmail}
                    onChange={(value) => updateForm("toEmail", value)}
                    readOnly={!isEditing}
                  />
                  <SelectField
                    label="Consumer"
                    value={form.consumerID}
                    onChange={(value) => updateForm("consumerID", value)}
                    options={[
                      { value: "", label: "None" },
                      ...consumerOptions.map((consumer) => ({
                        value: consumer.id,
                        label: consumer.name,
                      })),
                    ]}
                    disabled={!isEditing}
                  />
                </div>

                <div className="space-y-2">
                  <p className="text-xs font-medium tracking-[0.18em] text-gray-400 uppercase">
                    Body
                  </p>
                  <textarea
                    rows={14}
                    value={form.body}
                    onChange={(event) => updateForm("body", event.target.value)}
                    readOnly={!isEditing}
                    className={`w-full rounded-2xl border px-4 py-3 text-sm text-gray-800 outline-none dark:text-white ${
                      isEditing
                        ? "border-gray-200 bg-white placeholder:text-gray-400 dark:border-gray-700 dark:bg-gray-900"
                        : "border-gray-200 bg-white/70 dark:border-gray-800 dark:bg-gray-900/50"
                    }`}
                  />
                </div>

                {isEditing ? (
                  <div className="flex flex-wrap justify-end gap-3 border-t border-gray-200 pt-5 dark:border-gray-800">
                    <button
                      type="button"
                      onClick={() => {
                        setForm(templateToForm(selectedTemplate));
                        setIsEditing(false);
                        setDetailError("");
                      }}
                      className="inline-flex items-center rounded-xl border border-gray-200 bg-white px-4 py-3 text-sm font-semibold text-gray-700 transition hover:bg-gray-50 dark:border-gray-700 dark:bg-gray-900 dark:text-gray-200 dark:hover:bg-gray-800"
                    >
                      Cancel
                    </button>
                    <button
                      type="button"
                      onClick={() => void handleSave()}
                      disabled={isSaving}
                      className="inline-flex items-center rounded-xl bg-gray-900 px-4 py-3 text-sm font-semibold text-white transition hover:bg-gray-800 disabled:cursor-not-allowed disabled:opacity-60 dark:bg-white dark:text-gray-900 dark:hover:bg-gray-200"
                    >
                      {isSaving ? "Saving..." : "Save Template"}
                    </button>
                  </div>
                ) : null}
              </div>
            ) : (
              <div className="flex h-full min-h-[320px] items-center justify-center rounded-2xl border border-dashed border-gray-300 px-5 py-6 text-sm text-gray-500 dark:border-gray-700 dark:text-gray-400">
                Select a template to inspect it, or create a new one.
              </div>
            )}
          </div>
        </div>
      </ComponentCard>
    </div>
  );
}

function TemplateTag({ children }: { children: ReactNode }) {
  return (
    <span className="rounded-full border border-gray-200 bg-white px-3 py-1 text-xs font-medium text-gray-600 dark:border-gray-700 dark:bg-white/5 dark:text-gray-300">
      {children}
    </span>
  );
}

function Field({
  label,
  value,
  onChange,
  readOnly,
}: {
  label: string;
  value: string;
  onChange: (value: string) => void;
  readOnly: boolean;
}) {
  return (
    <div className="space-y-2">
      <p className="text-xs font-medium tracking-[0.18em] text-gray-400 uppercase">{label}</p>
      <input
        value={value}
        onChange={(event) => onChange(event.target.value)}
        readOnly={readOnly}
        className={`w-full rounded-2xl border px-4 py-3 text-sm text-gray-800 outline-none dark:text-white ${
          readOnly
            ? "border-gray-200 bg-white/70 dark:border-gray-800 dark:bg-gray-900/50"
            : "border-gray-200 bg-white dark:border-gray-700 dark:bg-gray-900"
        }`}
      />
    </div>
  );
}

function SelectField({
  label,
  value,
  onChange,
  options,
  disabled,
}: {
  label: string;
  value: string;
  onChange: (value: string) => void;
  options: Array<{ value: string; label: string }>;
  disabled: boolean;
}) {
  return (
    <div className="space-y-2">
      <p className="text-xs font-medium tracking-[0.18em] text-gray-400 uppercase">{label}</p>
      <select
        value={value}
        onChange={(event) => onChange(event.target.value)}
        disabled={disabled}
        className={`w-full rounded-2xl border px-4 py-3 text-sm text-gray-800 outline-none dark:text-white ${
          disabled
            ? "border-gray-200 bg-white/70 dark:border-gray-800 dark:bg-gray-900/50"
            : "border-gray-200 bg-white dark:border-gray-700 dark:bg-gray-900"
        }`}
      >
        {options.map((option) => (
          <option key={`${label}-${option.value}`} value={option.value}>
            {option.label}
          </option>
        ))}
      </select>
    </div>
  );
}

function mapTemplateResponse(item: TemplateSummaryItem | TemplateDetailResponse): TemplateItem {
  return {
    id: item.id,
    name: item.name,
    category: item.category,
    trafficClass: item.traffic_class || "transactional",
    subject: item.subject,
    from: item.from_email,
    to: item.to_email,
    status: item.status,
    variables: item.variables ?? [],
    consumerId: item.consumer_id || "",
    consumer: item.consumer_name || "none",
    body: item.text_body || item.html_body || "",
    createdAt: item.created_at,
    activeVersion: item.active_version,
    runtimeVersion: item.runtime_version,
    updatedAt: item.updated_at,
  };
}

function templateToForm(template: TemplateItem): TemplateFormState {
  return {
    name: template.name,
    category: template.category,
    trafficClass: template.trafficClass,
    status: template.status,
    consumerID: template.consumerId ?? "",
    subject: template.subject,
    fromEmail: template.from,
    toEmail: template.to,
    body: template.body,
    variablesText: template.variables.join(", "),
  };
}

function emptyTemplateForm(): TemplateFormState {
  return {
    name: "",
    category: "",
    trafficClass: "transactional",
    status: "draft",
    consumerID: "",
    subject: "",
    fromEmail: "",
    toEmail: "",
    body: "",
    variablesText: "",
  };
}

function splitVariables(value: string) {
  return value
    .split(",")
    .map((item) => item.trim())
    .filter((item) => item !== "");
}
