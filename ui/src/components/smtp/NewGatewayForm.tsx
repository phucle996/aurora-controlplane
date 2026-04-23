"use client";

import { useEffect, useMemo, useState } from "react";
import { useRouter } from "next/navigation";
import { SMTPPageShell } from "@/components/smtp/SMTPPageShell";
import { createGateway, listGateways } from "@/components/smtp/api";
import { useSMTPWorkspace } from "@/components/smtp/SMTPWorkspaceProvider";
import type { GatewayItem } from "@/components/smtp/types";

type GatewayFormState = {
  name: string;
  trafficClass: string;
  status: string;
  routingMode: string;
  priority: string;
  desiredShardCount: string;
  fallbackGatewayID: string;
};

export default function NewGatewayForm() {
  return (
    <SMTPPageShell>
      <NewGatewayFormContent />
    </SMTPPageShell>
  );
}

function NewGatewayFormContent() {
  const router = useRouter();
  const { workspace, workspaceID, isLoading: isWorkspaceLoading, error: workspaceError } = useSMTPWorkspace();
  const [form, setForm] = useState<GatewayFormState>({
    name: "",
    trafficClass: "transactional",
    status: "disabled",
    routingMode: "round_robin",
    priority: "100",
    desiredShardCount: "1",
    fallbackGatewayID: "",
  });
  const [gateways, setGateways] = useState<GatewayItem[]>([]);
  const [submitError, setSubmitError] = useState("");
  const [isSaving, setIsSaving] = useState(false);
  const [isLoadingGateways, setIsLoadingGateways] = useState(true);

  useEffect(() => {
    if (workspaceID === "") {
      setGateways([]);
      setIsLoadingGateways(false);
      return;
    }

    let cancelled = false;

    async function load() {
      setIsLoadingGateways(true);
      try {
        const items = await listGateways(workspaceID);
        if (!cancelled) {
          setGateways(items);
        }
      } catch (err) {
        if (!cancelled) {
          setSubmitError(err instanceof Error ? err.message : "Failed to load gateways.");
        }
      } finally {
        if (!cancelled) {
          setIsLoadingGateways(false);
        }
      }
    }

    void load();
    return () => {
      cancelled = true;
    };
  }, [workspaceID]);

  function updateField<K extends keyof GatewayFormState>(key: K, value: GatewayFormState[K]) {
    setForm((current) => ({ ...current, [key]: value }));
    setSubmitError("");
  }

  const fallbackOptions = useMemo(
    () => gateways.filter((gateway) => gateway.trafficClass === form.trafficClass),
    [form.trafficClass, gateways],
  );

  async function handleSubmit() {
    if (workspaceID === "") {
      setSubmitError("Choose a workspace first.");
      return;
    }
    if (workspace?.defaultZoneID == null || workspace.defaultZoneID.trim() === "") {
      setSubmitError("The selected workspace does not have a default zone yet.");
      return;
    }
    if (form.name.trim() === "") {
      setSubmitError("Gateway name is required.");
      return;
    }

    setIsSaving(true);
    setSubmitError("");
    try {
      const result = await createGateway(workspaceID, {
        zone_id: workspace.defaultZoneID,
        name: form.name.trim(),
        traffic_class: form.trafficClass,
        status: form.status,
        routing_mode: form.routingMode,
        priority: Number(form.priority || "100"),
        desired_shard_count: Math.max(1, Number(form.desiredShardCount || "1")),
        fallback_gateway_id: form.fallbackGatewayID,
        template_ids: [],
        endpoint_ids: [],
      });
      router.push(`/smtp/gateways/detail?id=${result.id}&workspace=${workspaceID}`);
    } catch (err) {
      setSubmitError(err instanceof Error ? err.message : "Failed to create gateway");
    } finally {
      setIsSaving(false);
    }
  }

  return (
    <div className="space-y-6">
      <div className="rounded-2xl border border-gray-200 bg-white dark:border-gray-800 dark:bg-white/[0.03]">
          <div className="flex items-start justify-between gap-4 px-6 py-5">
            <div>
              <h3 className="text-base font-medium text-gray-800 dark:text-white/90">Create Gateway</h3>
              <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
                Provision a new workspace-scoped SMTP gateway with shard and fallback policy.
              </p>
            </div>
            <div className="flex items-center gap-3">
              <button
                type="button"
                onClick={() => router.push(workspaceID === "" ? "/smtp/gateways" : `/smtp/gateways?workspace=${workspaceID}`)}
                className="inline-flex items-center rounded-xl border border-gray-200 bg-white px-4 py-3 text-sm font-semibold text-gray-700 transition hover:bg-gray-50 dark:border-gray-700 dark:bg-gray-900 dark:text-gray-200 dark:hover:bg-gray-800"
              >
                Cancel
              </button>
              <button
                type="button"
                onClick={() => void handleSubmit()}
                disabled={isSaving || isWorkspaceLoading}
                className="inline-flex items-center rounded-xl bg-gray-900 px-4 py-3 text-sm font-semibold text-white transition hover:bg-gray-800 disabled:cursor-not-allowed disabled:opacity-60 dark:bg-white dark:text-gray-900 dark:hover:bg-gray-200"
              >
                {isSaving ? "Creating..." : "Create Gateway"}
              </button>
            </div>
          </div>

          <div className="space-y-6 border-t border-gray-100 p-4 sm:p-6 dark:border-gray-800">
            {workspaceError !== "" ? (
              <ErrorPanel message={workspaceError} />
            ) : submitError !== "" ? (
              <ErrorPanel message={submitError} />
            ) : null}

            <div className="rounded-2xl border border-gray-200 bg-gray-50 px-4 py-4 dark:border-gray-800 dark:bg-gray-900/40">
              <p className="text-xs font-medium tracking-[0.18em] text-gray-400 uppercase">Workspace / zone</p>
              <p className="mt-2 text-sm font-semibold text-gray-900 dark:text-white">
                {workspace?.name ?? "No workspace selected"}
              </p>
              <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
                {workspace?.defaultZoneName
                  ? `Default zone: ${workspace.defaultZoneName}`
                  : "This workspace has no default zone yet."}
              </p>
            </div>

            <div className="grid gap-4 xl:grid-cols-2">
              <Field
                label="Gateway Name"
                value={form.name}
                onChange={(value) => updateField("name", value)}
                placeholder="Aurora Transactional Gateway"
              />
              <Field
                label="Priority"
                value={form.priority}
                onChange={(value) => updateField("priority", value)}
                placeholder="100"
              />
              <SelectField
                label="Traffic Class"
                value={form.trafficClass}
                onChange={(value) => {
                  updateField("trafficClass", value);
                  updateField("fallbackGatewayID", "");
                }}
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
                  { value: "disabled", label: "Disabled" },
                  { value: "active", label: "Active" },
                  { value: "draining", label: "Draining" },
                ]}
              />
              <SelectField
                label="Routing Mode"
                value={form.routingMode}
                onChange={(value) => updateField("routingMode", value)}
                options={[
                  { value: "round_robin", label: "Round Robin" },
                  { value: "priority", label: "Priority" },
                  { value: "weighted", label: "Weighted" },
                ]}
              />
              <Field
                label="Desired Shards"
                value={form.desiredShardCount}
                onChange={(value) => updateField("desiredShardCount", value)}
                placeholder="1"
              />
              <SelectField
                label="Fallback Gateway"
                value={form.fallbackGatewayID}
                onChange={(value) => updateField("fallbackGatewayID", value)}
                options={[
                  { value: "", label: isLoadingGateways ? "Loading..." : "None" },
                  ...fallbackOptions.map((gateway) => ({
                    value: gateway.id,
                    label: gateway.name,
                  })),
                ]}
              />
            </div>
          </div>
      </div>
    </div>
  );
}

function Field({
  label,
  value,
  onChange,
  placeholder,
}: {
  label: string;
  value: string;
  onChange: (value: string) => void;
  placeholder: string;
}) {
  return (
    <div className="space-y-2">
      <span className="text-xs font-medium tracking-[0.12em] text-gray-400 uppercase">{label}</span>
      <input
        value={value}
        onChange={(event) => onChange(event.target.value)}
        placeholder={placeholder}
        className="w-full rounded-2xl border border-gray-200 bg-white px-4 py-3 text-sm text-gray-800 outline-none placeholder:text-gray-400 dark:border-gray-700 dark:bg-gray-900/40 dark:text-white dark:placeholder:text-gray-500"
      />
    </div>
  );
}

function SelectField({
  label,
  value,
  onChange,
  options,
}: {
  label: string;
  value: string;
  onChange: (value: string) => void;
  options: Array<{ value: string; label: string }>;
}) {
  return (
    <div className="space-y-2">
      <span className="text-xs font-medium tracking-[0.12em] text-gray-400 uppercase">{label}</span>
      <select
        value={value}
        onChange={(event) => onChange(event.target.value)}
        className="w-full rounded-2xl border border-gray-200 bg-white px-4 py-3 text-sm text-gray-800 outline-none dark:border-gray-700 dark:bg-gray-900/40 dark:text-white"
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

function ErrorPanel({ message }: { message: string }) {
  return (
    <div className="rounded-2xl border border-error-200 bg-error-50 px-4 py-3 text-sm text-error-700 dark:border-error-500/30 dark:bg-error-500/10 dark:text-error-300">
      {message}
    </div>
  );
}
