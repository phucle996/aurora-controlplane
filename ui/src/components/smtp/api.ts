"use client";

import { parseAPIError } from "@/components/auth/auth-utils";
import type {
  ConsumerItem,
  ConsumerOption,
  DeliveryEndpoint,
  GatewayDetail,
  GatewayItem,
  SMTPOverview,
  SMTPWorkspaceOption,
  TemplateItem,
} from "@/components/smtp/types";

type APIEnvelope<T> = {
  data?: T;
  message?: string;
};

type ListPayload<T> = {
  items?: T[];
};

function workspacePath(workspaceID: string, suffix: string) {
  void workspaceID;
  return `/api/v1/smtp${suffix}`;
}

async function readData<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(path, {
    cache: "no-store",
    credentials: "include",
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

async function readItems<T>(path: string, init?: RequestInit): Promise<T[]> {
  const payload = await readData<ListPayload<T>>(path, init);
  return payload.items ?? [];
}

async function mutate<T>(path: string, init?: RequestInit): Promise<{ data: T | null; message: string }> {
  const response = await fetch(path, {
    cache: "no-store",
    credentials: "include",
    ...init,
  });
  if (!response.ok) {
    throw new Error(await parseAPIError(response));
  }
  const payload = (await response.json()) as APIEnvelope<T>;
  return {
    data: payload.data ?? null,
    message: payload.message ?? "",
  };
}

type WorkspaceOptionDTO = {
  id: string;
  name: string;
  slug: string;
  status: string;
  default_zone_id: string;
  default_zone_name: string;
};

type OverviewDTO = {
  metrics: {
    delivered_today: number;
    queued_now: number;
    active_gateways: number;
    total_gateways: number;
    live_templates: number;
    total_templates: number;
  };
  delivery_throughput: Array<{
    label: string;
    delivered: number;
    queued: number;
    retries: number;
  }>;
  health_distribution: {
    healthy: number;
    warning: number;
    stopped: number;
  };
  queue_mix: Array<{
    category: string;
    pending: number;
    processing: number;
    retries: number;
  }>;
  gateways: GatewayItemDTO[];
  timeline: Array<{
    id: string;
    entity_type: string;
    entity_name: string;
    action: string;
    actor_name: string;
    note: string;
    created_at: string;
  }>;
};

type ConsumerDTO = {
  id: string;
  zone_id: string;
  name: string;
  transport_type: string;
  source: string;
  consumer_group: string;
  worker_concurrency: number;
  ack_timeout_seconds: number;
  batch_size: number;
  status: string;
  note: string;
  connection_config?: Record<string, unknown>;
  desired_shard_count: number;
  has_secret: boolean;
  created_at?: string;
  updated_at?: string;
};

type ConsumerOptionDTO = {
  id: string;
  label: string;
  status: string;
};

type TemplateDTO = {
  id: string;
  name: string;
  category: string;
  traffic_class: string;
  subject: string;
  from_email: string;
  to_email: string;
  status: string;
  variables?: string[];
  consumer_id?: string;
  consumer_name?: string;
  text_body?: string;
  html_body?: string;
  active_version?: number;
  runtime_version?: number;
  created_at?: string;
  updated_at?: string;
};

type GatewayItemDTO = {
  id: string;
  name: string;
  traffic_class: string;
  status: string;
  routing_mode: string;
  priority: number;
  desired_shard_count: number;
  template_count: number;
  endpoint_count: number;
  ready_shards: number;
  pending_shards: number;
  draining_shards: number;
  fallback_gateway_name: string;
  updated_at?: string;
};

type GatewayDetailDTO = {
  id: string;
  name: string;
  traffic_class: string;
  status: string;
  routing_mode: string;
  priority: number;
  desired_shard_count: number;
  runtime_version: number;
  fallback_gateway?: {
    id: string;
    name: string;
    status: string;
  } | null;
  templates?: Array<{
    id: string;
    name: string;
    category: string;
    traffic_class: string;
    status: string;
    consumer_id?: string;
    consumer_name: string;
    selected: boolean;
    position: number;
  }>;
  endpoints?: Array<{
    id: string;
    name: string;
    host: string;
    port: number;
    username: string;
    status: DeliveryEndpoint["status"];
    selected: boolean;
    position: number;
  }>;
  ready_shards: number;
  pending_shards: number;
  draining_shards: number;
  created_at?: string;
  updated_at?: string;
};

type EndpointDTO = {
  id: string;
  name: string;
  provider_kind?: string;
  host: string;
  port: number;
  username: string;
  priority: number;
  weight: number;
  max_connections: number;
  max_parallel_sends: number;
  max_messages_per_second: number;
  burst: number;
  warmup_state: string;
  status: DeliveryEndpoint["status"];
  tls_mode: "none" | "starttls" | "tls" | "mtls";
  has_secret: boolean;
  has_ca_cert?: boolean;
  has_client_cert?: boolean;
  has_client_key?: boolean;
  created_at?: string;
  updated_at?: string;
};

function mapWorkspaceOption(item: WorkspaceOptionDTO): SMTPWorkspaceOption {
  return {
    id: item.id,
    name: item.name,
    slug: item.slug,
    status: item.status,
    defaultZoneID: item.default_zone_id,
    defaultZoneName: item.default_zone_name,
  };
}

function mapConsumer(item: ConsumerDTO): ConsumerItem {
  return {
    id: item.id,
    zoneId: item.zone_id,
    name: item.name,
    transportType: item.transport_type,
    source: item.source,
    consumerGroup: item.consumer_group,
    workerConcurrency: item.worker_concurrency,
    ackTimeoutSeconds: item.ack_timeout_seconds,
    batchSize: item.batch_size,
    status: item.status,
    note: item.note,
    connectionConfig: item.connection_config,
    desiredShardCount: item.desired_shard_count,
    hasSecret: item.has_secret,
    createdAt: item.created_at,
    updatedAt: item.updated_at,
  };
}

function mapTemplate(item: TemplateDTO): TemplateItem {
  return {
    id: item.id,
    name: item.name,
    category: item.category,
    trafficClass: item.traffic_class,
    subject: item.subject,
    from: item.from_email,
    to: item.to_email,
    status: item.status,
    variables: item.variables ?? [],
    consumerId: item.consumer_id ?? "",
    consumer: item.consumer_name ?? "",
    textBody: item.text_body ?? "",
    htmlBody: item.html_body ?? "",
    activeVersion: item.active_version,
    runtimeVersion: item.runtime_version,
    createdAt: item.created_at,
    updatedAt: item.updated_at,
  };
}

function mapGatewayItem(item: GatewayItemDTO): GatewayItem {
  return {
    id: item.id,
    name: item.name,
    trafficClass: item.traffic_class,
    status: item.status,
    routingMode: item.routing_mode,
    priority: item.priority,
    desiredShardCount: item.desired_shard_count,
    templateCount: item.template_count,
    endpointCount: item.endpoint_count,
    readyShards: item.ready_shards,
    pendingShards: item.pending_shards,
    drainingShards: item.draining_shards,
    fallbackGatewayName: item.fallback_gateway_name,
    updatedAt: item.updated_at,
  };
}

function mapGatewayDetail(item: GatewayDetailDTO): GatewayDetail {
  return {
    id: item.id,
    name: item.name,
    trafficClass: item.traffic_class,
    status: item.status,
    routingMode: item.routing_mode,
    priority: item.priority,
    desiredShardCount: item.desired_shard_count,
    runtimeVersion: item.runtime_version,
    fallbackGateway: item.fallback_gateway
      ? {
          id: item.fallback_gateway.id,
          name: item.fallback_gateway.name,
          status: item.fallback_gateway.status,
        }
      : null,
    templates: (item.templates ?? []).map((template) => ({
      id: template.id,
      name: template.name,
      category: template.category,
      trafficClass: template.traffic_class,
      status: template.status,
      consumerId: template.consumer_id ?? "",
      consumerName: template.consumer_name,
      selected: template.selected,
      position: template.position,
    })),
    endpoints: (item.endpoints ?? []).map((endpoint) => ({
      id: endpoint.id,
      name: endpoint.name,
      host: endpoint.host,
      port: endpoint.port,
      username: endpoint.username,
      status: endpoint.status,
      selected: endpoint.selected,
      position: endpoint.position,
    })),
    readyShards: item.ready_shards,
    pendingShards: item.pending_shards,
    drainingShards: item.draining_shards,
    createdAt: item.created_at,
    updatedAt: item.updated_at,
  };
}

function mapEndpoint(item: EndpointDTO): DeliveryEndpoint {
  return {
    id: item.id,
    name: item.name,
    providerKind: item.provider_kind ?? "smtp",
    host: item.host,
    port: item.port,
    username: item.username,
    priority: item.priority,
    weight: item.weight,
    maxConnections: item.max_connections,
    maxParallelSends: item.max_parallel_sends,
    maxMessagesPerSecond: item.max_messages_per_second,
    burst: item.burst,
    warmupState: item.warmup_state,
    status: item.status,
    tlsMode: item.tls_mode,
    hasSecret: item.has_secret,
    hasCACert: item.has_ca_cert ?? false,
    hasClientCert: item.has_client_cert ?? false,
    hasClientKey: item.has_client_key ?? false,
    createdAt: item.created_at,
    updatedAt: item.updated_at,
  };
}

function mapOverview(item: OverviewDTO): SMTPOverview {
  return {
    metrics: {
      deliveredToday: item.metrics.delivered_today,
      queuedNow: item.metrics.queued_now,
      activeGateways: item.metrics.active_gateways,
      totalGateways: item.metrics.total_gateways,
      liveTemplates: item.metrics.live_templates,
      totalTemplates: item.metrics.total_templates,
    },
    deliveryThroughput: item.delivery_throughput.map((point) => ({
      label: point.label,
      delivered: point.delivered,
      queued: point.queued,
      retries: point.retries,
    })),
    healthDistribution: {
      healthy: item.health_distribution.healthy,
      warning: item.health_distribution.warning,
      stopped: item.health_distribution.stopped,
    },
    queueMix: item.queue_mix.map((entry) => ({
      category: entry.category,
      pending: entry.pending,
      processing: entry.processing,
      retries: entry.retries,
    })),
    gateways: item.gateways.map(mapGatewayItem),
    timeline: item.timeline.map((entry) => ({
      id: entry.id,
      entityType: entry.entity_type,
      entityName: entry.entity_name,
      action: entry.action,
      actorName: entry.actor_name,
      note: entry.note,
      createdAt: entry.created_at,
    })),
  };
}

export async function listWorkspaceOptions(): Promise<SMTPWorkspaceOption[]> {
  const items = await readItems<WorkspaceOptionDTO>("/api/v1/workspaces/options");
  return items.map(mapWorkspaceOption);
}

export async function getSMTPOverview(workspaceID: string): Promise<SMTPOverview> {
  const data = await readData<OverviewDTO>(workspacePath(workspaceID, "/overview"));
  return mapOverview(data);
}

export async function listConsumers(workspaceID: string): Promise<ConsumerItem[]> {
  const items = await readItems<ConsumerDTO>(workspacePath(workspaceID, "/consumers"));
  return items.map(mapConsumer);
}

export async function getConsumer(workspaceID: string, consumerID: string): Promise<ConsumerItem> {
  const data = await readData<ConsumerDTO>(workspacePath(workspaceID, `/consumers/${consumerID}`));
  return mapConsumer(data);
}

export async function listConsumerOptions(workspaceID: string): Promise<ConsumerOption[]> {
  const items = await readItems<ConsumerOptionDTO>(workspacePath(workspaceID, "/templates/options/consumers"));
  return items.map((item) => ({
    id: item.id,
    label: item.label,
    status: item.status,
  }));
}

export async function createConsumer(workspaceID: string, input: Record<string, unknown>): Promise<ConsumerItem> {
  const result = await mutate<ConsumerDTO>(workspacePath(workspaceID, "/consumers"), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(input),
  });
  if (result.data == null) {
    throw new Error("Missing consumer payload.");
  }
  return mapConsumer(result.data);
}

export async function updateConsumer(workspaceID: string, consumerID: string, input: Record<string, unknown>): Promise<ConsumerItem> {
  const result = await mutate<ConsumerDTO>(workspacePath(workspaceID, `/consumers/${consumerID}`), {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(input),
  });
  if (result.data == null) {
    throw new Error("Missing consumer payload.");
  }
  return mapConsumer(result.data);
}

export async function deleteConsumer(workspaceID: string, consumerID: string): Promise<void> {
  await mutate<null>(workspacePath(workspaceID, `/consumers/${consumerID}`), {
    method: "DELETE",
  });
}

export async function tryConnectConsumer(workspaceID: string, input: Record<string, unknown>): Promise<string> {
  const result = await mutate<null>(workspacePath(workspaceID, "/consumers/try-connect"), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(input),
  });
  return result.message || "Consumer connection succeeded.";
}

export async function listTemplates(workspaceID: string): Promise<TemplateItem[]> {
  const items = await readItems<TemplateDTO>(workspacePath(workspaceID, "/templates"));
  return items.map(mapTemplate);
}

export async function getTemplate(workspaceID: string, templateID: string): Promise<TemplateItem> {
  const data = await readData<TemplateDTO>(workspacePath(workspaceID, `/templates/${templateID}`));
  return mapTemplate(data);
}

export async function createTemplate(workspaceID: string, input: Record<string, unknown>): Promise<TemplateItem> {
  const result = await mutate<TemplateDTO>(workspacePath(workspaceID, "/templates"), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(input),
  });
  if (result.data == null) {
    throw new Error("Missing template payload.");
  }
  return mapTemplate(result.data);
}

export async function updateTemplate(workspaceID: string, templateID: string, input: Record<string, unknown>): Promise<TemplateItem> {
  const result = await mutate<TemplateDTO>(workspacePath(workspaceID, `/templates/${templateID}`), {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(input),
  });
  if (result.data == null) {
    throw new Error("Missing template payload.");
  }
  return mapTemplate(result.data);
}

export async function deleteTemplate(workspaceID: string, templateID: string): Promise<void> {
  await mutate<null>(workspacePath(workspaceID, `/templates/${templateID}`), {
    method: "DELETE",
  });
}

export async function listGateways(workspaceID: string): Promise<GatewayItem[]> {
  const items = await readItems<GatewayItemDTO>(workspacePath(workspaceID, "/gateways"));
  return items.map(mapGatewayItem);
}

export async function getGatewayDetail(workspaceID: string, gatewayID: string): Promise<GatewayDetail> {
  const data = await readData<GatewayDetailDTO>(workspacePath(workspaceID, `/gateways/${gatewayID}/detail`));
  return mapGatewayDetail(data);
}

export async function createGateway(workspaceID: string, input: Record<string, unknown>): Promise<GatewayDetail> {
  const result = await mutate<GatewayDetailDTO>(workspacePath(workspaceID, "/gateways"), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(input),
  });
  if (result.data == null) {
    throw new Error("Missing gateway payload.");
  }
  return mapGatewayDetail(result.data);
}

export async function updateGateway(workspaceID: string, gatewayID: string, input: Record<string, unknown>): Promise<GatewayDetail> {
  const result = await mutate<GatewayDetailDTO>(workspacePath(workspaceID, `/gateways/${gatewayID}`), {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(input),
  });
  if (result.data == null) {
    throw new Error("Missing gateway payload.");
  }
  return mapGatewayDetail(result.data);
}

export async function deleteGateway(workspaceID: string, gatewayID: string): Promise<void> {
  await mutate<null>(workspacePath(workspaceID, `/gateways/${gatewayID}`), {
    method: "DELETE",
  });
}

export async function bindGatewayTemplates(workspaceID: string, gatewayID: string, templateIDs: string[]): Promise<GatewayDetail> {
  const result = await mutate<GatewayDetailDTO>(workspacePath(workspaceID, `/gateways/${gatewayID}/templates`), {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ template_ids: templateIDs }),
  });
  if (result.data == null) {
    throw new Error("Missing gateway payload.");
  }
  return mapGatewayDetail(result.data);
}

export async function bindGatewayEndpoints(workspaceID: string, gatewayID: string, endpointIDs: string[]): Promise<GatewayDetail> {
  const result = await mutate<GatewayDetailDTO>(workspacePath(workspaceID, `/gateways/${gatewayID}/endpoints`), {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ endpoint_ids: endpointIDs }),
  });
  if (result.data == null) {
    throw new Error("Missing gateway payload.");
  }
  return mapGatewayDetail(result.data);
}

async function mutateGatewayState(workspaceID: string, gatewayID: string, action: "start" | "drain" | "disable"): Promise<GatewayDetail> {
  const result = await mutate<GatewayDetailDTO>(workspacePath(workspaceID, `/gateways/${gatewayID}/${action}`), {
    method: "POST",
  });
  if (result.data == null) {
    throw new Error("Missing gateway payload.");
  }
  return mapGatewayDetail(result.data);
}

export function startGateway(workspaceID: string, gatewayID: string) {
  return mutateGatewayState(workspaceID, gatewayID, "start");
}

export function drainGateway(workspaceID: string, gatewayID: string) {
  return mutateGatewayState(workspaceID, gatewayID, "drain");
}

export function disableGateway(workspaceID: string, gatewayID: string) {
  return mutateGatewayState(workspaceID, gatewayID, "disable");
}

export async function listEndpoints(workspaceID: string): Promise<DeliveryEndpoint[]> {
  const items = await readItems<EndpointDTO>(workspacePath(workspaceID, "/endpoints"));
  return items.map(mapEndpoint);
}

export async function getEndpoint(workspaceID: string, endpointID: string): Promise<DeliveryEndpoint> {
  const data = await readData<EndpointDTO>(workspacePath(workspaceID, `/endpoints/${endpointID}`));
  return mapEndpoint(data);
}

export async function createEndpoint(workspaceID: string, input: Record<string, unknown>): Promise<DeliveryEndpoint> {
  const result = await mutate<EndpointDTO>(workspacePath(workspaceID, "/endpoints"), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(input),
  });
  if (result.data == null) {
    throw new Error("Missing endpoint payload.");
  }
  return mapEndpoint(result.data);
}

export async function updateEndpoint(workspaceID: string, endpointID: string, input: Record<string, unknown>): Promise<DeliveryEndpoint> {
  const result = await mutate<EndpointDTO>(workspacePath(workspaceID, `/endpoints/${endpointID}`), {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(input),
  });
  if (result.data == null) {
    throw new Error("Missing endpoint payload.");
  }
  return mapEndpoint(result.data);
}

export async function deleteEndpoint(workspaceID: string, endpointID: string): Promise<void> {
  await mutate<null>(workspacePath(workspaceID, `/endpoints/${endpointID}`), {
    method: "DELETE",
  });
}

export async function tryConnectEndpoint(workspaceID: string, input: Record<string, unknown>): Promise<string> {
  const result = await mutate<null>(workspacePath(workspaceID, "/endpoints/try-connect"), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(input),
  });
  return result.message || "Endpoint connection succeeded.";
}
