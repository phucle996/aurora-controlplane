"use client";

import Link from "next/link";
import { useEffect, useMemo, useState } from "react";
import ComponentCard from "@/components/common/ComponentCard";
import type { ConsumerItem } from "@/components/smtp/types";

type ConsumerListResponse = {
  items?: Array<{
    id: string;
    name: string;
    transport_type: string;
    source: string;
    consumer_group: string;
    status: string;
    note: string;
    created_at: string;
    updated_at: string;
  }>;
};

type ConsumerDetailResponse = {
  id: string;
  name: string;
  transport_type: string;
  source: string;
  consumer_group: string;
  status: string;
  note: string;
  created_at: string;
  updated_at: string;
};

export function ConsumerTab({
  search,
  onSearchChange,
}: {
  search: string;
  onSearchChange: (value: string) => void;
}) {
  const [consumers, setConsumers] = useState<ConsumerItem[]>([]);
  const [selectedConsumerID, setSelectedConsumerID] = useState("");
  const [selectedConsumer, setSelectedConsumer] = useState<ConsumerItem | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState("");
  const [isDetailLoading, setIsDetailLoading] = useState(false);
  const [detailError, setDetailError] = useState("");

  useEffect(() => {
    let cancelled = false;

    async function loadConsumers() {
      setIsLoading(true);
      setError("");

      try {
        const response = await fetch("/api/v1/smtp/consumers", {
          method: "GET",
          cache: "no-store",
        });
        if (!response.ok) {
          throw new Error("Failed to load SMTP consumers");
        }

        const result = (await response.json()) as ConsumerListResponse;
        if (cancelled) {
          return;
        }

        const items = (result.items ?? []).map(mapConsumerResponse);
        setConsumers(items);
        if (items.length === 0) {
          setSelectedConsumerID("");
          setSelectedConsumer(null);
          return;
        }

        const hasSelected = items.some((item) => item.id === selectedConsumerID);
        if (!hasSelected && selectedConsumerID !== "") {
          setSelectedConsumerID("");
          setSelectedConsumer(null);
        }
      } catch (err) {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : "Failed to load SMTP consumers");
        }
      } finally {
        if (!cancelled) {
          setIsLoading(false);
        }
      }
    }

    void loadConsumers()
    return () => {
      cancelled = true;
    };
  }, []);

  useEffect(() => {
    if (selectedConsumerID === "") {
      setSelectedConsumer(null);
      setDetailError("");
      return;
    }

    let cancelled = false;

    async function loadConsumerDetail() {
      setIsDetailLoading(true);
      setDetailError("");

      try {
        const response = await fetch(`/api/v1/smtp/consumers/${selectedConsumerID}`, {
          method: "GET",
          cache: "no-store",
        });
        if (!response.ok) {
          throw new Error("Failed to load SMTP consumer detail");
        }

        const result = (await response.json()) as ConsumerDetailResponse;
        if (cancelled) {
          return;
        }
        setSelectedConsumer(mapConsumerResponse(result));
      } catch (err) {
        if (!cancelled) {
          setDetailError(
            err instanceof Error ? err.message : "Failed to load SMTP consumer detail",
          );
        }
      } finally {
        if (!cancelled) {
          setIsDetailLoading(false);
        }
      }
    }

    void loadConsumerDetail()
    return () => {
      cancelled = true;
    };
  }, [selectedConsumerID]);

  const keyword = search.trim().toLowerCase();
  const filteredConsumers = useMemo(
    () =>
      consumers.filter((consumer) => {
        if (keyword === "") {
          return true;
        }

        return (
          consumer.name.toLowerCase().includes(keyword) ||
          consumer.stream.toLowerCase().includes(keyword) ||
          consumer.status.toLowerCase().includes(keyword) ||
          consumer.detail.toLowerCase().includes(keyword)
        );
      }),
    [consumers, keyword],
  );

  return (
    <div className="space-y-6">
      <ComponentCard
        title="Consumer Lanes"
        desc="Track SMTP consumers by status, transport, source stream, and consumer group."
        headerAction={
          <Link
            href="/smtp/consumers/new"
            className="inline-flex items-center rounded-xl bg-gray-900 px-4 py-2.5 text-sm font-semibold text-white transition hover:bg-gray-800 dark:bg-white dark:text-gray-900 dark:hover:bg-gray-200"
          >
            New Consumer
          </Link>
        }
      >
        <div className="space-y-4">
          <div className="rounded-2xl border border-gray-200 bg-gray-50 px-4 py-3 dark:border-gray-800 dark:bg-gray-900/40">
            <input
              type="text"
              value={search}
              onChange={(event) => onSearchChange(event.target.value)}
              placeholder="Search consumers by name, stream, or status"
              className="w-full bg-transparent text-sm text-gray-800 outline-none placeholder:text-gray-400 dark:text-white dark:placeholder:text-gray-500"
            />
          </div>

          <div className="grid gap-3">
            {isLoading && (
              <div className="rounded-2xl border border-gray-200 bg-gray-50 px-5 py-5 dark:border-gray-800 dark:bg-gray-900/40">
                <p className="text-sm text-gray-500 dark:text-gray-400">
                  Loading consumers...
                </p>
              </div>
            )}

            {!isLoading && error !== "" && (
              <div className="rounded-2xl border border-error-200 bg-error-50 px-5 py-5 dark:border-error-500/30 dark:bg-error-500/10">
                <p className="text-sm text-error-700 dark:text-error-300">{error}</p>
              </div>
            )}

            {!isLoading && error === "" && filteredConsumers.length > 0 && (
              <div className="overflow-hidden rounded-2xl border border-gray-200 bg-white dark:border-gray-800 dark:bg-gray-900/40">
                <div className="overflow-x-auto">
                  <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-800">
                    <thead className="bg-gray-50 dark:bg-gray-900/70">
                      <tr>
                        <TableHead>Status</TableHead>
                        <TableHead>Name</TableHead>
                        <TableHead>Consumer ID</TableHead>
                        <TableHead>Transport</TableHead>
                        <TableHead>Stream</TableHead>
                        <TableHead>Consumer Group</TableHead>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-gray-200 dark:divide-gray-800">
                      {filteredConsumers.map((consumer) => (
                        <tr
                          key={consumer.id}
                          onClick={() => setSelectedConsumerID(consumer.id)}
                          className="cursor-pointer transition hover:bg-gray-50 dark:hover:bg-white/5"
                        >
                          <td className="whitespace-nowrap px-5 py-4">
                            <StatusCell status={consumer.status} />
                          </td>
                          <td className="px-5 py-4">
                            <div className="min-w-[180px]">
                              <p className="text-sm font-semibold text-gray-900 dark:text-white">
                                {consumer.name}
                              </p>
                              <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
                                {consumer.detail || "No note"}
                              </p>
                            </div>
                          </td>
                          <td className="px-5 py-4">
                            <code className="text-xs text-gray-600 dark:text-gray-300">
                              {consumer.id}
                            </code>
                          </td>
                          <td className="whitespace-nowrap px-5 py-4 text-sm text-gray-700 dark:text-gray-200">
                            {consumer.transportType ?? "-"}
                          </td>
                          <td className="px-5 py-4 text-sm text-gray-700 dark:text-gray-200">
                            <span className="break-all">{consumer.stream || "-"}</span>
                          </td>
                          <td className="px-5 py-4 text-sm text-gray-700 dark:text-gray-200">
                            {consumer.consumerGroup || "-"}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </div>
            )}
          </div>

          {!isLoading && error === "" && filteredConsumers.length === 0 && (
            <div className="rounded-2xl border border-gray-200 bg-gray-50 px-5 py-5 dark:border-gray-800 dark:bg-gray-900/40">
              <p className="text-sm text-gray-500 dark:text-gray-400">
                No consumer matches your search.
              </p>
            </div>
          )}
        </div>
      </ComponentCard>

      <ConsumerDrawer
        consumer={selectedConsumer}
        onClose={() => {
          setSelectedConsumerID("");
          setSelectedConsumer(null);
          setDetailError("");
        }}
        isLoading={isDetailLoading}
        error={detailError}
      />
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
    normalized === "active" || normalized === "running" || normalized === "ready" || normalized === "healthy"
      ? "bg-emerald-500"
      : normalized === "disabled" || normalized === "unhealthy" || normalized === "failed" || normalized === "error"
        ? "bg-rose-500"
        : "bg-amber-500";

  return (
    <div className="flex items-center gap-2">
      <span className={`h-2.5 w-2.5 rounded-full ${toneClass}`} />
      <span className="text-sm capitalize text-gray-700 dark:text-gray-200">{status}</span>
    </div>
  );
}

function ConsumerDrawer({
  consumer,
  onClose,
  isLoading,
  error,
}: {
  consumer: ConsumerItem | null;
  onClose: () => void;
  isLoading: boolean;
  error: string;
}) {
  return (
    <>
      <div
        className={`fixed inset-0 z-40 bg-gray-900/40 transition ${
          consumer ? "pointer-events-auto opacity-100" : "pointer-events-none opacity-0"
        }`}
        onClick={onClose}
      />
      <aside
        className={`fixed top-0 right-0 z-50 flex h-full w-full max-w-[420px] flex-col border-l border-gray-200 bg-white shadow-2xl transition-transform duration-300 dark:border-gray-800 dark:bg-gray-900 ${
          consumer ? "translate-x-0" : "translate-x-full"
        }`}
      >
        <div className="flex items-start justify-between gap-4 border-b border-gray-200 px-6 py-5 dark:border-gray-800">
          <div>
            <p className="text-xs font-medium tracking-[0.22em] text-gray-400 uppercase">
              Consumer Detail
            </p>
            <h2 className="mt-2 text-xl font-semibold text-gray-900 dark:text-white">
              {consumer?.name ?? "Consumer"}
            </h2>
          </div>
          <button
            type="button"
            onClick={onClose}
            className="inline-flex h-10 w-10 items-center justify-center rounded-2xl border border-gray-200 text-gray-500 transition hover:bg-gray-50 dark:border-gray-700 dark:text-gray-300 dark:hover:bg-gray-800"
          >
            x
          </button>
        </div>

        <div className="flex-1 space-y-5 overflow-y-auto px-6 py-6">
          {isLoading ? (
            <DrawerInfo label="State" value="Loading consumer detail..." />
          ) : error !== "" ? (
            <DrawerInfo label="Error" value={error} />
          ) : (
            <>
              <DrawerInfo
                label="Note"
                value={consumer?.detail ?? "Select a consumer from the list to inspect it."}
              />
              <DrawerInfo label="Source" value={consumer?.stream ?? "-"} />
              <DrawerInfo label="Transport" value={consumer?.transportType ?? "-"} />
              <DrawerInfo label="Consumer group" value={consumer?.consumerGroup ?? "-"} />
              <DrawerInfo label="Status" value={consumer?.status ?? "-"} />
              <DrawerInfo label="Consumer ID" value={consumer?.id ?? "-"} />
              <DrawerInfo
                label="Created at"
                value={formatDateTime(consumer?.createdAt)}
              />
              <DrawerInfo
                label="Updated at"
                value={formatDateTime(consumer?.updatedAt)}
              />
            </>
          )}
        </div>
      </aside>
    </>
  );
}

function DrawerInfo({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-2xl border border-gray-200 bg-gray-50 p-4 dark:border-gray-800 dark:bg-gray-900/40">
      <p className="text-xs font-medium tracking-[0.18em] text-gray-400 uppercase">
        {label}
      </p>
      <p className="mt-2 text-sm leading-6 text-gray-700 dark:text-gray-200">
        {value}
      </p>
    </div>
  );
}

function mapConsumerResponse(
  item: NonNullable<ConsumerListResponse["items"]>[number] | ConsumerDetailResponse,
): ConsumerItem {
  return {
    id: item.id,
    name: item.name,
    stream: item.source,
    status: item.status,
    detail: item.note,
    transportType: item.transport_type,
    consumerGroup: item.consumer_group,
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
