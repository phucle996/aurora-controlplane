"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, useMemo, useState } from "react";
import ComponentCard from "@/components/common/ComponentCard";
import { listGateways } from "@/components/smtp/api";
import { useSMTPWorkspace } from "@/components/smtp/SMTPWorkspaceProvider";
import type { GatewayItem } from "@/components/smtp/types";

export function GatewayTab() {
  const router = useRouter();
  const { workspace, workspaceID, isLoading: isWorkspaceLoading, error: workspaceError } = useSMTPWorkspace();
  const [gateways, setGateways] = useState<GatewayItem[]>([]);
  const [search, setSearch] = useState("");
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState("");

  useEffect(() => {
    if (workspaceID === "") {
      setGateways([]);
      setIsLoading(false);
      setError("");
      return;
    }

    let cancelled = false;

    async function load() {
      setIsLoading(true);
      setError("");
      try {
        const items = await listGateways(workspaceID);
        if (!cancelled) {
          setGateways(items);
        }
      } catch (err) {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : "Failed to load SMTP gateways");
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
  }, [workspaceID]);

  const keyword = search.trim().toLowerCase();
  const filteredGateways = useMemo(
    () =>
      gateways.filter((gateway) => {
        if (keyword === "") {
          return true;
        }
        return (
          gateway.name.toLowerCase().includes(keyword) ||
          gateway.trafficClass.toLowerCase().includes(keyword) ||
          gateway.status.toLowerCase().includes(keyword) ||
          gateway.routingMode.toLowerCase().includes(keyword) ||
          gateway.id.toLowerCase().includes(keyword)
        );
      }),
    [gateways, keyword],
  );

  const workspaceQuery = workspaceID === "" ? "" : `?workspace=${workspaceID}`;

  if (isWorkspaceLoading) {
    return <StateCard title="SMTP Gateways" message="Resolving workspace context..." />;
  }

  if (workspaceError !== "") {
    return <ErrorCard title="SMTP Gateways" message={workspaceError} />;
  }

  if (workspace == null) {
    return <StateCard title="SMTP Gateways" message="No workspace is available for SMTP yet." />;
  }

  return (
    <div className="space-y-6">
      <ComponentCard
        title="Gateways"
        desc="Gateways own routing mode, shard posture, and endpoint bindings for the active workspace."
        headerAction={
          <Link
            href={`/smtp/gateways/new${workspaceQuery}`}
            className="inline-flex items-center rounded-xl bg-gray-900 px-4 py-2.5 text-sm font-semibold text-white transition hover:bg-gray-800 dark:bg-white dark:text-gray-900 dark:hover:bg-gray-200"
          >
            New Gateway
          </Link>
        }
      >
        <div className="space-y-4">
          <div className="rounded-2xl border border-gray-200 bg-gray-50 px-4 py-3 dark:border-gray-800 dark:bg-gray-900/40">
            <input
              type="text"
              value={search}
              onChange={(event) => setSearch(event.target.value)}
              placeholder="Search gateways by name, traffic class, status, or routing mode"
              className="w-full bg-transparent text-sm text-gray-800 outline-none placeholder:text-gray-400 dark:text-white dark:placeholder:text-gray-500"
            />
          </div>

          {isLoading ? (
            <StatePanel message="Loading gateways..." />
          ) : error !== "" ? (
            <ErrorPanel message={error} />
          ) : filteredGateways.length === 0 ? (
            <StatePanel message="No gateway matches your search." />
          ) : (
            <div className="overflow-hidden rounded-2xl border border-gray-200 bg-white dark:border-gray-800 dark:bg-gray-900/40">
              <div className="overflow-x-auto">
                <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-800">
                  <thead className="bg-gray-50 dark:bg-gray-900/70">
                    <tr>
                      <TableHead>Status</TableHead>
                      <TableHead>Name</TableHead>
                      <TableHead>Gateway ID</TableHead>
                      <TableHead>Traffic Class</TableHead>
                      <TableHead>Routing</TableHead>
                      <TableHead>Bindings</TableHead>
                      <TableHead>Shards</TableHead>
                      <TableHead>Updated</TableHead>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-gray-200 dark:divide-gray-800">
                    {filteredGateways.map((gateway) => (
                      <tr
                        key={gateway.id}
                        onClick={() => router.push(`/smtp/gateways/detail?id=${gateway.id}${workspaceQuery ? `&${workspaceQuery.slice(1)}` : ""}`)}
                        className="cursor-pointer transition hover:bg-gray-50 dark:hover:bg-white/5"
                      >
                        <td className="whitespace-nowrap px-5 py-4">
                          <StatusCell status={gateway.status} />
                        </td>
                        <td className="px-5 py-4">
                          <div className="min-w-[180px]">
                            <p className="text-sm font-semibold text-gray-900 dark:text-white">
                              {gateway.name}
                            </p>
                            <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
                              {gateway.fallbackGatewayName !== ""
                                ? `Fallback: ${gateway.fallbackGatewayName}`
                                : "No fallback gateway"}
                            </p>
                          </div>
                        </td>
                        <td className="px-5 py-4">
                          <code className="text-xs text-gray-600 dark:text-gray-300">{gateway.id}</code>
                        </td>
                        <td className="whitespace-nowrap px-5 py-4 text-sm text-gray-700 dark:text-gray-200">
                          {gateway.trafficClass}
                        </td>
                        <td className="whitespace-nowrap px-5 py-4 text-sm text-gray-700 dark:text-gray-200">
                          {gateway.routingMode}
                        </td>
                        <td className="whitespace-nowrap px-5 py-4 text-sm text-gray-700 dark:text-gray-200">
                          {gateway.templateCount} tpl / {gateway.endpointCount} ep
                        </td>
                        <td className="whitespace-nowrap px-5 py-4 text-sm text-gray-700 dark:text-gray-200">
                          {gateway.readyShards}/{gateway.desiredShardCount}
                          <span className="ml-2 text-xs text-gray-400">
                            p:{gateway.pendingShards} d:{gateway.drainingShards}
                          </span>
                        </td>
                        <td className="whitespace-nowrap px-5 py-4 text-sm text-gray-700 dark:text-gray-200">
                          {formatDateTime(gateway.updatedAt)}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          )}
        </div>
      </ComponentCard>
    </div>
  );
}

function TableHead({ children }: { children: React.ReactNode }) {
  return (
    <th className="px-5 py-3 text-left text-xs font-semibold tracking-[0.18em] text-gray-400 uppercase dark:text-gray-500">
      {children}
    </th>
  );
}

function StatusCell({ status }: { status: string }) {
  const normalized = status.trim().toLowerCase();
  const toneClass =
    normalized === "active" || normalized === "ready" || normalized === "healthy"
      ? "bg-emerald-500"
      : normalized === "disabled" || normalized === "unhealthy" || normalized === "failed" || normalized === "error" || normalized === "maintenance"
        ? "bg-rose-500"
        : "bg-amber-500";

  return (
    <div className="flex items-center gap-2">
      <span className={`h-2.5 w-2.5 rounded-full ${toneClass}`} />
      <span className="text-sm capitalize text-gray-700 dark:text-gray-200">{status}</span>
    </div>
  );
}

function formatDateTime(value?: string) {
  if (!value) {
    return "-";
  }
  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) {
    return value;
  }
  return parsed.toLocaleString();
}

function StateCard({ title, message }: { title: string; message: string }) {
  return (
    <ComponentCard title={title} desc="Workspace-scoped SMTP routing.">
      <StatePanel message={message} />
    </ComponentCard>
  );
}

function StatePanel({ message }: { message: string }) {
  return (
    <div className="rounded-2xl border border-gray-200 bg-gray-50 px-5 py-5 dark:border-gray-800 dark:bg-gray-900/40">
      <p className="text-sm text-gray-500 dark:text-gray-400">{message}</p>
    </div>
  );
}

function ErrorCard({ title, message }: { title: string; message: string }) {
  return (
    <ComponentCard title={title} desc="Workspace-scoped SMTP routing.">
      <ErrorPanel message={message} />
    </ComponentCard>
  );
}

function ErrorPanel({ message }: { message: string }) {
  return (
    <div className="rounded-2xl border border-error-200 bg-error-50 px-5 py-5 dark:border-error-500/30 dark:bg-error-500/10">
      <p className="text-sm text-error-700 dark:text-error-300">{message}</p>
    </div>
  );
}
