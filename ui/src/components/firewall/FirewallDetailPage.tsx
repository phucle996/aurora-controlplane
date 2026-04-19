"use client";

import { useEffect, useMemo, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import PageBreadcrumb from "@/components/common/PageBreadCrumb";
import ComponentCard from "@/components/common/ComponentCard";
import Input from "@/components/form/input/InputField";
import Button from "@/components/ui/button/Button";
import { useToast } from "@/components/ui/toast/ToastProvider";
import {
  getHypervisorFirewall,
  normalizeFirewall,
  updateHypervisorFirewall,
  type HypervisorFirewallRule,
} from "@/components/hypervisor/api";

function policyClasses(policy: "allow" | "deny") {
  return policy === "allow"
    ? "bg-brand-500/10 text-brand-600 dark:text-brand-400"
    : "bg-amber-500/12 text-amber-600 dark:text-amber-400";
}

function actionClasses(action: "allow" | "deny") {
  return action === "allow"
    ? "bg-brand-500/10 text-brand-600 dark:text-brand-400"
    : "bg-amber-500/12 text-amber-600 dark:text-amber-400";
}

function tabClasses(active: boolean) {
  return `rounded-xl px-4 py-2.5 text-sm font-medium transition ${
    active
      ? "bg-white text-gray-900 shadow-theme-xs dark:bg-gray-800 dark:text-white"
      : "text-gray-500 hover:text-gray-900 dark:text-gray-400 dark:hover:text-white"
  }`;
}

export default function FirewallDetailPage() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const { pushToast } = useToast();
  const firewallID = searchParams.get("id")?.trim() ?? "";
  const [direction, setDirection] = useState<"inbound" | "outbound">("inbound");
  const [search, setSearch] = useState("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [firewall, setFirewall] = useState<ReturnType<typeof normalizeFirewall> | null>(null);
  const [rules, setRules] = useState<HypervisorFirewallRule[]>([]);
  const [draftInboundPolicy, setDraftInboundPolicy] = useState<"allow" | "deny">("deny");
  const [draftOutboundPolicy, setDraftOutboundPolicy] = useState<"allow" | "deny">("allow");
  const [savingPolicies, setSavingPolicies] = useState(false);
  const [refreshing, setRefreshing] = useState(false);

  async function load({ silent = false }: { silent?: boolean } = {}) {
    if (!firewallID) {
      setError("Missing firewall id.");
      setLoading(false);
      return;
    }

    try {
      if (silent) {
        setRefreshing(true);
      } else {
        setLoading(true);
      }
      setError("");
      const payload = await getHypervisorFirewall(firewallID);
      const normalizedFirewall = normalizeFirewall(payload.firewall);
      setFirewall(normalizedFirewall);
      setDraftInboundPolicy(normalizedFirewall.defaultInboundPolicy);
      setDraftOutboundPolicy(normalizedFirewall.defaultOutboundPolicy);
      setRules(payload.rules);
    } catch (err) {
      setFirewall(null);
      setRules([]);
      setError(err instanceof Error ? err.message : "Failed to load firewall.");
    } finally {
      setLoading(false);
      setRefreshing(false);
    }
  }

  useEffect(() => {
    void load();
  }, [firewallID]);

  const filteredRules = useMemo(
    () =>
      rules.filter((rule) => {
        if (rule.direction !== direction) return false;
        const query = search.trim().toLowerCase();
        if (query === "") return true;
        return (
          rule.name.toLowerCase().includes(query) ||
          rule.protocol.toLowerCase().includes(query) ||
          rule.port_range.toLowerCase().includes(query) ||
          rule.sources.some((item) => item.toLowerCase().includes(query)) ||
          rule.destinations.some((item) => item.toLowerCase().includes(query))
        );
      }),
    [direction, rules, search],
  );

  async function savePolicies() {
    if (!firewall) return;
    try {
      setSavingPolicies(true);
      const updated = normalizeFirewall(
        await updateHypervisorFirewall(firewall.id, {
          default_inbound_policy: draftInboundPolicy,
          default_outbound_policy: draftOutboundPolicy,
        }),
      );
      setFirewall(updated);
      setDraftInboundPolicy(updated.defaultInboundPolicy);
      setDraftOutboundPolicy(updated.defaultOutboundPolicy);
      pushToast({ kind: "success", message: "Firewall default policies updated." });
    } catch (err) {
      pushToast({
        kind: "error",
        message: err instanceof Error ? err.message : "Failed to update firewall policies.",
      });
    } finally {
      setSavingPolicies(false);
    }
  }

  if (loading) {
    return (
      <div className="space-y-6">
        <PageBreadcrumb pageTitle="Firewall Detail" />
        <section className="rounded-3xl border border-gray-200 bg-white p-8 text-center dark:border-gray-800 dark:bg-white/[0.03]">
          <p className="text-sm text-gray-500 dark:text-gray-400">Loading firewall...</p>
        </section>
      </div>
    );
  }

  if (!firewall) {
    return (
      <div className="space-y-6">
        <PageBreadcrumb pageTitle="Firewall Detail" />
        <section className="rounded-3xl border border-gray-200 bg-white p-8 text-center dark:border-gray-800 dark:bg-white/[0.03]">
          <h1 className="text-2xl font-semibold text-gray-900 dark:text-white">Firewall not found</h1>
          <p className="mt-3 text-sm text-gray-500 dark:text-gray-400">
            {error || "Firewall nay co the da bi xoa hoac lien ket detail khong con hop le."}
          </p>
          <div className="mt-6">
            <Button className="rounded-xl px-5" onClick={() => router.push("/firewall")}>
              Back to Firewalls
            </Button>
          </div>
        </section>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <PageBreadcrumb pageTitle="Firewall Detail" />

      <section className="rounded-3xl border border-gray-200 bg-white p-6 dark:border-gray-800 dark:bg-white/[0.03]">
        <div className="flex flex-col gap-4 xl:flex-row xl:items-start xl:justify-between">
          <div>
            <h1 className="text-3xl font-semibold text-gray-900 dark:text-white">{firewall.name}</h1>
            <p className="mt-2 text-sm text-gray-500 dark:text-gray-400">
              {firewall.target || "No target"} · {rules.length} total rules
            </p>
          </div>

          <div className="flex flex-wrap gap-3">
            <Button variant="outline" className="rounded-xl px-5" onClick={() => void load({ silent: true })} disabled={refreshing}>
              {refreshing ? "Refreshing..." : "Refresh"}
            </Button>
            <Button variant="outline" className="rounded-xl px-5" onClick={() => router.push("/firewall")}>
              Back to List
            </Button>
            <Button className="rounded-xl px-5" onClick={() => router.push("/firewall/new")}>
              Add Firewall
            </Button>
          </div>
        </div>
      </section>

      <section className="grid gap-6 xl:grid-cols-[minmax(0,1.6fr)_360px]">
        <div className="space-y-6">
          <section className="rounded-2xl border border-gray-200 bg-white dark:border-gray-800 dark:bg-white/[0.03]">
            <div className="flex flex-wrap items-center justify-between gap-4 border-b border-gray-200 px-6 py-5 dark:border-gray-800">
              <div>
                <h2 className="text-base font-medium text-gray-900 dark:text-white">Firewall Rules</h2>
                <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
                  Chi tiet rule inbound va outbound cho firewall nay.
                </p>
              </div>

              <div className="flex w-full flex-col gap-3 xl:w-auto xl:min-w-[420px]">
                <Input
                  value={search}
                  onChange={(event) => setSearch(event.target.value)}
                  placeholder="Search rules by name, protocol, port, source, or destination"
                />
                <div className="inline-flex rounded-2xl border border-gray-200 bg-gray-50 p-1 dark:border-gray-800 dark:bg-gray-900/60">
                  <button type="button" onClick={() => setDirection("inbound")} className={tabClasses(direction === "inbound")}>
                    Inbound
                  </button>
                  <button type="button" onClick={() => setDirection("outbound")} className={tabClasses(direction === "outbound")}>
                    Outbound
                  </button>
                </div>
              </div>
            </div>

            <div className="overflow-hidden rounded-b-2xl">
              <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-800">
                <thead className="bg-gray-50 dark:bg-gray-900/60">
                  <tr className="text-left text-xs uppercase tracking-[0.18em] text-gray-400 dark:text-gray-500">
                    <th className="px-4 py-3 font-medium">Rule</th>
                    <th className="px-4 py-3 font-medium">Protocol / Port</th>
                    <th className="px-4 py-3 font-medium">{direction === "inbound" ? "Sources" : "Destinations"}</th>
                    <th className="px-4 py-3 font-medium">{direction === "inbound" ? "Destinations" : "Sources"}</th>
                    <th className="px-4 py-3 font-medium">Action</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-200 dark:divide-gray-800">
                  {filteredRules.length === 0 ? (
                    <tr>
                      <td colSpan={5} className="px-4 py-10 text-center text-sm text-gray-500 dark:text-gray-400">
                        {search.trim() === ""
                          ? `No ${direction} rules are stored for this firewall yet.`
                          : `No ${direction} rules matched the current search.`}
                      </td>
                    </tr>
                  ) : null}
                  {filteredRules.map((rule) => (
                    <tr key={rule.id} className="bg-white dark:bg-transparent">
                      <td className="px-4 py-4">
                        <p className="font-medium text-gray-900 dark:text-white">{rule.name}</p>
                        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{rule.id}</p>
                      </td>
                      <td className="px-4 py-4 text-sm text-gray-700 dark:text-gray-300">
                        <div className="space-y-1">
                          <p className="font-medium uppercase">{rule.protocol}</p>
                          <p>{rule.port_range}</p>
                        </div>
                      </td>
                      <td className="px-4 py-4">
                        <TagList items={rule.sources} />
                      </td>
                      <td className="px-4 py-4">
                        <TagList items={rule.destinations} />
                      </td>
                      <td className="px-4 py-4">
                        <span className={`inline-flex rounded-full px-3 py-1 text-xs font-semibold uppercase ${actionClasses(rule.action)}`}>
                          {rule.action}
                        </span>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </section>
        </div>

        <div className="space-y-6">
          <ComponentCard title="Default Policy" desc="Edit the fallback action applied when no explicit firewall rule matches.">
            <div className="space-y-5">
              <PolicyPicker label="Inbound Default Policy" value={draftInboundPolicy} onChange={setDraftInboundPolicy} />
              <PolicyPicker label="Outbound Default Policy" value={draftOutboundPolicy} onChange={setDraftOutboundPolicy} />
              <Button
                className="w-full rounded-xl"
                onClick={() => void savePolicies()}
                disabled={
                  savingPolicies ||
                  (firewall.defaultInboundPolicy === draftInboundPolicy &&
                    firewall.defaultOutboundPolicy === draftOutboundPolicy)
                }
              >
                {savingPolicies ? "Saving..." : "Save Default Policy"}
              </Button>
            </div>
          </ComponentCard>

          <ComponentCard title="Firewall Snapshot" desc="Tong quan nhanh de review policy hien tai.">
            <div className="space-y-4 text-sm">
              <MetaRow label="Target" value={firewall.target || "No target"} />
              <MetaRow label="Inbound Rules" value={`${rules.filter((rule) => rule.direction === "inbound").length}`} />
              <MetaRow label="Outbound Rules" value={`${rules.filter((rule) => rule.direction === "outbound").length}`} />
              <MetaRow label="State" value={firewall.status} />
            </div>
          </ComponentCard>
        </div>
      </section>
    </div>
  );
}

function PolicyPicker({
  label,
  value,
  onChange,
}: {
  label: string;
  value: "allow" | "deny";
  onChange: (value: "allow" | "deny") => void;
}) {
  return (
    <div className="space-y-3">
      <p className="text-sm font-medium text-gray-900 dark:text-white">{label}</p>
      <div className="inline-flex rounded-2xl border border-gray-200 bg-gray-50 p-1 dark:border-gray-800 dark:bg-gray-900/60">
        <button type="button" onClick={() => onChange("allow")} className={tabClasses(value === "allow")}>
          Allow
        </button>
        <button type="button" onClick={() => onChange("deny")} className={tabClasses(value === "deny")}>
          Deny
        </button>
      </div>
      <span className={`inline-flex rounded-full px-3 py-1 text-xs font-semibold uppercase ${policyClasses(value)}`}>
        {value}
      </span>
    </div>
  );
}

function TagList({ items }: { items: string[] }) {
  if (items.length === 0) {
    return <span className="text-sm text-gray-400 dark:text-gray-500">None</span>;
  }

  return (
    <div className="flex flex-wrap gap-2">
      {items.map((item) => (
        <span
          key={item}
          className="inline-flex rounded-full border border-gray-200 bg-gray-50 px-2.5 py-1 text-xs text-gray-700 dark:border-gray-800 dark:bg-gray-900/60 dark:text-gray-300"
        >
          {item}
        </span>
      ))}
    </div>
  );
}

function MetaRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-center justify-between gap-3 rounded-2xl border border-gray-200 bg-gray-50 px-4 py-4 dark:border-gray-800 dark:bg-gray-900/60">
      <span className="text-gray-500 dark:text-gray-400">{label}</span>
      <span className="font-medium text-gray-900 dark:text-white">{value}</span>
    </div>
  );
}
