"use client";

import { useEffect, useMemo, useState } from "react";
import { useRouter } from "next/navigation";
import PageBreadcrumb from "@/components/common/PageBreadCrumb";
import Badge from "@/components/ui/badge/Badge";
import Button from "@/components/ui/button/Button";
import { PlusIcon } from "@/icons";
import {
  listWorkspaceNamespaces,
  listWorkspaceNetworkPolicies,
  type WorkspaceNamespace,
  type WorkspaceNetworkPolicyListItem,
} from "@/components/workspace/api";
import { workspaceStatusColor } from "@/components/workspace/data";

export default function WorkspaceNetworkPoliciesPage() {
  const router = useRouter();
  const [policies, setPolicies] = useState<WorkspaceNetworkPolicyListItem[]>([]);
  const [namespaces, setNamespaces] = useState<WorkspaceNamespace[]>([]);
  const [namespaceFilter, setNamespaceFilter] = useState("all");
  const [policyTypeFilter, setPolicyTypeFilter] = useState("all");

  useEffect(() => {
    let cancelled = false;

    async function load() {
      const [nextNamespaces, nextPolicies] = await Promise.all([
        listWorkspaceNamespaces(),
        listWorkspaceNetworkPolicies(),
      ]);
      if (cancelled) {
        return;
      }
      setNamespaces(nextNamespaces);
      setPolicies(nextPolicies);
    }

    void load();
    return () => {
      cancelled = true;
    };
  }, []);

  const filteredPolicies = useMemo(() => {
    return policies.filter((item) => {
      const matchesNamespace =
        namespaceFilter === "all" || item.namespace === namespaceFilter;
      const matchesType =
        policyTypeFilter === "all" || item.policy_type === policyTypeFilter;
      return matchesNamespace && matchesType;
    });
  }, [namespaceFilter, policies, policyTypeFilter]);

  return (
    <div className="space-y-6">
      <PageBreadcrumb pageTitle="Network Policies" />

      <section className="rounded-3xl border border-gray-200 bg-white p-6 dark:border-gray-800 dark:bg-white/[0.03]">
        <div className="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
          <div className="space-y-2">
            <h1 className="text-3xl font-semibold text-gray-900 dark:text-white">
              Namespace network policies
            </h1>
            <p className="max-w-3xl text-sm leading-7 text-gray-500 dark:text-gray-400">
              Policies apply only to namespaced Kubernetes workloads. Targeting is
              done by resource labels so the same namespace can host multiple apps,
              databases, and storage-facing components safely.
            </p>
          </div>

          <Button
            className="rounded-xl px-5"
            startIcon={<PlusIcon className="size-4" />}
            onClick={() => router.push("/workspace/network-policies/new")}
          >
            New policy
          </Button>
        </div>
      </section>

      <section className="rounded-3xl border border-gray-200 bg-white dark:border-gray-800 dark:bg-white/[0.03]">
        <div className="grid gap-4 border-b border-gray-200 px-6 py-5 dark:border-gray-800 md:grid-cols-2">
          <select
            value={namespaceFilter}
            onChange={(event) => setNamespaceFilter(event.target.value)}
            className="h-11 rounded-lg border border-gray-300 bg-transparent px-4 text-sm text-gray-800 shadow-theme-xs outline-none transition focus:border-brand-300 focus:ring-3 focus:ring-brand-500/10 dark:border-gray-700 dark:bg-gray-900 dark:text-white/90 dark:focus:border-brand-800"
          >
            <option value="all">All namespaces</option>
            {namespaces.map((item) => (
              <option key={item.id} value={item.display_name}>
                {item.display_name}
              </option>
            ))}
          </select>

          <select
            value={policyTypeFilter}
            onChange={(event) => setPolicyTypeFilter(event.target.value)}
            className="h-11 rounded-lg border border-gray-300 bg-transparent px-4 text-sm text-gray-800 shadow-theme-xs outline-none transition focus:border-brand-300 focus:ring-3 focus:ring-brand-500/10 dark:border-gray-700 dark:bg-gray-900 dark:text-white/90 dark:focus:border-brand-800"
          >
            <option value="all">All policy types</option>
            <option value="Ingress">Ingress</option>
            <option value="Egress">Egress</option>
            <option value="Ingress/Egress">Ingress/Egress</option>
          </select>
        </div>

        <div className="overflow-x-auto">
          <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-800">
            <thead>
              <tr className="text-left text-xs uppercase tracking-[0.18em] text-gray-400 dark:text-gray-500">
                <th className="px-6 py-4 font-medium">Policy</th>
                <th className="px-6 py-4 font-medium">Namespace</th>
                <th className="px-6 py-4 font-medium">Policies</th>
                <th className="px-6 py-4 font-medium">Type</th>
                <th className="px-6 py-4 font-medium">Rules</th>
                <th className="px-6 py-4 font-medium">State</th>
                <th className="px-6 py-4 font-medium">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-200 dark:divide-gray-800">
              {filteredPolicies.length === 0 ? (
                <tr>
                  <td
                    colSpan={7}
                    className="px-6 py-12 text-center text-sm text-gray-500 dark:text-gray-400"
                  >
                    No network policies match the current filters.
                  </td>
                </tr>
              ) : null}

              {filteredPolicies.map((item) => (
                <tr key={item.id} className="align-top">
                  <td className="px-6 py-5">
                    <div className="space-y-1">
                      <p className="text-sm font-semibold text-gray-900 dark:text-white">
                        {item.name}
                      </p>
                      <p className="text-xs text-gray-500 dark:text-gray-400">
                        Updated {new Date(item.updated_at).toLocaleString()}
                      </p>
                    </div>
                  </td>
                  <td className="px-6 py-5 text-sm text-gray-700 dark:text-gray-300">
                    {item.namespace}
                  </td>
                  <td className="px-6 py-5">
                    <div className="space-y-1 text-sm text-gray-700 dark:text-gray-300">
                      <p>{item.ingress_rule_count} ingress rule(s)</p>
                      <p>{item.egress_rule_count} egress rule(s)</p>
                    </div>
                  </td>
                  <td className="px-6 py-5 text-sm text-gray-700 dark:text-gray-300">
                    {item.policy_type}
                  </td>
                  <td className="px-6 py-5 text-sm text-gray-700 dark:text-gray-300">
                    {item.rules_summary}
                  </td>
                  <td className="px-6 py-5">
                    <Badge color={workspaceStatusColor(item.status)}>{item.status}</Badge>
                  </td>
                  <td className="px-6 py-5">
                    <Button
                      variant="outline"
                      className="rounded-xl px-4"
                      onClick={() =>
                        router.push(
                          `/workspace/network-policies/editor?id=${encodeURIComponent(item.id)}`
                        )
                      }
                    >
                      Edit
                    </Button>
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
