"use client";

import { useEffect, useMemo, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import ComponentCard from "@/components/common/ComponentCard";
import { DeliveryFlowCanvas } from "@/components/smtp/DeliveryFlowCanvas";
import {
  bindGatewayEndpoints,
  bindGatewayTemplates,
  deleteGateway,
  disableGateway,
  drainGateway,
  getGatewayDetail,
  startGateway,
} from "@/components/smtp/api";
import { SMTPPageShell } from "@/components/smtp/SMTPPageShell";
import { useSMTPWorkspace } from "@/components/smtp/SMTPWorkspaceProvider";
import type { DeliveryEndpoint, GatewayDetail, GatewayItem, TemplateItem } from "@/components/smtp/types";

export default function GatewayDetailPage() {
  const searchParams = useSearchParams();
  const router = useRouter();
  const { workspace, workspaceID, isLoading: isWorkspaceLoading, error: workspaceError } = useSMTPWorkspace();
  const gatewayID = searchParams.get("id") ?? "";

  const [gateway, setGateway] = useState<GatewayDetail | null>(null);
  const [selectedTemplates, setSelectedTemplates] = useState<TemplateItem[]>([]);
  const [availableTemplates, setAvailableTemplates] = useState<TemplateItem[]>([]);
  const [selectedEndpoints, setSelectedEndpoints] = useState<DeliveryEndpoint[]>([]);
  const [availableEndpoints, setAvailableEndpoints] = useState<DeliveryEndpoint[]>([]);
  const [fallbackGateway, setFallbackGateway] = useState<GatewayItem | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState("");
  const [isMutating, setIsMutating] = useState(false);
  const [isSavingTemplates, setIsSavingTemplates] = useState(false);
  const [isSavingEndpoints, setIsSavingEndpoints] = useState(false);

  useEffect(() => {
    if (workspaceID === "" || gatewayID === "") {
      setGateway(null);
      setIsLoading(false);
      return;
    }

    let cancelled = false;

    async function load() {
      setIsLoading(true);
      setError("");
      try {
        const detail = await getGatewayDetail(workspaceID, gatewayID);
        if (cancelled) {
          return;
        }

        setGateway(detail);

        const templates = detail.templates.map<TemplateItem>((item) => ({
          id: item.id,
          name: item.name,
          category: item.category,
          trafficClass: item.trafficClass,
          subject: "",
          from: "",
          to: "",
          status: item.status,
          variables: [],
          consumerId: item.consumerId ?? "",
          consumer: item.consumerName,
          textBody: "",
          htmlBody: "",
        }));
        setAvailableTemplates(templates);
        setSelectedTemplates(templates.filter((item) => detail.templates.some((row) => row.id === item.id && row.selected)));

        const endpoints = detail.endpoints.map<DeliveryEndpoint>((item) => ({
          id: item.id,
          name: item.name,
          providerKind: "smtp",
          host: item.host,
          port: item.port,
          username: item.username,
          priority: 0,
          weight: 0,
          maxConnections: 0,
          maxParallelSends: 0,
          maxMessagesPerSecond: 0,
          burst: 0,
          warmupState: "",
          tlsMode: "starttls",
          status: item.status,
          hasSecret: false,
        }));
        setAvailableEndpoints(endpoints);
        setSelectedEndpoints(endpoints.filter((item) => detail.endpoints.some((row) => row.id === item.id && row.selected)));

        setFallbackGateway(
          detail.fallbackGateway == null
            ? null
            : {
                id: detail.fallbackGateway.id,
                name: detail.fallbackGateway.name,
                trafficClass: detail.trafficClass,
                status: detail.fallbackGateway.status,
                routingMode: "",
                priority: 0,
                desiredShardCount: 0,
                templateCount: 0,
                endpointCount: 0,
                readyShards: 0,
                pendingShards: 0,
                drainingShards: 0,
                fallbackGatewayName: "",
              },
        );
      } catch (err) {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : "Failed to load SMTP gateway detail");
          setGateway(null);
        }
      } finally {
        if (!cancelled) {
          setIsLoading(false);
        }
      }
    }

    void load();
    return () => {
      cancelled = true;
    };
  }, [gatewayID, workspaceID]);

  const workspaceQuery = workspaceID === "" ? "" : `?workspace=${workspaceID}`;
  const fallbackGatewayName = fallbackGateway?.name ?? "";
  const fallbackGatewayStatus = fallbackGateway?.status ?? "";

  async function runGatewayMutation(action: "start" | "drain" | "disable" | "delete") {
    if (workspaceID === "" || gateway == null) {
      return;
    }
    if (action === "delete" && !window.confirm(`Delete gateway "${gateway.name}"?`)) {
      return;
    }

    setIsMutating(true);
    setError("");
    try {
      if (action === "delete") {
        await deleteGateway(workspaceID, gateway.id);
        router.push(`/smtp/gateways${workspaceQuery}`);
        return;
      }

      const next =
        action === "start"
          ? await startGateway(workspaceID, gateway.id)
          : action === "drain"
            ? await drainGateway(workspaceID, gateway.id)
            : await disableGateway(workspaceID, gateway.id);
      setGateway(next);
    } catch (err) {
      setError(err instanceof Error ? err.message : `Failed to ${action} gateway`);
    } finally {
      setIsMutating(false);
    }
  }

  async function saveTemplates(templateIDs: string[]) {
    if (workspaceID === "" || gateway == null) {
      return;
    }
    setIsSavingTemplates(true);
    setError("");
    try {
      const next = await bindGatewayTemplates(workspaceID, gateway.id, templateIDs);
      setGateway(next);
      const selectedSet = new Set(templateIDs);
      setSelectedTemplates(availableTemplates.filter((item) => selectedSet.has(item.id)));
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to save gateway templates");
      throw err;
    } finally {
      setIsSavingTemplates(false);
    }
  }

  async function saveEndpoints(endpointIDs: string[]) {
    if (workspaceID === "" || gateway == null) {
      return;
    }
    setIsSavingEndpoints(true);
    setError("");
    try {
      const next = await bindGatewayEndpoints(workspaceID, gateway.id, endpointIDs);
      setGateway(next);
      const selectedSet = new Set(endpointIDs);
      setSelectedEndpoints(availableEndpoints.filter((item) => selectedSet.has(item.id)));
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to save gateway endpoints");
      throw err;
    } finally {
      setIsSavingEndpoints(false);
    }
  }

  const summaryRows = useMemo(
    () => [
      { label: "Traffic Class", value: gateway?.trafficClass ?? "-" },
      { label: "Routing Mode", value: gateway?.routingMode ?? "-" },
      { label: "Priority", value: gateway != null ? String(gateway.priority) : "-" },
      { label: "Desired Shards", value: gateway != null ? String(gateway.desiredShardCount) : "-" },
      { label: "Fallback Gateway", value: fallbackGatewayName || "None" },
      { label: "Runtime Version", value: gateway != null ? String(gateway.runtimeVersion) : "-" },
    ],
    [fallbackGatewayName, gateway],
  );

  return (
    <SMTPPageShell>
      <div className="space-y-6">
        <p className="text-xs font-medium tracking-[0.2em] text-gray-400 uppercase">Gateway Detail</p>

        {isWorkspaceLoading ? (
          <StatePanel message="Resolving workspace context..." />
        ) : workspaceError !== "" ? (
          <ErrorPanel message={workspaceError} />
        ) : workspace == null ? (
          <StatePanel message="No workspace is available for SMTP yet." />
        ) : isLoading ? (
          <StatePanel message="Loading gateway detail..." />
        ) : error !== "" && gateway == null ? (
          <ErrorPanel message={error} />
        ) : gateway == null ? (
          <StatePanel message="Gateway not found." />
        ) : (
          <>
            {error !== "" ? <ErrorPanel message={error} /> : null}

            <ComponentCard
              title={gateway.name}
              desc={`Gateway ID: ${gateway.id}`}
              headerAction={
                <div className="flex flex-wrap items-center gap-2">
                  <StatusPill status={gateway.status} />
                  <ActionButton
                    label="Start"
                    onClick={() => void runGatewayMutation("start")}
                    disabled={isMutating}
                  />
                  <ActionButton
                    label="Drain"
                    onClick={() => void runGatewayMutation("drain")}
                    disabled={isMutating}
                    tone="amber"
                  />
                  <ActionButton
                    label="Disable"
                    onClick={() => void runGatewayMutation("disable")}
                    disabled={isMutating}
                  />
                  <ActionButton
                    label="Delete"
                    onClick={() => void runGatewayMutation("delete")}
                    disabled={isMutating}
                    tone="rose"
                  />
                </div>
              }
            >
              <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
                {summaryRows.map((row) => (
                  <DetailRow key={row.label} label={row.label} value={row.value} />
                ))}
              </div>
            </ComponentCard>

            <ComponentCard
              title="Routing Flow"
              desc="Manage which templates route into this gateway and which endpoints this gateway can use."
            >
              <DeliveryFlowCanvas
                gatewayName={gateway.name}
                gatewayStatus={gateway.status}
                trafficClass={gateway.trafficClass}
                fallbackGatewayName={fallbackGatewayName || undefined}
                fallbackGatewayStatus={fallbackGatewayStatus || undefined}
                selectedTemplates={selectedTemplates}
                selectedEndpoints={selectedEndpoints}
                availableTemplates={availableTemplates}
                availableEndpoints={availableEndpoints}
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

function DetailRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-2xl border border-gray-200 bg-gray-50 px-4 py-4 dark:border-gray-800 dark:bg-gray-900/40">
      <p className="text-xs font-medium tracking-[0.16em] text-gray-400 uppercase">{label}</p>
      <p className="mt-2 text-sm font-semibold text-gray-900 dark:text-white">{value}</p>
    </div>
  );
}

function StatusPill({ status }: { status: string }) {
  const normalized = status.trim().toLowerCase();
  const toneClass =
    normalized === "active"
      ? "border-emerald-200 bg-emerald-50 text-emerald-700 dark:border-emerald-500/30 dark:bg-emerald-500/10 dark:text-emerald-300"
      : normalized === "draining"
        ? "border-amber-200 bg-amber-50 text-amber-700 dark:border-amber-500/30 dark:bg-amber-500/10 dark:text-amber-300"
        : "border-gray-200 bg-gray-50 text-gray-700 dark:border-gray-700 dark:bg-gray-900 dark:text-gray-200";

  return (
    <span className={`inline-flex items-center rounded-full border px-3 py-1 text-xs font-semibold uppercase ${toneClass}`}>
      {status}
    </span>
  );
}

function ActionButton({
  label,
  onClick,
  disabled,
  tone = "slate",
}: {
  label: string;
  onClick: () => void;
  disabled: boolean;
  tone?: "slate" | "amber" | "rose";
}) {
  const className =
    tone === "amber"
      ? "border-amber-200 bg-amber-50 text-amber-700 hover:bg-amber-100 dark:border-amber-500/30 dark:bg-amber-500/10 dark:text-amber-300 dark:hover:bg-amber-500/20"
      : tone === "rose"
        ? "border-error-200 bg-error-50 text-error-700 hover:bg-error-100 dark:border-error-500/30 dark:bg-error-500/10 dark:text-error-300 dark:hover:bg-error-500/20"
        : "border-gray-200 bg-white text-gray-700 hover:bg-gray-50 dark:border-gray-700 dark:bg-gray-900 dark:text-gray-200 dark:hover:bg-gray-800";

  return (
    <button
      type="button"
      onClick={onClick}
      disabled={disabled}
      className={`inline-flex items-center rounded-xl border px-4 py-2.5 text-sm font-semibold transition disabled:cursor-not-allowed disabled:opacity-60 ${className}`}
    >
      {label}
    </button>
  );
}

function StatePanel({ message }: { message: string }) {
  return (
    <div className="rounded-2xl border border-gray-200 bg-gray-50 px-5 py-5 dark:border-gray-800 dark:bg-gray-900/40">
      <p className="text-sm text-gray-500 dark:text-gray-400">{message}</p>
    </div>
  );
}

function ErrorPanel({ message }: { message: string }) {
  return (
    <div className="rounded-2xl border border-error-200 bg-error-50 px-5 py-5 dark:border-error-500/30 dark:bg-error-500/10">
      <p className="text-sm text-error-700 dark:text-error-300">{message}</p>
    </div>
  );
}
