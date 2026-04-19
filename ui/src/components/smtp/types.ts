export type SMTPTab = {
  key: string;
  label: string;
  caption: string;
  href: string;
};

export type LaneRoute = {
  templateName: string;
  endpointName: string;
  routeType: string;
  priority: number;
  status: string;
};

export type LaneItem = {
  id: string;
  name: string;
  trafficClass: string;
  priority: number;
  consumerId?: string;
  consumer?: string;
  status: string;
  shardCount?: number;
  readyShards?: number;
  drainingShards?: number;
  pendingShards?: number;
  runtimeVersion?: number;
  createdAt?: string;
  updatedAt?: string;
};

export type ConsumerItem = {
  id: string;
  name: string;
  stream: string;
  status: string;
  detail: string;
  transportType?: string;
  consumerGroup?: string;
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
  body: string;
  activeVersion?: number;
  runtimeVersion?: number;
  createdAt?: string;
  updatedAt?: string;
};

export type DeliveryEndpoint = {
  id: string;
  name: string;
  providerKind?: string;
  host: string;
  port: number;
  username: string;
  priority: number;
  weight: number;
  tlsMode: "none" | "starttls" | "tls" | "mtls";
  status: "active" | "draining" | "disabled" | "standby" | "maintenance";
  hasCACert?: boolean;
  hasClientCert?: boolean;
  hasClientKey?: boolean;
  provider?: string;
  note?: string;
  createdAt?: string;
  updatedAt?: string;
};
