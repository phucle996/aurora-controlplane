export type FirewallDirection = "inbound" | "outbound";
export type FirewallStatus = "active" | "disabled";
export type FirewallAction = "allow" | "deny";
export type FirewallDefaultPolicy = "allow" | "deny";

export type FirewallRule = {
  id: string;
  name: string;
  direction: FirewallDirection;
  protocol: "tcp" | "udp" | "icmp" | "all";
  portRange: string;
  sources: string[];
  destinations: string[];
  action: FirewallAction;
};

export type FirewallItem = {
  id: string;
  name: string;
  target: string;
  attachedResources: number;
  defaultInboundPolicy: FirewallDefaultPolicy;
  defaultOutboundPolicy: FirewallDefaultPolicy;
  status: FirewallStatus;
  rules: FirewallRule[];
};

export const firewallItems: FirewallItem[] = [
  {
    id: "fw-cluster-public-edge",
    name: "Public Edge Firewall",
    target: "public-edge",
    attachedResources: 3,
    defaultInboundPolicy: "deny",
    defaultOutboundPolicy: "allow",
    status: "active",
    rules: [
      {
        id: "rule-001",
        name: "Allow HTTPS",
        direction: "inbound",
        protocol: "tcp",
        portRange: "443",
        sources: ["0.0.0.0/0", "::/0"],
        destinations: ["edge-proxy-01"],
        action: "allow",
      },
      {
        id: "rule-002",
        name: "Allow SSH Admin",
        direction: "inbound",
        protocol: "tcp",
        portRange: "22",
        sources: ["203.113.0.0/16", "10.20.0.0/16"],
        destinations: ["edge-proxy-01"],
        action: "allow",
      },
      {
        id: "rule-003",
        name: "Allow Metrics Export",
        direction: "outbound",
        protocol: "tcp",
        portRange: "9090",
        sources: ["edge-proxy-01"],
        destinations: ["10.17.0.0/16", "metrics.internal"],
        action: "allow",
      },
      {
        id: "rule-004",
        name: "Deny Unknown SMTP",
        direction: "outbound",
        protocol: "tcp",
        portRange: "25",
        sources: ["edge-proxy-01"],
        destinations: ["0.0.0.0/0"],
        action: "deny",
      },
    ],
  },
  {
    id: "fw-auth-relay",
    name: "Auth Relay Firewall",
    target: "mail-worker-01",
    attachedResources: 2,
    defaultInboundPolicy: "deny",
    defaultOutboundPolicy: "deny",
    status: "active",
    rules: [
      {
        id: "rule-101",
        name: "Allow Relay API",
        direction: "inbound",
        protocol: "tcp",
        portRange: "8080-8082",
        sources: ["10.17.0.0/16"],
        destinations: ["mail-worker-01", "queue-consumer-02"],
        action: "allow",
      },
      {
        id: "rule-102",
        name: "Allow Redis Egress",
        direction: "outbound",
        protocol: "tcp",
        portRange: "6379",
        sources: ["mail-worker-01"],
        destinations: ["redis.internal", "10.17.21.10"],
        action: "allow",
      },
    ],
  },
  {
    id: "fw-batch-jobs",
    name: "Batch Jobs Firewall",
    target: "analytics-batch-01",
    attachedResources: 1,
    defaultInboundPolicy: "deny",
    defaultOutboundPolicy: "deny",
    status: "disabled",
    rules: [
      {
        id: "rule-201",
        name: "Allow Warehouse Sync",
        direction: "outbound",
        protocol: "tcp",
        portRange: "5432",
        sources: ["analytics-batch-01"],
        destinations: ["warehouse.internal"],
        action: "allow",
      },
    ],
  },
];

export function findFirewallByID(id: string) {
  return firewallItems.find((item) => item.id === id);
}
