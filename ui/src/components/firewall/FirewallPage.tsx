"use client";

import { useEffect, useMemo, useState } from "react";
import { useRouter } from "next/navigation";
import PageBreadcrumb from "@/components/common/PageBreadCrumb";
import Input from "@/components/form/input/InputField";
import Button from "@/components/ui/button/Button";
import { PlusIcon } from "@/icons";
import { listHypervisorFirewalls, normalizeFirewall } from "@/components/hypervisor/api";

function statusClasses(status: "active" | "disabled") {
  return status === "active"
    ? "bg-emerald-500/12 text-emerald-600 dark:text-emerald-400"
    : "bg-rose-500/12 text-rose-600 dark:text-rose-400";
}

function statusDotClasses(status: "active" | "disabled") {
  return status === "active" ? "bg-emerald-500" : "bg-rose-500";
}

function policyClasses(policy: "allow" | "deny") {
  return policy === "allow"
    ? "bg-brand-500/10 text-brand-600 dark:text-brand-400"
    : "bg-amber-500/12 text-amber-600 dark:text-amber-400";
}

export default function FirewallPage() {
  const router = useRouter();
  const [search, setSearch] = useState("");
  const [items, setItems] = useState<ReturnType<typeof normalizeFirewall>[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  useEffect(() => {
    async function load() {
      try {
        setLoading(true);
        setError("");
        const response = await listHypervisorFirewalls();
        setItems(response.map(normalizeFirewall));
      } catch (err) {
        setError(err instanceof Error ? err.message : "Failed to load firewalls.");
      } finally {
        setLoading(false);
      }
    }

    void load();
  }, []);

  const filteredItems = useMemo(() => {
    const query = search.trim().toLowerCase();
    if (query === "") return items;

    return items.filter((item) => {
      return (
        item.name.toLowerCase().includes(query) ||
        item.target.toLowerCase().includes(query) ||
        item.id.toLowerCase().includes(query)
      );
    });
  }, [items, search]);

  return (
    <div className="space-y-6">
      <PageBreadcrumb pageTitle="Firewall" />

      <section className="rounded-3xl border border-gray-200 bg-white dark:border-gray-800 dark:bg-white/[0.03]">
        <div className="flex flex-col gap-4 border-b border-gray-200 px-6 py-5 dark:border-gray-800 xl:flex-row xl:items-center xl:justify-between">
          <div>
            <h1 className="text-2xl font-semibold text-gray-900 dark:text-white">Firewalls</h1>
            <p className="mt-2 max-w-2xl text-sm leading-6 text-gray-500 dark:text-gray-400">
              Quan ly cac firewall policy cho workload computing, sau do di sau vao rule inbound va outbound trong trang detail.
            </p>
          </div>

          <Button className="rounded-xl px-5" onClick={() => router.push("/firewall/new")}>
            <PlusIcon className="size-4" />
            <span>Add Firewall</span>
          </Button>
        </div>

        <div className="space-y-4 px-6 py-5">
          <div className="relative max-w-md">
            <span className="pointer-events-none absolute left-4 top-1/2 -translate-y-1/2 text-gray-400">
              <svg width="20" height="20" viewBox="0 0 20 20" fill="none" aria-hidden="true">
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
              placeholder="Tim firewall theo ten, target, hoac id..."
              className="pl-11"
            />
          </div>

          <div className="overflow-hidden rounded-2xl border border-gray-200 dark:border-gray-800">
            <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-800">
              <thead className="bg-gray-50 dark:bg-gray-900/60">
                <tr className="text-left text-xs uppercase tracking-[0.18em] text-gray-400 dark:text-gray-500">
                  <th className="px-4 py-3 font-medium">Firewall</th>
                  <th className="px-4 py-3 font-medium">Target</th>
                  <th className="px-4 py-3 font-medium">Default Inbound</th>
                  <th className="px-4 py-3 font-medium">Default Outbound</th>
                  <th className="px-4 py-3 font-medium">Rules</th>
                  <th className="px-4 py-3 font-medium">State</th>
                  <th className="px-4 py-3 font-medium text-right">Action</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-200 dark:divide-gray-800">
                {loading ? (
                  <tr>
                    <td colSpan={7} className="px-4 py-10 text-center text-sm text-gray-500 dark:text-gray-400">
                      Loading firewalls...
                    </td>
                  </tr>
                ) : null}
                {!loading && error ? (
                  <tr>
                    <td colSpan={7} className="px-4 py-10 text-center text-sm text-rose-600 dark:text-rose-400">
                      {error}
                    </td>
                  </tr>
                ) : null}
                {!loading && !error && filteredItems.length === 0 ? (
                  <tr>
                    <td colSpan={7} className="px-4 py-10 text-center text-sm text-gray-500 dark:text-gray-400">
                      No firewalls found for the current filters.
                    </td>
                  </tr>
                ) : null}
                {filteredItems.map((item) => (
                  <tr key={item.id} className="bg-white transition hover:bg-gray-50 dark:bg-transparent dark:hover:bg-white/[0.02]">
                    <td className="px-4 py-4">
                      <p className="font-medium text-gray-900 dark:text-white">{item.name}</p>
                      <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{item.id}</p>
                    </td>
                    <td className="px-4 py-4 text-sm text-gray-700 dark:text-gray-300">{item.target || "No target"}</td>
                    <td className="px-4 py-4">
                      <span className={`inline-flex rounded-full px-3 py-1 text-xs font-semibold uppercase ${policyClasses(item.defaultInboundPolicy)}`}>
                        {item.defaultInboundPolicy}
                      </span>
                    </td>
                    <td className="px-4 py-4">
                      <span className={`inline-flex rounded-full px-3 py-1 text-xs font-semibold uppercase ${policyClasses(item.defaultOutboundPolicy)}`}>
                        {item.defaultOutboundPolicy}
                      </span>
                    </td>
                    <td className="px-4 py-4 text-sm text-gray-700 dark:text-gray-300">See detail</td>
                    <td className="px-4 py-4">
                      <span className={`inline-flex items-center rounded-full px-3 py-1 text-xs font-semibold ${statusClasses(item.status)}`}>
                        <span className={`mr-2 h-2 w-2 rounded-full ${statusDotClasses(item.status)}`} />
                        {item.status}
                      </span>
                    </td>
                    <td className="px-4 py-4 text-right">
                      <Button
                        variant="outline"
                        className="rounded-xl px-4"
                        onClick={() => router.push(`/firewall/detail?id=${encodeURIComponent(item.id)}`)}
                      >
                        View
                      </Button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      </section>
    </div>
  );
}
