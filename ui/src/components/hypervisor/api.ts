"use client";

import { parseAPIError } from "@/components/auth/auth-utils";

export type HypervisorVirtualMachine = {
  id: string;
  owner_user_id: string;
  package_id: string;
  package_code: string;
  package_name: string;
  package_status: string;
  node_id: string;
  domain_uuid: string;
  name: string;
  description: string;
  zone: string;
  image: string;
  status: string;
  power_state: string;
  vcpu: number;
  ram_gb: number;
  disk_gb: number;
  primary_ip: string;
  last_seen_at: string;
  created_at: string;
  updated_at: string;
  runtime?: HypervisorVirtualMachineRuntime;
};

export type HypervisorVirtualMachineRuntime = {
  virtual_machine_id: string;
  node_id: string;
  domain_uuid: string;
  power_state: string;
  reason: string;
  cpu_usage_percent: number;
  ram_used_bytes: number;
  disk_read_bytes_per_sec: number;
  disk_write_bytes_per_sec: number;
  network_rx_bytes_per_sec: number;
  network_tx_bytes_per_sec: number;
  attached_gpus: Array<{
    vendor: string;
    model: string;
    driver_version: string;
    pci_address: string;
    uuid: string;
    memory_total_mb: number;
    serial: string;
  }>;
  gpu_metrics: Array<{
    vendor: string;
    model: string;
    pci_address: string;
    uuid: string;
    memory_total_mb: number;
    memory_used_mb: number;
    utilization_gpu_percent: number;
    utilization_memory_percent: number;
    temperature_celsius: number;
    power_watts: number;
    collected_at: string;
  }>;
  last_event_at: string;
  updated_at: string;
};

export type HypervisorVirtualMachineRuntimeState = {
  power_state: string;
  reason: string;
  last_event_at: string;
  updated_at: string;
};

export type HypervisorVirtualMachineState = {
  id: string;
  status: string;
  power_state: string;
  primary_ip: string;
  last_seen_at: string;
  updated_at: string;
  runtime?: HypervisorVirtualMachineRuntimeState;
};

export type HypervisorMetricPoint = {
  timestamp: string;
  value: number;
};

export type HypervisorMetricSeries = {
  name: string;
  unit: string;
  latest: number;
  points: HypervisorMetricPoint[];
};

export type HypervisorMetricStreamPayload = {
  scope: "virtual_machine" | "node";
  id: string;
  generated_at: string;
  window_sec: number;
  step_sec: number;
  series: HypervisorMetricSeries[];
};

export type HypervisorVirtualMachineCommand = {
  id: string;
  owner_user_id: string;
  node_id: string;
  virtual_machine_id: string;
  action: string;
  status: string;
  name: string;
  description: string;
  zone: string;
  image: string;
  vcpu: number;
  ram_gb: number;
  disk_gb: number;
  sent_at: string;
  completed_at: string;
  created_at: string;
  updated_at: string;
};

export type HypervisorFirewall = {
  ID?: string;
  OwnerUserID?: string;
  Name?: string;
  Target?: string;
  Status?: string;
  DefaultInboundPolicy?: string;
  DefaultOutboundPolicy?: string;
  CreatedAt?: string;
  UpdatedAt?: string;
  id?: string;
  owner_user_id?: string;
  name?: string;
  target?: string;
  status?: string;
  default_inbound_policy?: string;
  default_outbound_policy?: string;
  created_at?: string;
  updated_at?: string;
};

export type HypervisorFirewallRule = {
  id: string;
  firewall_id: string;
  direction: "inbound" | "outbound";
  name: string;
  protocol: "tcp" | "udp" | "icmp" | "all" | string;
  port_range: string;
  action: "allow" | "deny";
  sources: string[];
  destinations: string[];
  created_at: string;
  updated_at: string;
};

export type CreateVirtualMachineInput = {
  node_id?: string;
  package_id: string;
  name: string;
  description: string;
  zone: string;
  image: string;
  auth_mode: "password" | "ssh";
  password?: string;
  ssh_public_key?: string;
};

export type PlanPackageSpec = {
  vcpu: number;
  ram_gb: number;
  disk_gb: number;
};

export type PlanPackage = {
  id: string;
  resource_type: string;
  code: string;
  name: string;
  description: string;
  status: string;
  created_at: string;
  retired_at?: string;
  spec?: PlanPackageSpec;
};

export type HypervisorNode = {
  node_id: string;
  hostname: string;
  zone: string;
  status: string;
};

export type CreateFirewallInput = {
  name: string;
  target: string;
  default_inbound_policy: "allow" | "deny";
  default_outbound_policy: "allow" | "deny";
};

export type UpdateFirewallInput = {
  default_inbound_policy: "allow" | "deny";
  default_outbound_policy: "allow" | "deny";
};

export type CreateFirewallRuleInput = {
  direction: "inbound" | "outbound";
  name: string;
  protocol: "tcp" | "udp" | "icmp" | "all" | string;
  port_range: string;
  action: "allow" | "deny";
  sources: string[];
  destinations: string[];
};

async function readData<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(path, {
    cache: "no-store",
    ...init,
  });

  if (!response.ok) {
    throw new Error(await parseAPIError(response));
  }

  const payload = (await response.json()) as { data?: T };
  if (payload.data === undefined) {
    throw new Error("Missing API data payload.");
  }

  return payload.data;
}

export async function listHypervisorVirtualMachines(): Promise<HypervisorVirtualMachine[]> {
  const payload = await readData<{ items: HypervisorVirtualMachine[] }>("/api/v1/hypervisor/virtual-machines");
  return payload.items ?? [];
}

export async function getHypervisorVirtualMachine(id: string): Promise<HypervisorVirtualMachine> {
  return readData<HypervisorVirtualMachine>(`/api/v1/hypervisor/virtual-machines/${encodeURIComponent(id)}`);
}

export async function createHypervisorVirtualMachine(input: CreateVirtualMachineInput): Promise<{
  id: string;
  package_id: string;
  package_code: string;
  package_name: string;
  dispatch_state: string;
}> {
  return readData(`/api/v1/hypervisor/virtual-machines`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify(input),
  });
}

export async function listActiveVPSPackages(): Promise<PlanPackage[]> {
  const payload = await readData<{ items: PlanPackage[] }>(
    "/api/v1/plan/packages?resource_type=vps&status=active",
  );
  return payload.items ?? [];
}

export async function listHypervisorNodes(): Promise<HypervisorNode[]> {
  const payload = await readData<{ items: HypervisorNode[] }>("/api/v1/hypervisor/nodes");
  return payload.items ?? [];
}

export async function getHypervisorVirtualMachineState(id: string): Promise<HypervisorVirtualMachineState> {
  return readData<HypervisorVirtualMachineState>(`/api/v1/hypervisor/virtual-machines/${encodeURIComponent(id)}/state`);
}

export async function runHypervisorVirtualMachineAction(
  id: string,
  action: "start" | "stop" | "restart" | "delete",
): Promise<{
  dispatch_state: string;
}> {
  return readData(`/api/v1/hypervisor/virtual-machines/${encodeURIComponent(id)}/${action}`, {
    method: action === "delete" ? "DELETE" : "POST",
  });
}

export async function listHypervisorFirewalls(): Promise<HypervisorFirewall[]> {
  const payload = await readData<{ items: HypervisorFirewall[] }>("/api/v1/hypervisor/firewalls");
  return payload.items ?? [];
}

export async function getHypervisorFirewall(id: string): Promise<{
  firewall: HypervisorFirewall;
  rules: HypervisorFirewallRule[];
}> {
  return readData(`/api/v1/hypervisor/firewalls/${encodeURIComponent(id)}`);
}

export async function createHypervisorFirewall(input: CreateFirewallInput): Promise<HypervisorFirewall> {
  return readData("/api/v1/hypervisor/firewalls", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify(input),
  });
}

export async function updateHypervisorFirewall(id: string, input: UpdateFirewallInput): Promise<HypervisorFirewall> {
  return readData(`/api/v1/hypervisor/firewalls/${encodeURIComponent(id)}`, {
    method: "PATCH",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify(input),
  });
}

export async function createHypervisorFirewallRule(id: string, input: CreateFirewallRuleInput): Promise<HypervisorFirewallRule> {
  return readData(`/api/v1/hypervisor/firewalls/${encodeURIComponent(id)}/rules`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify(input),
  });
}

export function buildHypervisorMetricsWebSocketURL(id: string): string {
  if (typeof window === "undefined") {
    return "";
  }

  const url = new URL(window.location.origin);
  url.protocol = url.protocol === "https:" ? "wss:" : "ws:";
  url.pathname = "/api/v1/hypervisor/metrics/ws";
  url.searchParams.set("id", id);
  return url.toString();
}

export function normalizeFirewall(item: HypervisorFirewall) {
  return {
    id: item.id ?? item.ID ?? "",
    ownerUserId: item.owner_user_id ?? item.OwnerUserID ?? "",
    name: item.name ?? item.Name ?? "",
    target: item.target ?? item.Target ?? "",
    status: (item.status ?? item.Status ?? "disabled") as "active" | "disabled",
    defaultInboundPolicy: (item.default_inbound_policy ??
      item.DefaultInboundPolicy ??
      "deny") as "allow" | "deny",
    defaultOutboundPolicy: (item.default_outbound_policy ??
      item.DefaultOutboundPolicy ??
      "allow") as "allow" | "deny",
    createdAt: item.created_at ?? item.CreatedAt ?? "",
    updatedAt: item.updated_at ?? item.UpdatedAt ?? "",
  };
}
