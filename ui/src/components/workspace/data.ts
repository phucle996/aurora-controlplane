"use client";

export {
  type WorkspaceInventoryItem,
  type WorkspaceNamespace,
  type WorkspaceNamespaceCatalog,
  type WorkspaceNamespaceStatus,
  type WorkspaceNetworkPolicy,
  type WorkspaceNetworkPolicyRule,
  type WorkspaceNetworkPolicyStatus,
} from "@/components/workspace/api";

export type WorkspaceResourceType =
  | "virtual-machine"
  | "database"
  | "object-storage"
  | "application";

export const GLOBAL_NAMESPACE = "__global__";

// namespaceSlug keeps user-entered namespace names Kubernetes-safe before submission.
export function namespaceSlug(value: string) {
  return value
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9-]+/g, "-")
    .replace(/^-+|-+$/g, "")
    .replace(/--+/g, "-");
}

// namespaceCounts rolls up inventory items so namespace rows can show resource totals.
export function namespaceCounts(items: Array<{ namespace: string }>) {
  return items.reduce<Record<string, number>>((acc, item) => {
    if (item.namespace === GLOBAL_NAMESPACE) {
      return acc;
    }
    acc[item.namespace] = (acc[item.namespace] || 0) + 1;
    return acc;
  }, {});
}

// derivePolicyType derives the editor/list label from ingress and egress rules.
export function derivePolicyType(policy: {
  ingress_rules: unknown[];
  egress_rules: unknown[];
}) {
  const hasIngress = policy.ingress_rules.length > 0;
  const hasEgress = policy.egress_rules.length > 0;
  if (hasIngress && hasEgress) return "Ingress/Egress";
  if (hasIngress) return "Ingress";
  return "Egress";
}

export function workspaceResourceLabel(type: string) {
  switch (type) {
    case "virtual-machine":
      return "Virtual machine";
    case "database":
      return "Database";
    case "object-storage":
      return "Object storage";
    case "application":
      return "Application";
    default:
      return type;
  }
}

export function workspaceStatusColor(status: string) {
  switch (status) {
    case "ready":
    case "available":
    case "running":
    case "enforced":
      return "success";
    case "degraded":
    case "pending":
    case "deploying":
    case "rendering":
    case "applying":
    case "terminating":
      return "warning";
    case "stopped":
    case "failed":
    case "error":
      return "error";
    default:
      return "primary";
  }
}

export function workspaceNamespaceDisplayStatus(item: {
  status: string;
  runtime_status?: string;
}) {
  const runtimeStatus = item.runtime_status?.trim();
  if (runtimeStatus && runtimeStatus !== "deleting") {
    return runtimeStatus;
  }
  if (item.status === "deleting") {
    return "terminating";
  }
  if (runtimeStatus === "deleting") {
    return "terminating";
  }
  return item.status;
}
