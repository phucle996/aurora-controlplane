"use client";

import { useEffect, useMemo, useState } from "react";
import { createPortal } from "react-dom";
import dynamic from "next/dynamic";
import type { ApexOptions } from "apexcharts";
import ComponentCard from "@/components/common/ComponentCard";
import { getSMTPOverview } from "@/components/smtp/api";
import { useSMTPWorkspace } from "@/components/smtp/SMTPWorkspaceProvider";
import type { SMTPOverview } from "@/components/smtp/types";

const ReactApexChart = dynamic(() => import("react-apexcharts"), {
  ssr: false,
});

const baseDeliveryOptions: ApexOptions = {
  chart: {
    type: "area",
    toolbar: { show: false },
    fontFamily: "Outfit, sans-serif",
  },
  colors: ["#1d4ed8", "#60a5fa", "#f97316"],
  stroke: {
    curve: "smooth",
    width: [3, 3, 2],
  },
  fill: {
    type: "gradient",
    gradient: {
      opacityFrom: 0.35,
      opacityTo: 0.03,
    },
  },
  dataLabels: { enabled: false },
  legend: {
    position: "top",
    horizontalAlign: "left",
    fontFamily: "Outfit, sans-serif",
  },
  grid: {
    borderColor: "#e5e7eb",
    strokeDashArray: 4,
  },
  xaxis: {
    categories: [],
    axisBorder: { show: false },
    axisTicks: { show: false },
  },
  yaxis: {
    labels: {
      style: {
        colors: ["#6b7280"],
      },
    },
  },
  tooltip: {
    shared: true,
    intersect: false,
  },
};

const healthOptions: ApexOptions = {
  chart: {
    type: "donut",
    toolbar: { show: false },
    fontFamily: "Outfit, sans-serif",
  },
  labels: ["Healthy", "Warning", "Stopped"],
  colors: ["#10b981", "#f59e0b", "#ef4444"],
  stroke: {
    colors: ["#ffffff"],
  },
  dataLabels: {
    enabled: false,
  },
  legend: {
    position: "bottom",
    fontFamily: "Outfit, sans-serif",
  },
  plotOptions: {
    pie: {
      donut: {
        size: "72%",
      },
    },
  },
};

const baseQueueMixOptions: ApexOptions = {
  chart: {
    type: "bar",
    stacked: true,
    toolbar: { show: false },
    fontFamily: "Outfit, sans-serif",
  },
  colors: ["#2563eb", "#14b8a6", "#f97316"],
  plotOptions: {
    bar: {
      horizontal: true,
      barHeight: "52%",
      borderRadius: 6,
    },
  },
  dataLabels: { enabled: false },
  grid: {
    borderColor: "#eef2f7",
  },
  xaxis: {
    categories: [],
    axisBorder: { show: false },
    axisTicks: { show: false },
  },
  legend: {
    position: "top",
    horizontalAlign: "left",
  },
};

export function GeneralTab() {
  const { workspace, workspaceID, isLoading: isWorkspaceLoading, error: workspaceError } = useSMTPWorkspace();
  const [overview, setOverview] = useState<SMTPOverview | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState("");
  const [isChartReady, setIsChartReady] = useState(false);
  const [isTimelineDialogOpen, setIsTimelineDialogOpen] = useState(false);
  const [timelineSearch, setTimelineSearch] = useState("");

  useEffect(() => {
    if (workspaceID === "") {
      setOverview(null);
      setIsLoading(false);
      setError("");
      return;
    }

    let cancelled = false;

    async function loadOverview() {
      setIsLoading(true);
      setError("");

      try {
        const result = await getSMTPOverview(workspaceID);
        if (!cancelled) {
          setOverview(result);
        }
      } catch (err) {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : "Failed to load SMTP overview");
        }
      } finally {
        if (!cancelled) {
          setIsLoading(false);
        }
      }
    }

    void loadOverview();
    return () => {
      cancelled = true;
    };
  }, [workspaceID]);

  useEffect(() => {
    setIsChartReady(true);
  }, []);

  const throughput = useMemo(() => overview?.deliveryThroughput ?? [], [overview]);
  const queueMix = useMemo(() => overview?.queueMix ?? [], [overview]);
  const timeline = useMemo(() => overview?.timeline ?? [], [overview]);
  const timelinePreview = useMemo(() => timeline.slice(0, 10), [timeline]);
  const metrics = overview?.metrics;
  const health = overview?.healthDistribution;
  const filteredTimeline = useMemo(() => {
    const keyword = timelineSearch.trim().toLowerCase();
    if (keyword === "") {
      return timeline;
    }
    return timeline.filter((item) => {
      const haystack = [
        item.entityName,
        item.entityType,
        item.action,
        item.actorName,
        item.note,
        formatTimelineTitle(item),
        formatTimelineDetail(item),
      ]
        .join(" ")
        .toLowerCase();
      return haystack.includes(keyword);
    });
  }, [timeline, timelineSearch]);

  const deliverySeries = useMemo(
    () => [
      {
        name: "Delivered",
        data: throughput.map((item) => item.delivered),
      },
      {
        name: "Queued",
        data: throughput.map((item) => item.queued),
      },
      {
        name: "Retries",
        data: throughput.map((item) => item.retries),
      },
    ],
    [throughput],
  );

  const deliveryOptions = useMemo<ApexOptions>(
    () => ({
      ...baseDeliveryOptions,
      xaxis: {
        ...(baseDeliveryOptions.xaxis ?? {}),
        categories: throughput.map((item) => item.label),
      },
    }),
    [throughput],
  );

  const queueMixSeries = useMemo(
    () => [
      { name: "Pending", data: queueMix.map((item) => item.pending) },
      { name: "Processing", data: queueMix.map((item) => item.processing) },
      { name: "Retry", data: queueMix.map((item) => item.retries) },
    ],
    [queueMix],
  );

  const queueMixOptions = useMemo<ApexOptions>(
    () => ({
      ...baseQueueMixOptions,
      xaxis: {
        ...(baseQueueMixOptions.xaxis ?? {}),
        categories: queueMix.map((item) => item.category),
      },
    }),
    [queueMix],
  );

  const peakHour = useMemo(() => {
    if (throughput.length === 0) {
      return { label: "--:--", value: 0 };
    }
    return throughput.reduce(
      (best, item) =>
        item.delivered > best.value ? { label: item.label, value: item.delivered } : best,
      { label: throughput[0].label, value: throughput[0].delivered },
    );
  }, [throughput]);

  const retryWindow = useMemo(() => {
    if (throughput.length === 0) {
      return { label: "--:--", value: 0 };
    }
    return throughput.reduce(
      (best, item) =>
        item.retries > best.value ? { label: item.label, value: item.retries } : best,
      { label: throughput[0].label, value: throughput[0].retries },
    );
  }, [throughput]);

  const currentMix = useMemo(() => {
    if (queueMix.length === 0) {
      return { label: "No traffic", value: 0 };
    }
    return queueMix.reduce((best, item) => {
      const total = item.pending + item.processing + item.retries;
      return total > best.value ? { label: item.category, value: total } : best;
    }, { label: queueMix[0].category, value: queueMix[0].pending + queueMix[0].processing + queueMix[0].retries });
  }, [queueMix]);

  if (isWorkspaceLoading) {
    return (
      <ComponentCard title="SMTP Overview" desc="Resolving workspace context...">
        <div className="rounded-2xl border border-gray-200 bg-gray-50 px-4 py-4 text-sm text-gray-500 dark:border-gray-800 dark:bg-gray-900/30 dark:text-gray-400">
          Loading workspaces...
        </div>
      </ComponentCard>
    );
  }

  if (workspaceError !== "") {
    return (
      <ComponentCard title="SMTP Overview" desc="Workspace resolution failed.">
        <div className="rounded-2xl border border-rose-200 bg-rose-50 px-4 py-4 text-sm text-rose-700 dark:border-rose-900/60 dark:bg-rose-950/30 dark:text-rose-200">
          {workspaceError}
        </div>
      </ComponentCard>
    );
  }

  if (workspace == null) {
    return (
      <ComponentCard title="SMTP Overview" desc="Choose a workspace to load SMTP data.">
        <div className="rounded-2xl border border-gray-200 bg-gray-50 px-4 py-4 text-sm text-gray-500 dark:border-gray-800 dark:bg-gray-900/30 dark:text-gray-400">
          No workspace is available for SMTP yet.
        </div>
      </ComponentCard>
    );
  }

  if (isLoading) {
    return (
      <div className="space-y-6">
        <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
          {Array.from({ length: 4 }).map((_, index) => (
            <div
              key={index}
              className="h-[148px] animate-pulse rounded-3xl border border-gray-200 bg-gray-100 dark:border-gray-800 dark:bg-gray-900"
            />
          ))}
        </div>
        <div className="grid gap-6 xl:grid-cols-[1.55fr_0.95fr]">
          <div className="h-[540px] animate-pulse rounded-2xl border border-gray-200 bg-gray-100 dark:border-gray-800 dark:bg-gray-900" />
          <div className="grid gap-6">
            <div className="h-[300px] animate-pulse rounded-2xl border border-gray-200 bg-gray-100 dark:border-gray-800 dark:bg-gray-900" />
            <div className="h-[280px] animate-pulse rounded-2xl border border-gray-200 bg-gray-100 dark:border-gray-800 dark:bg-gray-900" />
          </div>
        </div>
      </div>
    );
  }

  if (error !== "") {
    return (
      <ComponentCard title="SMTP Overview" desc="Runtime aggregation could not be loaded from backend.">
        <div className="rounded-2xl border border-rose-200 bg-rose-50 px-4 py-4 text-sm text-rose-700 dark:border-rose-900/60 dark:bg-rose-950/30 dark:text-rose-200">
          {error}
        </div>
      </ComponentCard>
    );
  }

  return (
    <div className="space-y-6">
      <section className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
        <MetricCard
          title="Delivered Today"
          value={formatNumber(metrics?.deliveredToday ?? 0)}
          delta={`${formatNumber(throughput.reduce((sum, item) => sum + item.delivered, 0))} in window`}
          tone="blue"
          caption="Messages marked sent today"
        />
        <MetricCard
          title="Queued Right Now"
          value={formatNumber(metrics?.queuedNow ?? 0)}
          delta={`${formatNumber(queueMix.reduce((sum, item) => sum + item.retries, 0))} retry`}
          tone="emerald"
          caption="Pending + processing + failed mail"
        />
        <MetricCard
          title="Active Gateways"
          value={`${metrics?.activeGateways ?? 0} / ${metrics?.totalGateways ?? 0}`}
          delta={`${metrics?.totalGateways ?? 0} listed`}
          tone="amber"
          caption="Only active gateways can accept routed mail"
        />
        <MetricCard
          title="Template Coverage"
          value={formatNumber(metrics?.totalTemplates ?? 0)}
          delta={`${metrics?.liveTemplates ?? 0} live`}
          tone="slate"
          caption="Templates currently registered in SMTP"
        />
      </section>

      <section className="grid gap-6 xl:grid-cols-[1.55fr_0.95fr]">
        <ComponentCard
          title="Delivery Throughput"
          desc="Queued, delivered, and retry volume across the current 24h operating window."
        >
          <div className="flex min-h-[420px] flex-col">
            <div className="grid gap-4 md:grid-cols-3">
              <InlineStat
                label="Peak hour"
                value={peakHour.label}
                subtext={`${formatNumber(peakHour.value)} delivered`}
              />
              <InlineStat
                label="Retry window"
                value={retryWindow.label}
                subtext={`${formatNumber(retryWindow.value)} retries`}
              />
              <InlineStat
                label="Current mix"
                value={currentMix.label}
                subtext={`${formatNumber(currentMix.value)} queued units`}
              />
            </div>
            <div className="mt-auto w-full overflow-hidden pt-4">
              {isChartReady ? (
                <ReactApexChart
                  options={deliveryOptions}
                  series={deliverySeries}
                  type="area"
                  height={320}
                  width="100%"
                />
              ) : (
                <ChartSkeleton height={320} />
              )}
            </div>
          </div>
        </ComponentCard>

        <div className="grid gap-6">
          <ComponentCard
            title="Health Distribution"
            desc="Operational split of SMTP infrastructure and delivery posture."
          >
            <div className="flex flex-col gap-5 lg:flex-row lg:items-center">
              <div className="mx-auto w-full max-w-[260px]">
                {isChartReady ? (
                  <ReactApexChart
                    options={healthOptions}
                    series={[
                      health?.healthy ?? 0,
                      health?.warning ?? 0,
                      health?.stopped ?? 0,
                    ]}
                    type="donut"
                    height={260}
                  />
                ) : (
                  <ChartSkeleton height={260} />
                )}
              </div>
              <div className="space-y-3">
                <StatusLegend
                  label="Healthy"
                  value={`${formatNumber(health?.healthy ?? 0)} surfaces`}
                  tone="emerald"
                />
                <StatusLegend
                  label="Warning"
                  value={`${formatNumber(health?.warning ?? 0)} surfaces`}
                  tone="amber"
                />
                <StatusLegend
                  label="Stopped"
                  value={`${formatNumber(health?.stopped ?? 0)} surfaces`}
                  tone="rose"
                />
              </div>
            </div>
          </ComponentCard>

          <ComponentCard
            title="Queue Mix"
            desc="How traffic is distributed across categories that currently feed SMTP."
          >
            <div className="w-full overflow-hidden">
              {isChartReady ? (
                <ReactApexChart
                  options={queueMixOptions}
                  series={queueMixSeries}
                  type="bar"
                  height={260}
                  width="100%"
                />
              ) : (
                <ChartSkeleton height={260} />
              )}
            </div>
          </ComponentCard>
        </div>
      </section>

      <section>
        <ComponentCard
          title="Runtime Timeline"
          desc="Recent SMTP configuration actions persisted by the control plane."
        >
          <div className="space-y-1">
            {timeline.length === 0 ? (
              <EmptyState text="No runtime actions have been recorded yet." />
            ) : (
              <div className="overflow-hidden rounded-2xl border border-gray-200 bg-gray-50/70 dark:border-gray-800 dark:bg-gray-900/30">
                {timelinePreview.map((item, index) => (
                  <div
                    key={item.id}
                    className={`grid gap-4 px-4 py-3 md:grid-cols-[160px_minmax(0,1fr)] ${
                      index === 0 ? "" : "border-t border-gray-200 dark:border-gray-800"
                    }`}
                  >
                    <div className="text-xs font-medium tracking-[0.12em] text-gray-400 uppercase whitespace-pre-line">
                      {formatTimelineTimestamp(item.createdAt)}
                    </div>
                    <div className="min-w-0">
                      <p className="text-sm font-semibold text-gray-900 dark:text-white">
                        {formatTimelineTitle(item)}
                      </p>
                      {formatTimelineDetail(item) !== "" ? (
                        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
                          {formatTimelineDetail(item)}
                        </p>
                      ) : null}
                    </div>
                  </div>
                ))}
              </div>
            )}
            {timeline.length > 10 ? (
              <div className="flex justify-end pt-3">
                <button
                  type="button"
                  onClick={() => setIsTimelineDialogOpen(true)}
                  className="inline-flex items-center rounded-xl border border-gray-200 bg-white px-4 py-2 text-sm font-semibold text-gray-700 transition hover:bg-gray-50 dark:border-gray-700 dark:bg-gray-900 dark:text-gray-200 dark:hover:bg-gray-800"
                >
                  Show more
                </button>
              </div>
            ) : null}
          </div>
        </ComponentCard>
      </section>

      <TimelineDialog
        open={isTimelineDialogOpen}
        search={timelineSearch}
        items={filteredTimeline}
        onClose={() => setIsTimelineDialogOpen(false)}
        onSearchChange={setTimelineSearch}
      />
    </div>
  );
}

function MetricCard({
  title,
  value,
  delta,
  caption,
  tone,
}: {
  title: string;
  value: string;
  delta: string;
  caption: string;
  tone: "blue" | "emerald" | "amber" | "slate";
}) {
  const toneClass = {
    blue: "from-blue-600 to-blue-500",
    emerald: "from-emerald-600 to-emerald-500",
    amber: "from-amber-500 to-orange-500",
    slate: "from-slate-800 to-slate-700",
  }[tone];

  return (
    <div className={`rounded-3xl bg-gradient-to-br ${toneClass} px-5 py-5 text-white shadow-theme-sm`}>
      <p className="text-xs font-medium tracking-[0.2em] uppercase text-white/70">{title}</p>
      <div className="mt-4 flex items-end justify-between gap-4">
        <h3 className="text-3xl font-semibold">{value}</h3>
        <span className="rounded-full bg-white/15 px-2.5 py-1 text-xs font-semibold text-white">
          {delta}
        </span>
      </div>
      <p className="mt-3 text-sm text-white/80">{caption}</p>
    </div>
  );
}

function InlineStat({
  label,
  value,
  subtext,
}: {
  label: string;
  value: string;
  subtext: string;
}) {
  return (
    <div className="rounded-2xl border border-gray-200 bg-gray-50 px-4 py-4 dark:border-gray-800 dark:bg-gray-900/40">
      <p className="text-xs font-medium tracking-[0.18em] text-gray-400 uppercase">{label}</p>
      <p className="mt-2 text-xl font-semibold text-gray-900 dark:text-white">{value}</p>
      <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{subtext}</p>
    </div>
  );
}

function StatusLegend({
  label,
  value,
  tone,
}: {
  label: string;
  value: string;
  tone: "emerald" | "amber" | "rose";
}) {
  const dotClass = {
    emerald: "bg-emerald-500",
    amber: "bg-amber-500",
    rose: "bg-rose-500",
  }[tone];

  return (
    <div className="flex items-center justify-between gap-4 rounded-2xl border border-gray-200 bg-gray-50 px-4 py-3 dark:border-gray-800 dark:bg-gray-900/40">
      <div className="flex items-center gap-3">
        <span className={`h-3 w-3 rounded-full ${dotClass}`} />
        <span className="text-sm font-medium text-gray-700 dark:text-gray-200">{label}</span>
      </div>
      <span className="text-sm text-gray-500 dark:text-gray-400">{value}</span>
    </div>
  );
}

function EmptyState({ text }: { text: string }) {
  return (
    <div className="rounded-2xl border border-dashed border-gray-300 px-4 py-5 text-sm text-gray-500 dark:border-gray-700 dark:text-gray-400">
      {text}
    </div>
  );
}

function ChartSkeleton({ height }: { height: number }) {
  return (
    <div
      className="w-full animate-pulse rounded-2xl border border-gray-200 bg-gray-50 dark:border-gray-800 dark:bg-gray-900/40"
      style={{ height }}
    />
  );
}

function TimelineDialog({
  open,
  search,
  items,
  onClose,
  onSearchChange,
}: {
  open: boolean;
  search: string;
  items: SMTPOverview["timeline"];
  onClose: () => void;
  onSearchChange: (value: string) => void;
}) {
  useEffect(() => {
    if (!open) {
      return;
    }
    function onKeyDown(event: KeyboardEvent) {
      if (event.key === "Escape") {
        onClose();
      }
    }
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [onClose, open]);

  if (!open || typeof document === "undefined") {
    return null;
  }

  return createPortal(
    <div className="fixed inset-0 z-[100100] flex items-center justify-center bg-slate-950/55 px-4 py-6 backdrop-blur-sm">
      <div className="flex max-h-[85vh] w-full max-w-5xl flex-col overflow-hidden rounded-[28px] border border-gray-200 bg-white shadow-2xl dark:border-gray-800 dark:bg-gray-950">
        <div className="flex items-start justify-between gap-4 border-b border-gray-200 px-6 py-5 dark:border-gray-800">
          <div>
            <h3 className="text-lg font-semibold text-gray-900 dark:text-white">Runtime Timeline</h3>
            <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
              Search recent runtime events persisted by the control plane.
            </p>
          </div>
          <button
            type="button"
            onClick={onClose}
            className="inline-flex h-10 w-10 items-center justify-center rounded-xl border border-gray-200 bg-white text-gray-500 transition hover:bg-gray-50 dark:border-gray-700 dark:bg-gray-900 dark:text-gray-300 dark:hover:bg-gray-800"
          >
            ×
          </button>
        </div>
        <div className="border-b border-gray-200 px-6 py-4 dark:border-gray-800">
          <input
            type="text"
            value={search}
            onChange={(event) => onSearchChange(event.target.value)}
            placeholder="Search entity, action, actor, or note"
            className="w-full rounded-2xl border border-gray-200 bg-gray-50 px-4 py-3 text-sm text-gray-800 outline-none transition focus:border-blue-400 focus:bg-white dark:border-gray-700 dark:bg-gray-900 dark:text-white dark:focus:border-blue-500"
          />
        </div>
        <div className="overflow-y-auto px-6 py-4">
          {items.length === 0 ? (
            <EmptyState text="No timeline entry matched your search." />
          ) : (
            <div className="overflow-hidden rounded-2xl border border-gray-200 bg-gray-50/70 dark:border-gray-800 dark:bg-gray-900/30">
              {items.map((item, index) => (
                <div
                  key={item.id}
                  className={`grid gap-4 px-4 py-3 md:grid-cols-[180px_minmax(0,1fr)] ${
                    index === 0 ? "" : "border-t border-gray-200 dark:border-gray-800"
                  }`}
                >
                  <div className="text-xs font-medium tracking-[0.12em] text-gray-400 uppercase whitespace-pre-line">
                    {formatTimelineTimestamp(item.createdAt)}
                  </div>
                  <div className="min-w-0">
                    <p className="text-sm font-semibold text-gray-900 dark:text-white">
                      {formatTimelineTitle(item)}
                    </p>
                    {formatTimelineDetail(item) !== "" ? (
                      <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
                        {formatTimelineDetail(item)}
                      </p>
                    ) : null}
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>,
    document.body,
  );
}

function formatNumber(value: number) {
  return new Intl.NumberFormat("en-US").format(value);
}

function formatTimelineTitle(item: NonNullable<SMTPOverview["timeline"]>[number]) {
  const entity =
    item.entityName ||
    (item.entityType ? item.entityType.replaceAll("_", " ") : "smtp item");
  const action = item.action ? item.action.replaceAll("_", " ") : "updated";
  return `${entity} ${action}`;
}

function formatTimelineDetail(item: NonNullable<SMTPOverview["timeline"]>[number]) {
  if (item.note?.trim()) {
    return item.note.trim();
  }
  if (item.actorName?.trim()) {
    return `by ${item.actorName.trim()}`;
  }
  return "";
}

function formatTimelineTimestamp(value: string) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return "--/--/----\n--:--:--";
  }
  return new Intl.DateTimeFormat("en-GB", {
    day: "2-digit",
    month: "2-digit",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  })
    .format(date)
    .replace(", ", "\n");
}
