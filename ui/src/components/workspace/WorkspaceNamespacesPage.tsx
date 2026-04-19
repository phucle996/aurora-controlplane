"use client";

import { useEffect, useMemo, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import PageBreadcrumb from "@/components/common/PageBreadCrumb";
import Input from "@/components/form/input/InputField";
import { Modal } from "@/components/ui/modal";
import Badge from "@/components/ui/badge/Badge";
import Button from "@/components/ui/button/Button";
import { Dropdown } from "@/components/ui/dropdown/Dropdown";
import { DropdownItem } from "@/components/ui/dropdown/DropdownItem";
import { useToast } from "@/components/ui/toast/ToastProvider";
import {
  ArrowDownIcon,
  ArrowUpIcon,
  ChevronDownIcon,
  MoreDotIcon,
  PlusIcon,
  TrashBinIcon,
} from "@/icons";
import {
  deleteWorkspaceNamespace,
  listWorkspaceNamespaces,
  type WorkspaceNamespace,
} from "@/components/workspace/api";
import {
  workspaceStatusColor,
  workspaceNamespaceDisplayStatus,
} from "@/components/workspace/data";

type NamespaceSortKey = "namespace" | "zone" | "resource" | "state";
type SortDirection = "asc" | "desc";
type PageSize = 10 | 25 | 50 | 100;
type ToolbarMenu = "zone" | "state" | "pageSize" | null;
type StateFilter = "all" | "ready" | "terminating" | "deleting" | "protected";

const PAGE_SIZE_OPTIONS: PageSize[] = [10, 25, 50, 100];
const STATE_FILTER_OPTIONS: Array<{ label: string; value: StateFilter }> = [
  { label: "All states", value: "all" },
  { label: "Ready", value: "ready" },
  { label: "Terminating", value: "terminating" },
  { label: "Deleting", value: "deleting" },
  { label: "Protected", value: "protected" },
];

export default function WorkspaceNamespacesPage() {
  const router = useRouter();
  const { pushToast } = useToast();
  const [namespaces, setNamespaces] = useState<WorkspaceNamespace[]>([]);
  const [pendingDelete, setPendingDelete] = useState<WorkspaceNamespace | null>(null);
  const [confirmValue, setConfirmValue] = useState("");
  const [openMenuId, setOpenMenuId] = useState<string | null>(null);
  const [openToolbarMenu, setOpenToolbarMenu] = useState<ToolbarMenu>(null);
  const [expandedNamespaceIds, setExpandedNamespaceIds] = useState<Set<string>>(
    () => new Set(),
  );
  const [searchTerm, setSearchTerm] = useState("");
  const [zoneFilter, setZoneFilter] = useState("all");
  const [stateFilter, setStateFilter] = useState<StateFilter>("all");
  const [sortKey, setSortKey] = useState<NamespaceSortKey>("namespace");
  const [sortDirection, setSortDirection] = useState<SortDirection>("asc");
  const [pageSize, setPageSize] = useState<PageSize>(10);
  const [pageIndex, setPageIndex] = useState(0);
  const lastLoadErrorRef = useRef<string>("");

  const zoneOptions = useMemo(() => {
    const zones = new Set(
      namespaces
        .map((item) => item.zone?.trim())
        .filter((item): item is string => Boolean(item)),
    );
    return ["all", ...Array.from(zones).sort((left, right) => left.localeCompare(right))];
  }, [namespaces]);

  const filteredNamespaces = useMemo(() => {
    const query = searchTerm.trim().toLowerCase();

    return namespaces.filter((item) => {
      const displayStatus = workspaceNamespaceDisplayStatus(item);
      const matchesSearch =
        query.length === 0 ||
        [item.display_name, item.description, item.zone, item.status, item.runtime_status]
          .filter(Boolean)
          .some((value) => String(value).toLowerCase().includes(query));
      const matchesZone = zoneFilter === "all" || (item.zone || "") === zoneFilter;
      const matchesState =
        stateFilter === "all" ||
        (stateFilter === "ready" && displayStatus === "ready") ||
        (stateFilter === "terminating" && displayStatus === "terminating") ||
        (stateFilter === "deleting" && item.status === "deleting") ||
        (stateFilter === "protected" && !item.can_delete);

      return matchesSearch && matchesZone && matchesState;
    });
  }, [namespaces, searchTerm, stateFilter, zoneFilter]);

  const sortedNamespaces = useMemo(() => {
    const nextNamespaces = [...filteredNamespaces];

    const compareText = (left: string, right: string) =>
      left.localeCompare(right, undefined, { sensitivity: "base", numeric: true });

    nextNamespaces.sort((left, right) => {
      let result = 0;

      switch (sortKey) {
        case "namespace":
          result = compareText(left.display_name || "", right.display_name || "");
          break;
        case "zone":
          result = compareText(left.zone || "", right.zone || "");
          break;
        case "resource":
          result = (left.resource_count || 0) - (right.resource_count || 0);
          break;
        case "state":
          result = compareText(
            workspaceNamespaceDisplayStatus(left),
            workspaceNamespaceDisplayStatus(right),
          );
          break;
      }

      return sortDirection === "asc" ? result : -result;
    });

    return nextNamespaces;
  }, [filteredNamespaces, sortDirection, sortKey]);

  const totalNamespaces = sortedNamespaces.length;
  const visiblePageSize = pageSize;
  const totalPages = Math.max(1, Math.ceil(totalNamespaces / pageSize));
  const currentPageStart = totalNamespaces === 0 ? 0 : pageIndex * visiblePageSize;
  const currentPageEnd =
    totalNamespaces === 0 ? 0 : Math.min(totalNamespaces, currentPageStart + visiblePageSize);
  const visibleNamespaces = sortedNamespaces.slice(currentPageStart, currentPageEnd);

  const namespaceSummary = useMemo(() => {
    const total = namespaces.length;
    const ready = namespaces.filter((item) => workspaceNamespaceDisplayStatus(item) === "ready").length;
    const terminating = namespaces.filter(
      (item) => workspaceNamespaceDisplayStatus(item) === "terminating",
    ).length;
    const protectedCount = namespaces.filter((item) => !item.can_delete).length;
    return { total, ready, terminating, protectedCount };
  }, [namespaces]);

  useEffect(() => {
    let cancelled = false;
    let timer: ReturnType<typeof setTimeout> | null = null;

    async function load() {
      try {
        const nextNamespaces = await listWorkspaceNamespaces();

        if (!cancelled) {
          setNamespaces(nextNamespaces);
          lastLoadErrorRef.current = "";
        }
      } catch (error) {
        if (!cancelled) {
          const message =
            error instanceof Error ? error.message : "Failed to load namespace resource counts.";
          if (lastLoadErrorRef.current === message) {
            return;
          }
          lastLoadErrorRef.current = message;
          pushToast({
            kind: "error",
            message,
          });
        }
      }
    }

    const schedule = async () => {
      await load();
      if (!cancelled) {
        timer = setTimeout(() => {
          void schedule();
        }, 3000);
      }
    };

    void schedule();
    return () => {
      cancelled = true;
      if (timer) {
        clearTimeout(timer);
      }
    };
  }, [pushToast]);

  useEffect(() => {
    setPageIndex(0);
    closeRowMenu();
    setOpenToolbarMenu(null);
  }, [pageSize, searchTerm, sortDirection, sortKey, stateFilter, zoneFilter]);

  useEffect(() => {
    const maxIndex = Math.max(0, totalPages - 1);
    if (pageIndex > maxIndex) {
      setPageIndex(maxIndex);
    }
  }, [pageIndex, pageSize, totalPages]);

  // openDeleteDialog primes the confirmation modal for a deletable namespace.
  function openDeleteDialog(namespace: WorkspaceNamespace) {
    setPendingDelete(namespace);
    setConfirmValue("");
  }

  function closeDeleteDialog() {
    setPendingDelete(null);
    setConfirmValue("");
  }

  function closeRowMenu() {
    setOpenMenuId(null);
  }

  function closeToolbarMenu() {
    setOpenToolbarMenu(null);
  }

  function toggleNamespaceExpanded(namespaceId: string) {
    setExpandedNamespaceIds((current) => {
      const next = new Set(current);
      if (next.has(namespaceId)) {
        next.delete(namespaceId);
      } else {
        next.add(namespaceId);
      }
      return next;
    });
  }

  function handleSort(nextKey: NamespaceSortKey) {
    if (sortKey === nextKey) {
      setSortDirection((current) => (current === "asc" ? "desc" : "asc"));
      return;
    }

    setSortKey(nextKey);
    setSortDirection("asc");
  }

  function handlePageSizeChange(nextPageSize: PageSize) {
    setPageSize(nextPageSize);
  }

  function handleZoneFilter(nextZone: string) {
    setZoneFilter(nextZone);
  }

  function handleStateFilter(nextState: StateFilter) {
    setStateFilter(nextState);
  }

  // confirmDelete removes a namespace only after the user types its exact name.
  async function confirmDelete() {
    if (!pendingDelete) {
      return;
    }

    if (confirmValue.trim() !== pendingDelete.display_name) {
      pushToast({
        kind: "error",
        message: "Namespace confirmation does not match.",
      });
      return;
    }

    try {
      await deleteWorkspaceNamespace(pendingDelete.id, pendingDelete.display_name);
      setNamespaces(await listWorkspaceNamespaces());
      pushToast({
        kind: "success",
        message: `${pendingDelete.display_name} is terminating in the runtime.`,
      });
      closeDeleteDialog();
      closeRowMenu();
    } catch (error) {
      pushToast({
        kind: "error",
        message: error instanceof Error ? error.message : "Failed to delete namespace.",
      });
    }
  }

  return (
    <div className="space-y-6">
      <PageBreadcrumb pageTitle="Namespaces" />

      <section className="overflow-visible rounded-[32px] border border-gray-200 bg-white dark:border-gray-800 dark:bg-white/[0.03]">
        <div className="grid gap-3 border-b border-gray-200 px-6 py-5 sm:grid-cols-2 xl:grid-cols-4 dark:border-gray-800">
          <MetricCard label="Namespaces" value={namespaceSummary.total} />
          <MetricCard label="Ready" value={namespaceSummary.ready} tone="success" />
          <MetricCard label="Terminating" value={namespaceSummary.terminating} tone="warning" />
          <MetricCard label="Protected" value={namespaceSummary.protectedCount} tone="primary" />
        </div>

        <div className="border-b border-gray-200 px-6 py-5 dark:border-gray-800">
          <div className="flex flex-col gap-4 xl:flex-row xl:items-end xl:justify-between">
            <div className="grid flex-1 gap-4 lg:grid-cols-[minmax(0,1.5fr)_minmax(0,0.8fr)_minmax(0,0.8fr)]">
              <div className="space-y-2">
                <label className="text-sm font-medium text-gray-700 dark:text-gray-300">
                  Search
                </label>
                <Input
                  value={searchTerm}
                  onChange={(event) => setSearchTerm(event.target.value)}
                  placeholder="Search namespace, description, zone, or cluster"
                  className="rounded-xl border-gray-200 bg-white dark:border-gray-800 dark:bg-white/[0.03]"
                />
              </div>

              <div className="space-y-2">
                <label className="text-sm font-medium text-gray-700 dark:text-gray-300">
                  Zone
                </label>
                <div className="relative">
                  <button
                    type="button"
                    className="dropdown-toggle flex h-11 w-full items-center justify-between rounded-xl border border-gray-200 bg-white px-4 text-sm text-gray-800 transition hover:border-gray-300 dark:border-gray-800 dark:bg-white/[0.03] dark:text-white"
                    onClick={() =>
                      setOpenToolbarMenu((current) => (current === "zone" ? null : "zone"))
                    }
                  >
                    <span className="truncate">
                      {zoneFilter === "all" ? "All zones" : zoneFilter}
                    </span>
                    <ChevronDownIcon className="size-4 text-gray-400" aria-hidden="true" />
                  </button>
                  <Dropdown
                    isOpen={openToolbarMenu === "zone"}
                    onClose={closeToolbarMenu}
                    className="w-72 p-1"
                  >
                    {zoneOptions.map((zone) => {
                      const active = zoneFilter === zone;
                      return (
                        <DropdownItem
                          key={zone}
                          onItemClick={() => {
                            handleZoneFilter(zone);
                            closeToolbarMenu();
                          }}
                          baseClassName="flex w-full items-center justify-between rounded-lg px-4 py-2 text-left text-sm text-gray-700 hover:bg-gray-100 hover:text-gray-900 dark:text-gray-300 dark:hover:bg-white/[0.04] dark:hover:text-white"
                        >
                          <span>{zone === "all" ? "All zones" : zone}</span>
                          {active ? <span className="text-brand-500">✓</span> : null}
                        </DropdownItem>
                      );
                    })}
                  </Dropdown>
                </div>
              </div>

              <div className="space-y-2">
                <label className="text-sm font-medium text-gray-700 dark:text-gray-300">
                  State
                </label>
                <div className="relative">
                  <button
                    type="button"
                    className="dropdown-toggle flex h-11 w-full items-center justify-between rounded-xl border border-gray-200 bg-white px-4 text-sm text-gray-800 transition hover:border-gray-300 dark:border-gray-800 dark:bg-white/[0.03] dark:text-white"
                    onClick={() =>
                      setOpenToolbarMenu((current) => (current === "state" ? null : "state"))
                    }
                  >
                    <span className="truncate">
                      {STATE_FILTER_OPTIONS.find((item) => item.value === stateFilter)?.label ?? "All states"}
                    </span>
                    <ChevronDownIcon className="size-4 text-gray-400" aria-hidden="true" />
                  </button>
                  <Dropdown
                    isOpen={openToolbarMenu === "state"}
                    onClose={closeToolbarMenu}
                    className="w-56 p-1"
                  >
                    {STATE_FILTER_OPTIONS.map((option) => {
                      const active = stateFilter === option.value;
                      return (
                        <DropdownItem
                          key={option.value}
                          onItemClick={() => {
                            handleStateFilter(option.value);
                            closeToolbarMenu();
                          }}
                          baseClassName="flex w-full items-center justify-between rounded-lg px-4 py-2 text-left text-sm text-gray-700 hover:bg-gray-100 hover:text-gray-900 dark:text-gray-300 dark:hover:bg-white/[0.04] dark:hover:text-white"
                        >
                          <span>{option.label}</span>
                          {active ? <span className="text-brand-500">✓</span> : null}
                        </DropdownItem>
                      );
                    })}
                  </Dropdown>
                </div>
              </div>
            </div>

            <Button
              className="rounded-xl px-5"
              startIcon={<PlusIcon className="size-4" />}
              onClick={() => router.push("/workspace/namespaces/new")}
            >
              New namespace
            </Button>
          </div>
        </div>

        <div className="relative overflow-x-auto">
          <table className="min-w-full table-fixed divide-y divide-gray-200 dark:divide-gray-800">
            <colgroup>
              <col className="w-[38%]" />
              <col className="w-[22%]" />
              <col className="w-[14%]" />
              <col className="w-[18%]" />
              <col className="w-[8%]" />
            </colgroup>
            <thead className="bg-gray-50/70 dark:bg-white/[0.02]">
              <tr className="text-left text-xs uppercase tracking-[0.18em] text-gray-400 dark:text-gray-500">
                <ThSort
                  label="Namespace"
                  active={sortKey === "namespace"}
                  direction={sortDirection}
                  onClick={() => handleSort("namespace")}
                />
                <ThSort
                  label="Zone"
                  active={sortKey === "zone"}
                  direction={sortDirection}
                  onClick={() => handleSort("zone")}
                />
                <ThSort
                  label="Resources"
                  active={sortKey === "resource"}
                  direction={sortDirection}
                  align="right"
                  onClick={() => handleSort("resource")}
                />
                <ThSort
                  label="State"
                  active={sortKey === "state"}
                  direction={sortDirection}
                  onClick={() => handleSort("state")}
                />
                <th className="px-8 py-4 font-medium text-right">Manage</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-200 dark:divide-gray-800">
              {namespaces.length === 0 ? (
                <tr>
                  <td colSpan={5} className="px-8 py-16">
                    <div className="mx-auto max-w-md text-center">
                      <div className="mx-auto mb-4 flex h-14 w-14 items-center justify-center rounded-2xl bg-brand-50 text-brand-500 dark:bg-brand-500/10 dark:text-brand-400">
                        <PlusIcon className="size-5" />
                      </div>
                      <h2 className="text-lg font-semibold text-gray-900 dark:text-white">
                        No namespaces yet
                      </h2>
                      <p className="mt-2 text-sm leading-7 text-gray-500 dark:text-gray-400">
                        Create a namespace to group resources and keep runtime state
                        isolated per team or app.
                      </p>
                      <div className="mt-6">
                        <Button
                          className="rounded-xl px-5"
                          startIcon={<PlusIcon className="size-4" />}
                          onClick={() => router.push("/workspace/namespaces/new")}
                        >
                          Add namespace
                        </Button>
                      </div>
                    </div>
                  </td>
                </tr>
              ) : null}

              {visibleNamespaces.map((item) => {
                const resourceCount = item.resource_count || 0;
                const canDelete = item.can_delete && resourceCount === 0;
                const displayStatus = workspaceNamespaceDisplayStatus(item);
                const runtimeStatus = item.runtime_status?.trim();
                const statusCopy =
                  runtimeStatus && runtimeStatus !== item.status ? runtimeStatus : "";
                const isExpanded = expandedNamespaceIds.has(item.id);
                const description = item.description?.trim() || "";

                return (
                  <tr
                    key={item.id}
                    className={[
                      "align-top transition-colors",
                      "hover:bg-gray-50/70 dark:hover:bg-white/[0.02]",
                      item.status === "deleting" ? "bg-warning-50/35 dark:bg-warning-500/5" : "",
                    ].join(" ")}
                  >
                    <td className="px-8 py-5 align-top">
                      <div className="space-y-2">
                        <div className="flex flex-wrap items-center gap-2">
                          <button
                            type="button"
                            className="inline-flex items-center rounded-lg px-1 py-0.5 text-left transition hover:bg-brand-50 dark:hover:bg-brand-500/10"
                            onClick={() => toggleNamespaceExpanded(item.id)}
                            aria-expanded={isExpanded}
                            aria-controls={`namespace-detail-${item.id}`}
                          >
                            <span className="text-sm font-semibold text-gray-900 transition hover:text-brand-600 dark:text-white dark:hover:text-brand-300">
                              {item.display_name}
                            </span>
                          </button>

                          {item.is_default ? (
                            <Badge size="sm" color="primary">
                              Default
                            </Badge>
                          ) : null}
                          {item.status === "deleting" ? (
                            <Badge size="sm" color="warning">
                              Terminating
                            </Badge>
                          ) : null}
                          {!item.can_delete && item.status !== "deleting" ? (
                            <Badge size="sm" color="warning">
                              Protected
                            </Badge>
                          ) : null}
                        </div>
                        {isExpanded ? (
                          <p
                            id={`namespace-detail-${item.id}`}
                            className="max-w-3xl text-sm leading-7 text-gray-600 dark:text-gray-300"
                          >
                            {description || "No description provided for this namespace."}
                          </p>
                        ) : null}
                      </div>
                    </td>
                    <td className="px-8 py-5 align-top text-sm text-gray-700 dark:text-gray-300">
                      <span className="inline-flex rounded-full bg-gray-100 px-2.5 py-1 text-xs font-medium text-gray-700 dark:bg-white/[0.04] dark:text-gray-300">
                        {item.zone || "-"}
                      </span>
                    </td>
                    <td className="px-8 py-5 align-top text-right text-sm font-medium tabular-nums text-gray-800 dark:text-gray-200">
                      {resourceCount}
                    </td>
                    <td className="px-8 py-5 align-top">
                      <div className="space-y-2">
                        <Badge color={workspaceStatusColor(displayStatus)}>{displayStatus}</Badge>
                        {statusCopy ? (
                          <p className="text-xs text-gray-500 dark:text-gray-400">
                            Runtime: {statusCopy}
                          </p>
                        ) : null}
                      </div>
                    </td>
                    <td className="px-8 py-5 align-top text-right">
                      <div className="relative inline-flex justify-end overflow-visible">
                        {item.status === "deleting" ? (
                          <span className="inline-flex rounded-full bg-warning-50 px-3 py-1 text-xs font-medium text-warning-700 dark:bg-warning-500/10 dark:text-warning-400">
                            In runtime teardown
                          </span>
                        ) : canDelete ? (
                          <>
                            <button
                              type="button"
                              aria-label={`Open actions for ${item.display_name}`}
                              className="dropdown-toggle inline-flex h-10 w-10 items-center justify-center rounded-full border border-gray-200 bg-white text-gray-500 transition hover:border-gray-300 hover:bg-gray-50 hover:text-gray-700 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500/30 dark:border-gray-800 dark:bg-white/[0.03] dark:text-gray-400 dark:hover:border-gray-700 dark:hover:bg-white/[0.05] dark:hover:text-white"
                              onClick={() =>
                                setOpenMenuId((current) => (current === item.id ? null : item.id))
                              }
                            >
                              <MoreDotIcon className="size-5" aria-hidden="true" />
                            </button>

                            <Dropdown
                              isOpen={openMenuId === item.id}
                              onClose={closeRowMenu}
                              placement="top"
                              className="w-56 p-1"
                            >
                              <DropdownItem
                                onItemClick={() => {
                                  openDeleteDialog(item);
                                  closeRowMenu();
                                }}
                                className="rounded-lg px-3 py-2 text-sm text-error-600 hover:bg-error-50 hover:text-error-700 dark:text-error-400 dark:hover:bg-error-500/10 dark:hover:text-error-300"
                              >
                                <span className="inline-flex items-center gap-2">
                                  <TrashBinIcon className="size-4" aria-hidden="true" />
                                  Delete namespace
                                </span>
                              </DropdownItem>
                            </Dropdown>
                          </>
                        ) : (
                          <span className="inline-flex rounded-full bg-gray-100 px-3 py-1 text-xs font-medium text-gray-600 dark:bg-white/[0.04] dark:text-gray-400">
                            Protected namespace
                          </span>
                        )}
                      </div>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>

        <div className="flex flex-col gap-4 border-t border-gray-200 px-6 py-4 text-sm text-gray-500 dark:border-gray-800 dark:text-gray-400 xl:flex-row xl:items-center xl:justify-between">
          <p>
            Showing{" "}
            <span className="font-medium text-gray-900 dark:text-white">
              {totalNamespaces === 0 ? 0 : currentPageStart + 1}
            </span>{" "}
            to{" "}
            <span className="font-medium text-gray-900 dark:text-white">{currentPageEnd}</span>{" "}
            of{" "}
            <span className="font-medium text-gray-900 dark:text-white">{totalNamespaces}</span>{" "}
            namespaces.
          </p>

          <div className="flex flex-wrap items-center gap-4">
            <div className="relative flex items-center gap-3">
              <span className="text-sm font-medium text-gray-700 dark:text-gray-300">
                Rows per page
              </span>
              <button
                type="button"
                className="dropdown-toggle inline-flex h-10 min-w-24 items-center justify-between rounded-xl border border-gray-200 bg-white px-3 text-sm text-gray-800 transition hover:border-gray-300 dark:border-gray-800 dark:bg-white/[0.03] dark:text-white"
                onClick={() =>
                  setOpenToolbarMenu((current) => (current === "pageSize" ? null : "pageSize"))
                }
              >
                <span>{pageSize}</span>
                <ChevronDownIcon className="size-4 text-gray-400" aria-hidden="true" />
              </button>
              <Dropdown
                isOpen={openToolbarMenu === "pageSize"}
                onClose={closeToolbarMenu}
                placement="top"
                className="w-36 p-1"
              >
                {PAGE_SIZE_OPTIONS.map((option) => {
                  const active = pageSize === option;
                  return (
                    <DropdownItem
                      key={option}
                      onItemClick={() => {
                        handlePageSizeChange(option);
                        closeToolbarMenu();
                      }}
                      baseClassName="flex w-full items-center justify-between rounded-lg px-4 py-2 text-left text-sm text-gray-700 hover:bg-gray-100 hover:text-gray-900 dark:text-gray-300 dark:hover:bg-white/[0.04] dark:hover:text-white"
                    >
                      <span>{option}</span>
                      {active ? <span className="text-brand-500">✓</span> : null}
                    </DropdownItem>
                  );
                })}
              </Dropdown>
            </div>

            <div className="flex items-center gap-2">
              <Button
                size="sm"
                variant="outline"
                className="rounded-xl px-4"
                disabled={pageIndex === 0 || totalNamespaces === 0}
                onClick={() => setPageIndex((current) => Math.max(current - 1, 0))}
              >
                Previous
              </Button>
              <span className="rounded-full border border-gray-200 bg-white px-3 py-2 text-xs font-medium text-gray-600 dark:border-gray-800 dark:bg-white/[0.03] dark:text-gray-400">
                Page {pageIndex + 1} of {totalPages}
              </span>
              <Button
                size="sm"
                variant="outline"
                className="rounded-xl px-4"
                disabled={totalNamespaces === 0 || pageIndex >= totalPages - 1}
                onClick={() => setPageIndex((current) => Math.min(current + 1, totalPages - 1))}
              >
                Next
              </Button>
            </div>
          </div>
        </div>
      </section>

      <Modal isOpen={pendingDelete !== null} onClose={closeDeleteDialog} className="max-w-xl p-0">
        {pendingDelete ? (
          <div className="space-y-6 p-6">
            <div className="space-y-2">
              <p className="text-xs font-semibold uppercase tracking-[0.24em] text-error-500">
                Delete namespace
              </p>
              <h2 className="text-2xl font-semibold text-gray-900 dark:text-white">
                Confirm namespace removal
              </h2>
              <p className="text-sm leading-7 text-gray-500 dark:text-gray-400">
                Type{" "}
                <span className="font-semibold text-gray-900 dark:text-white">
                  {pendingDelete.display_name}
                </span>{" "}
                to remove this namespace from the workspace catalog.
              </p>
            </div>

            <div className="space-y-2">
              <label className="text-sm font-medium text-gray-700 dark:text-gray-300">
                Namespace name
              </label>
              <Input
                value={confirmValue}
                onChange={(event) => setConfirmValue(event.target.value)}
                placeholder={pendingDelete.display_name}
              />
            </div>

            <div className="flex gap-3">
              <Button
                className="rounded-xl px-5"
                onClick={confirmDelete}
                disabled={confirmValue.trim() !== pendingDelete.display_name}
              >
                Delete namespace
              </Button>
              <Button
                variant="outline"
                className="rounded-xl px-5"
                onClick={closeDeleteDialog}
              >
                Cancel
              </Button>
            </div>
          </div>
        ) : null}
      </Modal>
    </div>
  );
}

function MetricCard(props: { label: string; value: number; tone?: "primary" | "success" | "warning" }) {
  const tone = props.tone ?? "primary";
  const toneClasses = {
    primary: "bg-brand-50 text-brand-600 dark:bg-brand-500/10 dark:text-brand-300",
    success: "bg-success-50 text-success-700 dark:bg-success-500/10 dark:text-success-300",
    warning: "bg-warning-50 text-warning-700 dark:bg-warning-500/10 dark:text-warning-300",
  }[tone];

  return (
    <div className="rounded-2xl border border-gray-200 bg-white px-4 py-4 shadow-theme-xs dark:border-gray-800 dark:bg-white/[0.03]">
      <p className="text-xs font-medium uppercase tracking-[0.18em] text-gray-400 dark:text-gray-500">
        {props.label}
      </p>
      <div
        className={`mt-3 inline-flex rounded-2xl px-3 py-2 text-2xl font-semibold tabular-nums ${toneClasses}`}
      >
        {props.value}
      </div>
    </div>
  );
}

function ThSort(props: {
  label: string;
  active: boolean;
  direction: SortDirection;
  onClick: () => void;
  align?: "left" | "right";
}) {
  const icon =
    props.active && props.direction === "desc" ? (
      <ArrowDownIcon className="size-3.5 text-brand-500" aria-hidden="true" />
    ) : (
      <ArrowUpIcon
        className={[
          "size-3.5",
          props.active ? "text-brand-500" : "text-gray-300 transition group-hover:text-gray-500 dark:text-gray-600",
        ].join(" ")}
        aria-hidden="true"
      />
    );

  return (
    <th
      className={[
        "px-6 py-4 font-medium",
        props.align === "right" ? "text-right" : "text-left",
      ].join(" ")}
      aria-sort={props.active ? (props.direction === "asc" ? "ascending" : "descending") : "none"}
    >
      <button
        type="button"
        onClick={props.onClick}
        className={[
          "group inline-flex items-center gap-1.5",
          props.align === "right" ? "ml-auto justify-end" : "justify-start",
        ].join(" ")}
      >
        <span>{props.label}</span>
        {icon}
      </button>
    </th>
  );
}
