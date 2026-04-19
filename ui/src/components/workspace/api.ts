"use client";

import { parseAPIError } from "@/components/auth/auth-utils";

type APIEnvelope<T> = { data?: T };

function buildQuery(params: Record<string, string | undefined>) {
  const query = new URLSearchParams();
  for (const [key, value] of Object.entries(params)) {
    if (value && value.trim() !== "") {
      query.set(key, value.trim());
    }
  }
  return query.toString();
}

export type WorkspaceNamespaceStatus = "ready" | "pending" | "deleting";

export type WorkspaceNamespace = {
  id: string;
  display_name: string;
  zone: string;
  description: string;
  status: WorkspaceNamespaceStatus;
  runtime_status?: string;
  is_default: boolean;
  can_delete: boolean;
  resource_count: number;
};

export type WorkspaceZoneOption = {
  id: string;
  name: string;
};

export type WorkspaceNamespaceCatalog = {
  zones: WorkspaceZoneOption[];
};

export type CreateWorkspaceNamespaceInput = {
  display_name: string;
  cluster_id: string;
  description: string;
};

export type WorkspaceInventoryItem = {
  id: string;
  resource_type: "virtual-machine" | "database" | "object-storage" | "application" | string;
  name: string;
  namespace: string;
  zone: string;
  cluster: string;
  status: string;
  endpoint: string;
  labels: string[];
  policy_capable: boolean;
  detail_href?: string;
  created_at: string;
};

export type WorkspaceNetworkPolicyStatus = "draft" | "enforced";

export type WorkspaceNetworkPolicyRule = {
  id: string;
  source_raw?: string;
  destination_raw?: string;
  source?: string;
  destination?: string;
  description?: string;
};

export type WorkspaceNetworkPolicy = {
  id: string;
  namespace_id: string;
  namespace: string;
  name: string;
  default_ingress_behavior: "allow" | "deny";
  default_egress_behavior: "allow" | "deny";
  policy_type: "Ingress" | "Egress" | "Ingress/Egress";
  rules_summary: string;
  status: WorkspaceNetworkPolicyStatus;
  ingress_rules: WorkspaceNetworkPolicyRule[];
  egress_rules: WorkspaceNetworkPolicyRule[];
  updated_at: string;
};

export type WorkspaceNetworkPolicyListItem = {
  id: string;
  namespace: string;
  name: string;
  policy_type: "Ingress" | "Egress" | "Ingress/Egress";
  rules_summary: string;
  status: WorkspaceNetworkPolicyStatus;
  ingress_rule_count: number;
  egress_rule_count: number;
  updated_at: string;
};

export type UpsertWorkspaceNetworkPolicyInput = {
  name: string;
  namespace_id: string;
  default_ingress_behavior?: "allow" | "deny";
  default_egress_behavior?: "allow" | "deny";
  rules_summary: string;
  status: WorkspaceNetworkPolicyStatus;
  ingress_rules: Array<{ source: string; destination: string; description?: string }>;
  egress_rules: Array<{ source: string; destination: string; description?: string }>;
};

export type WorkspaceMarketplaceCatalogItem = {
  resource_definition_id: string;
  template_id: string;
  template_name: string;
  slug: string;
  name: string;
  summary: string;
  description: string;
  resource_type: string;
  resource_model: string;
  versions: Array<{
    resource_definition_id: string;
    resource_version: string;
  }>;
  default_version: string;
};

export type WorkspaceMarketplacePlanOption = {
  code: string;
  name: string;
};

export type WorkspaceMarketplaceDeployMetadata = {
  resource: WorkspaceMarketplaceCatalogItem;
  plans: WorkspaceMarketplacePlanOption[];
};

export type WorkspaceMarketplaceDeployNamespace = {
  id: string;
  display_name: string;
};

export type WorkspaceMarketplaceDeployJob = {
  id: string;
  job_name: string;
  display_name: string;
  description: string;
  execution_mode: "manual" | "cron" | string;
};

export type WorkspaceMarketplaceDeployBootstrap = {
  resource: WorkspaceMarketplaceCatalogItem;
  plans: WorkspaceMarketplacePlanOption[];
  namespaces: WorkspaceMarketplaceDeployNamespace[];
  jobs: WorkspaceMarketplaceDeployJob[];
};

export type WorkspaceJobDefinition = {
  id: string;
  resource_type: string;
  resource_model: string;
  job_name: string;
  display_name: string;
  description: string;
  execution_mode: "manual" | "cron" | string;
  manifest_template: string;
  status: "draft" | "published" | "disabled" | string;
  created_at: string;
  updated_at: string;
};

export type CreateWorkspaceMarketplaceDeploymentInput = {
  resource_definition_id: string;
  template_id: string;
  namespace_id: string;
  name: string;
  plan: string;
  version: string;
  job_definition_id?: string;
  params?: Record<string, unknown>;
};

async function readData<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(path, {
    cache: "no-store",
    ...init,
  });
  if (!response.ok) {
    throw new Error(await parseAPIError(response));
  }
  const payload = (await response.json()) as APIEnvelope<T>;
  if (payload.data === undefined) {
    throw new Error("Missing API data payload.");
  }
  return payload.data;
}

async function readOptionalData<T>(
  path: string,
  init?: RequestInit,
  optionalStatuses: number[] = [404],
): Promise<T | null> {
  const response = await fetch(path, {
    cache: "no-store",
    ...init,
  });
  if (optionalStatuses.includes(response.status)) {
    return null;
  }
  if (!response.ok) {
    throw new Error(await parseAPIError(response));
  }
  const payload = (await response.json()) as APIEnvelope<T>;
  if (payload.data === undefined) {
    return null;
  }
  return payload.data;
}

type ResourcePlatformNamespaceListItem = {
  id: string;
  display_name: string;
  runtime_namespace?: string;
  runtime_status?: string;
  cluster_id?: string;
  cluster_name?: string;
  zone_name?: string;
  description?: string;
  status?: WorkspaceNamespaceStatus;
  resource_count?: number;
  created_at?: string;
  updated_at?: string;
};

type ResourcePlatformClusterListItem = {
  id: string;
  name?: string;
  zone_id?: string;
  zone_name?: string;
};

type ResourcePlatformNetworkPolicyListItem = {
  id: string;
  namespace_id: string;
  namespace_name: string;
  name: string;
  policy_type?: "Ingress" | "Egress" | "Ingress/Egress";
  rules_summary?: string;
  status?: WorkspaceNetworkPolicyStatus;
  updated_at?: string;
};

type ResourcePlatformNetworkPolicyRule = {
  id: string;
  direction?: "ingress" | "egress" | string;
  source_raw?: string;
  destination_raw?: string;
  description?: string;
};

type ResourcePlatformNetworkPolicyDetail = ResourcePlatformNetworkPolicyListItem & {
  runtime_namespace?: string;
  zone_name?: string;
  default_ingress_behavior?: "allow" | "deny";
  default_egress_behavior?: "allow" | "deny";
  rules?: ResourcePlatformNetworkPolicyRule[];
};

function normalizeWorkspaceNamespace(
  item: ResourcePlatformNamespaceListItem,
): WorkspaceNamespace {
  const displayName = item.display_name ?? "";
  const resourceCount =
    typeof item.resource_count === "number" ? item.resource_count : 0;
  const isDefault = displayName === "default";

  return {
    id: item.id,
    display_name: displayName,
    zone: item.zone_name ?? "",
    description: item.description ?? "",
    status: item.status ?? "ready",
    runtime_status: item.runtime_status ?? item.status ?? "ready",
    is_default: isDefault,
    can_delete: !isDefault,
    resource_count: resourceCount,
  };
}

export async function listWorkspaceNamespaces(): Promise<WorkspaceNamespace[]> {
  const payload = await readData<
    { items?: WorkspaceNamespace[] | ResourcePlatformNamespaceListItem[] } | ResourcePlatformNamespaceListItem[]
  >("/api/v1/resource-platform/namespaces");

  const items = Array.isArray(payload) ? payload : payload.items ?? [];
  return items.map((item) => normalizeWorkspaceNamespace(item as ResourcePlatformNamespaceListItem));
}

export async function getWorkspaceNamespaceCatalog(): Promise<WorkspaceNamespaceCatalog> {
  const payload = await readData<
    { items?: ResourcePlatformClusterListItem[] } | ResourcePlatformClusterListItem[]
  >("/api/v1/resource-platform/k8s/clusters");
  const items = Array.isArray(payload) ? payload : payload.items ?? [];

  return {
    zones: items.map((item) => ({
      id: item.id,
      name: item.zone_name || item.name || "Unnamed cluster",
    })),
  };
}

export async function createWorkspaceNamespace(input: CreateWorkspaceNamespaceInput): Promise<WorkspaceNamespace> {
  const response = await fetch("/api/v1/resource-platform/namespaces", {
    cache: "no-store",
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      display_name: input.display_name,
      cluster_id: input.cluster_id,
      description: input.description,
    }),
  });
  if (!response.ok) {
    throw new Error(await parseAPIError(response));
  }

  return {
    id:
      typeof crypto !== "undefined" && "randomUUID" in crypto
        ? crypto.randomUUID()
        : `workspace-namespace-${Date.now()}`,
    display_name: input.display_name,
    zone: "",
    description: input.description,
    status: "ready",
    runtime_status: "ready",
    is_default: false,
    can_delete: true,
    resource_count: 0,
  };
}

export async function deleteWorkspaceNamespace(id: string, confirmDisplayName: string): Promise<void> {
  const response = await fetch(`/api/v1/resource-platform/namespaces/${encodeURIComponent(id)}`, {
    cache: "no-store",
    method: "DELETE",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ confirm_display_name: confirmDisplayName }),
  });
  if (!response.ok) {
    throw new Error(await parseAPIError(response));
  }
}

export async function listWorkspaceInventory(): Promise<WorkspaceInventoryItem[]> {
  const payload = await readOptionalData<{ items?: WorkspaceInventoryItem[] } | WorkspaceInventoryItem[]>(
    "/api/v1/workspace/inventory",
  );
  if (payload == null) {
    return [];
  }
  return Array.isArray(payload) ? payload : payload.items ?? [];
}

export async function listWorkspaceNetworkPolicies(): Promise<WorkspaceNetworkPolicyListItem[]> {
  const payload = await readData<
    { items?: ResourcePlatformNetworkPolicyListItem[] } | ResourcePlatformNetworkPolicyListItem[]
  >("/api/v1/resource-platform/network-policies");
  const items = Array.isArray(payload) ? payload : payload.items ?? [];

  return items.map((item) => ({
    id: item.id,
    namespace: item.namespace_name,
    name: item.name,
    policy_type: item.policy_type ?? "Ingress",
    rules_summary: item.rules_summary ?? "",
    status: item.status ?? "draft",
    ingress_rule_count: 0,
    egress_rule_count: 0,
    updated_at: item.updated_at ?? "",
  }));
}

export async function getWorkspaceNetworkPolicy(id: string): Promise<WorkspaceNetworkPolicy> {
  const item = await readData<ResourcePlatformNetworkPolicyDetail>(
    `/api/v1/resource-platform/network-policies/${encodeURIComponent(id)}`,
  );

  const ingressRules = (item.rules ?? [])
    .filter((rule) => String(rule.direction).toLowerCase() === "ingress")
    .map((rule) => ({
      id: rule.id,
      source_raw: rule.source_raw,
      destination_raw: rule.destination_raw,
      source: rule.source_raw,
      destination: rule.destination_raw,
      description: rule.description,
    }));
  const egressRules = (item.rules ?? [])
    .filter((rule) => String(rule.direction).toLowerCase() === "egress")
    .map((rule) => ({
      id: rule.id,
      source_raw: rule.source_raw,
      destination_raw: rule.destination_raw,
      source: rule.source_raw,
      destination: rule.destination_raw,
      description: rule.description,
    }));

  return {
    id: item.id,
    namespace_id: item.namespace_id,
    namespace: item.namespace_name,
    name: item.name,
    default_ingress_behavior: item.default_ingress_behavior ?? "deny",
    default_egress_behavior: item.default_egress_behavior ?? "deny",
    policy_type: item.policy_type ?? "Ingress",
    rules_summary: item.rules_summary ?? "",
    status: item.status ?? "draft",
    ingress_rules: ingressRules,
    egress_rules: egressRules,
    updated_at: item.updated_at ?? "",
  };
}

export async function createWorkspaceNetworkPolicy(input: UpsertWorkspaceNetworkPolicyInput): Promise<WorkspaceNetworkPolicy> {
  const rules = [
    ...input.ingress_rules.map((rule, index) => ({
      direction: "ingress",
      sort_order: index,
      source_raw: rule.source,
      destination_raw: rule.destination,
      description: rule.description,
    })),
    ...input.egress_rules.map((rule, index) => ({
      direction: "egress",
      sort_order: input.ingress_rules.length + index,
      source_raw: rule.source,
      destination_raw: rule.destination,
      description: rule.description,
    })),
  ];

  const response = await fetch("/api/v1/resource-platform/network-policies", {
    cache: "no-store",
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      namespace_id: input.namespace_id,
      name: input.name,
      default_ingress_behavior: input.default_ingress_behavior ?? "deny",
      default_egress_behavior: input.default_egress_behavior ?? "deny",
      policy_type:
        input.ingress_rules.length > 0 && input.egress_rules.length > 0
          ? "Ingress/Egress"
          : input.ingress_rules.length > 0
            ? "Ingress"
            : "Egress",
      rules,
    }),
  });
  if (!response.ok) {
    throw new Error(await parseAPIError(response));
  }

  return {
    id:
      typeof crypto !== "undefined" && "randomUUID" in crypto
        ? crypto.randomUUID()
        : `workspace-network-policy-${Date.now()}`,
    namespace_id: input.namespace_id,
    namespace: "",
    name: input.name,
    default_ingress_behavior: input.default_ingress_behavior ?? "deny",
    default_egress_behavior: input.default_egress_behavior ?? "deny",
    policy_type:
      input.ingress_rules.length > 0 && input.egress_rules.length > 0
        ? "Ingress/Egress"
        : input.ingress_rules.length > 0
          ? "Ingress"
          : "Egress",
    rules_summary: input.rules_summary,
    status: input.status,
    ingress_rules: [],
    egress_rules: [],
    updated_at: new Date().toISOString(),
  };
}

export async function updateWorkspaceNetworkPolicy(id: string, input: UpsertWorkspaceNetworkPolicyInput): Promise<WorkspaceNetworkPolicy> {
  const rules = [
    ...input.ingress_rules.map((rule, index) => ({
      direction: "ingress",
      sort_order: index,
      source_raw: rule.source,
      destination_raw: rule.destination,
      description: rule.description,
    })),
    ...input.egress_rules.map((rule, index) => ({
      direction: "egress",
      sort_order: input.ingress_rules.length + index,
      source_raw: rule.source,
      destination_raw: rule.destination,
      description: rule.description,
    })),
  ];

  const response = await fetch(`/api/v1/resource-platform/network-policies/${encodeURIComponent(id)}`, {
    cache: "no-store",
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      name: input.name,
      default_ingress_behavior: input.default_ingress_behavior ?? "deny",
      default_egress_behavior: input.default_egress_behavior ?? "deny",
      policy_type:
        input.ingress_rules.length > 0 && input.egress_rules.length > 0
          ? "Ingress/Egress"
          : input.ingress_rules.length > 0
            ? "Ingress"
            : "Egress",
      status: input.status,
      rules,
    }),
  });
  if (!response.ok) {
    throw new Error(await parseAPIError(response));
  }

  return {
    id,
    namespace_id: input.namespace_id,
    namespace: "",
    name: input.name,
    default_ingress_behavior: input.default_ingress_behavior ?? "deny",
    default_egress_behavior: input.default_egress_behavior ?? "deny",
    policy_type:
      input.ingress_rules.length > 0 && input.egress_rules.length > 0
        ? "Ingress/Egress"
        : input.ingress_rules.length > 0
          ? "Ingress"
          : "Egress",
    rules_summary: input.rules_summary,
    status: input.status,
    ingress_rules: [],
    egress_rules: [],
    updated_at: new Date().toISOString(),
  };
}

export async function listWorkspaceMarketplaceCatalog(): Promise<WorkspaceMarketplaceCatalogItem[]> {
  const payload = await readOptionalData<{ items?: WorkspaceMarketplaceCatalogItem[] } | WorkspaceMarketplaceCatalogItem[]>(
    "/api/v1/workspace/marketplace/catalog",
  );
  if (payload == null) {
    return [];
  }
  return Array.isArray(payload) ? payload : payload.items ?? [];
}

export async function getWorkspaceMarketplaceDeployMetadata(resource: string): Promise<WorkspaceMarketplaceDeployMetadata> {
  return readData<WorkspaceMarketplaceDeployMetadata>(
    `/api/v1/workspace/marketplace/deploy-metadata?resource=${encodeURIComponent(resource)}`,
  );
}

export async function getWorkspaceMarketplaceDeployBootstrap(
  resource: string,
): Promise<WorkspaceMarketplaceDeployBootstrap> {
  return readData<WorkspaceMarketplaceDeployBootstrap>(
    `/api/v1/workspace/marketplace/deploy-bootstrap?resource=${encodeURIComponent(resource)}`,
  );
}

export async function listWorkspaceJobDefinitions(options?: {
  resourceType?: string;
  resourceModel?: string;
  status?: string;
}): Promise<WorkspaceJobDefinition[]> {
  const query = buildQuery({
    resource_type: options?.resourceType,
    resource_model: options?.resourceModel,
    status: options?.status,
  });
  const payload = await readData<
    { items?: WorkspaceJobDefinition[] } | WorkspaceJobDefinition[]
  >(`/api/v1/resource-platform/job-definitions${query ? `?${query}` : ""}`);

  const items = Array.isArray(payload) ? payload : payload.items ?? [];
  return items;
}

export async function createWorkspaceMarketplaceDeployment(
  input: CreateWorkspaceMarketplaceDeploymentInput,
): Promise<WorkspaceInventoryItem> {
  return readData<WorkspaceInventoryItem>("/api/v1/workspace/marketplace/deployments", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(input),
  });
}
