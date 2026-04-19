"use client";

import Link from "next/link";
import { useEffect, useMemo, useState } from "react";
import PageBreadcrumb from "@/components/common/PageBreadCrumb";
import Input from "@/components/form/input/InputField";
import Badge from "@/components/ui/badge/Badge";
import { useToast } from "@/components/ui/toast/ToastProvider";
import {
  listWorkspaceInventory,
  listWorkspaceNamespaces,
  type WorkspaceInventoryItem,
} from "@/components/workspace/api";
import {
  GLOBAL_NAMESPACE,
  namespaceCounts,
  workspaceResourceLabel,
  workspaceStatusColor,
  type WorkspaceNamespace,
} from "@/components/workspace/data";

// formatRelative keeps timestamps compact in the inventory table.
function formatRelative(value: string) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return "-";
  }

  const diffMinutes = Math.max(Math.floor((Date.now() - date.getTime()) / 60000), 0);
  if (diffMinutes < 1) return "just now";
  if (diffMinutes < 60) return `${diffMinutes}m ago`;
  const diffHours = Math.floor(diffMinutes / 60);
  if (diffHours < 24) return `${diffHours}h ago`;
  return `${Math.floor(diffHours / 24)}d ago`;
}

export default function WorkspacePage() {
  const { pushToast } = useToast();
  const [namespaces, setNamespaces] = useState<WorkspaceNamespace[]>([]);
  const [inventory, setInventory] = useState<WorkspaceInventoryItem[]>([]);
  const [search, setSearch] = useState("");
  const [namespaceFilter, setNamespaceFilter] = useState("all");
  const [resourceTypeFilter, setResourceTypeFilter] = useState("all");
  const [statusFilter, setStatusFilter] = useState("all");

  useEffect(() => {
    let cancelled = false;

    async function load() {
      try {
        const [nextNamespaces, nextInventory] = await Promise.all([
          listWorkspaceNamespaces(),
          listWorkspaceInventory(),
        ]);

        if (cancelled) {
          return;
        }

        setNamespaces(nextNamespaces);
        setInventory(nextInventory);
      } catch (error) {
        if (cancelled) {
          return;
        }

        pushToast({
          kind: "error",
          message:
            error instanceof Error
              ? error.message
              : "Failed to load workspace inventory.",
        });
      }
    }

    void load();
    return () => {
      cancelled = true;
    };
  }, [pushToast]);

  const filteredInventory = useMemo(() => {
    const query = search.trim().toLowerCase();

    return inventory.filter((item) => {
      const matchesSearch =
        query === "" ||
        item.name.toLowerCase().includes(query) ||
        item.endpoint.toLowerCase().includes(query) ||
        item.cluster.toLowerCase().includes(query) ||
        item.labels.some((label) => label.includes(query));
      const matchesNamespace =
        namespaceFilter === "all" || item.namespace === namespaceFilter;
      const matchesType =
        resourceTypeFilter === "all" || item.resource_type === resourceTypeFilter;
      const matchesStatus =
        statusFilter === "all" || item.status === statusFilter;
      return matchesSearch && matchesNamespace && matchesType && matchesStatus;
    });
  }, [inventory, namespaceFilter, resourceTypeFilter, search, statusFilter]);

  const counts = useMemo(() => namespaceCounts(inventory), [inventory]);

  return (
    <div className="space-y-6">
      <PageBreadcrumb pageTitle="My Workspace" />

      <section className="rounded-3xl border border-gray-200 bg-white p-6 dark:border-gray-800 dark:bg-white/[0.03] lg:p-8">
        <div className="grid gap-5 lg:grid-cols-[minmax(0,1.4fr)_repeat(3,minmax(0,1fr))]">
          <div className="space-y-3">
            <div className="inline-flex items-center rounded-full bg-brand-500/10 px-3 py-1 text-xs font-semibold uppercase tracking-[0.24em] text-brand-600 dark:text-brand-400">
              Unified inventory
            </div>
            <h1 className="text-3xl font-semibold text-gray-900 dark:text-white">
              Workspace resources across Kubernetes and global compute
            </h1>
            <p className="text-sm leading-7 text-gray-500 dark:text-gray-400">
              Every resource except virtual machines is placed inside a workspace
              namespace. The inventory below lets you filter by namespace, type,
              status, or label-aware search terms.
            </p>
          </div>
        </div>
      </section>

      <section className="rounded-3xl border border-gray-200 bg-white dark:border-gray-800 dark:bg-white/[0.03]">
        <div className="grid gap-4 border-b border-gray-200 px-6 py-5 dark:border-gray-800 xl:grid-cols-[minmax(0,1fr)_260px_220px_220px]">
          <Input
            type="text"
            value={search}
            onChange={(event) => setSearch(event.target.value)}
            placeholder="Search resource name, endpoint, cluster, or labels..."
          />

          <select
            value={namespaceFilter}
            onChange={(event) => setNamespaceFilter(event.target.value)}
            className="h-11 rounded-lg border border-gray-300 bg-transparent px-4 text-sm text-gray-800 shadow-theme-xs outline-none transition focus:border-brand-300 focus:ring-3 focus:ring-brand-500/10 dark:border-gray-700 dark:bg-gray-900 dark:text-white/90 dark:focus:border-brand-800"
          >
            <option value="all">All namespaces</option>
            <option value={GLOBAL_NAMESPACE}>Global / not namespaced</option>
            {namespaces.map((item) => (
              <option key={item.id} value={item.display_name}>
                {item.display_name} ({counts[item.display_name] || 0})
              </option>
            ))}
          </select>

          <select
            value={resourceTypeFilter}
            onChange={(event) => setResourceTypeFilter(event.target.value)}
            className="h-11 rounded-lg border border-gray-300 bg-transparent px-4 text-sm text-gray-800 shadow-theme-xs outline-none transition focus:border-brand-300 focus:ring-3 focus:ring-brand-500/10 dark:border-gray-700 dark:bg-gray-900 dark:text-white/90 dark:focus:border-brand-800"
          >
            <option value="all">All resource types</option>
            {Array.from(new Set(inventory.map((item) => item.resource_type))).map((item) => (
              <option key={item} value={item}>
                {workspaceResourceLabel(item)}
              </option>
            ))}
          </select>

          <select
            value={statusFilter}
            onChange={(event) => setStatusFilter(event.target.value)}
            className="h-11 rounded-lg border border-gray-300 bg-transparent px-4 text-sm text-gray-800 shadow-theme-xs outline-none transition focus:border-brand-300 focus:ring-3 focus:ring-brand-500/10 dark:border-gray-700 dark:bg-gray-900 dark:text-white/90 dark:focus:border-brand-800"
          >
            <option value="all">All statuses</option>
            {Array.from(new Set(inventory.map((item) => item.status))).map((item) => (
              <option key={item} value={item}>
                {item}
              </option>
            ))}
          </select>
        </div>

        <div className="overflow-x-auto">
          <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-800">
            <thead>
              <tr className="text-left text-xs uppercase tracking-[0.18em] text-gray-400 dark:text-gray-500">
                <th className="px-6 py-4 font-medium">Resource</th>
                <th className="px-6 py-4 font-medium">Type</th>
                <th className="px-6 py-4 font-medium">Namespace</th>
                <th className="px-6 py-4 font-medium">Target</th>
                <th className="px-6 py-4 font-medium">Labels</th>
                <th className="px-6 py-4 font-medium">Status</th>
                <th className="px-6 py-4 font-medium">Updated</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-200 dark:divide-gray-800">
              {filteredInventory.length === 0 ? (
                <tr>
                  <td
                    colSpan={7}
                    className="px-6 py-12 text-center text-sm text-gray-500 dark:text-gray-400"
                  >
                    No workspace resources match the current filters.
                  </td>
                </tr>
              ) : null}

              {filteredInventory.map((item) => (
                <tr key={item.id} className="align-top">
                  <td className="px-6 py-5">
                    <div className="space-y-1">
                      {item.detail_href ? (
                        <Link
                          href={item.detail_href}
                          className="text-sm font-semibold text-gray-900 transition hover:text-brand-500 dark:text-white"
                        >
                          {item.name}
                        </Link>
                      ) : (
                        <p className="text-sm font-semibold text-gray-900 dark:text-white">
                          {item.name}
                        </p>
                      )}
                      <p className="text-xs text-gray-500 dark:text-gray-400">
                        {item.endpoint}
                      </p>
                    </div>
                  </td>
                  <td className="px-6 py-5 text-sm text-gray-700 dark:text-gray-300">
                    {workspaceResourceLabel(item.resource_type)}
                  </td>
                  <td className="px-6 py-5">
                    {item.namespace === GLOBAL_NAMESPACE ? (
                      <span className="text-sm text-gray-500 dark:text-gray-400">
                        Global
                      </span>
                    ) : (
                      <Link
                        href={`/workspace/network-policies/new?namespace=${encodeURIComponent(item.namespace)}`}
                        className="text-sm font-medium text-brand-600 hover:text-brand-700 dark:text-brand-400"
                      >
                        {item.namespace}
                      </Link>
                    )}
                  </td>
                  <td className="px-6 py-5">
                    <div className="space-y-1 text-sm text-gray-700 dark:text-gray-300">
                      <p>{item.cluster}</p>
                      <p className="text-xs text-gray-500 dark:text-gray-400">
                        {item.zone}
                      </p>
                    </div>
                  </td>
                  <td className="px-6 py-5">
                    <div className="flex flex-wrap gap-2">
                      {item.labels.slice(0, 3).map((label) => (
                        <span
                          key={label}
                          className="rounded-full bg-gray-100 px-2.5 py-1 text-xs text-gray-600 dark:bg-gray-800 dark:text-gray-300"
                        >
                          {label}
                        </span>
                      ))}
                      {item.labels.length > 3 ? (
                        <span className="rounded-full bg-gray-100 px-2.5 py-1 text-xs text-gray-600 dark:bg-gray-800 dark:text-gray-300">
                          +{item.labels.length - 3}
                        </span>
                      ) : null}
                    </div>
                  </td>
                  <td className="px-6 py-5">
                    <Badge color={workspaceStatusColor(item.status)}>{item.status}</Badge>
                  </td>
                  <td className="px-6 py-5 text-sm text-gray-500 dark:text-gray-400">
                    {formatRelative(item.created_at)}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </section>
    </div>
  );
}

function MetricCard(props: { label: string; value: string }) {
  return (
    <div className="rounded-3xl border border-gray-200 bg-gray-50 px-5 py-5 dark:border-gray-800 dark:bg-gray-900/60">
      <p className="text-xs font-semibold uppercase tracking-[0.24em] text-gray-400 dark:text-gray-500">
        {props.label}
      </p>
      <p className="mt-3 text-3xl font-semibold text-gray-900 dark:text-white">
        {props.value}
      </p>
    </div>
  );
}
