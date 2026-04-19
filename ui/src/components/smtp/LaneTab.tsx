"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, useMemo, useState } from "react";
import ComponentCard from "@/components/common/ComponentCard";
import type { LaneItem } from "@/components/smtp/types";

type LaneListResponse = {
  items?: Array<{
    id: string;
    name: string;
    traffic_class: string;
    priority: number;
    consumer_id: string;
    consumer_name: string;
    status: string;
    shard_count?: number;
    ready_shards?: number;
    draining_shards?: number;
    pending_shards?: number;
    created_at: string;
    updated_at: string;
  }>;
};

export function LaneTab() {
  const router = useRouter();
  const [lanes, setLanes] = useState<LaneItem[]>([]);
  const [search, setSearch] = useState("");
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState("");

  useEffect(() => {
    let cancelled = false;

    async function loadLanes() {
      setIsLoading(true);
      setError("");
      try {
        const response = await fetch("/api/v1/smtp/lanes", { cache: "no-store" });
        if (!response.ok) {
          throw new Error("Failed to load SMTP lanes");
        }
        const result = (await response.json()) as LaneListResponse;
        if (!cancelled) {
          setLanes((result.items ?? []).map(mapLaneResponse));
        }
      } catch (err) {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : "Failed to load SMTP lanes");
        }
      } finally {
        if (!cancelled) {
          setIsLoading(false);
        }
      }
    }

    void loadLanes();
    return () => {
      cancelled = true;
    };
  }, []);

  const keyword = search.trim().toLowerCase();
  const filteredLanes = useMemo(
    () =>
      lanes.filter((lane) => {
        if (keyword === "") {
          return true;
        }
        return (
          lane.name.toLowerCase().includes(keyword) ||
          lane.trafficClass.toLowerCase().includes(keyword) ||
          lane.status.toLowerCase().includes(keyword) ||
          lane.id.toLowerCase().includes(keyword)
        );
      }),
    [keyword, lanes],
  );

  return (
    <div className="space-y-6">
      <ComponentCard
        title="Lane"
        desc="Delivery lanes group traffic classes, shard state, and endpoint routing in one place."
        headerAction={
          <Link
            href="/smtp/lanes/new"
            className="inline-flex items-center rounded-xl bg-gray-900 px-4 py-2.5 text-sm font-semibold text-white transition hover:bg-gray-800 dark:bg-white dark:text-gray-900 dark:hover:bg-gray-200"
          >
            Add Lane
          </Link>
        }
      >
        <div className="space-y-4">
          <div className="rounded-2xl border border-gray-200 bg-gray-50 px-4 py-3 dark:border-gray-800 dark:bg-gray-900/40">
            <input
              type="text"
              value={search}
              onChange={(event) => setSearch(event.target.value)}
              placeholder="Search lanes by name, traffic class, status, or id"
              className="w-full bg-transparent text-sm text-gray-800 outline-none placeholder:text-gray-400 dark:text-white dark:placeholder:text-gray-500"
            />
          </div>

          {isLoading ? (
            <div className="rounded-2xl border border-gray-200 bg-gray-50 px-5 py-5 dark:border-gray-800 dark:bg-gray-900/40">
              <p className="text-sm text-gray-500 dark:text-gray-400">Loading lanes...</p>
            </div>
          ) : error !== "" ? (
            <div className="rounded-2xl border border-error-200 bg-error-50 px-5 py-5 dark:border-error-500/30 dark:bg-error-500/10">
              <p className="text-sm text-error-700 dark:text-error-300">{error}</p>
            </div>
          ) : filteredLanes.length === 0 ? (
            <div className="rounded-2xl border border-gray-200 bg-gray-50 px-5 py-5 dark:border-gray-800 dark:bg-gray-900/40">
              <p className="text-sm text-gray-500 dark:text-gray-400">No lane matches your search.</p>
            </div>
          ) : (
            <div className="overflow-hidden rounded-2xl border border-gray-200 bg-white dark:border-gray-800 dark:bg-gray-900/40">
              <div className="overflow-x-auto">
                <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-800">
                  <thead className="bg-gray-50 dark:bg-gray-900/70">
                    <tr>
                      <TableHead>Status</TableHead>
                      <TableHead>Name</TableHead>
                      <TableHead>Lane ID</TableHead>
                      <TableHead>Traffic Class</TableHead>
                      <TableHead>Priority</TableHead>
                      <TableHead>Shards</TableHead>
                      <TableHead>Updated</TableHead>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-gray-200 dark:divide-gray-800">
                    {filteredLanes.map((lane) => (
                      <tr
                        key={lane.id}
                        onClick={() => router.push(`/smtp/lanes/detail?id=${lane.id}`)}
                        className="cursor-pointer transition hover:bg-gray-50 dark:hover:bg-white/5"
                      >
                        <td className="whitespace-nowrap px-5 py-4">
                          <StatusCell status={lane.status} />
                        </td>
                        <td className="px-5 py-4 text-sm font-semibold text-gray-900 dark:text-white">
                          {lane.name}
                        </td>
                        <td className="px-5 py-4">
                          <code className="text-xs text-gray-600 dark:text-gray-300">{lane.id}</code>
                        </td>
                        <td className="whitespace-nowrap px-5 py-4 text-sm text-gray-700 dark:text-gray-200">
                          {lane.trafficClass}
                        </td>
                        <td className="whitespace-nowrap px-5 py-4 text-sm text-gray-700 dark:text-gray-200">
                          {lane.priority}
                        </td>
                        <td className="whitespace-nowrap px-5 py-4 text-sm text-gray-700 dark:text-gray-200">
                          {lane.readyShards ?? 0}/{lane.shardCount ?? 0}
                        </td>
                        <td className="whitespace-nowrap px-5 py-4 text-sm text-gray-700 dark:text-gray-200">
                          {formatDateTime(lane.updatedAt)}
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

function mapLaneResponse(item: NonNullable<LaneListResponse["items"]>[number]): LaneItem {
  return {
    id: item.id,
    name: item.name,
    trafficClass: item.traffic_class || "transactional",
    priority: item.priority ?? 100,
    consumerId: item.consumer_id || "",
    consumer: item.consumer_name || "",
    status: item.status,
    shardCount: item.shard_count ?? 0,
    readyShards: item.ready_shards ?? 0,
    drainingShards: item.draining_shards ?? 0,
    pendingShards: item.pending_shards ?? 0,
    createdAt: item.created_at,
    updatedAt: item.updated_at,
  };
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
