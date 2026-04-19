export type VMStatus = "running" | "stopped" | "starting" | "maintenance";

export type VMNetworkInterface = {
  id: string;
  ipAddress: string;
  status: "attached" | "detached";
  label: string;
};

export type VMTask = {
  id: string;
  name: string;
  status: "ready" | "queued" | "paused";
};

export type VMImageFamily = "linux" | "apps" | "custom";

export type VMImageOption = {
  id: string;
  family: VMImageFamily;
  name: string;
  description: string;
};

export type VMItem = {
  id: string;
  name: string;
  os: string;
  zone: string;
  status: VMStatus;
  vcpu: number;
  ramGb: number;
  diskGb: number;
  ipAddress: string;
  uptime: string;
  cpuUsage: number;
  memoryUsage: number;
  diskUsage: number;
  networkMbps: number;
  sshCommand: string;
  interfaces: VMNetworkInterface[];
  tasks: VMTask[];
};

export const vmItems: VMItem[] = [
  {
    id: "vm-b7b0c1b6-2ab1-4a7e-9bdb-011f801be11a",
    name: "mail-worker-01",
    os: "Ubuntu 24.04 LTS",
    zone: "ap-southeast-1a",
    status: "running",
    vcpu: 4,
    ramGb: 8,
    diskGb: 120,
    ipAddress: "10.17.24.11",
    uptime: "12d 04h",
    cpuUsage: 32,
    memoryUsage: 58,
    diskUsage: 46,
    networkMbps: 148,
    sshCommand: "ssh ubuntu@10.17.24.11",
    interfaces: [
      { id: "nic-001", ipAddress: "10.17.24.11", status: "attached", label: "Public Interface" },
      { id: "nic-002", ipAddress: "172.16.10.11", status: "attached", label: "Private Backend" },
    ],
    tasks: [
      { id: "task-001", name: "Console", status: "ready" },
      { id: "task-002", name: "SSH Session", status: "ready" },
      { id: "task-003", name: "Snapshot Daily", status: "queued" },
    ],
  },
  {
    id: "vm-79965561-d838-4f49-a8ce-cc65fdbde512",
    name: "queue-consumer-02",
    os: "Ubuntu 22.04 LTS",
    zone: "ap-southeast-1b",
    status: "running",
    vcpu: 2,
    ramGb: 4,
    diskGb: 80,
    ipAddress: "10.17.24.29",
    uptime: "3d 17h",
    cpuUsage: 44,
    memoryUsage: 61,
    diskUsage: 37,
    networkMbps: 86,
    sshCommand: "ssh ubuntu@10.17.24.29",
    interfaces: [
      { id: "nic-010", ipAddress: "10.17.24.29", status: "attached", label: "Public Interface" },
    ],
    tasks: [
      { id: "task-010", name: "Console", status: "ready" },
      { id: "task-011", name: "SSH Session", status: "ready" },
    ],
  },
  {
    id: "vm-52f66544-67d4-4bc7-a95b-c8c7cbca28be",
    name: "billing-api-blue",
    os: "Debian 12",
    zone: "ap-southeast-1a",
    status: "starting",
    vcpu: 8,
    ramGb: 16,
    diskGb: 160,
    ipAddress: "10.17.19.77",
    uptime: "booting",
    cpuUsage: 18,
    memoryUsage: 24,
    diskUsage: 41,
    networkMbps: 22,
    sshCommand: "ssh debian@10.17.19.77",
    interfaces: [
      { id: "nic-020", ipAddress: "10.17.19.77", status: "attached", label: "Primary NIC" },
    ],
    tasks: [
      { id: "task-020", name: "Boot Sequence", status: "queued" },
      { id: "task-021", name: "Health Check", status: "queued" },
    ],
  },
  {
    id: "vm-06c7b2b3-d097-4484-bf86-73ce4d0ddcd0",
    name: "campaign-render-03",
    os: "Rocky Linux 9",
    zone: "ap-southeast-1c",
    status: "maintenance",
    vcpu: 8,
    ramGb: 32,
    diskGb: 240,
    ipAddress: "10.17.40.18",
    uptime: "maintenance",
    cpuUsage: 8,
    memoryUsage: 19,
    diskUsage: 62,
    networkMbps: 12,
    sshCommand: "ssh rocky@10.17.40.18",
    interfaces: [
      { id: "nic-030", ipAddress: "10.17.40.18", status: "attached", label: "Primary NIC" },
      { id: "nic-031", ipAddress: "172.16.44.18", status: "detached", label: "Maintenance VLAN" },
    ],
    tasks: [
      { id: "task-030", name: "Patch Window", status: "paused" },
      { id: "task-031", name: "Disk Check", status: "queued" },
    ],
  },
  {
    id: "vm-9d4c0f1f-cd84-48f3-810c-5a6ce7a3b002",
    name: "analytics-batch-01",
    os: "Ubuntu 24.04 LTS",
    zone: "ap-southeast-1b",
    status: "stopped",
    vcpu: 16,
    ramGb: 64,
    diskGb: 320,
    ipAddress: "10.17.51.90",
    uptime: "stopped",
    cpuUsage: 0,
    memoryUsage: 0,
    diskUsage: 29,
    networkMbps: 0,
    sshCommand: "ssh ubuntu@10.17.51.90",
    interfaces: [
      { id: "nic-040", ipAddress: "10.17.51.90", status: "attached", label: "Primary NIC" },
    ],
    tasks: [
      { id: "task-040", name: "Resume Job Queue", status: "paused" },
    ],
  },
  {
    id: "vm-fc9e1b95-a042-4f08-8c0d-6b0dbb70337f",
    name: "edge-proxy-01",
    os: "AlmaLinux 9",
    zone: "ap-southeast-1a",
    status: "running",
    vcpu: 2,
    ramGb: 2,
    diskGb: 40,
    ipAddress: "10.17.12.4",
    uptime: "29d 09h",
    cpuUsage: 21,
    memoryUsage: 49,
    diskUsage: 51,
    networkMbps: 213,
    sshCommand: "ssh almalinux@10.17.12.4",
    interfaces: [
      { id: "nic-050", ipAddress: "10.17.12.4", status: "attached", label: "Edge NIC" },
      { id: "nic-051", ipAddress: "172.16.1.4", status: "attached", label: "Service Mesh" },
    ],
    tasks: [
      { id: "task-050", name: "Console", status: "ready" },
      { id: "task-051", name: "SSH Session", status: "ready" },
      { id: "task-052", name: "TLS Rotation", status: "queued" },
    ],
  },
];

export const vmImageOptions: VMImageOption[] = [
  { id: "ubuntu-24", family: "linux", name: "Ubuntu 24.04", description: "LTS image for general workloads" },
  { id: "debian-12", family: "linux", name: "Debian 12", description: "Stable base image for long-running services" },
  { id: "alpine-3", family: "linux", name: "Alpine 3.20", description: "Small footprint image for lightweight nodes" },
  { id: "windows-server", family: "linux", name: "Windows Server", description: "Microsoft Windows Server image for enterprise workloads" },
  { id: "docker-host", family: "apps", name: "Docker Host", description: "Preloaded with Docker engine and compose" },
  { id: "node-runtime", family: "apps", name: "Node Runtime", description: "Node.js and PM2 ready image" },
  { id: "golden-image", family: "custom", name: "Golden Image", description: "Private base image from your snapshots" },
];

export function statusClasses(status: VMStatus) {
  switch (status) {
    case "running":
      return "bg-emerald-500/12 text-emerald-600 dark:text-emerald-400";
    case "stopped":
      return "bg-rose-500/12 text-rose-600 dark:text-rose-400";
    case "starting":
      return "bg-amber-500/12 text-amber-600 dark:text-amber-400";
    default:
      return "bg-blue-light-500/12 text-blue-light-700 dark:text-blue-light-400";
  }
}

export function statusDotClasses(status: VMStatus) {
  switch (status) {
    case "running":
      return "bg-emerald-500";
    case "stopped":
      return "bg-rose-500";
    case "starting":
      return "bg-amber-500";
    default:
      return "bg-blue-light-500";
  }
}

export function initialsForVM(name: string) {
  return name
    .split("-")
    .slice(0, 2)
    .map((part) => part.charAt(0).toUpperCase())
    .join("");
}

export function findVMByID(id: string) {
  return vmItems.find((item) => item.id === id);
}
