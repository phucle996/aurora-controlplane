"use client";

import type { ChangeEvent, ReactNode } from "react";
import { useMemo, useState } from "react";
import { useRouter } from "next/navigation";
import PageBreadcrumb from "@/components/common/PageBreadCrumb";
import ComponentCard from "@/components/common/ComponentCard";
import Input from "@/components/form/input/InputField";
import Button from "@/components/ui/button/Button";
import { useToast } from "@/components/ui/toast/ToastProvider";
import {
  createHypervisorFirewall,
  createHypervisorFirewallRule,
  normalizeFirewall,
} from "@/components/hypervisor/api";

type RuleForm = {
  id: string;
  name: string;
  protocol: "tcp" | "udp" | "icmp" | "all";
  portRange: string;
  action: "allow" | "deny";
  sources: string[];
  destinations: string[];
  sourceInput: string;
  destinationInput: string;
};

function createRule(id: string): RuleForm {
  return {
    id,
    name: "",
    protocol: "tcp",
    portRange: "",
    action: "allow",
    sources: [],
    destinations: [],
    sourceInput: "",
    destinationInput: "",
  };
}

export default function NewFirewallPage() {
  const router = useRouter();
  const { pushToast } = useToast();
  const [name, setName] = useState("");
  const [target, setTarget] = useState("");
  const [defaultInboundPolicy, setDefaultInboundPolicy] = useState<"allow" | "deny">("deny");
  const [defaultOutboundPolicy, setDefaultOutboundPolicy] = useState<"allow" | "deny">("allow");
  const [inboundRules, setInboundRules] = useState<RuleForm[]>([createRule("inbound-1")]);
  const [outboundRules, setOutboundRules] = useState<RuleForm[]>([createRule("outbound-1")]);
  const [submitting, setSubmitting] = useState(false);

  const totalRules = inboundRules.length + outboundRules.length;

  const summary = useMemo(() => {
    return {
      inboundSources: inboundRules.reduce((count, rule) => count + rule.sources.length, 0),
      outboundDestinations: outboundRules.reduce((count, rule) => count + rule.destinations.length, 0),
    };
  }, [inboundRules, outboundRules]);

  function updateRule(
    direction: "inbound" | "outbound",
    ruleID: string,
    updater: (rule: RuleForm) => RuleForm,
  ) {
    const setter = direction === "inbound" ? setInboundRules : setOutboundRules;
    setter((current) => current.map((rule) => (rule.id === ruleID ? updater(rule) : rule)));
  }

  function addRule(direction: "inbound" | "outbound") {
    const setter = direction === "inbound" ? setInboundRules : setOutboundRules;
    const nextID = `${direction}-${Date.now()}-${Math.random().toString(16).slice(2, 6)}`;
    setter((current) => [...current, createRule(nextID)]);
  }

  function removeRule(direction: "inbound" | "outbound", ruleID: string) {
    const setter = direction === "inbound" ? setInboundRules : setOutboundRules;
    setter((current) => current.filter((rule) => rule.id !== ruleID));
  }

  function addToken(direction: "inbound" | "outbound", ruleID: string, type: "source" | "destination") {
    updateRule(direction, ruleID, (rule) => {
      const raw = type === "source" ? rule.sourceInput : rule.destinationInput;
      const values = raw
        .split(",")
        .map((item) => item.trim())
        .filter(Boolean);
      if (values.length === 0) return rule;

      if (type === "source") {
        return {
          ...rule,
          sources: Array.from(new Set([...rule.sources, ...values])),
          sourceInput: "",
        };
      }

      return {
        ...rule,
        destinations: Array.from(new Set([...rule.destinations, ...values])),
        destinationInput: "",
      };
    });
  }

  function removeToken(direction: "inbound" | "outbound", ruleID: string, type: "source" | "destination", value: string) {
    updateRule(direction, ruleID, (rule) => {
      if (type === "source") {
        return {
          ...rule,
          sources: rule.sources.filter((item) => item !== value),
        };
      }
      return {
        ...rule,
        destinations: rule.destinations.filter((item) => item !== value),
      };
    });
  }

  async function handleCreate() {
    if (name.trim() === "") {
      pushToast({ kind: "error", message: "Firewall name is required." });
      return;
    }

    const allRules = [
      ...inboundRules.map((rule) => ({ ...rule, direction: "inbound" as const })),
      ...outboundRules.map((rule) => ({ ...rule, direction: "outbound" as const })),
    ].filter((rule) => {
      return (
        rule.name.trim() !== "" ||
        rule.portRange.trim() !== "" ||
        rule.sources.length > 0 ||
        rule.destinations.length > 0
      );
    });

    const invalidRule = allRules.find((rule) => rule.name.trim() === "");
    if (invalidRule) {
      pushToast({ kind: "error", message: "Every firewall rule must have a name before saving." });
      return;
    }

    try {
      setSubmitting(true);
      const created = normalizeFirewall(
        await createHypervisorFirewall({
          name: name.trim(),
          target: target.trim(),
          default_inbound_policy: defaultInboundPolicy,
          default_outbound_policy: defaultOutboundPolicy,
        }),
      );

      if (allRules.length > 0) {
        await Promise.all(
          allRules.map((rule) =>
            createHypervisorFirewallRule(created.id, {
              direction: rule.direction,
              name: rule.name.trim(),
              protocol: rule.protocol,
              port_range: rule.portRange.trim() || "any",
              action: rule.action,
              sources: rule.sources,
              destinations: rule.destinations,
            }),
          ),
        );
      }

      pushToast({
        kind: "success",
        message:
          allRules.length > 0
            ? `Firewall created with ${allRules.length} persisted rule(s).`
            : "Firewall created successfully.",
      });
      router.push(`/firewall/detail?id=${encodeURIComponent(created.id)}`);
    } catch (err) {
      pushToast({
        kind: "error",
        message: err instanceof Error ? err.message : "Failed to create firewall.",
      });
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <div className="space-y-6">
      <PageBreadcrumb pageTitle="Add Firewall" />

      <section className="grid gap-6 xl:grid-cols-[minmax(0,1.7fr)_360px]">
        <div className="space-y-6">
          <ComponentCard title="Firewall Information" desc="Thong tin co ban cho bo rule moi.">
            <div className="grid gap-5 md:grid-cols-2">
              <FieldBlock label="Firewall Name">
                <Input value={name} onChange={(event) => setName(event.target.value)} placeholder="vd: Public Edge Firewall" />
              </FieldBlock>
              <FieldBlock label="Target">
                <Input value={target} onChange={(event) => setTarget(event.target.value)} placeholder="vd: public-edge" />
              </FieldBlock>
            </div>
          </ComponentCard>

          <ComponentCard title="Default Policy" desc="Chinh sach mac dinh khi traffic khong match rule nao.">
            <div className="grid gap-5 md:grid-cols-2">
              <PolicyPicker label="Inbound Default Policy" value={defaultInboundPolicy} onChange={setDefaultInboundPolicy} />
              <PolicyPicker label="Outbound Default Policy" value={defaultOutboundPolicy} onChange={setDefaultOutboundPolicy} />
            </div>
          </ComponentCard>

          <RuleSection
            title="Inbound Rules"
            desc="Moi rule co the gan nhieu source va nhieu destination."
            rules={inboundRules}
            direction="inbound"
            onAddRule={() => addRule("inbound")}
            onRemoveRule={(ruleID) => removeRule("inbound", ruleID)}
            onUpdateRule={(ruleID, updater) => updateRule("inbound", ruleID, updater)}
            onAddToken={(ruleID, type) => addToken("inbound", ruleID, type)}
            onRemoveToken={(ruleID, type, value) => removeToken("inbound", ruleID, type, value)}
          />

          <RuleSection
            title="Outbound Rules"
            desc="Moi rule co the gan nhieu source va nhieu destination."
            rules={outboundRules}
            direction="outbound"
            onAddRule={() => addRule("outbound")}
            onRemoveRule={(ruleID) => removeRule("outbound", ruleID)}
            onUpdateRule={(ruleID, updater) => updateRule("outbound", ruleID, updater)}
            onAddToken={(ruleID, type) => addToken("outbound", ruleID, type)}
            onRemoveToken={(ruleID, type, value) => removeToken("outbound", ruleID, type, value)}
          />
        </div>

        <div className="space-y-6 xl:sticky xl:top-24 xl:self-start">
          <ComponentCard title="Firewall Preview" desc="Tom tat nhanh cau hinh truoc khi tao.">
            <div className="space-y-4">
              <SummaryRow label="Name" value={name.trim() || "Unnamed firewall"} />
              <SummaryRow label="Target" value={target.trim() || "No target yet"} />
              <SummaryRow label="Inbound Default" value={defaultInboundPolicy} />
              <SummaryRow label="Outbound Default" value={defaultOutboundPolicy} />
              <SummaryRow label="Total Rules" value={`${totalRules}`} />
              <SummaryRow label="Inbound Sources" value={`${summary.inboundSources}`} />
              <SummaryRow label="Outbound Destinations" value={`${summary.outboundDestinations}`} />
            </div>

            <div className="mt-6 flex flex-col gap-3">
              <Button className="w-full rounded-xl" onClick={() => void handleCreate()} disabled={submitting}>
                {submitting ? "Creating..." : "Create Firewall"}
              </Button>
              <Button variant="outline" className="w-full rounded-xl" onClick={() => router.push("/firewall")}>
                Back to Firewalls
              </Button>
            </div>
          </ComponentCard>
        </div>
      </section>
    </div>
  );
}

function FieldBlock({ label, children }: { label: string; children: ReactNode }) {
  return (
    <label className="block space-y-2">
      <span className="text-sm font-medium text-gray-700 dark:text-gray-300">{label}</span>
      {children}
    </label>
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
        <button type="button" onClick={() => onChange("allow")} className={policyTabClasses(value === "allow")}>
          Allow
        </button>
        <button type="button" onClick={() => onChange("deny")} className={policyTabClasses(value === "deny")}>
          Deny
        </button>
      </div>
    </div>
  );
}

function RuleSection({
  title,
  desc,
  rules,
  direction,
  onAddRule,
  onRemoveRule,
  onUpdateRule,
  onAddToken,
  onRemoveToken,
}: {
  title: string;
  desc: string;
  rules: RuleForm[];
  direction: "inbound" | "outbound";
  onAddRule: () => void;
  onRemoveRule: (ruleID: string) => void;
  onUpdateRule: (ruleID: string, updater: (rule: RuleForm) => RuleForm) => void;
  onAddToken: (ruleID: string, type: "source" | "destination") => void;
  onRemoveToken: (ruleID: string, type: "source" | "destination", value: string) => void;
}) {
  return (
    <ComponentCard
      title={title}
      desc={desc}
      headerAction={
        <Button variant="outline" className="rounded-xl px-4" onClick={onAddRule}>
          Add Rule
        </Button>
      }
    >
      <div className="space-y-5">
        {rules.map((rule, index) => (
          <div key={rule.id} className="rounded-2xl border border-gray-200 bg-gray-50 p-5 dark:border-gray-800 dark:bg-gray-900/60">
            <div className="mb-5 flex items-center justify-between gap-3">
              <div>
                <p className="font-medium text-gray-900 dark:text-white">Rule {index + 1}</p>
                <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
                  {direction === "inbound" ? "Inbound policy rule" : "Outbound policy rule"}
                </p>
              </div>
              {rules.length > 1 ? (
                <Button variant="outline" className="rounded-xl px-4" onClick={() => onRemoveRule(rule.id)}>
                  Remove
                </Button>
              ) : null}
            </div>

            <div className="grid gap-5 md:grid-cols-2">
              <FieldBlock label="Rule Name">
                <Input
                  value={rule.name}
                  onChange={(event) => onUpdateRule(rule.id, (current) => ({ ...current, name: event.target.value }))}
                  placeholder="vd: Allow SSH Admin"
                />
              </FieldBlock>
              <FieldBlock label="Port Range">
                <Input
                  value={rule.portRange}
                  onChange={(event) => onUpdateRule(rule.id, (current) => ({ ...current, portRange: event.target.value }))}
                  placeholder="22 or 443 or 8080-8090"
                />
              </FieldBlock>
              <FieldBlock label="Protocol">
                <select
                  value={rule.protocol}
                  onChange={(event) =>
                    onUpdateRule(rule.id, (current) => ({
                      ...current,
                      protocol: event.target.value as RuleForm["protocol"],
                    }))
                  }
                  className="h-11 w-full rounded-lg border border-gray-300 bg-transparent px-4 text-sm text-gray-800 shadow-theme-xs outline-none transition focus:border-brand-300 focus:ring-3 focus:ring-brand-500/10 dark:border-gray-700 dark:bg-gray-900 dark:text-white/90 dark:focus:border-brand-800"
                >
                  <option value="tcp">TCP</option>
                  <option value="udp">UDP</option>
                  <option value="icmp">ICMP</option>
                  <option value="all">ALL</option>
                </select>
              </FieldBlock>
              <FieldBlock label="Action">
                <select
                  value={rule.action}
                  onChange={(event) =>
                    onUpdateRule(rule.id, (current) => ({
                      ...current,
                      action: event.target.value as RuleForm["action"],
                    }))
                  }
                  className="h-11 w-full rounded-lg border border-gray-300 bg-transparent px-4 text-sm text-gray-800 shadow-theme-xs outline-none transition focus:border-brand-300 focus:ring-3 focus:ring-brand-500/10 dark:border-gray-700 dark:bg-gray-900 dark:text-white/90 dark:focus:border-brand-800"
                >
                  <option value="allow">Allow</option>
                  <option value="deny">Deny</option>
                </select>
              </FieldBlock>
            </div>

            <div className="mt-5 grid gap-5 md:grid-cols-2">
              <TokenEditor
                label="Sources"
                value={rule.sourceInput}
                items={rule.sources}
                placeholder="Nhap 1 hoac nhieu source, cach nhau boi dau phay"
                onChange={(event) =>
                  onUpdateRule(rule.id, (current) => ({ ...current, sourceInput: event.target.value }))
                }
                onAdd={() => onAddToken(rule.id, "source")}
                onRemove={(value) => onRemoveToken(rule.id, "source", value)}
              />
              <TokenEditor
                label="Destinations"
                value={rule.destinationInput}
                items={rule.destinations}
                placeholder="Nhap 1 hoac nhieu destination, cach nhau boi dau phay"
                onChange={(event) =>
                  onUpdateRule(rule.id, (current) => ({ ...current, destinationInput: event.target.value }))
                }
                onAdd={() => onAddToken(rule.id, "destination")}
                onRemove={(value) => onRemoveToken(rule.id, "destination", value)}
              />
            </div>
          </div>
        ))}
      </div>
    </ComponentCard>
  );
}

function TokenEditor({
  label,
  value,
  items,
  placeholder,
  onChange,
  onAdd,
  onRemove,
}: {
  label: string;
  value: string;
  items: string[];
  placeholder: string;
  onChange: (event: ChangeEvent<HTMLInputElement>) => void;
  onAdd: () => void;
  onRemove: (value: string) => void;
}) {
  return (
    <div className="space-y-3">
      <p className="text-sm font-medium text-gray-900 dark:text-white">{label}</p>
      <div className="flex gap-3">
        <Input value={value} onChange={onChange} placeholder={placeholder} />
        <Button variant="outline" className="rounded-xl px-4" onClick={onAdd}>
          Add
        </Button>
      </div>
      <div className="flex flex-wrap gap-2">
        {items.map((item) => (
          <button
            key={item}
            type="button"
            onClick={() => onRemove(item)}
            className="inline-flex items-center rounded-full border border-gray-200 bg-white px-3 py-1.5 text-xs text-gray-700 transition hover:border-rose-300 hover:text-rose-600 dark:border-gray-800 dark:bg-gray-950/40 dark:text-gray-300 dark:hover:border-rose-500/40 dark:hover:text-rose-400"
          >
            {item}
          </button>
        ))}
      </div>
    </div>
  );
}

function policyTabClasses(active: boolean) {
  return `rounded-xl px-4 py-2.5 text-sm font-medium transition ${
    active
      ? "bg-white text-gray-900 shadow-theme-xs dark:bg-gray-800 dark:text-white"
      : "text-gray-500 hover:text-gray-900 dark:text-gray-400 dark:hover:text-white"
  }`;
}

function SummaryRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-center justify-between gap-3 rounded-2xl border border-gray-200 bg-gray-50 px-4 py-4 dark:border-gray-800 dark:bg-gray-900/60">
      <span className="text-sm text-gray-500 dark:text-gray-400">{label}</span>
      <span className="text-sm font-semibold text-gray-900 dark:text-white">{value}</span>
    </div>
  );
}
