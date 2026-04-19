"use client";

import type { ChangeEvent, ReactNode } from "react";
import { useEffect, useMemo, useState } from "react";
import { useRouter } from "next/navigation";
import PasswordChecklist from "@/components/auth/PasswordChecklist";
import { isStrongPassword } from "@/components/auth/auth-utils";
import PageBreadcrumb from "@/components/common/PageBreadCrumb";
import ComponentCard from "@/components/common/ComponentCard";
import Input from "@/components/form/input/InputField";
import TextArea from "@/components/form/input/TextArea";
import Button from "@/components/ui/button/Button";
import { useToast } from "@/components/ui/toast/ToastProvider";
import { EyeCloseIcon, EyeIcon } from "@/icons";
import {
  createHypervisorVirtualMachine,
  listActiveVPSPackages,
  listHypervisorNodes,
  type HypervisorNode,
  type PlanPackage,
} from "@/components/hypervisor/api";
import { vmImageOptions, type VMImageFamily } from "@/components/virtual-machines/data";

const familyTabs: { id: VMImageFamily; label: string }[] = [
  { id: "linux", label: "Linux" },
  { id: "apps", label: "Apps" },
  { id: "custom", label: "Custom Image" },
];

export default function NewVirtualMachinePage() {
  const router = useRouter();
  const { pushToast } = useToast();
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [family, setFamily] = useState<VMImageFamily>("linux");
  const [imageID, setImageID] = useState("ubuntu-24");
  const [planID, setPlanID] = useState("");
  const [plans, setPlans] = useState<PlanPackage[]>([]);
  const [nodes, setNodes] = useState<HypervisorNode[]>([]);
  const [plansLoading, setPlansLoading] = useState(true);
  const [plansError, setPlansError] = useState("");
  const [zonesLoading, setZonesLoading] = useState(true);
  const [zonesError, setZonesError] = useState("");
  const [zone, setZone] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [authMode, setAuthMode] = useState<"password" | "ssh">("password");
  const [rootPassword, setRootPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [showRootPassword, setShowRootPassword] = useState(false);
  const [showConfirmPassword, setShowConfirmPassword] = useState(false);
  const [sshKey, setSSHKey] = useState("");
  const [generatedPrivateKey, setGeneratedPrivateKey] = useState("");
  const [tags, setTags] = useState("");
  const [project, setProject] = useState("default");
  const [backupsEnabled, setBackupsEnabled] = useState(true);
  const [monitoringEnabled, setMonitoringEnabled] = useState(true);
  const [privateNetworkingEnabled, setPrivateNetworkingEnabled] = useState(true);
  const [ipv6Enabled, setIPv6Enabled] = useState(false);

  const visibleImages = useMemo(
    () => vmImageOptions.filter((item) => item.family === family),
    [family],
  );

  const selectedImage =
    visibleImages.find((item) => item.id === imageID) ??
    vmImageOptions.find((item) => item.id === imageID) ??
    visibleImages[0];

  const selectedPlan =
    plans.find((item) => item.id === planID && item.spec) ??
    plans.find((item) => item.spec) ??
    null;
  const availableZones = useMemo(() => {
    const counts = new Map<string, number>();
    for (const node of nodes) {
      const currentZone = node.zone.trim();
      if (!currentZone) {
        continue;
      }
      counts.set(currentZone, (counts.get(currentZone) ?? 0) + 1);
    }
    return Array.from(counts.entries())
      .sort(([left], [right]) => left.localeCompare(right))
      .map(([name, nodeCount]) => ({ name, nodeCount }));
  }, [nodes]);

  const passwordReady = isStrongPassword(rootPassword) && rootPassword === confirmPassword;
  const accessSummary =
    authMode === "ssh"
      ? sshKey.trim()
        ? "SSH key"
        : "SSH key pending"
      : passwordReady
        ? "Password"
        : "Password pending";

  const addonSummary = useMemo(
    () =>
      [
        backupsEnabled ? "Backups" : null,
        monitoringEnabled ? "Monitoring" : null,
        privateNetworkingEnabled ? "Private Net" : null,
        ipv6Enabled ? "IPv6" : null,
      ]
        .filter(Boolean)
        .join(", ") || "None",
    [backupsEnabled, ipv6Enabled, monitoringEnabled, privateNetworkingEnabled],
  );

  useEffect(() => {
    let cancelled = false;

    async function loadProvisioningOptions() {
      try {
        setPlansLoading(true);
        setZonesLoading(true);
        setPlansError("");
        setZonesError("");
        const [packageItems, nodeItems] = await Promise.all([listActiveVPSPackages(), listHypervisorNodes()]);
        const items = packageItems.filter((item) => item.spec);
        if (cancelled) {
          return;
        }
        setPlans(items);
        setNodes(nodeItems);
        setPlanID((current) =>
          current && items.some((item) => item.id === current) ? current : (items[0]?.id ?? ""),
        );
        setZone((current) =>
          current && nodeItems.some((item) => item.zone.trim() === current)
            ? current
            : (nodeItems.find((item) => item.zone.trim())?.zone.trim() ?? ""),
        );
      } catch (err) {
        if (cancelled) {
          return;
        }
        const message = err instanceof Error ? err.message : "Failed to load provisioning options.";
        setPlans([]);
        setNodes([]);
        setPlanID("");
        setZone("");
        setPlansError(message);
        setZonesError(message);
      } finally {
        if (!cancelled) {
          setPlansLoading(false);
          setZonesLoading(false);
        }
      }
    }

    void loadProvisioningOptions();
    return () => {
      cancelled = true;
    };
  }, []);

  function randomChars(length: number, alphabet: string) {
    const bytes = new Uint8Array(length);
    globalThis.crypto.getRandomValues(bytes);
    return Array.from(bytes, (byte) => alphabet[byte % alphabet.length]).join("");
  }

  function generatePassword() {
    const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz23456789!@#$%^&*()_+-=";
    const nextPassword = `Vm!${randomChars(14, alphabet)}`;
    setRootPassword(nextPassword);
    setConfirmPassword(nextPassword);
    pushToast({
      kind: "success",
      message: "Strong password generated.",
    });
  }

  function generateSSHKeyPair() {
    const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/";
    const publicBody = randomChars(44, alphabet);
    const privateBody = randomChars(160, alphabet).match(/.{1,64}/g)?.join("\n") ?? "";
    setSSHKey(`ssh-ed25519 ${publicBody} aurora@controlplane`);
    setGeneratedPrivateKey(
      `-----BEGIN OPENSSH PRIVATE KEY-----\n${privateBody}\n-----END OPENSSH PRIVATE KEY-----`,
    );
    pushToast({
      kind: "success",
      message: "SSH key pair generated.",
    });
  }

  async function handleCreate() {
    if (name.trim() === "") {
      pushToast({ kind: "error", message: "VPS name is required." });
      return;
    }
    if (!selectedPlan?.spec) {
      pushToast({ kind: "error", message: plansError || "An active VPS package is required." });
      return;
    }
    if (zone.trim() === "") {
      pushToast({ kind: "error", message: zonesError || "Assign at least one hypervisor node to a zone first." });
      return;
    }
    if (authMode === "password" && !passwordReady) {
      pushToast({ kind: "error", message: "Root password does not satisfy the required rules." });
      return;
    }
    if (authMode === "ssh" && sshKey.trim() === "") {
      pushToast({ kind: "error", message: "Public SSH key is required when SSH mode is selected." });
      return;
    }

    try {
      setSubmitting(true);
      const payload = await createHypervisorVirtualMachine({
        name: name.trim(),
        description: description.trim(),
        package_id: selectedPlan.id,
        zone: zone.trim(),
        image: selectedImage?.id ?? "ubuntu-24",
        auth_mode: authMode,
        password: authMode === "password" ? rootPassword : undefined,
        ssh_public_key: authMode === "ssh" ? sshKey.trim() : undefined,
      });

      pushToast({
        kind: "success",
        message:
          payload.dispatch_state === "dispatched"
            ? "Virtual machine provisioning dispatched to the node."
            : "Virtual machine provisioning queued successfully.",
      });
      if (tags.trim() !== "" || project !== "default") {
        pushToast({
          kind: "info",
          message: "Tags, project, and add-on settings are not persisted by the current hypervisor API yet.",
        });
      }

      router.push(`/virtual-machines/detail?id=${encodeURIComponent(payload.id)}`);
    } catch (err) {
      pushToast({
        kind: "error",
        message: err instanceof Error ? err.message : "Failed to create virtual machine.",
      });
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <div className="space-y-6">
      <PageBreadcrumb pageTitle="New Virtual Machine" />

      <section className="grid gap-6 xl:grid-cols-[minmax(0,1.6fr)_360px]">
        <div className="space-y-6">
          <ComponentCard title="1. VPS Information" desc="Basic identity and notes for this virtual machine.">
            <div className="grid gap-5">
              <FieldBlock label="VPS Name">
                <Input
                  value={name}
                  onChange={(event) => setName(event.target.value)}
                  placeholder="vd: mail-relay-01"
                />
              </FieldBlock>
              <FieldBlock label="Description">
                <TextArea
                  rows={4}
                  value={description}
                  onChange={setDescription}
                  placeholder="Ghi chu nhanh ve muc dich, owner, hoac workload chinh cua may ao nay."
                />
              </FieldBlock>
            </div>
          </ComponentCard>

          <ComponentCard title="2. Select OS / Image" desc="Chon mot image base phu hop voi workload du kien.">
            <div className="space-y-5">
              <div className="inline-flex rounded-2xl border border-gray-200 bg-gray-50 p-1 dark:border-gray-800 dark:bg-gray-900/60">
                {familyTabs.map((tab) => (
                  <button
                    key={tab.id}
                    type="button"
                    onClick={() => {
                      setFamily(tab.id);
                      const nextImage = vmImageOptions.find((item) => item.family === tab.id);
                      if (nextImage) setImageID(nextImage.id);
                    }}
                    className={`rounded-xl px-4 py-2.5 text-sm font-medium transition ${
                      family === tab.id
                        ? "bg-white text-gray-900 shadow-theme-xs dark:bg-gray-800 dark:text-white"
                        : "text-gray-500 hover:text-gray-900 dark:text-gray-400 dark:hover:text-white"
                    }`}
                  >
                    {tab.label}
                  </button>
                ))}
              </div>

              <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
                {visibleImages.map((image) => {
                  const active = image.id === imageID;
                  return (
                    <button
                      key={image.id}
                      type="button"
                      onClick={() => setImageID(image.id)}
                      className={`rounded-2xl border px-4 py-4 text-left transition ${
                        active
                          ? "border-brand-400 bg-brand-500/10 shadow-theme-xs dark:border-brand-500/50"
                          : "border-gray-200 bg-gray-50 hover:border-brand-300 dark:border-gray-800 dark:bg-gray-900/60 dark:hover:border-brand-500/40"
                      }`}
                    >
                      <div className="flex items-start justify-between gap-3">
                        <div className="flex min-w-0 items-start gap-3">
                          <div className="mt-0.5 shrink-0">
                            <ImageIcon imageID={image.id} family={image.family} active={active} />
                          </div>
                          <div>
                            <p className="font-medium text-gray-900 dark:text-white">{image.name}</p>
                            <p className="mt-2 text-sm leading-6 text-gray-500 dark:text-gray-400">
                              {image.description}
                            </p>
                          </div>
                        </div>
                        <span
                          className={`mt-1 h-4 w-4 rounded-full border-2 ${
                            active
                              ? "border-brand-500 bg-brand-500"
                              : "border-gray-300 dark:border-gray-600"
                          }`}
                        />
                      </div>
                    </button>
                  );
                })}
              </div>
            </div>
          </ComponentCard>

          <ComponentCard title="3. Choose Resource Package" desc="CPU, memory va disk duoc resolve tu catalog package dang active.">
            {plansLoading ? (
              <div className="rounded-2xl border border-dashed border-gray-200 px-4 py-8 text-sm text-gray-500 dark:border-gray-800 dark:text-gray-400">
                Loading active VPS packages...
              </div>
            ) : null}
            {!plansLoading && plansError ? (
              <div className="rounded-2xl border border-dashed border-rose-200 px-4 py-8 text-sm text-rose-600 dark:border-rose-500/30 dark:text-rose-400">
                {plansError}
              </div>
            ) : null}
            {!plansLoading && !plansError && plans.length === 0 ? (
              <div className="rounded-2xl border border-dashed border-gray-200 px-4 py-8 text-sm text-gray-500 dark:border-gray-800 dark:text-gray-400">
                No active VPS package is available yet. Create or seed a package before provisioning a VM.
              </div>
            ) : null}
            {!plansLoading && !plansError && plans.length > 0 ? (
              <div className="grid gap-4 md:grid-cols-3">
                {plans.map((plan) => {
                  const active = plan.id === planID;
                  return (
                    <button
                      key={plan.id}
                      type="button"
                      onClick={() => setPlanID(plan.id)}
                      className={`rounded-3xl border px-5 py-5 text-left transition ${
                        active
                          ? "border-brand-400 bg-brand-500/10 shadow-theme-xs dark:border-brand-500/50"
                          : "border-gray-200 bg-gray-50 hover:border-brand-300 dark:border-gray-800 dark:bg-gray-900/60 dark:hover:border-brand-500/40"
                      }`}
                    >
                      <div className="flex h-full flex-col gap-5">
                        <div className="flex items-start justify-between gap-3">
                          <div>
                            <p className="font-medium text-gray-900 dark:text-white">{plan.name}</p>
                            <p className="mt-1 text-xs uppercase tracking-[0.18em] text-gray-400 dark:text-gray-500">
                              {plan.code}
                            </p>
                          </div>
                          <span
                            className={`mt-1 h-4 w-4 rounded-full border-2 ${
                              active
                                ? "border-brand-500 bg-brand-500"
                                : "border-gray-300 dark:border-gray-600"
                            }`}
                          />
                        </div>

                        <p className="text-sm leading-6 text-gray-500 dark:text-gray-400">
                          {plan.description || "Compute package from the shared resource catalog."}
                        </p>

                        <div className="space-y-3">
                          <PlanLine label="vCPU" value={`${plan.spec?.vcpu ?? 0} cores`} />
                          <PlanLine label="Memory" value={`${plan.spec?.ram_gb ?? 0} GB RAM`} />
                          <PlanLine label="Storage" value={`${plan.spec?.disk_gb ?? 0} GB SSD`} />
                        </div>
                      </div>
                    </button>
                  );
                })}
              </div>
            ) : null}
          </ComponentCard>

          <ComponentCard title="4. Select Zone" desc="Chon zone de dat workload, khong can chon region rieng.">
            <div className="grid gap-4 md:grid-cols-3">
              {availableZones.map((item) => {
                const active = item.name === zone;
                return (
                  <button
                    key={item.name}
                    type="button"
                    onClick={() => setZone(item.name)}
                    className={`rounded-2xl border px-4 py-4 text-left transition ${
                      active
                        ? "border-brand-400 bg-brand-500/10 shadow-theme-xs dark:border-brand-500/50"
                        : "border-gray-200 bg-gray-50 hover:border-brand-300 dark:border-gray-800 dark:bg-gray-900/60 dark:hover:border-brand-500/40"
                    }`}
                  >
                    <div className="flex items-start justify-between gap-3">
                      <div>
                        <p className="font-medium text-gray-900 dark:text-white">{item.name}</p>
                        <p className="mt-2 text-sm text-gray-500 dark:text-gray-400">
                          {item.nodeCount} {item.nodeCount === 1 ? "hypervisor node" : "hypervisor nodes"} available
                        </p>
                      </div>
                      <span
                        className={`mt-1 h-4 w-4 rounded-full border-2 ${
                          active
                            ? "border-brand-500 bg-brand-500"
                            : "border-gray-300 dark:border-gray-600"
                        }`}
                      />
                    </div>
                  </button>
                );
              })}
            </div>
            {!zonesLoading && zonesError ? (
              <div className="rounded-2xl border border-dashed border-rose-200 px-4 py-8 text-sm text-rose-600 dark:border-rose-500/30 dark:text-rose-400">
                {zonesError}
              </div>
            ) : null}
            {!zonesLoading && !zonesError && availableZones.length === 0 ? (
              <div className="rounded-2xl border border-dashed border-gray-200 px-4 py-8 text-sm text-gray-500 dark:border-gray-800 dark:text-gray-400">
                No zoned hypervisor node is available yet. Assign a node to a zone in admin before provisioning a VM.
              </div>
            ) : null}

          </ComponentCard>

          <ComponentCard title="5. Authentication & Security" desc="Thong tin dang nhap root va khoa SSH cho lan ket noi dau tien.">
            <div className="space-y-5">
              <div className="grid gap-4 md:grid-cols-2">
                <AuthModeCard
                  active={authMode === "password"}
                  title="Password"
                  description="Dang nhap lan dau bang root password."
                  onClick={() => setAuthMode("password")}
                />
                <AuthModeCard
                  active={authMode === "ssh"}
                  title="SSH Key"
                  description="Chi cap quyen truy cap bang public key."
                  onClick={() => setAuthMode("ssh")}
                />
              </div>

              {authMode === "password" ? (
                <div className="space-y-5">
                  <div className="flex flex-wrap items-center justify-between gap-3 rounded-2xl border border-gray-200 bg-gray-50 px-4 py-3 dark:border-gray-800 dark:bg-gray-900/60">
                    <div>
                      <p className="font-medium text-gray-900 dark:text-white">Root password</p>
                      <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
                        Mat khau phai dat cung rule nhu luong dang ky tai khoan.
                      </p>
                    </div>
                    <Button variant="outline" className="rounded-xl px-4" onClick={generatePassword}>
                      Generate Password
                    </Button>
                  </div>

                  <div className="grid gap-5 md:grid-cols-2">
                    <FieldBlock label="Root Password">
                      <PasswordField
                        visible={showRootPassword}
                        onToggle={() => setShowRootPassword((value) => !value)}
                        value={rootPassword}
                        onChange={(event) => setRootPassword(event.target.value)}
                        placeholder="Nhap mat khau root"
                        error={rootPassword.length > 0 && !isStrongPassword(rootPassword)}
                      />
                    </FieldBlock>
                    <FieldBlock label="Confirm Password">
                      <PasswordField
                        visible={showConfirmPassword}
                        onToggle={() => setShowConfirmPassword((value) => !value)}
                        value={confirmPassword}
                        onChange={(event) => setConfirmPassword(event.target.value)}
                        placeholder="Nhap lai mat khau"
                        error={confirmPassword.length > 0 && rootPassword !== confirmPassword}
                      />
                    </FieldBlock>
                  </div>

                  <PasswordChecklist password={rootPassword} confirmPassword={confirmPassword} />
                </div>
              ) : (
                <div className="space-y-5">
                  <div className="flex flex-wrap items-center justify-between gap-3 rounded-2xl border border-gray-200 bg-gray-50 px-4 py-3 dark:border-gray-800 dark:bg-gray-900/60">
                    <div>
                      <p className="font-medium text-gray-900 dark:text-white">SSH access</p>
                      <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
                        Paste public key co san hoac tao nhanh mot cap key moi.
                      </p>
                    </div>
                    <Button variant="outline" className="rounded-xl px-4" onClick={generateSSHKeyPair}>
                      Generate SSH Key
                    </Button>
                  </div>

                  <FieldBlock label="Public SSH Key">
                    <TextArea
                      rows={4}
                      value={sshKey}
                      onChange={setSSHKey}
                      placeholder="ssh-ed25519 AAAA... user@machine"
                    />
                  </FieldBlock>

                  {generatedPrivateKey.trim() !== "" ? (
                    <FieldBlock label="Generated Private Key">
                      <TextArea rows={8} value={generatedPrivateKey} onChange={setGeneratedPrivateKey} />
                    </FieldBlock>
                  ) : null}
                </div>
              )}
            </div>
          </ComponentCard>

          <ComponentCard title="6. Additional Options" desc="Bat them cac tinh nang ho tro van hanh cho VPS nay.">
            <div className="grid gap-4 md:grid-cols-2">
              <OptionRow checked={backupsEnabled} onChange={setBackupsEnabled} title="Enable Backups" subtitle="Daily point-in-time backups" />
              <OptionRow checked={monitoringEnabled} onChange={setMonitoringEnabled} title="Monitoring" subtitle="Basic CPU, memory and disk metrics" />
              <OptionRow checked={privateNetworkingEnabled} onChange={setPrivateNetworkingEnabled} title="Private Networking" subtitle="Attach an internal service network" />
              <OptionRow checked={ipv6Enabled} onChange={setIPv6Enabled} title="IPv6 Support" subtitle="Provision a dual-stack public interface" />
            </div>
          </ComponentCard>

          <ComponentCard title="7. Finalize Setup" desc="Nhan tag va project de doi ngu van hanh quan ly de hon.">
            <div className="grid gap-5 md:grid-cols-2">
              <FieldBlock label="Tags">
                <Input value={tags} onChange={(event) => setTags(event.target.value)} placeholder="mail, relay, auth" />
              </FieldBlock>
              <FieldBlock label="Project">
                <select
                  value={project}
                  onChange={(event) => setProject(event.target.value)}
                  className="h-11 w-full rounded-lg border border-gray-300 bg-transparent px-4 text-sm text-gray-800 shadow-theme-xs outline-none transition focus:border-brand-300 focus:ring-3 focus:ring-brand-500/10 dark:border-gray-700 dark:bg-gray-900 dark:text-white/90 dark:focus:border-brand-800"
                >
                  <option value="default">Default Project</option>
                  <option value="auth-platform">Auth Platform</option>
                  <option value="campaign-engine">Campaign Engine</option>
                </select>
              </FieldBlock>
            </div>
          </ComponentCard>
        </div>

        <div className="space-y-6 xl:sticky xl:top-24 xl:self-start">
          <ComponentCard
            title="Provisioning Summary"
            desc="Tong hop package, image, zone va access mode duoc gui toi hypervisor API."
          >
            <div className="space-y-4">
              <PriceRow label="VPS Name" value={name.trim() || "Unspecified"} />
              <PriceRow label="Selected Image" value={selectedImage?.name ?? "None"} />
              <PriceRow label="Package" value={selectedPlan ? `${selectedPlan.name} (${selectedPlan.code})` : "No active package"} />
              <PriceRow label="Zone" value={zone} />
              <PriceRow label="Add-ons" value={addonSummary} />
            </div>

            <div className="mt-6 space-y-3 rounded-2xl border border-gray-200 bg-gray-50 p-4 dark:border-gray-800 dark:bg-gray-900/60">
              <PriceRow
                label="Compute"
                value={
                  selectedPlan?.spec
                    ? `${selectedPlan.spec.vcpu} vCPU / ${selectedPlan.spec.ram_gb} GB`
                    : "Pending package"
                }
              />
              <PriceRow
                label="Storage"
                value={selectedPlan?.spec ? `${selectedPlan.spec.disk_gb} GB SSD` : "Pending package"}
              />
              <PriceRow label="Access" value={accessSummary} />
            </div>

            <div className="mt-6 flex flex-col gap-3">
              <Button
                className="w-full rounded-xl"
                onClick={() => void handleCreate()}
                disabled={submitting || plansLoading || zonesLoading || !selectedPlan?.spec || zone.trim() === ""}
              >
                {submitting ? "Creating..." : "Create VPS"}
              </Button>
              <Button
                variant="outline"
                className="w-full rounded-xl"
                onClick={() => router.push("/virtual-machines")}
              >
                Back to inventory
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

function AuthModeCard({
  active,
  title,
  description,
  onClick,
}: {
  active: boolean;
  title: string;
  description: string;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={`rounded-2xl border px-4 py-4 text-left transition ${
        active
          ? "border-brand-400 bg-brand-500/10 shadow-theme-xs dark:border-brand-500/50"
          : "border-gray-200 bg-gray-50 hover:border-brand-300 dark:border-gray-800 dark:bg-gray-900/60 dark:hover:border-brand-500/40"
      }`}
    >
      <div className="flex items-start justify-between gap-3">
        <div>
          <p className="font-medium text-gray-900 dark:text-white">{title}</p>
          <p className="mt-2 text-sm leading-6 text-gray-500 dark:text-gray-400">{description}</p>
        </div>
        <span
          className={`mt-1 h-4 w-4 rounded-full border-2 ${
            active ? "border-brand-500 bg-brand-500" : "border-gray-300 dark:border-gray-600"
          }`}
        />
      </div>
    </button>
  );
}

function PasswordField({
  visible,
  onToggle,
  value,
  onChange,
  placeholder,
  error,
}: {
  visible: boolean;
  onToggle: () => void;
  value: string;
  onChange: (event: ChangeEvent<HTMLInputElement>) => void;
  placeholder: string;
  error?: boolean;
}) {
  return (
    <div className="relative">
      <Input
        type={visible ? "text" : "password"}
        value={value}
        onChange={onChange}
        placeholder={placeholder}
        error={error}
      />
      <button
        type="button"
        onClick={onToggle}
        className="absolute right-4 top-1/2 z-30 -translate-y-1/2 text-gray-500 transition-colors hover:text-gray-700 dark:text-gray-400 dark:hover:text-white"
        aria-label={visible ? "Hide password" : "Show password"}
      >
        {visible ? <EyeIcon className="fill-current" /> : <EyeCloseIcon className="fill-current" />}
      </button>
    </div>
  );
}

function OptionRow({
  checked,
  onChange,
  title,
  subtitle,
}: {
  checked: boolean;
  onChange: (value: boolean) => void;
  title: string;
  subtitle: string;
}) {
  return (
    <button
      type="button"
      onClick={() => onChange(!checked)}
      className={`flex items-start gap-4 rounded-2xl border px-4 py-4 text-left transition ${
        checked
          ? "border-brand-400 bg-brand-500/10 dark:border-brand-500/50"
          : "border-gray-200 bg-gray-50 hover:border-brand-300 dark:border-gray-800 dark:bg-gray-900/60 dark:hover:border-brand-500/40"
      }`}
    >
      <span
        className={`mt-0.5 inline-flex h-5 w-5 shrink-0 items-center justify-center rounded-md border ${
          checked
            ? "border-brand-500 bg-brand-500 text-white"
            : "border-gray-300 bg-white dark:border-gray-700 dark:bg-gray-900"
        }`}
      >
        {checked ? (
          <svg width="12" height="12" viewBox="0 0 12 12" fill="none" xmlns="http://www.w3.org/2000/svg">
            <path
              d="M2.5 6.25L4.75 8.5L9.5 3.75"
              stroke="currentColor"
              strokeWidth="1.6"
              strokeLinecap="round"
              strokeLinejoin="round"
            />
          </svg>
        ) : null}
      </span>
      <span className="min-w-0">
        <span className="block text-sm font-medium text-gray-900 dark:text-white">{title}</span>
        <span className="mt-1 block text-sm text-gray-500 dark:text-gray-400">{subtitle}</span>
      </span>
    </button>
  );
}

function PriceRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-start justify-between gap-4 border-b border-dashed border-gray-200 pb-3 text-sm last:border-b-0 last:pb-0 dark:border-gray-800">
      <span className="text-gray-500 dark:text-gray-400">{label}</span>
      <span className="text-right font-medium text-gray-900 dark:text-white">{value}</span>
    </div>
  );
}

function PlanLine({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-center justify-between gap-3 rounded-2xl bg-white/80 px-3 py-3 text-sm dark:bg-gray-950/40">
      <span className="text-gray-500 dark:text-gray-400">{label}</span>
      <span className="font-medium text-gray-900 dark:text-white">{value}</span>
    </div>
  );
}

function ImageIcon({
  imageID,
  family,
  active,
}: {
  imageID: string;
  family: VMImageFamily;
  active: boolean;
}) {
  const tone = active
    ? "border-brand-400 bg-brand-500/10 text-brand-600 dark:border-brand-500/50 dark:text-brand-400"
    : "border-gray-200 bg-white text-gray-500 dark:border-gray-800 dark:bg-gray-900/80 dark:text-gray-300";

  if (imageID.includes("ubuntu")) {
    return (
      <span className={`inline-flex h-11 w-11 items-center justify-center rounded-2xl border ${tone}`}>
        <svg width="24" height="24" viewBox="0 0 512 512" fill="none" aria-hidden="true">
          <path
            fill="currentColor"
            d="M256.417 508.37c-139.417 0-252.5-113.082-252.5-252.54C3.918 116.375 117 3.332 256.418 3.332c139.458 0 252.5 113.043 252.5 252.5S395.875 508.37 256.417 508.37zM98.837 225.507c-16.702 0-30.284 13.583-30.284 30.363 0 16.702 13.582 30.284 30.284 30.284 16.78 0 30.442-13.582 30.442-30.284 0-16.78-13.661-30.363-30.442-30.363zm216.688 137.957c-14.412 8.291-19.426 26.888-11.135 41.3 8.45 14.649 27.007 19.584 41.419 11.174 14.57-8.252 19.584-26.889 11.095-41.38-8.37-14.608-26.889-19.544-41.38-11.055v-.04zM164.775 255.87c0-29.968 14.925-56.54 37.707-72.571l-22.15-37.155c-26.612 17.729-46.394 44.854-54.567 76.6 9.674 7.896 15.675 19.741 15.675 33.126 0 13.306-6.001 25.23-15.675 33.048 8.173 31.745 27.994 58.831 54.567 76.52l22.15-37.075c-22.782-15.991-37.707-42.603-37.707-72.493zm88.602-88.72c46.394 0 84.377 35.575 88.326 80.942l43.274-.79c-2.172-33.324-16.741-63.53-39.168-85.403-11.648 4.264-25.033 3.672-36.483-3.08-11.53-6.634-18.676-17.807-20.69-29.969a132.869 132.869 0 00-35.259-4.738c-20.926 0-40.708 4.896-58.397 13.583l21.124 37.786c11.293-5.33 24.046-8.292 37.273-8.292v-.04zm0 177.362c-13.266 0-25.98-3.001-37.273-8.292l-21.124 37.786c17.65 8.647 37.47 13.543 58.397 13.543 12.28 0 24.046-1.58 35.26-4.738 1.973-12.2 9.16-23.454 20.689-30.047 11.569-6.594 24.835-7.305 36.483-2.883 22.466-22.19 36.957-52.197 39.168-85.561l-43.274-.632c-3.949 45.249-41.932 80.784-88.326 80.784v.04zm62.148-196.235c14.49 8.37 33.008 3.435 41.379-11.135 8.489-14.49 3.475-33.127-11.016-41.497-14.451-8.292-33.009-3.357-41.419 11.173-8.41 14.452-3.395 33.048 11.056 41.42v.039z"
          />
        </svg>
      </span>
    );
  }

  if (imageID.includes("debian")) {
    return (
      <span className={`inline-flex h-11 w-11 items-center justify-center rounded-2xl border ${tone}`}>
        <svg width="24" height="24" viewBox="0 0 512 512" fill="none" aria-hidden="true">
          <path
            fill="currentColor"
            d="M293.64 269.591c-7.952.108 1.506 4.093 11.888 5.695a105.704 105.704 0 007.79-6.704c-6.466 1.579-13.043 1.614-19.678 1.01m42.69-10.64c4.732-6.535 8.19-13.694 9.403-21.095-1.063 5.272-3.924 9.831-6.623 14.637-14.864 9.361-1.398-5.557-.008-11.229-15.98 20.12-2.195 12.064-2.772 17.687m15.76-41.003c.96-14.325-2.822-9.797-4.093-4.332 1.48.774 2.657 10.096 4.094 4.332m-88.61-195.76c4.244.762 9.169 1.347 8.48 2.356 4.644-1.013 5.699-1.949-8.48-2.357m8.48 2.36l-3 .62 2.792-.246.208-.373m132.358 198.83c.473 12.86-3.766 19.103-7.583 30.147l-6.877 3.435c-5.622 10.928.547 6.939-3.477 15.63-8.784 7.805-26.651 24.433-32.37 25.946-4.174-.088 2.827-4.921 3.743-6.82-11.752 8.08-9.43 12.119-27.41 17.029l-.523-1.175c-44.345 20.86-105.934-20.478-105.122-76.884-.473 3.585-1.344 2.684-2.33 4.136-2.287-29.023 13.405-58.174 39.867-70.069 25.885-12.819 56.23-7.559 74.77 9.72-10.181-13.343-30.455-27.483-54.48-26.158-23.532.373-45.547 15.326-52.89 31.56-12.06 7.594-13.458 29.266-18.71 33.225-7.067 51.95 13.296 74.388 47.74 100.793 5.419 3.654 1.526 4.209 2.261 6.99-11.444-5.361-21.926-13.452-30.54-23.356 4.567 6.697 9.504 13.205 15.884 18.314-10.793-3.654-25.207-26.146-29.42-27.063 18.607 33.306 75.471 58.412 105.253 45.96-13.782.504-31.283.28-46.767-5.442-6.5-3.346-15.342-10.278-13.763-11.58 40.637 15.188 82.622 11.507 117.78-16.684 8.944-6.97 18.718-18.82 21.54-18.985-4.255 6.396.724 3.08-2.541 8.726 8.918-14.375-3.87-5.857 9.218-24.83l4.837 6.658c-1.798-11.93 14.818-26.42 13.131-45.292 3.812-5.772 4.251 6.207.208 19.489 5.61-14.73 1.479-17.098 2.923-29.255 1.555 4.09 3.6 8.426 4.651 12.739-3.658-14.233 3.755-23.975 5.584-32.243-1.798-.8-5.638 6.292-6.515-10.52.13-7.3 2.033-3.824 2.764-5.622-1.436-.828-5.194-6.423-7.486-17.159 1.66-2.522 4.433 6.539 6.697 6.912-1.456-8.537-3.959-15.045-4.055-21.599-6.596-13.782-2.333 1.841-7.686-5.914-7.02-21.9 5.826-5.083 6.693-15.03 10.64 15.415 16.708 39.309 19.492 49.201-2.126-12.064-5.557-23.751-9.75-35.061 3.23 1.363-5.206-24.822 4.201-7.482-10.046-36.967-42.997-71.505-73.31-87.712 3.704 3.392 8.39 7.659 6.711 8.325-15.075-8.976-12.426-9.677-14.582-13.47-12.284-4.994-13.089.4-21.222.012-23.15-12.277-27.614-10.975-48.916-18.669l.97 4.532c-15.34-5.11-17.87 1.937-34.444.016-1.013-.79 5.31-2.85 10.512-3.608-14.825 1.956-14.132-2.923-28.642.539 3.578-2.507 7.351-4.167 11.167-6.304-12.087.736-28.861 7.04-23.686 1.306-19.72 8.802-54.75 21.152-74.404 39.585l-.62-4.132c-9.007 10.817-39.277 32.297-41.688 46.294l-2.407.562c-4.69 7.933-7.717 16.928-11.436 25.096-6.13 10.447-8.988 4.02-8.114 5.656-12.06 24.449-18.048 44.989-23.22 61.832 3.69 5.514.085 33.17 1.483 55.312-6.054 109.338 76.737 215.496 167.238 240.014 13.265 4.736 32.989 4.559 49.767 5.048-19.797-5.665-22.358-2.996-41.638-9.723-13.917-6.55-16.963-14.032-26.817-22.581l3.905 6.893c-19.327-6.835-11.237-8.464-26.963-13.443l4.17-5.437c-6.265-.474-16.593-10.552-19.415-16.139l-6.85.27c-8.234-10.159-12.62-17.475-12.3-23.147l-2.214 3.943c-2.511-4.31-30.287-38.096-15.877-30.229-2.68-2.452-6.238-3.985-10.097-10.997l2.935-3.354c-6.94-8.919-12.766-20.36-12.323-24.168 3.697 4.999 6.265 5.93 8.807 6.79-17.513-43.456-18.495-2.396-31.761-44.234l2.807-.227c-2.153-3.243-3.458-6.762-5.187-10.213l1.217-12.172c-12.607-14.579-3.527-61.99-1.706-87.993 1.263-10.575 10.524-21.83 17.571-39.478l-4.29-.74c8.206-14.313 46.852-57.484 64.75-55.262 8.669-10.893-1.717-.042-3.411-2.784 19.042-19.712 25.03-13.928 37.884-17.47 13.862-8.23-11.895 3.207-5.326-3.139 23.963-6.119 16.982-13.913 48.242-17.02 3.3 1.875-7.651 2.895-10.4 5.333 19.965-9.766 63.182-7.544 91.25 5.422 32.57 15.218 69.164 60.218 70.608 102.55l1.644.438c-.832 16.828 2.576 36.29-3.33 54.165l4.016-8.476m-197.486 57.13l-1.117 5.576c5.226 7.105 9.377 14.798 16.046 20.343-4.798-9.372-8.367-13.246-14.93-25.919m12.354-.485c-2.765-3.062-4.405-6.743-6.238-10.413 1.756 6.45 5.345 11.992 8.683 17.63l-2.445-7.217m218.665-47.53l-1.17 2.934a141.476 141.476 0 01-13.86 44.234 139.075 139.075 0 0015.03-47.168M265.056 18.376c5.376-1.976 13.22-1.082 18.923-2.376-7.432.624-14.833.997-22.142 1.937l3.22.439M76.264 118.76c1.24 11.471-8.633 15.923 2.184 8.36 5.799-13.054-2.264-3.6-2.184-8.36m-12.71 53.094c2.487-7.647 2.941-12.245 3.892-16.666-6.885 8.8-3.169 10.678-3.893 16.666"
          />
        </svg>
      </span>
    );
  }

  if (imageID.includes("alpine")) {
    return (
      <span className={`inline-flex h-11 w-11 items-center justify-center rounded-2xl border ${tone}`}>
        <svg width="24" height="24" viewBox="0 0 24 24" fill="none" aria-hidden="true">
          <path d="M4 18L10.4 7.2L13.8 12.4L16.3 8.6L20 18H4Z" fill="currentColor" fillOpacity="0.18" />
          <path d="M4 18L10.4 7.2L13.8 12.4L16.3 8.6L20 18" stroke="currentColor" strokeWidth="1.8" strokeLinejoin="round" />
        </svg>
      </span>
    );
  }

  if (imageID.includes("windows")) {
    return (
      <span className={`inline-flex h-11 w-11 items-center justify-center rounded-2xl border ${tone}`}>
        <svg width="24" height="24" viewBox="0 0 512 512.02" fill="none" aria-hidden="true">
          <path
            fill="currentColor"
            d="M0 512.02h242.686V269.335H0V512.02zm0-269.334h242.686V0H0v242.686zm269.314 0H512V0H269.314v242.686zm0 269.334H512V269.335H269.314V512.02z"
          />
        </svg>
      </span>
    );
  }

  if (imageID.includes("docker")) {
    return (
      <span className={`inline-flex h-11 w-11 items-center justify-center rounded-2xl border ${tone}`}>
        <svg width="24" height="24" viewBox="0 0 24 24" fill="none" aria-hidden="true">
          <rect x="6" y="8" width="3" height="3" rx="0.8" fill="currentColor" />
          <rect x="10" y="8" width="3" height="3" rx="0.8" fill="currentColor" />
          <rect x="14" y="8" width="3" height="3" rx="0.8" fill="currentColor" />
          <rect x="10" y="4" width="3" height="3" rx="0.8" fill="currentColor" />
          <path d="M4.5 13H18.2C18 15.8 15.8 18 12.5 18C9.2 18 5.9 16.7 4.5 13Z" stroke="currentColor" strokeWidth="1.8" strokeLinejoin="round" />
        </svg>
      </span>
    );
  }

  if (imageID.includes("node")) {
    return (
      <span className={`inline-flex h-11 w-11 items-center justify-center rounded-2xl border ${tone}`}>
        <svg width="24" height="24" viewBox="0 0 24 24" fill="none" aria-hidden="true">
          <path d="M12 3.8L18.7 7.7V16.3L12 20.2L5.3 16.3V7.7L12 3.8Z" stroke="currentColor" strokeWidth="1.8" />
          <path d="M12 8.2V15.8" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" />
        </svg>
      </span>
    );
  }

  return (
    <span className={`inline-flex h-11 w-11 items-center justify-center rounded-2xl border ${tone}`}>
      {family === "custom" ? (
        <svg width="22" height="22" viewBox="0 0 24 24" fill="none" aria-hidden="true">
          <path d="M6 7.5C6 6.12 7.12 5 8.5 5H15.5C16.88 5 18 6.12 18 7.5V16.5C18 17.88 16.88 19 15.5 19H8.5C7.12 19 6 17.88 6 16.5V7.5Z" stroke="currentColor" strokeWidth="1.8" />
          <path d="M9 9H15M9 12H15M9 15H13" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" />
        </svg>
      ) : (
        <svg width="22" height="22" viewBox="0 0 24 24" fill="none" aria-hidden="true">
          <rect x="5" y="5" width="14" height="14" rx="3" stroke="currentColor" strokeWidth="1.8" />
          <path d="M9 9H15V15H9V9Z" fill="currentColor" fillOpacity="0.18" />
        </svg>
      )}
    </span>
  );
}
