export type SMTPTab = {
  key: string;
  label: string;
  caption: string;
  href: string;
};

export type SMTPWorkspaceOption = {
  id: string;
  name: string;
  slug: string;
  status: string;
  defaultZoneID: string;
  defaultZoneName: string;
};

export type SMTPOverview = {
  metrics: {
    deliveredToday: number;
    queuedNow: number;
    activeGateways: number;
    totalGateways: number;
    liveTemplates: number;
    totalTemplates: number;
  };
  deliveryThroughput: Array<{
    label: string;
    delivered: number;
    queued: number;
    retries: number;
  }>;
  healthDistribution: {
    healthy: number;
    warning: number;
    stopped: number;
  };
  queueMix: Array<{
    category: string;
    pending: number;
    processing: number;
    retries: number;
  }>;
  gateways: GatewayItem[];
  timeline: Array<{
    id: string;
    entityType: string;
    entityName: string;
    action: string;
    actorName: string;
    note: string;
    createdAt: string;
  }>;
};

export type ConsumerOption = {
  id: string;
  label: string;
  status: string;
};

export type ConsumerItem = {
  id: string;
  zoneId: string;
  name: string;
  transportType: string;
  source: string;
  consumerGroup: string;
  workerConcurrency: number;
  ackTimeoutSeconds: number;
  batchSize: number;
  status: string;
  note: string;
  connectionConfig?: Record<string, unknown>;
  desiredShardCount: number;
  hasSecret: boolean;
  createdAt?: string;
  updatedAt?: string;
};

export type TemplateItem = {
  id: string;
  name: string;
  category: string;
  trafficClass: string;
  subject: string;
  from: string;
  to: string;
  status: string;
  variables: string[];
  consumerId?: string;
  consumer: string;
  textBody: string;
  htmlBody: string;
  activeVersion?: number;
  runtimeVersion?: number;
  createdAt?: string;
  updatedAt?: string;
};

export type GatewayItem = {
  id: string;
  name: string;
  trafficClass: string;
  status: string;
  routingMode: string;
  priority: number;
  desiredShardCount: number;
  templateCount: number;
  endpointCount: number;
  readyShards: number;
  pendingShards: number;
  drainingShards: number;
  fallbackGatewayName: string;
  updatedAt?: string;
};

export type GatewayBindingTemplate = {
  id: string;
  name: string;
  category: string;
  trafficClass: string;
  status: string;
  consumerId?: string;
  consumerName: string;
  selected: boolean;
  position: number;
};

export type GatewayBindingEndpoint = {
  id: string;
  name: string;
  host: string;
  port: number;
  username: string;
  status: DeliveryEndpoint["status"];
  selected: boolean;
  position: number;
};

export type GatewayDetail = {
  id: string;
  name: string;
  trafficClass: string;
  status: string;
  routingMode: string;
  priority: number;
  desiredShardCount: number;
  runtimeVersion: number;
  fallbackGateway: { id: string; name: string; status: string } | null;
  templates: GatewayBindingTemplate[];
  endpoints: GatewayBindingEndpoint[];
  readyShards: number;
  pendingShards: number;
  drainingShards: number;
  createdAt?: string;
  updatedAt?: string;
};

export type DeliveryEndpoint = {
  id: string;
  name: string;
  providerKind: string;
  host: string;
  port: number;
  username: string;
  priority: number;
  weight: number;
  maxConnections: number;
  maxParallelSends: number;
  maxMessagesPerSecond: number;
  burst: number;
  warmupState: string;
  tlsMode: "none" | "starttls" | "tls" | "mtls";
  status: "active" | "draining" | "disabled";
  hasSecret: boolean;
  hasCACert?: boolean;
  hasClientCert?: boolean;
  hasClientKey?: boolean;
  createdAt?: string;
  updatedAt?: string;
};

export type GatewayMutationAction = "start" | "drain" | "disable";
