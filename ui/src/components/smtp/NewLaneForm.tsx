"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { parseAPIError } from "@/components/auth/auth-utils";
import { SMTPPageShell } from "@/components/smtp/SMTPPageShell";

type LaneListResponse = {
  items?: Array<{
    id: string;
    name: string;
    traffic_class: string;
  }>;
};

type LaneFormState = {
  name: string;
  trafficClass: string;
  status: string;
  routingMode: string;
  priority: string;
  fallbackLaneID: string;
};

export default function NewLaneForm() {
  const router = useRouter();
  const [form, setForm] = useState<LaneFormState>({
    name: "",
    trafficClass: "transactional",
    status: "disabled",
    routingMode: "priority",
    priority: "100",
    fallbackLaneID: "",
  });
  const [lanes, setLanes] = useState<Array<{ id: string; name: string; trafficClass: string }>>([]);
  const [submitError, setSubmitError] = useState("");
  const [isSaving, setIsSaving] = useState(false);

  useEffect(() => {
    let cancelled = false;

    async function loadLanes() {
      try {
        const response = await fetch("/api/v1/smtp/lanes", { cache: "no-store" });
        if (!response.ok) {
          return;
        }
        const result = (await response.json()) as LaneListResponse;
        if (!cancelled) {
          setLanes(
            (result.items ?? []).map((item) => ({
              id: item.id,
              name: item.name,
              trafficClass: item.traffic_class || "transactional",
            })),
          );
        }
      } catch {
        if (!cancelled) {
          setLanes([]);
        }
      }
    }

    void loadLanes();
    return () => {
      cancelled = true;
    };
  }, []);

  function updateField<K extends keyof LaneFormState>(key: K, value: LaneFormState[K]) {
    setForm((current) => ({ ...current, [key]: value }));
    setSubmitError("");
  }

  async function handleSubmit() {
    setIsSaving(true);
    setSubmitError("");
    try {
      const response = await fetch("/api/v1/smtp/lanes", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        credentials: "include",
        body: JSON.stringify({
          name: form.name.trim(),
          traffic_class: form.trafficClass,
          status: form.status,
          routing_mode: form.routingMode,
          priority: Number(form.priority || "100"),
          fallback_lane_id: form.fallbackLaneID,
        }),
      });
      if (!response.ok) {
        throw new Error(await parseAPIError(response));
      }
      const result = (await response.json()) as { id: string };
      router.push(`/smtp/lanes/detail?id=${result.id}`);
    } catch (err) {
      setSubmitError(err instanceof Error ? err.message : "Failed to create lane");
    } finally {
      setIsSaving(false);
    }
  }

  const fallbackOptions = lanes.filter(
    (lane) => lane.trafficClass === form.trafficClass && lane.id !== form.fallbackLaneID,
  );

  return (
    <SMTPPageShell>
      <div className="space-y-6">
        <div className="rounded-2xl border border-gray-200 bg-white dark:border-gray-800 dark:bg-white/[0.03]">
          <div className="flex items-start justify-between gap-4 px-6 py-5">
            <div>
              <h3 className="text-base font-medium text-gray-800 dark:text-white/90">Create Lane</h3>
              <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
                Provision a new delivery lane for a traffic class, routing mode, and fallback chain.
              </p>
            </div>
            <div className="flex items-center gap-3">
              <button
                type="button"
                onClick={() => router.push("/smtp/lanes")}
                className="inline-flex items-center rounded-xl border border-gray-200 bg-white px-4 py-3 text-sm font-semibold text-gray-700 transition hover:bg-gray-50 dark:border-gray-700 dark:bg-gray-900 dark:text-gray-200 dark:hover:bg-gray-800"
              >
                Cancel
              </button>
              <button
                type="button"
                onClick={() => void handleSubmit()}
                disabled={isSaving}
                className="inline-flex items-center rounded-xl bg-gray-900 px-4 py-3 text-sm font-semibold text-white transition hover:bg-gray-800 disabled:cursor-not-allowed disabled:opacity-60 dark:bg-white dark:text-gray-900 dark:hover:bg-gray-200"
              >
                {isSaving ? "Creating..." : "Create Lane"}
              </button>
            </div>
          </div>

          <div className="space-y-6 border-t border-gray-100 p-4 sm:p-6 dark:border-gray-800">
            {submitError !== "" ? (
              <div className="rounded-2xl border border-error-200 bg-error-50 px-4 py-3 text-sm text-error-700 dark:border-error-500/30 dark:bg-error-500/10 dark:text-error-300">
                {submitError}
              </div>
            ) : null}

            <div className="grid gap-4 xl:grid-cols-2">
              <Field label="Lane Name" value={form.name} onChange={(value) => updateField("name", value)} placeholder="Aurora Transactional Lane" />
              <Field label="Priority" value={form.priority} onChange={(value) => updateField("priority", value)} placeholder="100" />
              <SelectField
                label="Traffic Class"
                value={form.trafficClass}
                onChange={(value) => {
                  updateField("trafficClass", value);
                  updateField("fallbackLaneID", "");
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
                  { value: "priority", label: "Priority" },
                  { value: "weighted", label: "Weighted" },
                ]}
              />
              <SelectField
                label="Fallback Lane"
                value={form.fallbackLaneID}
                onChange={(value) => updateField("fallbackLaneID", value)}
                options={[
                  { value: "", label: "None" },
                  ...fallbackOptions.map((lane) => ({
                    value: lane.id,
                    label: lane.name,
                  })),
                ]}
              />
            </div>
          </div>
        </div>
      </div>
    </SMTPPageShell>
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
