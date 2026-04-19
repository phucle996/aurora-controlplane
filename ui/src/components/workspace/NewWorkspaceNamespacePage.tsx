"use client";

import { useEffect, useMemo, useState } from "react";
import { useRouter } from "next/navigation";
import PageBreadcrumb from "@/components/common/PageBreadCrumb";
import Input from "@/components/form/input/InputField";
import Button from "@/components/ui/button/Button";
import { useToast } from "@/components/ui/toast/ToastProvider";
import {
  createWorkspaceNamespace,
  getWorkspaceNamespaceCatalog,
  listWorkspaceNamespaces,
  type WorkspaceZoneOption,
} from "@/components/workspace/api";
import { namespaceSlug } from "@/components/workspace/data";

export default function NewWorkspaceNamespacePage() {
  const router = useRouter();
  const { pushToast } = useToast();
  const [displayName, setDisplayName] = useState("");
  const [description, setDescription] = useState("");
  const [clusterID, setClusterID] = useState("");
  const [zones, setZones] = useState<WorkspaceZoneOption[]>([]);
  const [existingNamespaces, setExistingNamespaces] = useState<{ display_name: string }[]>([]);

  const normalizedName = displayName.trim();
  const slug = namespaceSlug(displayName);
  const selectedZone = useMemo(
    () => zones.find((item) => item.id === clusterID) ?? null,
    [clusterID, zones]
  );
  const hasDuplicate = existingNamespaces.some((item) => item.display_name === slug);
  const nameError =
    normalizedName !== "" && (slug.length === 0 || slug.length > 63 || slug !== normalizedName.toLowerCase());
  const isValid =
    slug.length > 0 &&
    slug.length <= 63 &&
    !nameError &&
    clusterID.trim() !== "" &&
    !hasDuplicate;
  const previewName = normalizedName || "Type a namespace name";
  const previewZone = selectedZone?.name || "Choose a zone";

  useEffect(() => {
    let cancelled = false;

    async function load() {
      try {
        const [namespaces, catalog] = await Promise.all([
          listWorkspaceNamespaces(),
          getWorkspaceNamespaceCatalog(),
        ]);
        if (cancelled) {
          return;
        }
        setExistingNamespaces(namespaces);
        setZones(catalog.zones ?? []);
        const firstZone = catalog.zones?.[0];
        if (firstZone) {
          setClusterID(firstZone.id);
        }
      } catch (error) {
        if (cancelled) {
          return;
        }
        pushToast({
          kind: "error",
          message:
            error instanceof Error ? error.message : "Failed to load namespace catalog.",
        });
      }
    }

    void load();
    return () => {
      cancelled = true;
    };
  }, [pushToast]);

  // submitNamespace creates a namespace through the workspace backend.
  async function submitNamespace() {
    if (!isValid) {
      pushToast({
        kind: "error",
        message: "Namespace must be a lowercase slug and unique in this workspace.",
      });
      return;
    }

    try {
      await createWorkspaceNamespace({
        display_name: displayName,
        description,
        cluster_id: clusterID,
      });
      pushToast({
        kind: "success",
        message: `${slug} added to the workspace namespace catalog.`,
      });
      router.push("/workspace/namespaces");
    } catch (error) {
      pushToast({
        kind: "error",
        message: error instanceof Error ? error.message : "Failed to create namespace.",
      });
    }
  }

  return (
    <div className="space-y-6">
      <PageBreadcrumb pageTitle="Create Namespace" />

      <section className="grid gap-6 xl:grid-cols-[minmax(0,1.3fr)_360px]">
        <div className="rounded-3xl border border-gray-200 bg-white p-6 dark:border-gray-800 dark:bg-white/[0.03]">
          <div className="space-y-5">
            <div>
              <h1 className="text-3xl font-semibold text-gray-900 dark:text-white">
                Create namespace
              </h1>
              <p className="mt-2 text-sm leading-7 text-gray-500 dark:text-gray-400">
                Give the namespace a clear name, pick where it should live, and add a
                short note for your team.
              </p>
            </div>

            <div className="grid gap-5 md:grid-cols-2">
              <div className="space-y-2 md:col-span-2">
                <label className="text-sm font-medium text-gray-700 dark:text-gray-300">
                  Namespace name
                </label>
                <Input
                  value={displayName}
                  onChange={(event) => setDisplayName(event.target.value)}
                  placeholder="analytics"
                  hint={nameError ? "Use lowercase letters, numbers, and dashes only." : "Use a short name that is easy to scan in lists and filters."}
                  error={nameError}
                />
              </div>

              <div className="space-y-2 md:col-span-2">
                <label className="text-sm font-medium text-gray-700 dark:text-gray-300">
                  Zone
                </label>
                <select
                  value={clusterID}
                  onChange={(event) => setClusterID(event.target.value)}
                  className="h-11 w-full rounded-lg border border-gray-300 bg-transparent px-4 text-sm text-gray-800 shadow-theme-xs outline-none transition focus:border-brand-300 focus:ring-3 focus:ring-brand-500/10 dark:border-gray-700 dark:bg-gray-900 dark:text-white/90 dark:focus:border-brand-800"
                >
                  {!clusterID ? <option value="">Choose a zone</option> : null}
                  {zones.map((item) => (
                    <option key={item.id} value={item.id}>
                      {item.name}
                    </option>
                  ))}
                </select>
                <p className="text-xs text-gray-500 dark:text-gray-400">
                  This decides where the namespace is created and which runtime cluster will keep it in sync.
                </p>
              </div>

              <div className="space-y-2 md:col-span-2">
                <label className="text-sm font-medium text-gray-700 dark:text-gray-300">
                  Description
                </label>
                <textarea
                  rows={4}
                  value={description}
                  onChange={(event) => setDescription(event.target.value)}
                  placeholder="Analytics-facing services and dashboards."
                  className="w-full rounded-lg border border-gray-300 bg-transparent px-4 py-3 text-sm text-gray-800 shadow-theme-xs outline-none transition focus:border-brand-300 focus:ring-3 focus:ring-brand-500/10 dark:border-gray-700 dark:bg-gray-900 dark:text-white/90 dark:focus:border-brand-800"
                />
                <p className="text-xs text-gray-500 dark:text-gray-400">
                  Keep this short. It appears in the namespace detail view and helps teammates understand the purpose quickly.
                </p>
              </div>
            </div>

            <div className="flex gap-3">
              <Button className="rounded-xl px-5" onClick={submitNamespace}>
                Create namespace
              </Button>
              <Button
                variant="outline"
                className="rounded-xl px-5"
                onClick={() => router.push("/workspace/namespaces")}
              >
                Cancel
              </Button>
            </div>
          </div>
        </div>

        <aside className="rounded-3xl border border-gray-200 bg-white p-6 dark:border-gray-800 dark:bg-white/[0.03]">
          <p className="text-xs font-semibold uppercase tracking-[0.24em] text-gray-400 dark:text-gray-500">
            Namespace preview
          </p>
          <div className="mt-5 space-y-4">
            <SummaryRow label="Display name" value={previewName} />
            <SummaryRow label="Zone" value={previewZone} />
          </div>
        </aside>
      </section>
    </div>
  );
}

function SummaryRow(props: { label: string; value: string }) {
  return (
    <div className="flex items-start justify-between gap-3 border-b border-dashed border-gray-200 pb-3 text-sm last:border-b-0 last:pb-0 dark:border-gray-800">
      <span className="text-gray-500 dark:text-gray-400">{props.label}</span>
      <span className="text-right font-medium text-gray-900 dark:text-white">
        {props.value}
      </span>
    </div>
  );
}
