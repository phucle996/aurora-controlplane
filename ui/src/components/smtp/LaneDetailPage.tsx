"use client";

import { useEffect, useMemo, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { parseAPIError } from "@/components/auth/auth-utils";
import ComponentCard from "@/components/common/ComponentCard";
import { DeliveryFlowCanvas } from "@/components/smtp/DeliveryFlowCanvas";
import { SMTPPageShell } from "@/components/smtp/SMTPPageShell";
import type { DeliveryEndpoint, LaneItem, TemplateItem } from "@/components/smtp/types";

type LaneDetailResponse = {
  lane: {
    id: string;
    name: string;
    traffic_class: string;
    status: string;
    routing_mode: string;
    priority: number;
    fallback_lane_id?: string;
  };
  fallback_lane?: {
    id: string;
    name: string;
    status: string;
  };
  templates: Array<{
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
  endpoints: Array<{
    id: string;
    name: string;
    host: string;
    port: number;
    username: string;
    status: "active" | "draining" | "disabled" | "standby" | "maintenance";
    selected: boolean;
    position: number;
  }>;
};

type LaneDetail = {
  id: string;
  name: string;
  trafficClass: string;
  status: string;
  routingMode: string;
  priority: number;
  fallbackLaneID?: string;
  runtimeVersion?: number;
  createdAt?: string;
  updatedAt?: string;
};

export default function LaneDetailPage() {
  const searchParams = useSearchParams();
  const router = useRouter();
  const laneID = searchParams.get("id") ?? "";

  const [lane, setLane] = useState<LaneDetail | null>(null);
  const [fallbackLane, setFallbackLane] = useState<LaneItem | null>(null);
  const [templates, setTemplates] = useState<TemplateItem[]>([]);
  const [selectedTemplates, setSelectedTemplates] = useState<TemplateItem[]>([]);
  const [endpoints, setEndpoints] = useState<DeliveryEndpoint[]>([]);
  const [selectedEndpoints, setSelectedEndpoints] = useState<DeliveryEndpoint[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState("");
  const [isMutating, setIsMutating] = useState(false);
  const [isSavingTemplates, setIsSavingTemplates] = useState(false);
  const [isSavingEndpoints, setIsSavingEndpoints] = useState(false);

  useEffect(() => {
    if (laneID === "") {
      return;
    }

    let cancelled = false;

    async function loadData() {
      setIsLoading(true);
      setError("");
      try {
        const response = await fetch(`/api/v1/smtp/lanes/${laneID}/detail`, {
          cache: "no-store",
          credentials: "include",
        });
        if (!response.ok) {
          throw new Error(await parseAPIError(response));
        }
        const result = (await response.json()) as LaneDetailResponse;

        if (cancelled) {
          return;
        }

        const availableTemplates = result.templates.map(mapTemplateResponse);
        const availableEndpoints = result.endpoints.map(mapEndpointResponse);

        setLane(mapLaneDetail(result));
        setFallbackLane(mapFallbackLane(result));
        setTemplates(availableTemplates);
        setSelectedTemplates(
          availableTemplates.filter((template) =>
            result.templates.some((item) => item.id === template.id && item.selected),
          ),
        );
        setEndpoints(availableEndpoints);
        setSelectedEndpoints(
          availableEndpoints.filter((endpoint) =>
            result.endpoints.some((item) => item.id === endpoint.id && item.selected),
          ),
        );
      } catch (err) {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : "Failed to load lane detail");
        }
      } finally {
        if (!cancelled) {
          setIsLoading(false);
        }
      }
    }

    void loadData();
    return () => {
      cancelled = true;
    };
  }, [laneID]);

  const fallbackLaneName = useMemo(() => {
    if (fallbackLane == null) {
      return "";
    }
    return fallbackLane.name;
  }, [fallbackLane]);

  const fallbackLaneStatus = useMemo(() => {
    if (fallbackLane == null) {
      return "";
    }
    return fallbackLane.status;
  }, [fallbackLane]);

  async function mutateLane(action: "start" | "drain" | "disable" | "delete") {
    if (lane == null) {
      return;
    }
    if (action === "delete" && !window.confirm(`Delete lane "${lane.name}"?`)) {
      return;
    }

    setIsMutating(true);
    setError("");
    try {
      const response = await fetch(
        action === "delete"
          ? `/api/v1/smtp/lanes/${lane.id}`
          : `/api/v1/smtp/lanes/${lane.id}/${action}`,
        {
          method: action === "delete" ? "DELETE" : "POST",
          credentials: "include",
        },
      );
      if (!response.ok) {
        throw new Error(await parseAPIError(response));
      }

      if (action === "delete") {
        router.push("/smtp/lanes");
        return;
      }

      setLane((current) =>
        current == null
          ? current
          : {
              ...current,
              status:
                action === "start" ? "active" : action === "drain" ? "draining" : "disabled",
            },
      );
    } catch (err) {
      setError(err instanceof Error ? err.message : `Failed to ${action} lane`);
    } finally {
      setIsMutating(false);
    }
  }

  async function saveTemplates(templateIDs: string[]) {
    if (lane == null) {
      return;
    }
    setIsSavingTemplates(true);
    setError("");
    try {
      const response = await fetch(`/api/v1/smtp/lanes/${lane.id}/templates`, {
        method: "PUT",
        credentials: "include",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ template_ids: templateIDs }),
      });
      if (!response.ok) {
        throw new Error(await parseAPIError(response));
      }
      const selectedSet = new Set(templateIDs);
      setSelectedTemplates(templates.filter((item) => selectedSet.has(item.id)));
    } catch (err) {
      const message = err instanceof Error ? err.message : "Failed to save lane templates";
      setError(message);
      throw err;
    } finally {
      setIsSavingTemplates(false);
    }
  }

  async function saveEndpoints(endpointIDs: string[]) {
    if (lane == null) {
      return;
    }
    setIsSavingEndpoints(true);
    setError("");
    try {
      const response = await fetch(`/api/v1/smtp/lanes/${lane.id}/endpoints`, {
        method: "PUT",
        credentials: "include",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ endpoint_ids: endpointIDs }),
      });
      if (!response.ok) {
        throw new Error(await parseAPIError(response));
      }
      const selectedSet = new Set(endpointIDs);
      setSelectedEndpoints(endpoints.filter((item) => selectedSet.has(item.id)));
    } catch (err) {
      const message = err instanceof Error ? err.message : "Failed to save lane endpoints";
      setError(message);
      throw err;
    } finally {
      setIsSavingEndpoints(false);
    }
  }

  return (
    <SMTPPageShell>
      <div className="space-y-6">
        <p className="text-xs font-medium tracking-[0.2em] text-gray-400 uppercase">Lane Detail</p>

        {isLoading ? (
          <div className="rounded-2xl border border-gray-200 bg-gray-50 px-5 py-5 dark:border-gray-800 dark:bg-gray-900/40">
            <p className="text-sm text-gray-500 dark:text-gray-400">Loading lane detail...</p>
          </div>
        ) : error !== "" && lane == null ? (
          <div className="rounded-2xl border border-error-200 bg-error-50 px-5 py-5 dark:border-error-500/30 dark:bg-error-500/10">
            <p className="text-sm text-error-700 dark:text-error-300">{error}</p>
          </div>
        ) : lane == null ? (
          <div className="rounded-2xl border border-gray-200 bg-gray-50 px-5 py-5 dark:border-gray-800 dark:bg-gray-900/40">
            <p className="text-sm text-gray-500 dark:text-gray-400">Lane not found.</p>
          </div>
        ) : (
          <>
            {error !== "" ? (
              <div className="rounded-2xl border border-error-200 bg-error-50 px-5 py-5 dark:border-error-500/30 dark:bg-error-500/10">
                <p className="text-sm text-error-700 dark:text-error-300">{error}</p>
              </div>
            ) : null}

            <ComponentCard
              title={lane.name}
              desc={`Lane ID: ${lane.id}`}
              headerAction={
                <div className="flex flex-wrap items-center gap-2">
                  <StatusPill status={lane.status} />
                  <button
                    type="button"
                    onClick={() => void mutateLane("start")}
                    disabled={isMutating}
                    className="inline-flex items-center rounded-xl border border-gray-200 bg-white px-4 py-2.5 text-sm font-semibold text-gray-700 transition hover:bg-gray-50 disabled:cursor-not-allowed disabled:opacity-60 dark:border-gray-700 dark:bg-gray-900 dark:text-gray-200 dark:hover:bg-gray-800"
                  >
                    Start
                  </button>
                  <button
                    type="button"
                    onClick={() => void mutateLane("drain")}
                    disabled={isMutating}
                    className="inline-flex items-center rounded-xl border border-amber-200 bg-amber-50 px-4 py-2.5 text-sm font-semibold text-amber-700 transition hover:bg-amber-100 disabled:cursor-not-allowed disabled:opacity-60 dark:border-amber-500/30 dark:bg-amber-500/10 dark:text-amber-300 dark:hover:bg-amber-500/20"
                  >
                    Drain
                  </button>
                  <button
                    type="button"
                    onClick={() => void mutateLane("disable")}
                    disabled={isMutating}
                    className="inline-flex items-center rounded-xl border border-gray-200 bg-white px-4 py-2.5 text-sm font-semibold text-gray-700 transition hover:bg-gray-50 disabled:cursor-not-allowed disabled:opacity-60 dark:border-gray-700 dark:bg-gray-900 dark:text-gray-200 dark:hover:bg-gray-800"
                  >
                    Disable
                  </button>
                  <button
                    type="button"
                    onClick={() => void mutateLane("delete")}
                    disabled={isMutating}
                    className="inline-flex items-center rounded-xl border border-error-200 bg-error-50 px-4 py-2.5 text-sm font-semibold text-error-700 transition hover:bg-error-100 disabled:cursor-not-allowed disabled:opacity-60 dark:border-error-500/30 dark:bg-error-500/10 dark:text-error-300 dark:hover:bg-error-500/20"
                  >
                    Delete
                  </button>
                </div>
              }
            >
              <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
                <DetailRow label="Traffic Class" value={lane.trafficClass} />
                <DetailRow label="Routing Mode" value={lane.routingMode} />
                <DetailRow label="Priority" value={String(lane.priority)} />
                <DetailRow label="Fallback Lane" value={fallbackLaneName || "None"} />
              </div>
            </ComponentCard>

            <ComponentCard
              title="Routing Flow"
              desc="Manage which templates route into this lane and which endpoints this lane can deliver through."
            >
              <DeliveryFlowCanvas
                laneID={lane.id}
                laneName={lane.name}
                laneStatus={lane.status}
                trafficClass={lane.trafficClass}
                fallbackLaneName={fallbackLaneName || undefined}
                fallbackLaneStatus={fallbackLaneStatus || undefined}
                selectedTemplates={selectedTemplates}
                selectedEndpoints={selectedEndpoints}
                availableTemplates={templates}
                availableEndpoints={endpoints}
                onSaveTemplates={saveTemplates}
                onSaveEndpoints={saveEndpoints}
                isSavingTemplates={isSavingTemplates}
                isSavingEndpoints={isSavingEndpoints}
              />
            </ComponentCard>
          </>
        )}
      </div>
    </SMTPPageShell>
  );
}

function mapLaneDetail(item: LaneDetailResponse): LaneDetail {
  return {
    id: item.lane.id,
    name: item.lane.name,
    trafficClass: item.lane.traffic_class || "transactional",
    status: item.lane.status,
    routingMode: item.lane.routing_mode || "priority",
    priority: item.lane.priority ?? 100,
    fallbackLaneID: item.lane.fallback_lane_id || "",
  };
}

function mapFallbackLane(item: LaneDetailResponse): LaneItem | null {
  if (!item.fallback_lane || item.fallback_lane.id === "") {
    return null;
  }
  return {
    id: item.fallback_lane.id,
    name: item.fallback_lane.name,
    trafficClass: item.lane.traffic_class || "transactional",
    priority: item.lane.priority ?? 100,
    status: item.fallback_lane.status,
  };
}

function mapTemplateResponse(item: LaneDetailResponse["templates"][number]): TemplateItem {
  return {
    id: item.id,
    name: item.name,
    category: item.category,
    trafficClass: item.traffic_class || "transactional",
    subject: "",
    from: "",
    to: "",
    status: item.status,
    variables: [],
    consumerId: item.consumer_id || "",
    consumer: item.consumer_name || "none",
    body: "",
  };
}

function mapEndpointResponse(item: LaneDetailResponse["endpoints"][number]): DeliveryEndpoint {
  return {
    id: item.id,
    name: item.name,
    host: item.host,
    port: item.port,
    username: item.username,
    priority: 100,
    weight: 1,
    tlsMode: "starttls",
    status: item.status ?? "maintenance",
  };
}

function DetailRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-2xl border border-gray-200 bg-gray-50 px-4 py-4 dark:border-gray-800 dark:bg-gray-900/50">
      <p className="text-xs font-medium tracking-[0.18em] text-gray-400 uppercase">{label}</p>
      <p className="mt-2 text-sm font-semibold text-gray-900 dark:text-white">{value}</p>
    </div>
  );
}

function StatusPill({ status }: { status: string }) {
  const normalized = status.trim().toLowerCase();
  const className =
    normalized === "active" || normalized === "ready" || normalized === "healthy"
      ? "border-emerald-200 bg-emerald-50 text-emerald-700 dark:border-emerald-500/30 dark:bg-emerald-500/10 dark:text-emerald-300"
      : normalized === "disabled" || normalized === "maintenance" || normalized === "unhealthy" || normalized === "failed" || normalized === "error"
        ? "border-rose-200 bg-rose-50 text-rose-700 dark:border-rose-500/30 dark:bg-rose-500/10 dark:text-rose-300"
        : "border-amber-200 bg-amber-50 text-amber-700 dark:border-amber-500/30 dark:bg-amber-500/10 dark:text-amber-300";

  return (
    <span className={`rounded-full border px-3 py-1 text-xs font-semibold capitalize ${className}`}>
      {status}
    </span>
  );
}
