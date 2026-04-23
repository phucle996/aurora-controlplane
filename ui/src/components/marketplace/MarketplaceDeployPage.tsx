"use client";

import { useEffect, useMemo, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import PageBreadcrumb from "@/components/common/PageBreadCrumb";
import ComponentCard from "@/components/common/ComponentCard";
import Input from "@/components/form/input/InputField";
import TextArea from "@/components/form/input/TextArea";
import Badge from "@/components/ui/badge/Badge";
import Button from "@/components/ui/button/Button";
import { useToast } from "@/components/ui/toast/ToastProvider";
import { CloseIcon, PlusIcon } from "@/icons";
import {
  createWorkspaceMarketplaceDeployment,
  getWorkspaceMarketplaceDeployBootstrap,
  type WorkspaceMarketplaceCatalogItem,
  type WorkspaceMarketplaceDeployJob,
  type WorkspaceMarketplaceDeployNamespace,
  type WorkspaceMarketplacePlanOption,
} from "@/components/workspace/api";

type DeployFormState = {
  name: string;
  description: string;
  namespace_id: string;
  plan: string;
  version: string;
};

type SelectedJobState = {
  job: WorkspaceMarketplaceDeployJob;
  schedule: string;
};

type DeployPrefill = {
  name?: string;
  description?: string;
  namespaceId?: string;
  plan?: string;
  version?: string;
};

function buildDefaultForm(
  template: WorkspaceMarketplaceCatalogItem,
  plans: WorkspaceMarketplacePlanOption[],
  namespaces: WorkspaceMarketplaceDeployNamespace[],
  prefill: DeployPrefill,
): DeployFormState {
  const namespaceId =
    (prefill.namespaceId && namespaces.some((item) => item.id === prefill.namespaceId)
      ? prefill.namespaceId
      : "") || namespaces[0]?.id || "";
  const version =
    (prefill.version &&
      template.versions.some((item) => item.resource_version === prefill.version)
      ? prefill.version
      : "") || template.default_version;
  const plan =
    (prefill.plan && plans.some((item) => item.code === prefill.plan) ? prefill.plan : "") ||
    plans[0]?.code ||
    "";

  return {
    name: prefill.name?.trim() || `${template.slug}-workspace`,
    description: prefill.description?.trim() || template.summary,
    namespace_id: namespaceId,
    plan,
    version,
  };
}

function FieldBlock(props: {
  label: string;
  hint?: string;
  children: React.ReactNode;
}) {
  return (
    <div className="space-y-2">
      <div>
        <p className="text-sm font-medium text-gray-800 dark:text-white/90">{props.label}</p>
        {props.hint ? (
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{props.hint}</p>
        ) : null}
      </div>
      {props.children}
    </div>
  );
}

function SummaryLine(props: { label: string; value: string }) {
  return (
    <div className="flex items-start justify-between gap-3 border-b border-dashed border-gray-200 pb-3 text-sm last:border-b-0 last:pb-0 dark:border-gray-800">
      <span className="text-gray-500 dark:text-gray-400">{props.label}</span>
      <span className="text-right font-medium text-gray-900 dark:text-white">{props.value}</span>
    </div>
  );
}

export default function MarketplaceDeployPage() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const { pushToast } = useToast();
  const [template, setTemplate] = useState<WorkspaceMarketplaceCatalogItem | null>(null);
  const [plans, setPlans] = useState<WorkspaceMarketplacePlanOption[]>([]);
  const [namespaces, setNamespaces] = useState<WorkspaceMarketplaceDeployNamespace[]>([]);
  const [jobs, setJobs] = useState<WorkspaceMarketplaceDeployJob[]>([]);
  const [jobDrawerOpen, setJobDrawerOpen] = useState(false);
  const [jobSearch, setJobSearch] = useState("");
  const [selectedJobs, setSelectedJobs] = useState<SelectedJobState[]>([]);
  const [form, setForm] = useState<DeployFormState | null>(null);
  const [loadError, setLoadError] = useState("");
  const [submitting, setSubmitting] = useState(false);

  const resourceKey = searchParams.get("resource") || "";
  const prefill = useMemo<DeployPrefill>(
    () => ({
      name: searchParams.get("name") || "",
      description: searchParams.get("description") || "",
      namespaceId: searchParams.get("namespace_id") || "",
      plan: searchParams.get("plan") || "",
      version: searchParams.get("version") || "",
    }),
    [searchParams],
  );
  const selectedNamespace = useMemo(
    () => namespaces.find((item) => item.id === form?.namespace_id) ?? null,
    [form?.namespace_id, namespaces],
  );
  const selectedVersion = useMemo(
    () =>
      template?.versions.find((item) => item.resource_version === form?.version) ??
      template?.versions[0] ??
      null,
    [form?.version, template],
  );
  const selectedConfiguration = useMemo(
    () => plans.find((item) => item.code === form?.plan) ?? null,
    [form?.plan, plans],
  );
  const filteredJobs = useMemo(() => {
    const query = jobSearch.trim().toLowerCase();
    return jobs.filter((item) => {
      if (query === "") {
        return true;
      }
      return (
        item.display_name.toLowerCase().includes(query) ||
        item.job_name.toLowerCase().includes(query) ||
        item.description.toLowerCase().includes(query) ||
        item.execution_mode.toLowerCase().includes(query)
      );
    });
  }, [jobSearch, jobs]);
  const selectedJobCount = selectedJobs.length;
  const selectedJobModeSummary = useMemo(() => {
    const modes = new Set(selectedJobs.map((item) => item.job.execution_mode));
    if (modes.size === 0) {
      return "-";
    }
    if (modes.size === 1) {
      return Array.from(modes)[0];
    }
    return "mixed";
  }, [selectedJobs]);

  useEffect(() => {
    let cancelled = false;

    async function load() {
      if (resourceKey.trim() === "") {
        if (!cancelled) {
          setLoadError("Select a package from the marketplace first.");
        }
        return;
      }

      try {
        const metadata = await getWorkspaceMarketplaceDeployBootstrap(resourceKey);
        if (cancelled) {
          return;
        }
        setTemplate(metadata.resource);
        setPlans(metadata.plans);
        setNamespaces(metadata.namespaces);
        setJobs(metadata.jobs);
        setForm(buildDefaultForm(metadata.resource, metadata.plans, metadata.namespaces, prefill));
        setLoadError("");
      } catch (error) {
        if (!cancelled) {
          setLoadError(
            error instanceof Error
              ? error.message
              : "Deployment metadata is not available yet.",
          );
        }
      }
    }

    void load();
    return () => {
      cancelled = true;
    };
  }, [prefill, resourceKey]);

  useEffect(() => {
    if (!template || !form) {
      return;
    }
    const hasVersion = template.versions.some((item) => item.resource_version === form.version);
    if (hasVersion) {
      return;
    }
    const fallbackVersion =
      template.default_version || template.versions[0]?.resource_version || "";
    if (!fallbackVersion || fallbackVersion === form.version) {
      return;
    }
    setForm((current) => (current ? { ...current, version: fallbackVersion } : current));
  }, [form, template]);

  useEffect(() => {
    setSelectedJobs((current) =>
      current
        .map((item) => {
          const nextJob = jobs.find((job) => job.id === item.job.id);
          if (!nextJob) {
            return null;
          }
          return {
            job: nextJob,
            schedule: nextJob.execution_mode === "cron" ? item.schedule : "",
          };
        })
        .filter((item): item is SelectedJobState => item !== null),
    );
  }, [jobs]);

  async function submitDeployment() {
    if (!template || !selectedVersion || !form || !form.name.trim() || !form.namespace_id) {
      pushToast({
        kind: "error",
        message: "Deployment name, namespace, and version are required.",
      });
      return;
    }
    if (plans.length > 0 && !form.plan.trim()) {
      pushToast({
        kind: "error",
        message: "Choose a published plan for this deployment.",
      });
      return;
    }
    const missingScheduleJob = selectedJobs.find(
      (item) => item.job.execution_mode === "cron" && !item.schedule.trim(),
    );
    if (missingScheduleJob) {
      pushToast({
        kind: "error",
        message: `Schedule is required for ${missingScheduleJob.job.display_name}.`,
      });
      return;
    }

    try {
      setSubmitting(true);
      await createWorkspaceMarketplaceDeployment({
        resource_definition_id: selectedVersion.resource_definition_id,
        template_id: template.template_id,
        namespace_id: form.namespace_id,
        name: form.name.trim(),
        plan: form.plan,
        version: form.version,
        params: selectedJobs.length > 0
          ? {
              jobs: selectedJobs.map((item) => ({
                job_definition_id: item.job.id,
                job_name: item.job.job_name,
                display_name: item.job.display_name,
                execution_mode: item.job.execution_mode,
                schedule: item.job.execution_mode === "cron" ? item.schedule.trim() : "",
              })),
            }
          : undefined,
      });
      pushToast({
        kind: "success",
        message: `${form.name.trim()} queued for deployment in ${selectedNamespace?.display_name ?? "the selected namespace"}.`,
      });
      router.push("/marketplace");
    } catch (error) {
      pushToast({
        kind: "error",
        message:
          error instanceof Error
            ? error.message
            : "Failed to queue marketplace deployment.",
      });
    } finally {
      setSubmitting(false);
    }
  }

  function toggleConfiguration(code: string) {
    setForm((current) => (current ? { ...current, plan: code } : current));
  }

  function toggleVersion(version: string) {
    setForm((current) => (current ? { ...current, version } : current));
  }

  function toggleJob(job: WorkspaceMarketplaceDeployJob) {
    setSelectedJobs((current) => {
      const exists = current.find((item) => item.job.id === job.id);
      if (exists) {
        return current.filter((item) => item.job.id !== job.id);
      }
      return [
        ...current,
        {
          job,
          schedule: job.execution_mode === "cron" ? "" : "",
        },
      ];
    });
  }

  function updateJobSchedule(jobID: string, schedule: string) {
    setSelectedJobs((current) =>
      current.map((item) =>
        item.job.id === jobID ? { ...item, schedule } : item,
      ),
    );
  }

  function removeJob(jobID: string) {
    setSelectedJobs((current) => current.filter((item) => item.job.id !== jobID));
  }

  if (loadError) {
    return (
      <div className="space-y-6">
        <PageBreadcrumb pageTitle="Deploy package" />
        <section className="rounded-3xl border border-gray-200 bg-white p-6 dark:border-gray-800 dark:bg-white/[0.03]">
          <div className="space-y-4">
            <h1 className="text-2xl font-semibold text-gray-900 dark:text-white">
              Deploy package
            </h1>
            <p className="text-sm leading-7 text-gray-500 dark:text-gray-400">{loadError}</p>
            <Button className="rounded-xl px-5" onClick={() => router.push("/marketplace")}>
              Back to marketplace
            </Button>
          </div>
        </section>
      </div>
    );
  }

  if (!template || !form) {
    return null;
  }

  return (
    <div className="space-y-6">
      <PageBreadcrumb pageTitle="Deploy package" />

      <section className="grid gap-6 xl:grid-cols-[minmax(0,1.6fr)_360px]">
        <div className="space-y-6">
          <ComponentCard
            title="Deployment identity"
            desc="Name the deployment and keep a short note for operators."
          >
            <div className="grid gap-5">
              <FieldBlock label="Deployment name">
                <Input
                  value={form.name}
                  onChange={(event) =>
                    setForm((current) =>
                      current ? { ...current, name: event.target.value } : current,
                    )
                  }
                  placeholder="n8n-workspace"
                />
              </FieldBlock>

              <FieldBlock label="Description">
                <TextArea
                  rows={4}
                  value={form.description}
                  onChange={(value) =>
                    setForm((current) =>
                      current ? { ...current, description: value } : current,
                    )
                  }
                  placeholder="Describe who uses this deployment and what it serves."
                />
              </FieldBlock>
            </div>
          </ComponentCard>

          <ComponentCard
            title="Plan"
            desc="Choose a deployment configuration published for this resource model."
          >
            <div className="space-y-3">
              <p className="text-xs font-semibold uppercase tracking-[0.18em] text-gray-400 dark:text-gray-500">
                Published configurations
              </p>
              {plans.length > 0 ? (
                <div className="grid gap-3">
                  {plans.map((item) => {
                    const isSelected = form.plan === item.code;
                    return (
                      <button
                        key={item.code}
                        type="button"
                        onClick={() => toggleConfiguration(item.code)}
                        className={`rounded-2xl border p-4 text-left transition ${
                          isSelected
                            ? "border-brand-400 bg-brand-50 shadow-sm dark:border-brand-500/40 dark:bg-brand-500/10"
                            : "border-gray-200 bg-white hover:border-brand-300 hover:bg-brand-50/40 dark:border-gray-800 dark:bg-gray-950/40 dark:hover:border-brand-500/30 dark:hover:bg-brand-500/5"
                        }`}
                      >
                        <div className="flex items-center justify-between gap-3">
                          <div>
                            <p className="text-sm font-semibold text-gray-900 dark:text-white">
                              {item.name}
                            </p>
                            <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
                              {item.code}
                            </p>
                          </div>
                          {isSelected ? <Badge color="success">Selected</Badge> : null}
                        </div>
                      </button>
                    );
                  })}
                </div>
              ) : (
                <div className="rounded-2xl border border-dashed border-gray-200 bg-gray-50 p-5 dark:border-gray-800 dark:bg-gray-900/40">
                  <div className="flex items-start justify-between gap-3">
                    <div>
                      <p className="text-sm font-semibold text-gray-900 dark:text-white">
                        No published configurations yet
                      </p>
                      <p className="mt-1 text-sm leading-6 text-gray-500 dark:text-gray-400">
                        This package is looking for plan entries published for{" "}
                        {template.resource_type} / {template.resource_model}. Ask an admin to
                        publish a matching configuration before deploying.
                      </p>
                    </div>
                    <Badge color="warning">{template.resource_model || "-"}</Badge>
                  </div>
                  <div className="mt-4 flex flex-wrap gap-2">
                    <Badge color="light">{template.resource_type}</Badge>
                    <Badge color="light">{template.resource_model}</Badge>
                  </div>
                </div>
              )}
            </div>
          </ComponentCard>

          <ComponentCard
            title="Deployment shape"
            desc="Choose the workspace namespace and package version for this rollout."
          >
            <div className="space-y-5">
              <div className="grid gap-5 lg:grid-cols-2">
                <FieldBlock
                  label="Workspace namespace"
                  hint="Deployments always target a workspace namespace."
                >
                  <select
                    value={form.namespace_id}
                    onChange={(event) =>
                      setForm((current) =>
                        current ? { ...current, namespace_id: event.target.value } : current,
                      )
                    }
                    className="h-11 w-full rounded-lg border border-gray-300 bg-transparent px-4 text-sm text-gray-800 shadow-theme-xs outline-none transition focus:border-brand-300 focus:ring-3 focus:ring-brand-500/10 dark:border-gray-700 dark:bg-gray-900 dark:text-white/90 dark:focus:border-brand-800"
                    disabled={namespaces.length === 0}
                  >
                    {namespaces.map((item) => (
                      <option key={item.id} value={item.id}>
                        {item.display_name}
                      </option>
                    ))}
                  </select>
                </FieldBlock>

                <FieldBlock
                  label="Package version"
                  hint="Select the exact model version that should ship with this package."
                >
                  <select
                    value={form.version}
                    onChange={(event) => toggleVersion(event.target.value)}
                    className="h-11 w-full rounded-lg border border-gray-300 bg-transparent px-4 text-sm text-gray-800 shadow-theme-xs outline-none transition focus:border-brand-300 focus:ring-3 focus:ring-brand-500/10 dark:border-gray-700 dark:bg-gray-900 dark:text-white/90 dark:focus:border-brand-800"
                    disabled={template.versions.length === 0}
                  >
                    {template.versions.map((item) => {
                      const isDefault = template.default_version === item.resource_version;
                      return (
                        <option key={item.resource_definition_id} value={item.resource_version}>
                          {item.resource_version || "Version"}
                          {isDefault ? " (recommended)" : ""}
                        </option>
                      );
                    })}
                  </select>
                </FieldBlock>
              </div>

              {namespaces.length === 0 ? (
                <div className="mt-5 rounded-2xl border border-warning-200 bg-warning-50 px-4 py-4 text-sm text-warning-700 dark:border-warning-500/30 dark:bg-warning-500/10 dark:text-warning-400">
                  Create a workspace namespace before deploying a package.
                </div>
              ) : null}
            </div>
          </ComponentCard>

          <ComponentCard
            title="Job"
            desc="Attach one or more published jobs to this deployment."
          >
            <div className="space-y-4">
              <div className="flex flex-wrap items-start justify-between gap-4">
                <div className="space-y-1">
                  <p className="text-sm font-medium text-gray-900 dark:text-white">
                    Selected jobs
                  </p>
                  <p className="text-sm leading-6 text-gray-500 dark:text-gray-400">
                    Filtered by the selected package and version.
                  </p>
                </div>

                <Button
                  variant="outline"
                  className="rounded-xl px-4"
                  onClick={() => setJobDrawerOpen(true)}
                  startIcon={<PlusIcon className="size-4" />}
                >
                  Add job
                </Button>
              </div>

              {selectedJobs.length === 0 ? (
                <div className="rounded-2xl border border-dashed border-gray-200 bg-gray-50 p-5 text-sm text-gray-500 dark:border-gray-800 dark:bg-gray-900/50 dark:text-gray-400">
                  No jobs selected yet. Use Add job to attach one or more scheduled jobs.
                </div>
              ) : null}

              {selectedJobs.length > 0 ? (
                <div className="space-y-3">
                  {selectedJobs.map((selectedJob) => (
                    <div
                      key={selectedJob.job.id}
                      className="rounded-2xl border border-gray-200 bg-white p-4 dark:border-gray-800 dark:bg-white/[0.03]"
                    >
                      <div className="flex items-start justify-between gap-3">
                        <div className="space-y-1">
                          <p className="text-sm font-semibold text-gray-900 dark:text-white">
                            {selectedJob.job.display_name}
                          </p>
                          <p className="text-xs text-gray-500 dark:text-gray-400">
                            {selectedJob.job.job_name}
                          </p>
                        </div>
                        <div className="flex items-center gap-2">
                          <Badge color={selectedJob.job.execution_mode === "cron" ? "warning" : "primary"}>
                            {selectedJob.job.execution_mode}
                          </Badge>
                          <Button
                            variant="outline"
                            className="h-9 rounded-full px-3"
                            onClick={() => removeJob(selectedJob.job.id)}
                          >
                            Remove
                          </Button>
                        </div>
                      </div>
                      <p className="mt-3 text-sm leading-6 text-gray-500 dark:text-gray-400">
                        {selectedJob.job.description || "No description provided."}
                      </p>
                      {selectedJob.job.execution_mode === "cron" ? (
                        <div className="mt-4 rounded-2xl border border-gray-200 bg-gray-50 p-4 dark:border-gray-800 dark:bg-gray-900/50">
                          <FieldBlock
                            label="Schedule"
                            hint="Cron jobs run on this schedule. Use standard cron syntax or @once."
                          >
                            <Input
                              value={selectedJob.schedule}
                              onChange={(event) =>
                                updateJobSchedule(selectedJob.job.id, event.target.value)
                              }
                              placeholder="0 2 * * *"
                            />
                          </FieldBlock>
                        </div>
                      ) : null}
                    </div>
                  ))}
                </div>
              ) : null}
            </div>
          </ComponentCard>
        </div>

        <aside className="rounded-3xl border border-gray-200 bg-white p-6 dark:border-gray-800 dark:bg-white/[0.03]">
          <p className="text-xs font-semibold uppercase tracking-[0.24em] text-gray-400 dark:text-gray-500">
            Deployment summary
          </p>
          <div className="mt-5 space-y-4">
            <SummaryLine label="Configuration" value={selectedConfiguration?.name ?? form.plan ?? "-"} />
            <SummaryLine label="Version" value={form.version} />
            <SummaryLine label="Namespace" value={selectedNamespace?.display_name ?? "-"} />
            <SummaryLine label="Jobs" value={String(selectedJobCount)} />
            <SummaryLine label="Job mode" value={selectedJobModeSummary} />
            {selectedJobs.length > 0 ? (
              <div className="space-y-3 pt-2">
                {selectedJobs.map((selectedJob) => (
                  <div
                    key={selectedJob.job.id}
                    className="rounded-2xl border border-gray-200 bg-gray-50 p-4 dark:border-gray-800 dark:bg-gray-900/50"
                  >
                    <div className="flex items-center justify-between gap-3">
                      <p className="text-sm font-semibold text-gray-900 dark:text-white">
                        {selectedJob.job.display_name}
                      </p>
                      <Badge color={selectedJob.job.execution_mode === "cron" ? "warning" : "primary"}>
                        {selectedJob.job.execution_mode}
                      </Badge>
                    </div>
                    {selectedJob.job.execution_mode === "cron" ? (
                      <p className="mt-2 text-sm text-gray-500 dark:text-gray-400">
                        Schedule: {selectedJob.schedule || "-"}
                      </p>
                    ) : (
                      <p className="mt-2 text-sm text-gray-500 dark:text-gray-400">
                        Manual job
                      </p>
                    )}
                  </div>
                ))}
              </div>
            ) : null}
          </div>

          <div className="mt-8 grid gap-3">
            <Button
              className="w-full justify-center rounded-xl"
              onClick={submitDeployment}
              disabled={submitting || namespaces.length === 0}
            >
              {submitting ? "Queueing..." : "Queue deployment"}
            </Button>
            <Button
              variant="outline"
              className="w-full justify-center rounded-xl"
              onClick={() => router.push("/marketplace")}
            >
              Cancel
            </Button>
          </div>
        </aside>
      </section>

      {jobDrawerOpen ? (
        <div className="fixed inset-0 z-50">
          <button
            type="button"
            className="absolute inset-0 bg-black/50 backdrop-blur-[2px]"
            aria-label="Close job drawer"
            onClick={() => setJobDrawerOpen(false)}
          />
          <aside className="absolute right-0 top-0 flex h-full w-full max-w-[520px] flex-col border-l border-gray-200 bg-white shadow-2xl dark:border-gray-800 dark:bg-gray-950">
            <div className="flex items-center justify-between border-b border-gray-200 px-6 py-5 dark:border-gray-800">
              <div>
                <p className="text-lg font-semibold text-gray-900 dark:text-white">Add jobs</p>
                <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
                  Choose one or more jobs and set a schedule for each cron job.
                </p>
              </div>
              <Button
                variant="outline"
                className="h-10 rounded-full px-3"
                onClick={() => setJobDrawerOpen(false)}
                startIcon={<CloseIcon className="size-4" />}
              >
                Close
              </Button>
            </div>

            <div className="border-b border-gray-200 px-6 py-4 dark:border-gray-800">
              <Input
                type="text"
                value={jobSearch}
                onChange={(event) => setJobSearch(event.target.value)}
                placeholder="Search job name, description, model, or mode..."
              />
            </div>

            <div className="flex-1 overflow-y-auto px-6 py-5">
              {filteredJobs.length === 0 ? (
                <div className="rounded-2xl border border-dashed border-gray-200 px-5 py-10 text-sm text-gray-500 dark:border-gray-800 dark:text-gray-400">
                  No jobs match this package.
                </div>
              ) : (
                <div className="space-y-3">
                  {filteredJobs.map((job) => {
                    const selectedJob = selectedJobs.find((item) => item.job.id === job.id);
                    const isSelected = Boolean(selectedJob);
                    const isCron = job.execution_mode === "cron";
                    return (
                      <div
                        key={job.id}
                        className={`rounded-2xl border p-4 transition ${
                          isSelected
                            ? "border-brand-400 bg-brand-50 shadow-sm dark:border-brand-500/40 dark:bg-brand-500/10"
                            : "border-gray-200 bg-white hover:border-brand-300 hover:bg-brand-50/40 dark:border-gray-800 dark:bg-gray-950/40 dark:hover:border-brand-500/30 dark:hover:bg-brand-500/5"
                        }`}
                      >
                        <div className="flex items-start gap-3">
                          <input
                            type="checkbox"
                            className="mt-1 h-4 w-4 rounded border-gray-300 text-brand-500 focus:ring-brand-500"
                            checked={isSelected}
                            onChange={() => toggleJob(job)}
                          />
                          <div className="min-w-0 flex-1">
                            <div className="flex items-start justify-between gap-3">
                              <div className="space-y-1">
                                <p className="text-sm font-semibold text-gray-900 dark:text-white">
                                  {job.display_name}
                                </p>
                                <p className="text-xs text-gray-500 dark:text-gray-400">
                                  {job.job_name}
                                </p>
                              </div>
                              <Badge color={isCron ? "warning" : "primary"}>{job.execution_mode}</Badge>
                            </div>
                            <p className="mt-3 line-clamp-2 text-sm leading-6 text-gray-500 dark:text-gray-400">
                              {job.description || "No description provided."}
                            </p>
                            {isSelected && isCron ? (
                              <div className="mt-4 rounded-2xl border border-gray-200 bg-white p-4 dark:border-gray-800 dark:bg-gray-950/40">
                                <FieldBlock
                                  label="Schedule"
                                  hint="Each cron job can have its own schedule."
                                >
                                  <Input
                                    value={selectedJob?.schedule ?? ""}
                                    onChange={(event) =>
                                      updateJobSchedule(job.id, event.target.value)
                                    }
                                    placeholder="0 2 * * *"
                                  />
                                </FieldBlock>
                              </div>
                            ) : null}
                          </div>
                        </div>
                      </div>
                    );
                  })}
                </div>
              )}
            </div>
          </aside>
        </div>
      ) : null}
    </div>
  );
}
