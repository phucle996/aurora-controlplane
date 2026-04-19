"use client";

import type { ReactNode } from "react";
import { useEffect, useMemo, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import dynamic from "next/dynamic";
import type { ApexOptions } from "apexcharts";
import PageBreadcrumb from "@/components/common/PageBreadCrumb";
import ComponentCard from "@/components/common/ComponentCard";
import Button from "@/components/ui/button/Button";
import { useToast } from "@/components/ui/toast/ToastProvider";
import { TimeIcon } from "@/icons";
import {
  buildHypervisorMetricsWebSocketURL,
  getHypervisorFirewall,
  getHypervisorVirtualMachine,
  getHypervisorVirtualMachineState,
  listHypervisorFirewalls,
  normalizeFirewall,
  runHypervisorVirtualMachineAction,
  type HypervisorFirewallRule,
  type HypervisorMetricSeries,
  type HypervisorMetricStreamPayload,
  type HypervisorVirtualMachine,
} from "@/components/hypervisor/api";
import { initialsForVM, statusClasses, statusDotClasses, type VMStatus } from "@/components/virtual-machines/data";

const ReactApexChart = dynamic(() => import("react-apexcharts"), {
  ssr: false,
});

function tabButtonClasses(active: boolean) {
  return `border-b-2 px-1 pb-3 text-sm font-medium transition ${
    active
      ? "border-brand-500 text-gray-900 dark:text-white"
      : "border-transparent text-gray-500 hover:text-gray-800 dark:text-gray-400 dark:hover:text-white/80"
  }`;
}

function metricBar(value: number, tint: string) {
  return (
    <div className="mt-4 h-2.5 overflow-hidden rounded-full bg-gray-100 dark:bg-gray-800">
      <div className={`h-full rounded-full ${tint}`} style={{ width: `${value <= 0 ? 0 : Math.max(6, value)}%` }} />
    </div>
  );
}

function normalizeStatus(vm: HypervisorVirtualMachine): VMStatus {
  if (vm.power_state === "running" || vm.status === "running") return "running";
  if (vm.power_state === "stopped" || vm.status === "stopped") return "stopped";
  if (vm.power_state === "creating" || vm.status === "provisioning") return "starting";
  return "maintenance";
}

function relativeTime(value: string) {
  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) return "pending";
  const diffMs = Date.now() - parsed.getTime();
  const diffMinutes = Math.max(Math.floor(diffMs / 60000), 0);
  if (diffMinutes < 1) return "just now";
  if (diffMinutes < 60) return `${diffMinutes}m ago`;
  const diffHours = Math.floor(diffMinutes / 60);
  if (diffHours < 24) return `${diffHours}h ago`;
  return `${Math.floor(diffHours / 24)}d ago`;
}

function compactTimeLabel(value: string) {
  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) return "";
  return parsed.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
}

function zeroMetricSeries() {
  const now = Date.now();
  return Array.from({ length: 12 }, (_, index) => ({
    timestamp: new Date(now - (11 - index) * 30_000).toISOString(),
    value: 0,
  }));
}

function resourceChartOptions(color: string, unit: string, categories: string[]): ApexOptions {
  return {
    chart: {
      type: "area",
      toolbar: { show: false },
      sparkline: { enabled: false },
      fontFamily: "Outfit, sans-serif",
    },
    colors: [color],
    stroke: {
      curve: "smooth",
      width: 3,
    },
    fill: {
      type: "gradient",
      gradient: {
        opacityFrom: 0.34,
        opacityTo: 0.03,
      },
    },
    dataLabels: { enabled: false },
    grid: {
      borderColor: "#e5e7eb",
      strokeDashArray: 4,
    },
    xaxis: {
      categories,
      axisBorder: { show: false },
      axisTicks: { show: false },
      labels: {
        style: {
          colors: Array(categories.length).fill("#6b7280"),
          fontSize: "11px",
        },
      },
    },
    yaxis: {
      labels: {
        formatter: (value) => `${Math.round(value)}${unit}`,
        style: {
          colors: ["#6b7280"],
        },
      },
    },
    tooltip: {
      x: { show: true },
      y: {
        formatter: (value) => `${value.toFixed(unit === "%" ? 1 : 0)}${unit}`,
      },
    },
    legend: { show: false },
  };
}

export default function VirtualMachineDetailPage() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const { pushToast } = useToast();
  const vmID = searchParams.get("id")?.trim() ?? "";
  const [tab, setTab] = useState<"overview" | "firewall" | "snapshots">("overview");
  const [busyAction, setBusyAction] = useState<"" | "start" | "stop" | "restart">("");
  const [vm, setVM] = useState<HypervisorVirtualMachine | null>(null);
  const [liveSeries, setLiveSeries] = useState<Record<string, HypervisorMetricSeries>>({});
  const [firewall, setFirewall] = useState<ReturnType<typeof normalizeFirewall> | null>(null);
  const [firewallRules, setFirewallRules] = useState<HypervisorFirewallRule[]>([]);
  const [firewallLoading, setFirewallLoading] = useState(false);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  const normalizedStatus = useMemo(() => (vm ? normalizeStatus(vm) : "maintenance"), [vm]);
  const runtime = vm?.runtime;
  const primaryIP = vm?.primary_ip?.trim() || "";
  const sshCommand = primaryIP ? `ssh root@${primaryIP}` : "Pending primary IP";

  async function loadVM(options?: { silent?: boolean }) {
    const silent = options?.silent ?? false;
    if (!vmID) {
      setVM(null);
      setError("Missing virtual machine id.");
      setLoading(false);
      return;
    }

    try {
      if (!silent) {
        setLoading(true);
      }
      const nextVM = await getHypervisorVirtualMachine(vmID);
      setError("");
      setVM(nextVM);
    } catch (err) {
      const message = err instanceof Error ? err.message : "Failed to load virtual machine.";
      setError(message);
      if (!silent) {
        setVM(null);
      }
    } finally {
      if (!silent) {
        setLoading(false);
      }
    }
  }

  useEffect(() => {
    void loadVM({ silent: false });
  }, [vmID]);

  useEffect(() => {
    if (!vmID || normalizedStatus !== "running") {
      setLiveSeries({});
      return;
    }

    let socket: WebSocket | null = null;
    let reconnectTimer: number | null = null;
    let cancelled = false;

    const connect = () => {
      if (cancelled) {
        return;
      }

      socket = new WebSocket(buildHypervisorMetricsWebSocketURL(vmID));
      socket.onmessage = (event) => {
        try {
          const payload = JSON.parse(event.data) as HypervisorMetricStreamPayload | { type?: string; message?: string };
          if ("series" in payload && Array.isArray(payload.series)) {
            const next = payload.series.reduce<Record<string, HypervisorMetricSeries>>((acc, item) => {
              acc[item.name] = item;
              return acc;
            }, {});
            setLiveSeries(next);
          }
        } catch {
          // Ignore malformed frames and keep the last good metric snapshot.
        }
      };
      socket.onclose = () => {
        if (!cancelled) {
          reconnectTimer = window.setTimeout(connect, 3000);
        }
      };
      socket.onerror = () => {
        socket?.close();
      };
    };

    connect();

    return () => {
      cancelled = true;
      if (reconnectTimer !== null) {
        window.clearTimeout(reconnectTimer);
      }
      socket?.close();
    };
  }, [vmID, normalizedStatus]);

  useEffect(() => {
    async function loadFirewallForVM(currentVM: HypervisorVirtualMachine) {
      try {
        setFirewallLoading(true);
        const candidates = (await listHypervisorFirewalls()).map(normalizeFirewall);
        const wanted = [currentVM.id, currentVM.name, currentVM.domain_uuid]
          .map((value) => value.trim().toLowerCase())
          .filter((value) => value !== "");
        const selected =
          candidates.find((item) => item.target.trim().toLowerCase() === wanted[0]) ??
          candidates.find((item) => item.target.trim().toLowerCase() === wanted[1]) ??
          candidates.find((item) => item.target.trim().toLowerCase() === wanted[2]) ??
          null;

        if (!selected) {
          setFirewall(null);
          setFirewallRules([]);
          return;
        }

        const detail = await getHypervisorFirewall(selected.id);
        setFirewall(normalizeFirewall(detail.firewall));
        setFirewallRules(detail.rules);
      } catch {
        setFirewall(null);
        setFirewallRules([]);
      } finally {
        setFirewallLoading(false);
      }
    }

    if (!vm) {
      setFirewall(null);
      setFirewallRules([]);
      setFirewallLoading(false);
      return;
    }

    void loadFirewallForVM(vm);
  }, [vm]);

  async function runAction(action: "start" | "stop" | "restart") {
    if (!vm) return;
    try {
      setBusyAction(action);
      const payload = await runHypervisorVirtualMachineAction(vm.id, action);
      pushToast({
        kind: "success",
        message: `${action} command ${payload.dispatch_state === "dispatched" ? "dispatched" : "queued"}.`,
      });
      try {
        const nextState = await getHypervisorVirtualMachineState(vm.id);
        setVM((current) =>
          current
            ? {
                ...current,
                status: nextState.status,
                power_state: nextState.power_state,
                primary_ip: nextState.primary_ip,
                last_seen_at: nextState.last_seen_at,
                updated_at: nextState.updated_at,
                runtime: nextState.runtime
                  ? {
                      virtual_machine_id: current.runtime?.virtual_machine_id ?? current.id,
                      node_id: current.runtime?.node_id ?? current.node_id,
                      domain_uuid: current.runtime?.domain_uuid ?? current.domain_uuid,
                      power_state: nextState.runtime.power_state,
                      reason: nextState.runtime.reason,
                      cpu_usage_percent: current.runtime?.cpu_usage_percent ?? 0,
                      ram_used_bytes: current.runtime?.ram_used_bytes ?? 0,
                      disk_read_bytes_per_sec: current.runtime?.disk_read_bytes_per_sec ?? 0,
                      disk_write_bytes_per_sec: current.runtime?.disk_write_bytes_per_sec ?? 0,
                      network_rx_bytes_per_sec: current.runtime?.network_rx_bytes_per_sec ?? 0,
                      network_tx_bytes_per_sec: current.runtime?.network_tx_bytes_per_sec ?? 0,
                      attached_gpus: current.runtime?.attached_gpus ?? [],
                      gpu_metrics: current.runtime?.gpu_metrics ?? [],
                      last_event_at: nextState.runtime.last_event_at,
                      updated_at: nextState.runtime.updated_at,
                    }
                  : current.runtime,
              }
            : current,
        );
      } catch {
        // Keep the last known VM detail if the lightweight state refresh fails.
      }
    } catch (err) {
      pushToast({
        kind: "error",
        message: err instanceof Error ? err.message : "Failed to run VM action.",
      });
    } finally {
      setBusyAction("");
    }
  }

  function copyValue(value: string, successMessage: string) {
    if (value.trim() === "") {
      pushToast({ kind: "error", message: "No value is available to copy yet." });
      return;
    }
    void navigator.clipboard.writeText(value).then(
      () => pushToast({ kind: "success", message: successMessage }),
      () => pushToast({ kind: "error", message: "Failed to copy to clipboard." }),
    );
  }

  if (loading) {
    return (
      <div className="space-y-6">
        <PageBreadcrumb pageTitle="Virtual Machine Detail" />
        <section className="rounded-3xl border border-gray-200 bg-white p-8 text-center dark:border-gray-800 dark:bg-white/[0.03]">
          <p className="text-sm text-gray-500 dark:text-gray-400">Loading virtual machine...</p>
        </section>
      </div>
    );
  }

  if (!vm) {
    return (
      <div className="space-y-6">
        <PageBreadcrumb pageTitle="Virtual Machine Detail" />

        <section className="rounded-3xl border border-gray-200 bg-white p-8 text-center dark:border-gray-800 dark:bg-white/[0.03]">
          <div className="mx-auto max-w-xl space-y-4">
            <div className="inline-flex items-center rounded-full bg-rose-500/10 px-3 py-1 text-xs font-semibold uppercase tracking-[0.24em] text-rose-600 dark:text-rose-400">
              Not Found
            </div>
            <div>
              <h1 className="text-2xl font-semibold text-gray-900 dark:text-white">
                Virtual machine not found
              </h1>
              <p className="mt-2 text-sm leading-6 text-gray-500 dark:text-gray-400">
                {error || "VM nay co the da bi xoa, khong thuoc tai khoan hien tai, hoac lien ket detail khong con hop le."}
              </p>
            </div>
            <div className="flex flex-wrap items-center justify-center gap-3 pt-2">
              <Button className="rounded-xl px-5" onClick={() => router.push("/virtual-machines")}>
                Back to List
              </Button>
            </div>
          </div>
        </section>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <PageBreadcrumb pageTitle="Virtual Machine Detail" />

      <section className="rounded-3xl border border-gray-200 bg-white p-6 dark:border-gray-800 dark:bg-white/[0.03]">
        <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
          <div className="space-y-4">
            <div className="flex flex-col gap-4 sm:flex-row sm:items-center">
              <div className="flex h-16 w-16 items-center justify-center rounded-3xl bg-brand-500/10 text-lg font-semibold text-brand-600 dark:text-brand-400">
                {initialsForVM(vm.name)}
              </div>
              <div>
                <div className="flex flex-wrap items-center gap-3">
                  <h1 className="text-3xl font-semibold text-gray-900 dark:text-white">{vm.name}</h1>
                  <span
                    className={`inline-flex items-center rounded-full px-3 py-1 text-xs font-semibold ${statusClasses(normalizedStatus)}`}
                  >
                    <span className={`mr-2 h-2 w-2 rounded-full ${statusDotClasses(normalizedStatus)}`} />
                    {normalizedStatus}
                  </span>
                </div>
                <p className="mt-2 text-sm text-gray-500 dark:text-gray-400">
                  {vm.image || "Unknown image"} · {vm.zone || "Unassigned zone"} · {vm.package_name && vm.package_code
                    ? `${vm.package_name} (${vm.package_code}) · `
                    : ""}{vm.id}
                </p>
              </div>
            </div>
          </div>

          <div className="flex flex-wrap items-center gap-3">
            <Button
              variant="outline"
              className="rounded-xl px-5"
              onClick={() => void runAction("start")}
              disabled={busyAction !== "" || normalizedStatus === "running" || normalizedStatus === "starting"}
            >
              <ActionLabel busy={busyAction === "start"} idle="Khoi dong" loading="Dang khoi dong" />
            </Button>
            <Button
              variant="outline"
              className="rounded-xl px-5"
              onClick={() => void runAction("stop")}
              disabled={busyAction !== "" || normalizedStatus === "stopped"}
            >
              <ActionLabel busy={busyAction === "stop"} idle="Dung" loading="Dang dung" />
            </Button>
            <Button
              variant="outline"
              className="rounded-xl px-5"
              onClick={() => void runAction("restart")}
              disabled={busyAction !== "" || normalizedStatus !== "running"}
            >
              <ActionLabel busy={busyAction === "restart"} idle="Khoi dong lai" loading="Dang khoi dong lai" />
            </Button>
          </div>
        </div>
      </section>

      <section className="grid gap-6 xl:grid-cols-[minmax(0,1.6fr)_320px]">
        <div className="space-y-6">
          <ComponentCard title="Instance Summary" desc="Core runtime identity, compute shape, and connectivity for this VM.">
            <div className="grid gap-4 xl:grid-cols-2">
              <SummaryItem label="Operating System" value={vm.image || "Unknown image"} />
              <SummaryItem label="Zone" value={vm.zone || "Unassigned"} />
              <SummaryItem
                label="Package"
                value={vm.package_name ? `${vm.package_name} (${vm.package_code})` : "Unassigned package"}
              />
              <SummaryItem label="Package Status" value={vm.package_status || "unknown"} />
            </div>

            <div className="grid gap-4 md:grid-cols-4">
              <InlineMetric label="vCPU" value={`${vm.vcpu} vCPU`} />
              <InlineMetric label="Memory" value={`${vm.ram_gb} GB RAM`} />
              <InlineMetric label="Disk" value={`${vm.disk_gb} GB SSD`} />
              <InlineMetric label="Uptime" value={relativeTime(vm.last_seen_at)} icon={<TimeIcon className="size-4" />} />
            </div>
          </ComponentCard>

          <section className="rounded-2xl border border-gray-200 bg-white dark:border-gray-800 dark:bg-white/[0.03]">
            <div className="border-b border-gray-200 px-6 pt-5 dark:border-gray-800">
              <div className="flex flex-wrap items-center gap-6">
                <button type="button" className={tabButtonClasses(tab === "overview")} onClick={() => setTab("overview")}>
                  Tong quan
                </button>
                <button type="button" className={tabButtonClasses(tab === "firewall")} onClick={() => setTab("firewall")}>
                  Firewall
                </button>
                <button type="button" className={tabButtonClasses(tab === "snapshots")} onClick={() => setTab("snapshots")}>
                  Snapshots
                </button>
              </div>
            </div>

            <div className="space-y-6 p-6">
              {tab === "overview" ? <OverviewTab vm={vm} liveSeries={liveSeries} /> : null}
              {tab === "firewall" ? <FirewallTab firewall={firewall} rules={firewallRules} loading={firewallLoading} /> : null}
              {tab === "snapshots" ? <SnapshotsTab vm={vm} /> : null}
            </div>
          </section>
        </div>

        <div className="space-y-6">
          <ComponentCard title="Connectivity" desc="Ready-made values for copy/paste during maintenance and support.">
            <div className="space-y-4 text-sm">
              <KeyValue label="Primary IP" value={primaryIP || "Pending primary IP"} />
              <KeyValue label="SSH Command" value={sshCommand} monospace />
            </div>
            <div className="flex flex-wrap gap-3">
              <Button variant="outline" className="rounded-xl px-4" onClick={() => copyValue(primaryIP, "Primary IP copied.")}>
                Copy IP
              </Button>
              <Button variant="outline" className="rounded-xl px-4" onClick={() => copyValue(sshCommand, "SSH command copied.")}>
                Copy SSH
              </Button>
              <Button className="rounded-xl px-4" onClick={() => router.push("/virtual-machines")}>
                Back to List
              </Button>
            </div>
          </ComponentCard>
        </div>
      </section>
    </div>
  );
}

function ActionLabel({
  busy,
  idle,
  loading,
}: {
  busy: boolean;
  idle: string;
  loading: string;
}) {
  if (!busy) return <span>{idle}</span>;
  return (
    <>
      <svg
        className="size-4 animate-spin"
        viewBox="0 0 24 24"
        fill="none"
        xmlns="http://www.w3.org/2000/svg"
        aria-hidden="true"
      >
        <circle cx="12" cy="12" r="9" stroke="currentColor" strokeOpacity="0.25" strokeWidth="2" />
        <path d="M21 12A9 9 0 0 0 12 3" stroke="currentColor" strokeWidth="2" strokeLinecap="round" />
      </svg>
      <span>{loading}</span>
    </>
  );
}

function OverviewTab({
  vm,
  liveSeries,
}: {
  vm: HypervisorVirtualMachine;
  liveSeries: Record<string, HypervisorMetricSeries>;
}) {
  const runtime = vm.runtime;
  const allocatedMemoryBytes = vm.ram_gb * 1024 * 1024 * 1024;
  const isRunning = normalizeStatus(vm) === "running";
  const cpuUsage = isRunning ? (liveSeries.cpu_usage_percent?.latest ?? runtime?.cpu_usage_percent ?? 0) : 0;
  const memoryBytes = isRunning ? (liveSeries.ram_used_bytes?.latest ?? runtime?.ram_used_bytes ?? 0) : 0;
  const memoryPercent = allocatedMemoryBytes > 0 ? (memoryBytes / allocatedMemoryBytes) * 100 : 0;
  const diskRate = isRunning ? (liveSeries.disk_io_bytes_per_sec?.latest ?? ((runtime?.disk_read_bytes_per_sec ?? 0) + (runtime?.disk_write_bytes_per_sec ?? 0))) : 0;
  const networkRate = isRunning ? (liveSeries.network_bytes_per_sec?.latest ?? ((runtime?.network_rx_bytes_per_sec ?? 0) + (runtime?.network_tx_bytes_per_sec ?? 0))) : 0;
  const cpuPoints = isRunning ? (liveSeries.cpu_usage_percent?.points ?? zeroMetricSeries()) : zeroMetricSeries();
  const memoryPoints = isRunning
    ? ((liveSeries.ram_used_bytes?.points ?? zeroMetricSeries()).map((point) => ({
        ...point,
        value: allocatedMemoryBytes > 0 ? (point.value / allocatedMemoryBytes) * 100 : 0,
      })))
    : zeroMetricSeries();
  const diskPoints = isRunning ? (liveSeries.disk_io_bytes_per_sec?.points ?? zeroMetricSeries()) : zeroMetricSeries();
  const networkPoints = isRunning ? (liveSeries.network_bytes_per_sec?.points ?? zeroMetricSeries()) : zeroMetricSeries();

  return (
    <>
      <div className="grid gap-4 xl:grid-cols-4">
        <MetricCard
          title="CPU Usage"
          value={`${cpuUsage.toFixed(1)}%`}
          body={metricBar(cpuUsage, "bg-emerald-500")}
        />
        <MetricCard
          title="Memory Usage"
          value={`${formatBytes(memoryBytes)} / ${formatBytes(allocatedMemoryBytes)}`}
          body={metricBar(memoryPercent, "bg-brand-500")}
        />
        <MetricCard
          title="Disk IO"
          value={formatRate(diskRate)}
          body={metricBar(Math.min((diskRate / (1024 * 1024)) * 10, 100), "bg-amber-500")}
        />
        <MetricCard
          title="Network Throughput"
          value={formatRate(networkRate)}
          body={metricBar(Math.min((networkRate / (1024 * 1024)) * 10, 100), "bg-blue-light-500")}
        />
      </div>

      <div className="grid gap-4 xl:grid-cols-2">
        <ResourceChartCard
          title="CPU Usage Trend"
          subtitle="Last live CPU samples from the hypervisor node."
          value={`${cpuUsage.toFixed(1)}%`}
          color="#10b981"
          unit="%"
          points={cpuPoints}
        />
        <ResourceChartCard
          title="Memory Usage Trend"
          subtitle="Allocated memory pressure for this virtual machine."
          value={`${memoryPercent.toFixed(1)}%`}
          color="#3b82f6"
          unit="%"
          points={memoryPoints}
        />
        <ResourceChartCard
          title="Disk I/O Trend"
          subtitle="Read and write throughput aggregated over time."
          value={formatRate(diskRate)}
          color="#f59e0b"
          unit=" MB/s"
          points={diskPoints.map((point) => ({
            ...point,
            value: point.value / (1024 * 1024),
          }))}
        />
        <ResourceChartCard
          title="Network Throughput Trend"
          subtitle="Ingress and egress throughput reported by the guest interface."
          value={formatRate(networkRate)}
          color="#0ea5e9"
          unit=" MB/s"
          points={networkPoints.map((point) => ({
            ...point,
            value: point.value / (1024 * 1024),
          }))}
        />
      </div>
    </>
  );
}

function FirewallTab({
  firewall,
  rules,
  loading,
}: {
  firewall: ReturnType<typeof normalizeFirewall> | null;
  rules: HypervisorFirewallRule[];
  loading: boolean;
}) {
  const inboundRules = rules.filter((rule) => rule.direction === "inbound");
  const outboundRules = rules.filter((rule) => rule.direction === "outbound");

  if (loading) {
    return (
      <ComponentCard title="Attached Firewall" desc="Loading the firewall matched to this virtual machine target.">
        <div className="rounded-2xl border border-dashed border-gray-200 px-4 py-8 text-sm text-gray-500 dark:border-gray-800 dark:text-gray-400">
          Loading firewall...
        </div>
      </ComponentCard>
    );
  }

  if (!firewall) {
    return (
      <ComponentCard title="Attached Firewall" desc="Firewall information is resolved from the current target mapping.">
        <div className="rounded-2xl border border-dashed border-gray-200 px-4 py-8 text-sm text-gray-500 dark:border-gray-800 dark:text-gray-400">
          No firewall target currently matches this virtual machine.
        </div>
      </ComponentCard>
    );
  }

  return (
    <div className="space-y-6">
      <ComponentCard title="Attached Firewall" desc="Firewall currently matched to this virtual machine target.">
        <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
          <KeyValue label="Firewall Name" value={firewall.name || "Unnamed firewall"} />
          <KeyValue label="Target" value={firewall.target || "No target"} />
          <KeyValue label="Inbound Default" value={firewall.defaultInboundPolicy} />
          <KeyValue label="Outbound Default" value={firewall.defaultOutboundPolicy} />
        </div>
      </ComponentCard>

      <div className="grid gap-6 xl:grid-cols-2">
        <FirewallRuleCard title="Inbound Rules" rules={inboundRules} />
        <FirewallRuleCard title="Outbound Rules" rules={outboundRules} />
      </div>
    </div>
  );
}

function SnapshotsTab({ vm }: { vm: HypervisorVirtualMachine }) {
  return (
    <ComponentCard
      title="Snapshot API"
      desc="Snapshot inventory is not exposed by the current hypervisor HTTP API yet."
    >
      <div className="rounded-2xl border border-dashed border-gray-200 px-4 py-8 text-sm text-gray-500 dark:border-gray-800 dark:text-gray-400">
        Snapshot list and restore points will appear here once the hypervisor backend exposes snapshot endpoints for VM <span className="font-medium text-gray-900 dark:text-white">{vm.name}</span>.
      </div>
    </ComponentCard>
  );
}

function SummaryItem({
  label,
  value,
}: {
  label: string;
  value: string;
}) {
  return (
    <div className="rounded-2xl border border-gray-200 bg-gray-50 px-4 py-4 dark:border-gray-800 dark:bg-gray-900/60">
      <p className="text-xs uppercase tracking-[0.2em] text-gray-400 dark:text-gray-500">{label}</p>
      <p className="mt-3 break-all text-sm font-medium text-gray-900 dark:text-white">{value}</p>
    </div>
  );
}

function InlineMetric({ label, value, icon }: { label: string; value: string; icon?: ReactNode }) {
  return (
    <div className="rounded-2xl border border-gray-200 bg-gray-50 px-4 py-4 dark:border-gray-800 dark:bg-gray-900/60">
      <p className="text-xs uppercase tracking-[0.2em] text-gray-400 dark:text-gray-500">{label}</p>
      <div className="mt-3 flex items-center gap-2 text-sm font-medium text-gray-900 dark:text-white">
        {icon}
        <span>{value}</span>
      </div>
    </div>
  );
}

function MetricCard({
  title,
  value,
  body,
}: {
  title: string;
  value: string;
  body: ReactNode;
}) {
  return (
    <div className="rounded-2xl border border-gray-200 bg-white px-5 py-5 dark:border-gray-800 dark:bg-gray-900/60">
      <p className="text-sm font-medium text-gray-500 dark:text-gray-400">{title}</p>
      <p className="mt-3 text-2xl font-semibold text-gray-900 dark:text-white">{value}</p>
      {body}
    </div>
  );
}

function ResourceChartCard({
  title,
  subtitle,
  value,
  color,
  unit,
  points,
}: {
  title: string;
  subtitle: string;
  value: string;
  color: string;
  unit: string;
  points: Array<{ timestamp: string; value: number }>;
}) {
  const categories = points.map((point) => compactTimeLabel(point.timestamp));
  const series = [
    {
      name: title,
      data: points.map((point) => Number(point.value.toFixed(2))),
    },
  ];

  return (
    <ComponentCard title={title} desc={subtitle}>
      <div className="mb-4 flex items-end justify-between gap-3">
        <div>
          <p className="text-3xl font-semibold text-gray-900 dark:text-white">{value}</p>
          <p className="mt-1 text-xs uppercase tracking-[0.2em] text-gray-400 dark:text-gray-500">
            Live resource history
          </p>
        </div>
        <div
          className="inline-flex items-center rounded-full px-3 py-1 text-xs font-semibold"
          style={{ backgroundColor: `${color}1A`, color }}
        >
          {points.length} samples
        </div>
      </div>
      <ReactApexChart
        type="area"
        height={240}
        options={resourceChartOptions(color, unit, categories)}
        series={series}
      />
    </ComponentCard>
  );
}

function KeyValue({ label, value, monospace = false }: { label: string; value: string; monospace?: boolean }) {
  return (
    <div className="rounded-2xl border border-gray-200 bg-gray-50 px-4 py-4 dark:border-gray-800 dark:bg-gray-900/60">
      <p className="text-xs uppercase tracking-[0.2em] text-gray-400 dark:text-gray-500">{label}</p>
      <p className={`mt-3 break-all text-sm text-gray-900 dark:text-white ${monospace ? "font-mono" : "font-medium"}`}>
        {value}
      </p>
    </div>
  );
}

function FirewallRuleCard({
  title,
  rules,
}: {
  title: string;
  rules: HypervisorFirewallRule[];
}) {
  return (
    <ComponentCard title={title} desc="Rules currently resolved from the selected firewall.">
      {rules.length === 0 ? (
        <div className="rounded-2xl border border-dashed border-gray-200 px-4 py-8 text-sm text-gray-500 dark:border-gray-800 dark:text-gray-400">
          No rules in this direction.
        </div>
      ) : (
        <div className="space-y-3">
          {rules.map((rule) => (
            <div
              key={rule.id}
              className="rounded-2xl border border-gray-200 bg-gray-50 px-4 py-4 dark:border-gray-800 dark:bg-gray-900/60"
            >
              <div className="flex flex-wrap items-center justify-between gap-3">
                <div>
                  <p className="font-medium text-gray-900 dark:text-white">{rule.name}</p>
                  <p className="mt-1 text-xs uppercase tracking-[0.18em] text-gray-500 dark:text-gray-400">
                    {rule.protocol} · {rule.port_range}
                  </p>
                </div>
                <span
                  className={`inline-flex rounded-full px-3 py-1 text-xs font-semibold uppercase ${
                    rule.action === "allow"
                      ? "bg-brand-500/10 text-brand-600 dark:text-brand-400"
                      : "bg-amber-500/12 text-amber-600 dark:text-amber-400"
                  }`}
                >
                  {rule.action}
                </span>
              </div>
              <div className="mt-4 grid gap-3 md:grid-cols-2">
                <RuleMetaList label="Sources" items={rule.sources} />
                <RuleMetaList label="Destinations" items={rule.destinations} />
              </div>
            </div>
          ))}
        </div>
      )}
    </ComponentCard>
  );
}

function RuleMetaList({ label, items }: { label: string; items: string[] }) {
  return (
    <div className="space-y-2">
      <p className="text-xs uppercase tracking-[0.18em] text-gray-400 dark:text-gray-500">{label}</p>
      {items.length === 0 ? (
        <span className="text-sm text-gray-400 dark:text-gray-500">None</span>
      ) : (
        <div className="flex flex-wrap gap-2">
          {items.map((item) => (
            <span
              key={`${label}-${item}`}
              className="inline-flex rounded-full border border-gray-200 bg-white px-2.5 py-1 text-xs text-gray-700 dark:border-gray-700 dark:bg-gray-950/40 dark:text-gray-300"
            >
              {item}
            </span>
          ))}
        </div>
      )}
    </div>
  );
}

function formatBytes(value: number) {
  if (!Number.isFinite(value) || value <= 0) {
    return "0 B";
  }
  const units = ["B", "KB", "MB", "GB", "TB"];
  let size = value;
  let unitIndex = 0;
  while (size >= 1024 && unitIndex < units.length - 1) {
    size /= 1024;
    unitIndex += 1;
  }
  return `${size.toFixed(size >= 10 || unitIndex === 0 ? 0 : 1)} ${units[unitIndex]}`;
}

function formatRate(value: number) {
  return `${formatBytes(value)}/s`;
}
