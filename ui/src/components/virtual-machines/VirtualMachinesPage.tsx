"use client";

import { useEffect, useMemo, useState } from "react";
import { useRouter } from "next/navigation";
import PageBreadcrumb from "@/components/common/PageBreadCrumb";
import Button from "@/components/ui/button/Button";
import Input from "@/components/form/input/InputField";
import { useToast } from "@/components/ui/toast/ToastProvider";
import { MoreDotIcon, PlusIcon, TimeIcon } from "@/icons";
import { statusClasses, statusDotClasses, type VMStatus } from "@/components/virtual-machines/data";
import {
  listHypervisorVirtualMachines,
  runHypervisorVirtualMachineAction,
  type HypervisorVirtualMachine,
} from "@/components/hypervisor/api";

function normalizeStatus(item: HypervisorVirtualMachine): VMStatus {
  if (item.power_state === "running" || item.status === "running") {
    return "running";
  }
  if (item.power_state === "stopped" || item.status === "stopped") {
    return "stopped";
  }
  if (item.power_state === "creating" || item.status === "provisioning") {
    return "starting";
  }
  return "maintenance";
}

function relativeTime(value: string) {
  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) {
    return "pending";
  }

  const diffMs = Date.now() - parsed.getTime();
  const diffMinutes = Math.max(Math.floor(diffMs / 60000), 0);
  if (diffMinutes < 1) return "just now";
  if (diffMinutes < 60) return `${diffMinutes}m ago`;

  const diffHours = Math.floor(diffMinutes / 60);
  if (diffHours < 24) return `${diffHours}h ago`;

  const diffDays = Math.floor(diffHours / 24);
  return `${diffDays}d ago`;
}

export default function VirtualMachinesPage() {
  const router = useRouter();
  const { pushToast } = useToast();
  const [search, setSearch] = useState("");
  const [statusFilter, setStatusFilter] = useState("all");
  const [selectedVMIDs, setSelectedVMIDs] = useState<string[]>([]);
  const [items, setItems] = useState<HypervisorVirtualMachine[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [busyAction, setBusyAction] = useState<"" | "start" | "stop">("");

  async function loadItems() {
    try {
      setLoading(true);
      setError("");
      setItems(await listHypervisorVirtualMachines());
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load virtual machines.");
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    void loadItems();
  }, []);

  const filteredItems = useMemo(() => {
    const query = search.trim().toLowerCase();

    return items.filter((item) => {
      const normalizedStatus = normalizeStatus(item);
      const matchesStatus = statusFilter === "all" || normalizedStatus === statusFilter;
      const matchesSearch =
        query === "" ||
        item.name.toLowerCase().includes(query) ||
        item.id.toLowerCase().includes(query) ||
        item.image.toLowerCase().includes(query) ||
        item.zone.toLowerCase().includes(query) ||
        item.primary_ip.toLowerCase().includes(query);

      return matchesStatus && matchesSearch;
    });
  }, [items, search, statusFilter]);
  const allVisibleSelected =
    filteredItems.length > 0 && filteredItems.every((item) => selectedVMIDs.includes(item.id));

  function toggleVMSelection(vmID: string) {
    setSelectedVMIDs((current) =>
      current.includes(vmID) ? current.filter((item) => item !== vmID) : [...current, vmID],
    );
  }

  function toggleVisibleSelection() {
    if (allVisibleSelected) {
      setSelectedVMIDs((current) => current.filter((id) => !filteredItems.some((item) => item.id === id)));
      return;
    }

    setSelectedVMIDs((current) => Array.from(new Set([...current, ...filteredItems.map((item) => item.id)])));
  }

  async function runBulkAction(action: "start" | "stop") {
    if (selectedVMIDs.length === 0) {
      return;
    }

    try {
      setBusyAction(action);
      await Promise.all(selectedVMIDs.map((id) => runHypervisorVirtualMachineAction(id, action)));
      pushToast({
        kind: "success",
        message: `${action === "start" ? "Start" : "Stop"} command sent for ${selectedVMIDs.length} VM(s).`,
      });
      setSelectedVMIDs([]);
      await loadItems();
    } catch (err) {
      pushToast({
        kind: "error",
        message: err instanceof Error ? err.message : "Failed to send VM action.",
      });
    } finally {
      setBusyAction("");
    }
  }

  return (
    <div className="space-y-6">
      <PageBreadcrumb pageTitle="Virtual Machines" />

      <section className="rounded-3xl border border-gray-200 bg-white dark:border-gray-800 dark:bg-white/[0.03]">
        <div className="flex flex-col gap-4 border-b border-gray-200 px-6 py-5 dark:border-gray-800 xl:flex-row xl:items-center xl:justify-between">
          <div>
            <h1 className="text-2xl font-semibold text-gray-900 dark:text-white">
              Danh sach Virtual Machines
            </h1>
            <p className="mt-2 max-w-2xl text-sm leading-6 text-gray-500 dark:text-gray-400">
              Quan ly va thao tac tren cac may ao cua user hien tai.
            </p>
          </div>

          <div className="inline-flex items-center rounded-full bg-brand-500/10 px-3 py-1 text-xs font-semibold uppercase tracking-[0.24em] text-brand-600 dark:text-brand-400">
            {selectedVMIDs.length} selected
          </div>
        </div>

        <div className="flex flex-col gap-4 px-6 py-5 xl:flex-row xl:items-center xl:justify-between">
          <div className="flex flex-col gap-3 md:flex-row md:items-center">
            <div className="relative min-w-0 md:w-[340px]">
              <span className="pointer-events-none absolute left-4 top-1/2 -translate-y-1/2 text-gray-400">
                <svg
                  width="20"
                  height="20"
                  viewBox="0 0 20 20"
                  fill="none"
                  xmlns="http://www.w3.org/2000/svg"
                >
                  <path
                    d="M17.5 17.5L13.875 13.875M15.8333 9.16667C15.8333 12.8486 12.8486 15.8333 9.16667 15.8333C5.48477 15.8333 2.5 12.8486 2.5 9.16667C2.5 5.48477 5.48477 2.5 9.16667 2.5C12.8486 2.5 15.8333 5.48477 15.8333 9.16667Z"
                    stroke="currentColor"
                    strokeWidth="1.5"
                    strokeLinecap="round"
                    strokeLinejoin="round"
                  />
                </svg>
              </span>
              <Input
                type="text"
                value={search}
                onChange={(event) => setSearch(event.target.value)}
                placeholder="Tim kiem VM theo ten, ID, OS, zone..."
                className="pl-11"
              />
            </div>

            <select
              value={statusFilter}
              onChange={(event) => setStatusFilter(event.target.value)}
              className="h-11 rounded-lg border border-gray-300 bg-transparent px-4 text-sm text-gray-800 shadow-theme-xs outline-none transition focus:border-brand-300 focus:ring-3 focus:ring-brand-500/10 dark:border-gray-700 dark:bg-gray-900 dark:text-white/90 dark:focus:border-brand-800"
            >
              <option value="all">Tat ca trang thai</option>
              <option value="running">Running</option>
              <option value="starting">Starting</option>
              <option value="maintenance">Maintenance</option>
              <option value="stopped">Stopped</option>
            </select>
          </div>

          <div className="flex flex-wrap items-center gap-3">
            <Button className="rounded-xl px-5" onClick={() => router.push("/virtual-machines/new")}>
              <PlusIcon className="size-4" />
              <span>Tao VM</span>
            </Button>
            <Button
              variant="outline"
              className="rounded-xl px-5"
              onClick={() => void runBulkAction("start")}
              disabled={selectedVMIDs.length === 0 || busyAction !== ""}
            >
              Khoi dong
            </Button>
            <Button
              variant="outline"
              className="rounded-xl px-5"
              onClick={() => void runBulkAction("stop")}
              disabled={selectedVMIDs.length === 0 || busyAction !== ""}
            >
              Dung
            </Button>
          </div>
        </div>

        <div className="overflow-x-auto">
          <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-800">
            <thead>
              <tr className="text-left text-xs uppercase tracking-[0.18em] text-gray-400 dark:text-gray-500">
                <th className="px-6 py-4 font-medium">
                  <input
                    type="checkbox"
                    checked={allVisibleSelected}
                    onChange={toggleVisibleSelection}
                    aria-label="Select all visible virtual machines"
                    className="h-4 w-4 rounded border-gray-300 text-brand-500 focus:ring-brand-500/20 dark:border-gray-700 dark:bg-gray-900"
                  />
                </th>
                <th className="px-6 py-4 font-medium">Ten VM</th>
                <th className="px-6 py-4 font-medium">Trang thai</th>
                <th className="px-6 py-4 font-medium">He dieu hanh</th>
                <th className="px-6 py-4 font-medium">Zone</th>
                <th className="px-6 py-4 font-medium">vCPU / RAM</th>
                <th className="px-6 py-4 font-medium">IP Address</th>
                <th className="px-6 py-4 font-medium">Uptime</th>
                <th className="px-6 py-4 font-medium text-right">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-200 dark:divide-gray-800">
              {loading ? (
                <tr>
                  <td
                    colSpan={9}
                    className="px-6 py-10 text-center text-sm text-gray-500 dark:text-gray-400"
                  >
                    Loading virtual machines...
                  </td>
                </tr>
              ) : null}
              {!loading && error ? (
                <tr>
                  <td
                    colSpan={9}
                    className="px-6 py-10 text-center text-sm text-rose-600 dark:text-rose-400"
                  >
                    {error}
                  </td>
                </tr>
              ) : null}
              {!loading && !error && filteredItems.length === 0 ? (
                <tr>
                  <td
                    colSpan={9}
                    className="px-6 py-10 text-center text-sm text-gray-500 dark:text-gray-400"
                  >
                    No virtual machines found for the current filters.
                  </td>
                </tr>
              ) : null}
              {filteredItems.map((item) => (
                <tr
                  key={item.id}
                  className="transition hover:bg-gray-50/80 dark:hover:bg-white/[0.02]"
                >
                  <td className="px-6 py-4">
                    <input
                      type="checkbox"
                      checked={selectedVMIDs.includes(item.id)}
                      onChange={() => toggleVMSelection(item.id)}
                      aria-label={`Select ${item.name}`}
                      className="h-4 w-4 rounded border-gray-300 text-brand-500 focus:ring-brand-500/20 dark:border-gray-700 dark:bg-gray-900"
                    />
                  </td>
                  <td className="px-6 py-4">
                    <div className="flex items-center gap-3">
                      <div>
                        <p className="font-medium text-gray-900 dark:text-white">{item.name}</p>
                        {item.package_name || item.package_code ? (
                          <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
                            {[item.package_name, item.package_code].filter(Boolean).join(" · ")}
                          </p>
                        ) : null}
                        <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
                          {item.id}
                        </p>
                      </div>
                    </div>
                  </td>
                  <td className="px-6 py-4">
                    <span
                      className={`inline-flex items-center rounded-full px-3 py-1 text-xs font-semibold ${statusClasses(normalizeStatus(item))}`}
                    >
                      <span
                        className={`mr-2 h-2 w-2 rounded-full ${statusDotClasses(normalizeStatus(item))}`}
                      />
                      {normalizeStatus(item)}
                    </span>
                  </td>
                  <td className="px-6 py-4 text-sm text-gray-700 dark:text-gray-300">{item.image || "Unknown image"}</td>
                  <td className="px-6 py-4 text-sm text-gray-700 dark:text-gray-300">{item.zone || "Unassigned"}</td>
                  <td className="px-6 py-4 text-sm text-gray-700 dark:text-gray-300">
                    {item.vcpu} vCPU / {item.ram_gb} GB
                  </td>
                  <td className="px-6 py-4 text-sm text-gray-700 dark:text-gray-300">
                    {item.primary_ip || "Pending"}
                  </td>
                  <td className="px-6 py-4">
                    <div className="inline-flex items-center gap-2 text-sm text-gray-500 dark:text-gray-400">
                      <TimeIcon className="size-4" />
                      <span>{relativeTime(item.last_seen_at)}</span>
                    </div>
                  </td>
                  <td className="px-6 py-4">
                    <div className="flex items-center justify-end gap-2">
                      <button
                        type="button"
                        onClick={() => router.push(`/virtual-machines/detail?id=${encodeURIComponent(item.id)}`)}
                        className="inline-flex h-9 w-9 items-center justify-center rounded-xl border border-gray-200 text-gray-500 transition hover:border-brand-300 hover:text-brand-600 dark:border-gray-800 dark:text-gray-400 dark:hover:border-brand-500/40 dark:hover:text-brand-400"
                        aria-label={`View ${item.name}`}
                      >
                        <svg
                          width="18"
                          height="18"
                          viewBox="0 0 18 18"
                          fill="none"
                          xmlns="http://www.w3.org/2000/svg"
                        >
                          <path
                            d="M1.5 9C2.7 5.7 5.475 3.75 9 3.75C12.525 3.75 15.3 5.7 16.5 9C15.3 12.3 12.525 14.25 9 14.25C5.475 14.25 2.7 12.3 1.5 9Z"
                            stroke="currentColor"
                            strokeWidth="1.5"
                            strokeLinecap="round"
                            strokeLinejoin="round"
                          />
                          <circle cx="9" cy="9" r="2.25" stroke="currentColor" strokeWidth="1.5" />
                        </svg>
                      </button>
                      <button
                        type="button"
                        className="inline-flex h-9 w-9 items-center justify-center rounded-xl border border-gray-200 text-gray-500 transition hover:border-brand-300 hover:text-brand-600 dark:border-gray-800 dark:text-gray-400 dark:hover:border-brand-500/40 dark:hover:text-brand-400"
                        aria-label={`Open actions for ${item.name}`}
                      >
                        <MoreDotIcon className="size-5" />
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>

        <div className="flex flex-col gap-4 border-t border-gray-200 px-6 py-4 text-sm text-gray-500 dark:border-gray-800 dark:text-gray-400 md:flex-row md:items-center md:justify-between">
          <p>
            Hien thi <span className="font-medium text-gray-900 dark:text-white">{filteredItems.length}</span>{" "}
            trong tong so <span className="font-medium text-gray-900 dark:text-white">{items.length}</span>{" "}
            virtual machines.
          </p>
          <div className="flex items-center gap-2">
            <button
              type="button"
              className="rounded-lg border border-gray-200 px-3 py-2 transition hover:border-brand-300 hover:text-brand-600 dark:border-gray-800 dark:hover:border-brand-500/40 dark:hover:text-brand-400"
            >
              Truoc
            </button>
            <button
              type="button"
              className="rounded-lg bg-brand-500 px-3 py-2 font-medium text-white"
            >
              1
            </button>
            <button
              type="button"
              className="rounded-lg border border-gray-200 px-3 py-2 transition hover:border-brand-300 hover:text-brand-600 dark:border-gray-800 dark:hover:border-brand-500/40 dark:hover:text-brand-400"
            >
              2
            </button>
            <button
              type="button"
              className="rounded-lg border border-gray-200 px-3 py-2 transition hover:border-brand-300 hover:text-brand-600 dark:border-gray-800 dark:hover:border-brand-500/40 dark:hover:text-brand-400"
            >
              Sau
            </button>
          </div>
        </div>
      </section>
    </div>
  );
}
