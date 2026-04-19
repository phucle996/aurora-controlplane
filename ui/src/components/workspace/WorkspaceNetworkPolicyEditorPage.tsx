"use client";

import { useEffect, useMemo, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import PageBreadcrumb from "@/components/common/PageBreadCrumb";
import Input from "@/components/form/input/InputField";
import Badge from "@/components/ui/badge/Badge";
import Button from "@/components/ui/button/Button";
import { useToast } from "@/components/ui/toast/ToastProvider";
import {
  createWorkspaceNetworkPolicy,
  getWorkspaceNetworkPolicy,
  listWorkspaceInventory,
  listWorkspaceNamespaces,
  updateWorkspaceNetworkPolicy,
  type UpsertWorkspaceNetworkPolicyInput,
  type WorkspaceInventoryItem,
  type WorkspaceNamespace,
  type WorkspaceNetworkPolicyStatus,
} from "@/components/workspace/api";
import { derivePolicyType } from "@/components/workspace/data";

const ALL_ENDPOINTS = "*";

type DraftRule = {
  id: string;
  source: string;
  destination: string;
  description?: string;
};

type EditorDraft = {
  id: string;
  name: string;
  namespace_id: string;
  ingress_rules: DraftRule[];
  egress_rules: DraftRule[];
  rules_summary: string;
  status: WorkspaceNetworkPolicyStatus;
};

function createRule(): DraftRule {
  return {
    id:
      typeof crypto !== "undefined" && "randomUUID" in crypto
        ? crypto.randomUUID()
        : `workspace-policy-rule-${Date.now()}`,
    source: ALL_ENDPOINTS,
    destination: ALL_ENDPOINTS,
  };
}

function createDraft(namespaceID: string): EditorDraft {
  return {
    id:
      typeof crypto !== "undefined" && "randomUUID" in crypto
        ? crypto.randomUUID()
        : `workspace-policy-${Date.now()}`,
    name: "",
    namespace_id: namespaceID,
    ingress_rules: [],
    egress_rules: [],
    rules_summary: "",
    status: "draft",
  };
}

function ruleValue(rule: { source_raw?: string; destination_raw?: string; source?: string; destination?: string }) {
  return {
    source: rule.source_raw ?? rule.source ?? ALL_ENDPOINTS,
    destination: rule.destination_raw ?? rule.destination ?? ALL_ENDPOINTS,
  };
}

export default function WorkspaceNetworkPolicyEditorPage(props: {
  policyID?: string;
}) {
  const router = useRouter();
  const searchParams = useSearchParams();
  const { pushToast } = useToast();
  const effectivePolicyID = props.policyID || searchParams.get("id") || undefined;
  const [namespaces, setNamespaces] = useState<WorkspaceNamespace[]>([]);
  const [inventory, setInventory] = useState<WorkspaceInventoryItem[]>([]);
  const [draft, setDraft] = useState<EditorDraft | null>(null);

  useEffect(() => {
    let cancelled = false;

    async function load() {
      try {
        const [namespaceItems, inventoryItems] = await Promise.all([
          listWorkspaceNamespaces(),
          listWorkspaceInventory(),
        ]);
        const existing = effectivePolicyID
          ? await getWorkspaceNetworkPolicy(effectivePolicyID)
          : null;
        if (cancelled) {
          return;
        }

        setNamespaces(namespaceItems);
        setInventory(inventoryItems);

        const preferredNamespaceID =
          existing?.namespace_id ||
          namespaceItems.find(
            (item) => item.display_name === searchParams.get("namespace"),
          )?.id ||
          namespaceItems[0]?.id ||
          "";

        setDraft(
          existing
            ? {
                id: existing.id,
                name: existing.name,
                namespace_id: existing.namespace_id,
                ingress_rules: existing.ingress_rules.map((rule) => ({
                  id: rule.id,
                  ...ruleValue(rule),
                  description: rule.description,
                })),
                egress_rules: existing.egress_rules.map((rule) => ({
                  id: rule.id,
                  ...ruleValue(rule),
                  description: rule.description,
                })),
                rules_summary: existing.rules_summary,
                status: existing.status,
              }
            : createDraft(preferredNamespaceID),
        );
      } catch (error) {
        if (!cancelled) {
          pushToast({
            kind: "error",
            message:
              error instanceof Error
                ? error.message
                : "Failed to load resources for policy editing.",
          });
        }
      }
    }

    void load();
    return () => {
      cancelled = true;
    };
  }, [effectivePolicyID, pushToast, searchParams]);

  const selectedNamespace = useMemo(
    () => namespaces.find((item) => item.id === draft?.namespace_id) ?? null,
    [draft?.namespace_id, namespaces],
  );

  const namespacedResources = useMemo(
    () =>
      inventory.filter(
        (item) =>
          item.namespace === selectedNamespace?.display_name && item.policy_capable,
      ),
    [inventory, selectedNamespace?.display_name],
  );

  const resourceOptions = useMemo(
    () =>
      [
        ALL_ENDPOINTS,
        "0.0.0.0/0",
        "10.0.0.0/8",
        "120.0.0.0/24",
        ...inventory
          .filter((item) => item.policy_capable)
          .map((item) => `namespace:${item.namespace}/${item.name}`),
        ...namespacedResources.map((item) => item.name),
      ].filter((value, index, array) => array.indexOf(value) === index),
    [inventory, namespacedResources],
  );

  function updateRule(
    section: "ingress_rules" | "egress_rules",
    ruleID: string,
    field: "source" | "destination",
    value: string,
  ) {
    setDraft((current) =>
      current
        ? {
            ...current,
            [section]: current[section].map((rule) =>
              rule.id === ruleID ? { ...rule, [field]: value } : rule,
            ),
          }
        : current,
    );
  }

  function addRule(section: "ingress_rules" | "egress_rules") {
    setDraft((current) =>
      current ? { ...current, [section]: [...current[section], createRule()] } : current,
    );
  }

  function removeRule(section: "ingress_rules" | "egress_rules", ruleID: string) {
    setDraft((current) =>
      current
        ? {
            ...current,
            [section]: current[section].filter((rule) => rule.id !== ruleID),
          }
        : current,
    );
  }

  async function savePolicy() {
    if (!draft || !draft.name.trim() || !draft.namespace_id) {
      pushToast({
        kind: "error",
        message: "Policy name and namespace are required.",
      });
      return;
    }

    if (draft.ingress_rules.length + draft.egress_rules.length === 0) {
      pushToast({
        kind: "error",
        message: "Add at least one ingress or egress rule.",
      });
      return;
    }

    const payload: UpsertWorkspaceNetworkPolicyInput = {
      name: draft.name.trim(),
      namespace_id: draft.namespace_id,
      rules_summary: draft.rules_summary.trim(),
      status: draft.status,
      ingress_rules: draft.ingress_rules.map((rule) => ({
        source: rule.source.trim() || ALL_ENDPOINTS,
        destination: rule.destination.trim() || ALL_ENDPOINTS,
        description: rule.description?.trim(),
      })),
      egress_rules: draft.egress_rules.map((rule) => ({
        source: rule.source.trim() || ALL_ENDPOINTS,
        destination: rule.destination.trim() || ALL_ENDPOINTS,
        description: rule.description?.trim(),
      })),
    };

    try {
      if (effectivePolicyID) {
        await updateWorkspaceNetworkPolicy(effectivePolicyID, payload)
      } else {
        await createWorkspaceNetworkPolicy(payload)
      }
      pushToast({
        kind: "success",
        message: `${draft.name.trim()} saved for ${selectedNamespace?.display_name ?? "the selected namespace"}.`,
      });
      router.push("/workspace/network-policies");
    } catch (error) {
      pushToast({
        kind: "error",
        message:
          error instanceof Error ? error.message : "Failed to save network policy.",
      });
    }
  }

  if (!draft) {
    return null;
  }

  const effectivePolicyType = derivePolicyType(draft);
  const ingressDefault = draft.ingress_rules.length === 0 ? "allow" : "deny";

  return (
    <div className="space-y-6">
      <PageBreadcrumb
        pageTitle={effectivePolicyID ? "Edit Network Policy" : "New Network Policy"}
      />

      <section className="grid gap-6 xl:grid-cols-[minmax(0,1.35fr)_360px]">
        <div className="space-y-6">
          <div className="rounded-3xl border border-gray-200 bg-white p-6 dark:border-gray-800 dark:bg-white/[0.03]">
            <div className="space-y-5">
              <div>
                <h1 className="text-3xl font-semibold text-gray-900 dark:text-white">
                  {effectivePolicyID ? "Edit network policy" : "Create network policy"}
                </h1>
                <p className="mt-2 text-sm leading-7 text-gray-500 dark:text-gray-400">
                  Build ingress and egress rules by selecting source and destination
                  endpoints inside one workspace namespace. You can type resource
                  names, <code>namespace:&lt;namespace&gt;/&lt;resource&gt;</code>, CIDR
                  ranges like <code>120.0.0.0/24</code>, or <code>*</code> for all.
                  Virtual machines stay out of scope.
                </p>
              </div>

              <div className="grid gap-5 md:grid-cols-2">
                <div className="space-y-2 md:col-span-2">
                  <label className="text-sm font-medium text-gray-700 dark:text-gray-300">
                    Policy name
                  </label>
                  <Input
                    value={draft.name}
                    onChange={(event) =>
                      setDraft((current) =>
                        current ? { ...current, name: event.target.value } : current,
                      )
                    }
                    placeholder="default-egress"
                  />
                </div>

                <div className="space-y-2">
                  <label className="text-sm font-medium text-gray-700 dark:text-gray-300">
                    Namespace
                  </label>
                  <select
                    value={draft.namespace_id}
                    onChange={(event) =>
                      setDraft((current) =>
                        current
                          ? {
                              ...current,
                              namespace_id: event.target.value,
                              ingress_rules: [],
                              egress_rules: [],
                            }
                          : current,
                      )
                    }
                    className="h-11 w-full rounded-lg border border-gray-300 bg-transparent px-4 text-sm text-gray-800 shadow-theme-xs outline-none transition focus:border-brand-300 focus:ring-3 focus:ring-brand-500/10 dark:border-gray-700 dark:bg-gray-900 dark:text-white/90 dark:focus:border-brand-800"
                  >
                    {namespaces.map((item) => (
                      <option key={item.id} value={item.id}>
                        {item.display_name}
                      </option>
                    ))}
                  </select>
                </div>

                <div className="space-y-2">
                  <label className="text-sm font-medium text-gray-700 dark:text-gray-300">
                    Status
                  </label>
                  <select
                    value={draft.status}
                    onChange={(event) =>
                      setDraft((current) =>
                        current
                          ? {
                              ...current,
                              status: event.target.value as WorkspaceNetworkPolicyStatus,
                            }
                          : current,
                      )
                    }
                    className="h-11 w-full rounded-lg border border-gray-300 bg-transparent px-4 text-sm text-gray-800 shadow-theme-xs outline-none transition focus:border-brand-300 focus:ring-3 focus:ring-brand-500/10 dark:border-gray-700 dark:bg-gray-900 dark:text-white/90 dark:focus:border-brand-800"
                  >
                    <option value="draft">draft</option>
                    <option value="enforced">enforced</option>
                  </select>
                </div>

                <div className="space-y-2 md:col-span-2">
                  <label className="text-sm font-medium text-gray-700 dark:text-gray-300">
                    Rules summary
                  </label>
                  <textarea
                    rows={4}
                    value={draft.rules_summary}
                    onChange={(event) =>
                      setDraft((current) =>
                        current
                          ? { ...current, rules_summary: event.target.value }
                          : current,
                      )
                    }
                    placeholder="Allow app traffic to database and keep default deny for everything else."
                    className="w-full rounded-lg border border-gray-300 bg-transparent px-4 py-3 text-sm text-gray-800 shadow-theme-xs outline-none transition focus:border-brand-300 focus:ring-3 focus:ring-brand-500/10 dark:border-gray-700 dark:bg-gray-900 dark:text-white/90 dark:focus:border-brand-800"
                  />
                </div>
              </div>
            </div>
          </div>

          <PolicySection
            title="Ingress"
            description="Who can talk to resources inside this namespace. No ingress rules means default allow; once ingress rules exist, unmatched ingress is denied by default."
            rules={draft.ingress_rules}
            resourceOptions={resourceOptions}
            onAdd={() => addRule("ingress_rules")}
            onRemove={(ruleID) => removeRule("ingress_rules", ruleID)}
            onChange={(ruleID, field, value) =>
              updateRule("ingress_rules", ruleID, field, value)
            }
          />

          <PolicySection
            title="Egress"
            description="Where resources in this namespace are allowed to connect. Use resource names, namespace-scoped references, CIDR ranges, or *."
            rules={draft.egress_rules}
            resourceOptions={resourceOptions}
            onAdd={() => addRule("egress_rules")}
            onRemove={(ruleID) => removeRule("egress_rules", ruleID)}
            onChange={(ruleID, field, value) =>
              updateRule("egress_rules", ruleID, field, value)
            }
          />
        </div>

        <aside className="rounded-3xl border border-gray-200 bg-white p-6 dark:border-gray-800 dark:bg-white/[0.03]">
          <p className="text-xs font-semibold uppercase tracking-[0.24em] text-gray-400 dark:text-gray-500">
            Policy summary
          </p>
          <div className="mt-5 space-y-4">
            <SummaryRow label="Namespace" value={selectedNamespace?.display_name || "-"} />
            <SummaryRow
              label="Zone"
              value={selectedNamespace?.zone || "-"}
            />
            <SummaryRow
              label="Policy-capable resources"
              value={String(namespacedResources.length)}
            />
            <SummaryRow
              label="Ingress rules"
              value={draft.ingress_rules.length ? String(draft.ingress_rules.length) : "0"}
            />
            <SummaryRow
              label="Egress rules"
              value={draft.egress_rules.length ? String(draft.egress_rules.length) : "0"}
            />
            <SummaryRow label="Ingress default" value={ingressDefault} />
            <SummaryRow label="Policy type" value={effectivePolicyType} />
            <SummaryRow label="State" value={draft.status} />
          </div>

          <div className="mt-8 grid gap-3">
            <Button className="rounded-xl" onClick={savePolicy}>
              Save policy
            </Button>
            <Button
              variant="outline"
              className="rounded-xl"
              onClick={() => router.push("/workspace/network-policies")}
            >
              Cancel
            </Button>
          </div>
        </aside>
      </section>
    </div>
  );
}

function PolicySection(props: {
  title: string;
  description: string;
  rules: DraftRule[];
  resourceOptions: string[];
  onAdd: () => void;
  onRemove: (ruleID: string) => void;
  onChange: (
    ruleID: string,
    field: "source" | "destination",
    value: string,
  ) => void;
}) {
  const dataListID = `${props.title.toLowerCase()}-resource-options`;

  return (
    <div className="rounded-3xl border border-gray-200 bg-white p-6 dark:border-gray-800 dark:bg-white/[0.03]">
      <div className="space-y-4">
        <div className="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
          <div>
            <h2 className="text-lg font-semibold text-gray-900 dark:text-white">
              {props.title}
            </h2>
            <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
              {props.description}
            </p>
          </div>
          <Button variant="outline" className="rounded-xl px-4" onClick={props.onAdd}>
            Add rule
          </Button>
        </div>

        {props.rules.length === 0 ? (
          <div className="rounded-2xl border border-dashed border-gray-200 px-4 py-5 text-sm text-gray-500 dark:border-gray-800 dark:text-gray-400">
            No {props.title.toLowerCase()} rules yet.
          </div>
        ) : null}

        <div className="space-y-4">
          {props.rules.map((rule) => (
            <div
              key={rule.id}
              className="grid gap-4 rounded-2xl border border-gray-200 p-4 dark:border-gray-800 lg:grid-cols-[minmax(0,1fr)_minmax(0,1fr)_auto]"
            >
              <div className="space-y-2">
                <label className="text-sm font-medium text-gray-700 dark:text-gray-300">
                  Source
                </label>
                <Input
                  value={rule.source}
                  onChange={(event) =>
                    props.onChange(rule.id, "source", event.target.value)
                  }
                  placeholder="namespace:default/orders-pg-k3s or 120.0.0.0/24 or *"
                  list={dataListID}
                />
              </div>

              <div className="space-y-2">
                <label className="text-sm font-medium text-gray-700 dark:text-gray-300">
                  Destination
                </label>
                <Input
                  value={rule.destination}
                  onChange={(event) =>
                    props.onChange(rule.id, "destination", event.target.value)
                  }
                  placeholder="namespace:default/orders-archive or 10.0.0.0/8 or *"
                  list={dataListID}
                />
              </div>

              <div className="flex items-end">
                <Button
                  variant="outline"
                  className="rounded-xl px-4"
                  onClick={() => props.onRemove(rule.id)}
                >
                  Remove
                </Button>
              </div>
            </div>
          ))}
        </div>

        <datalist id={dataListID}>
          {props.resourceOptions.map((item) => (
            <option key={item} value={item} />
          ))}
        </datalist>
      </div>
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
