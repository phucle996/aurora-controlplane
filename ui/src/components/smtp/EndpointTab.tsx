"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { useDropzone } from "react-dropzone";
import ComponentCard from "@/components/common/ComponentCard";
import {
  createEndpoint,
  deleteEndpoint,
  getEndpoint,
  listEndpoints,
  tryConnectEndpoint,
  updateEndpoint,
} from "@/components/smtp/api";
import { useSMTPWorkspace } from "@/components/smtp/SMTPWorkspaceProvider";
import type { DeliveryEndpoint } from "@/components/smtp/types";

type TLSMode = "none" | "starttls" | "tls" | "mtls";
type EndpointStatus = DeliveryEndpoint["status"];
type CertInputMode = "manual" | "upload";
type EndpointFieldErrorKey = "name" | "host" | "port" | "clientCert" | "clientKey";

type EndpointFormState = {
  name: string;
  host: string;
  port: string;
  username: string;
  password: string;
  priority: string;
  weight: string;
  tlsMode: TLSMode;
  status: EndpointStatus;
  certInputMode: CertInputMode;
  caCertContent: string;
  clientCertContent: string;
  clientKeyContent: string;
  caCertFile: File | null;
  clientCertFile: File | null;
  clientKeyFile: File | null;
};

type EndpointFieldErrors = Partial<Record<EndpointFieldErrorKey, string>>;

export function EndpointTab({
  search,
  onSearchChange,
  mode,
  onCreate,
  onEdit,
  onView,
  selectedEndpointId,
  onSelectEndpoint,
}: {
  search: string;
  onSearchChange: (value: string) => void;
  mode: "view" | "create" | "edit";
  onCreate: () => void;
  onEdit: () => void;
  onView: () => void;
  selectedEndpointId: string;
  onSelectEndpoint: (value: string) => void;
}) {
  const { workspace, workspaceID, isLoading: isWorkspaceLoading, error: workspaceError } = useSMTPWorkspace();
  const [endpoints, setEndpoints] = useState<DeliveryEndpoint[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState("");
  const [detail, setDetail] = useState<DeliveryEndpoint | null>(null);
  const [isDetailLoading, setIsDetailLoading] = useState(false);
  const [detailError, setDetailError] = useState("");
  const [actionError, setActionError] = useState("");
  const [isSaving, setIsSaving] = useState(false);
  const [isTryingConnect, setIsTryingConnect] = useState(false);
  const [connectResult, setConnectResult] = useState<{
    tone: "success" | "error";
    message: string;
  } | null>(null);
  const [form, setForm] = useState<EndpointFormState>(emptyEndpointForm());
  const [fieldErrors, setFieldErrors] = useState<EndpointFieldErrors>({});

  const loadEndpoints = useCallback(
    async (preferredEndpointID?: string) => {
      if (workspaceID === "") {
        setEndpoints([]);
        onSelectEndpoint("");
        setIsLoading(false);
        setError("");
        return;
      }

      setIsLoading(true);
      setError("");

      try {
        const nextItems = await listEndpoints(workspaceID);

        setEndpoints(nextItems);

        if (nextItems.length === 0) {
          onSelectEndpoint("");
          return;
        }

        const nextSelected =
          preferredEndpointID && nextItems.some((endpoint) => endpoint.id === preferredEndpointID)
            ? preferredEndpointID
            : selectedEndpointId !== "" && nextItems.some((endpoint) => endpoint.id === selectedEndpointId)
              ? selectedEndpointId
              : nextItems[0].id;

        onSelectEndpoint(nextSelected);
      } catch (err) {
        setError(err instanceof Error ? err.message : "Failed to load SMTP endpoints");
      } finally {
        setIsLoading(false);
      }
    },
    [onSelectEndpoint, selectedEndpointId, workspaceID],
  );

  useEffect(() => {
    void loadEndpoints();
  }, [loadEndpoints]);

  useEffect(() => {
    if (selectedEndpointId === "" || mode === "create") {
      setDetail(null);
      setDetailError("");
      return;
    }

    let cancelled = false;

    async function loadDetail() {
      setIsDetailLoading(true);
      setDetailError("");

      try {
        const item = await getEndpoint(workspaceID, selectedEndpointId);
        if (cancelled) {
          return;
        }

        setDetail(item);
      } catch (err) {
        if (!cancelled) {
          setDetailError(
            err instanceof Error ? err.message : "Failed to load SMTP endpoint detail",
          );
        }
      } finally {
        if (!cancelled) {
          setIsDetailLoading(false);
        }
      }
    }

    void loadDetail();
    return () => {
      cancelled = true;
    };
  }, [selectedEndpointId, mode, workspaceID]);

  useEffect(() => {
    if (mode === "create") {
      setForm(emptyEndpointForm());
      setFieldErrors({});
      return;
    }
    if (mode === "edit" && detail != null) {
      setForm(endpointToForm(detail));
      setFieldErrors({});
    }
  }, [mode, detail]);

  const keyword = search.trim().toLowerCase();
  const filteredEndpoints = useMemo(
    () =>
      endpoints.filter((endpoint) => {
        if (keyword === "") {
          return true;
        }

        return (
          endpoint.name.toLowerCase().includes(keyword) ||
          endpoint.host.toLowerCase().includes(keyword) ||
          endpoint.status.toLowerCase().includes(keyword) ||
          endpoint.username.toLowerCase().includes(keyword)
        );
      }),
    [keyword, endpoints],
  );

  const selectedEndpoint =
    filteredEndpoints.find((endpoint) => endpoint.id === selectedEndpointId) ??
    filteredEndpoints[0] ??
    endpoints[0] ??
    null;

  function updateFormField<K extends keyof EndpointFormState>(field: K, value: EndpointFormState[K]) {
    setForm((current) => ({ ...current, [field]: value }));
    const key = endpointFieldErrorKey(field);
    if (key == null) {
      return;
    }
    setFieldErrors((current) => {
      if (current[key] == null) {
        return current;
      }
      const next = { ...current };
      delete next[key];
      return next;
    });
  }

  function clearFieldErrors(keys: EndpointFieldErrorKey[]) {
    setFieldErrors((current) => {
      let changed = false;
      const next = { ...current };
      for (const key of keys) {
        if (next[key] != null) {
          delete next[key];
          changed = true;
        }
      }
      return changed ? next : current;
    });
  }

  async function handleSave() {
    const nextErrors = validateEndpointForm(form, { requireName: true, existingDetail: detail });
    if (Object.keys(nextErrors).length > 0) {
      setFieldErrors(nextErrors);
      setActionError("Please complete the required endpoint fields.");
      setConnectResult(null);
      return;
    }

    setIsSaving(true);
    setActionError("");
    setConnectResult(null);
    setFieldErrors({});

    try {
      if (workspaceID === "") {
        throw new Error("Choose a workspace first.");
      }
      const payload = await buildEndpointPayload(form);

      const result =
        mode === "create"
          ? await createEndpoint(workspaceID, payload)
          : await updateEndpoint(workspaceID, selectedEndpointId, payload);

      await loadEndpoints(result.id);
      onView();
    } catch (err) {
      setActionError(err instanceof Error ? err.message : "Failed to save SMTP endpoint");
    } finally {
      setIsSaving(false);
    }
  }

  async function handleTryConnect() {
    const nextErrors = validateEndpointForm(form, { requireName: false, existingDetail: detail });
    if (Object.keys(nextErrors).length > 0) {
      setFieldErrors(nextErrors);
      setConnectResult({
        tone: "error",
        message: "Please complete the required fields before trying the connection.",
      });
      return;
    }

    setIsTryingConnect(true);
    setActionError("");
    setConnectResult(null);
    setFieldErrors({});

    try {
      if (workspaceID === "") {
        throw new Error("Choose a workspace first.");
      }
      const payload = await buildEndpointPayload(form);
      const message = await tryConnectEndpoint(workspaceID, payload);

      setConnectResult({
        tone: "success",
        message,
      });
    } catch (err) {
      setConnectResult({
        tone: "error",
        message: err instanceof Error ? err.message : "Failed to connect to SMTP endpoint",
      });
    } finally {
      setIsTryingConnect(false);
    }
  }

  async function handleDelete(endpointID: string) {
    if (!window.confirm("Delete this SMTP endpoint?")) {
      return;
    }

    setActionError("");
    try {
      if (workspaceID === "") {
        throw new Error("Choose a workspace first.");
      }
      await deleteEndpoint(workspaceID, endpointID);
      await loadEndpoints();
      onView();
    } catch (err) {
      setActionError(err instanceof Error ? err.message : "Failed to delete SMTP endpoint");
    }
  }

  function handleBeginEdit(endpointID: string) {
    onSelectEndpoint(endpointID);
    onEdit();
  }

  return (
    <div className="space-y-6">
      <div className="grid gap-6 xl:grid-cols-[4fr_6fr]">
        <ComponentCard
          title="SMTP Endpoints"
          desc="Current delivery endpoints available to the control plane."
        >
          <div className="space-y-4">
            <div className="flex items-center gap-3">
              <div className="flex-1 rounded-2xl border border-gray-200 bg-gray-50 px-4 py-3 dark:border-gray-800 dark:bg-gray-900/40">
                <input
                  type="text"
                  value={search}
                  onChange={(event) => onSearchChange(event.target.value)}
                  placeholder="Search endpoints by name, host, sender, or status"
                  className="w-full bg-transparent text-sm text-gray-800 outline-none placeholder:text-gray-400 dark:text-white dark:placeholder:text-gray-500"
                />
              </div>
              <button
                type="button"
                onClick={onCreate}
                className="inline-flex items-center rounded-xl bg-gray-900 px-4 py-3 text-sm font-semibold text-white transition hover:bg-gray-800 dark:bg-white dark:text-gray-900 dark:hover:bg-gray-200"
              >
                Add
              </button>
            </div>

            {actionError !== "" && (
              <div className="rounded-2xl border border-error-200 bg-error-50 px-4 py-4 text-sm text-error-700 dark:border-error-500/30 dark:bg-error-500/10 dark:text-error-300">
                {actionError}
              </div>
            )}

            <div className="space-y-3">
              {isWorkspaceLoading && (
                <div className="rounded-2xl border border-gray-200 bg-gray-50 px-4 py-5 text-sm text-gray-500 dark:border-gray-800 dark:bg-gray-900/40 dark:text-gray-400">
                  Loading workspace context...
                </div>
              )}

              {!isWorkspaceLoading && workspaceError !== "" && (
                <div className="rounded-2xl border border-error-200 bg-error-50 px-4 py-5 text-sm text-error-700 dark:border-error-500/30 dark:bg-error-500/10 dark:text-error-300">
                  {workspaceError}
                </div>
              )}

              {!isWorkspaceLoading && workspaceError === "" && workspaceID === "" && (
                <div className="rounded-2xl border border-gray-200 bg-gray-50 px-4 py-5 text-sm text-gray-500 dark:border-gray-800 dark:bg-gray-900/40 dark:text-gray-400">
                  Choose a workspace to manage SMTP endpoints.
                </div>
              )}

              {!isWorkspaceLoading && workspace != null && (
                <div className="rounded-2xl border border-gray-200 bg-gray-50 px-4 py-4 text-sm text-gray-600 dark:border-gray-800 dark:bg-gray-900/40 dark:text-gray-300">
                  Managing endpoints for <span className="font-semibold text-gray-900 dark:text-white">{workspace.name}</span>.
                </div>
              )}

              {isLoading && (
                <div className="rounded-2xl border border-gray-200 bg-gray-50 px-4 py-5 text-sm text-gray-500 dark:border-gray-800 dark:bg-gray-900/40 dark:text-gray-400">
                  Loading SMTP endpoints...
                </div>
              )}

              {!isLoading && error !== "" && (
                <div className="rounded-2xl border border-error-200 bg-error-50 px-4 py-5 text-sm text-error-700 dark:border-error-500/30 dark:bg-error-500/10 dark:text-error-300">
                  {error}
                </div>
              )}

              {filteredEndpoints.map((endpoint) => {
                const active = mode === "view" && endpoint.id === selectedEndpoint?.id;
                return (
                  <button
                    key={endpoint.id}
                    type="button"
                    onClick={() => {
                      onSelectEndpoint(endpoint.id);
                      onView();
                    }}
                    className={`group relative w-full overflow-hidden rounded-2xl border px-4 py-4 text-left transition ${
                      active
                        ? "border-gray-900 bg-white shadow-theme-sm dark:border-white dark:bg-gray-900"
                        : "border-gray-200 bg-gray-50 hover:border-gray-300 dark:border-gray-800 dark:bg-gray-900/40 dark:hover:border-gray-700"
                    }`}
                  >
                    <div className="min-w-0 pr-[44%]">
                      <h3 className="flex items-center gap-2 text-sm font-semibold text-gray-900 dark:text-white">
                        <StatusDot status={endpoint.status} />
                        {endpoint.name}
                      </h3>
                      <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
                        {endpoint.host}:{endpoint.port}
                      </p>
                    </div>

                    <div className="absolute inset-y-0 right-0 w-[44%] bg-white/88 opacity-0 backdrop-blur-[2px] transition group-hover:opacity-100 dark:bg-gray-950/82">
                      <div className="flex h-full items-center justify-center gap-2 px-3">
                        <button
                          type="button"
                          onClick={(event) => {
                            event.stopPropagation();
                            handleBeginEdit(endpoint.id);
                          }}
                          className="rounded-lg border border-gray-200 bg-white px-2.5 py-1 text-xs font-medium text-gray-700 shadow-theme-xs dark:border-gray-700 dark:bg-gray-900 dark:text-gray-200"
                        >
                          Edit
                        </button>
                        <button
                          type="button"
                          onClick={(event) => {
                            event.stopPropagation();
                            void handleDelete(endpoint.id);
                          }}
                          className="rounded-lg border border-gray-200 bg-white px-2.5 py-1 text-xs font-medium text-gray-700 shadow-theme-xs dark:border-gray-700 dark:bg-gray-900 dark:text-gray-200"
                        >
                          Delete
                        </button>
                      </div>
                    </div>
                  </button>
                );
              })}

              {!isLoading && error === "" && filteredEndpoints.length === 0 && (
                <div className="rounded-2xl border border-gray-200 bg-gray-50 px-4 py-5 text-sm text-gray-500 dark:border-gray-800 dark:bg-gray-900/40 dark:text-gray-400">
                  No SMTP endpoints found.
                </div>
              )}
            </div>
          </div>
        </ComponentCard>

        <div className="space-y-6">
          <ComponentCard
            title={mode === "edit" ? "Edit SMTP Endpoint" : mode === "create" ? "Add SMTP Endpoint" : "Endpoint Detail"}
            desc={
              mode === "create"
                ? "Create a new outbound SMTP relay with TLS or mTLS material."
                : mode === "edit"
                  ? "Update the selected SMTP relay."
                  : "Runtime view for the SMTP relay currently selected."
            }
          >
            {mode === "create" || mode === "edit" ? (
              <div className="rounded-2xl bg-white p-1 dark:bg-gray-900">
                <div className="space-y-6">
                  <div className="space-y-5">
                    <div className="grid gap-4 md:grid-cols-2">
                      <EndpointField label="Name" value={form.name} onChange={(value) => updateFormField("name", value)} placeholder="Aurora Primary Relay" error={fieldErrors.name} />
                      <EndpointField label="Host" value={form.host} onChange={(value) => updateFormField("host", value)} placeholder="smtp.example.com" error={fieldErrors.host} />
                      <EndpointField label="Port" value={form.port} onChange={(value) => updateFormField("port", value)} placeholder="587" type="number" error={fieldErrors.port} />
                      <EndpointField label="Username" value={form.username} onChange={(value) => updateFormField("username", value)} placeholder="smtp_user" />
                      <EndpointField label="Password" value={form.password} onChange={(value) => updateFormField("password", value)} placeholder={mode === "edit" ? "Leave blank to keep current password" : "••••••••"} type="password" />
                      <EndpointField label="Priority" value={form.priority} onChange={(value) => updateFormField("priority", value)} placeholder="100" type="number" />
                      <EndpointField label="Weight" value={form.weight} onChange={(value) => updateFormField("weight", value)} placeholder="1" type="number" />
                    </div>

                    <FieldGroup label="Security mode">
                      <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
                        {[
                          { key: "none", label: "No TLS" },
                          { key: "starttls", label: "STARTTLS" },
                          { key: "tls", label: "TLS" },
                          { key: "mtls", label: "mTLS" },
                        ].map((tlsOption) => (
                          <ChoiceCard
                            key={tlsOption.key}
                            active={form.tlsMode === tlsOption.key}
                            title={tlsOption.label}
                            onClick={() =>
                              {
                                clearFieldErrors(["clientCert", "clientKey"]);
                                setForm((current) => ({
                                ...current,
                                tlsMode: tlsOption.key as TLSMode,
                                }));
                              }
                            }
                          />
                        ))}
                      </div>
                    </FieldGroup>

                    <FieldGroup label="Endpoint status">
                      <div className="grid gap-3 md:grid-cols-3">
                        {[
                          { key: "active", label: "Active" },
                          { key: "draining", label: "Draining" },
                          { key: "disabled", label: "Disabled" },
                        ].map((statusOption) => (
                          <ChoiceCard
                            key={statusOption.key}
                            active={form.status === statusOption.key}
                            title={statusOption.label}
                            onClick={() =>
                              setForm((current) => ({
                                ...current,
                                status: statusOption.key as EndpointStatus,
                              }))
                            }
                          />
                        ))}
                      </div>
                    </FieldGroup>

                    {(form.tlsMode === "tls" || form.tlsMode === "mtls") && (
                      <FieldGroup label="TLS material">
                      <div className="rounded-2xl border border-gray-200 bg-gray-50 p-4 dark:border-gray-800 dark:bg-gray-900/40">
                          <div className="space-y-4">
                            <div className="flex flex-wrap gap-2">
                              <ChoicePill
                                active={form.certInputMode === "manual"}
                                onClick={() =>
                                  updateFormField("certInputMode", "manual")
                                }
                              >
                                Manual cert input
                              </ChoicePill>
                              <ChoicePill
                                active={form.certInputMode === "upload"}
                                onClick={() =>
                                  updateFormField("certInputMode", "upload")
                                }
                              >
                                Drag / drop cert files
                              </ChoicePill>
                            </div>

                            {form.certInputMode === "manual" ? (
                              <div className="space-y-4">
                                <CertTextarea
                                  label="CA certificate"
                                  value={form.caCertContent}
                                  onChange={(value) =>
                                    updateFormField("caCertContent", value)
                                  }
                                  placeholder="-----BEGIN CERTIFICATE-----"
                                />
                                {form.tlsMode === "mtls" && (
                                  <>
                                    <CertTextarea
                                      label="Client certificate"
                                      value={form.clientCertContent}
                                      onChange={(value) =>
                                        updateFormField("clientCertContent", value)
                                      }
                                      placeholder="-----BEGIN CERTIFICATE-----"
                                      error={fieldErrors.clientCert}
                                    />
                                    <CertTextarea
                                      label="Client private key"
                                      value={form.clientKeyContent}
                                      onChange={(value) =>
                                        updateFormField("clientKeyContent", value)
                                      }
                                      placeholder="-----BEGIN PRIVATE KEY-----"
                                      error={fieldErrors.clientKey}
                                    />
                                  </>
                                )}
                              </div>
                            ) : (
                              <div className="space-y-4">
                                <CertDropzone
                                  label="CA certificate file"
                                  file={form.caCertFile}
                                  onFileChange={(file) =>
                                    updateFormField("caCertFile", file)
                                  }
                                />
                                {form.tlsMode === "mtls" && (
                                  <>
                                    <CertDropzone
                                      label="Client certificate file"
                                      file={form.clientCertFile}
                                      onFileChange={(file) =>
                                        updateFormField("clientCertFile", file)
                                      }
                                      error={fieldErrors.clientCert}
                                    />
                                    <CertDropzone
                                      label="Client private key file"
                                      file={form.clientKeyFile}
                                      onFileChange={(file) =>
                                        updateFormField("clientKeyFile", file)
                                      }
                                      error={fieldErrors.clientKey}
                                    />
                                  </>
                                )}
                              </div>
                            )}

                            {mode === "edit" && (
                              <p className="text-xs text-gray-500 dark:text-gray-400">
                                Leaving secret or certificate fields empty keeps the current stored
                                material. Switching to No TLS clears saved certificates. Switching
                                to mTLS requires both a client certificate and private key unless
                                the endpoint already has them stored.
                              </p>
                            )}
                          </div>
                      </div>
                    </FieldGroup>
                    )}
                  </div>

                  <div>
                    {connectResult != null && (
                      <div
                        className={`mb-4 rounded-2xl px-4 py-4 text-sm ${
                          connectResult.tone === "success"
                            ? "border border-success-200 bg-success-50 text-success-700 dark:border-success-500/30 dark:bg-success-500/10 dark:text-success-300"
                            : "border border-error-200 bg-error-50 text-error-700 dark:border-error-500/30 dark:bg-error-500/10 dark:text-error-300"
                        }`}
                      >
                        {connectResult.message}
                      </div>
                    )}
                    <div className="flex flex-wrap gap-3 pt-2">
                      <button
                        type="button"
                        onClick={() => void handleSave()}
                        disabled={isSaving}
                        className="inline-flex items-center rounded-xl bg-gray-900 px-4 py-2.5 text-sm font-semibold text-white transition hover:bg-gray-800 disabled:opacity-60 dark:bg-white dark:text-gray-900 dark:hover:bg-gray-200"
                      >
                        {isSaving ? "Saving..." : mode === "create" ? "Create Endpoint" : "Save Changes"}
                      </button>
                      <button
                        type="button"
                        onClick={() => void handleTryConnect()}
                        disabled={isTryingConnect}
                        className="inline-flex items-center rounded-xl border border-gray-200 bg-white px-4 py-2.5 text-sm font-semibold text-gray-700 transition hover:bg-gray-50 disabled:opacity-60 dark:border-gray-700 dark:bg-gray-900/40 dark:text-gray-200 dark:hover:bg-gray-900/80"
                      >
                        {isTryingConnect ? "Trying..." : "Try Connect"}
                      </button>
                      <button
                        type="button"
                        onClick={onView}
                        className="inline-flex items-center rounded-xl border border-gray-200 bg-white px-4 py-2.5 text-sm font-semibold text-gray-700 transition hover:bg-gray-50 dark:border-gray-700 dark:bg-gray-900/40 dark:text-gray-200 dark:hover:bg-gray-900/80"
                      >
                        Cancel
                      </button>
                    </div>
                  </div>
                </div>
              </div>
            ) : (
              <>
                {isDetailLoading && (
                  <div className="rounded-2xl border border-gray-200 bg-gray-50 px-4 py-5 text-sm text-gray-500 dark:border-gray-800 dark:bg-gray-900/40 dark:text-gray-400">
                    Loading endpoint detail...
                  </div>
                )}

                {!isDetailLoading && detailError !== "" && (
                  <div className="rounded-2xl border border-error-200 bg-error-50 px-4 py-5 text-sm text-error-700 dark:border-error-500/30 dark:bg-error-500/10 dark:text-error-300">
                    {detailError}
                  </div>
                )}

                {!isDetailLoading && detailError === "" && (
                  <>
                    <div className="grid gap-4 md:grid-cols-2">
                      <MetricChip label="Endpoint ID" value={detail?.id ?? "-"} />
                      <MetricChip label="TLS mode" value={detail?.tlsMode ?? "-"} />
                      <MetricChip label="Host" value={detail?.host ?? "-"} />
                      <MetricChip label="Port" value={detail != null ? String(detail.port) : "-"} />
                      <MetricChip label="Username" value={detail?.username ?? "-"} />
                      <MetricChip label="Status" value={detail?.status ?? "-"} />
                      <MetricChip label="CA cert" value={detail?.hasCACert ? "present" : "none"} />
                      <MetricChip label="Client cert" value={detail?.hasClientCert ? "present" : "none"} />
                      <MetricChip label="Client key" value={detail?.hasClientKey ? "present" : "none"} />
                      <MetricChip label="Priority / Weight" value={detail == null ? "-" : `${detail.priority} / ${detail.weight}`} />
                    </div>

                    <div className="mt-5 rounded-2xl border border-gray-200 bg-gray-50 p-4 dark:border-gray-800 dark:bg-gray-900/40">
                      <p className="text-xs font-medium tracking-[0.18em] text-gray-400 uppercase">
                        Runtime
                      </p>
                      <p className="mt-3 text-sm leading-6 text-gray-600 dark:text-gray-300">
                        {detail == null
                          ? "-"
                          : `Updated ${detail.updatedAt ?? "-"}, relay connects with ${detail.tlsMode.toUpperCase()} and caches credentials in memory for outbound delivery.`}
                      </p>
                    </div>
                  </>
                )}
              </>
            )}
          </ComponentCard>
        </div>
      </div>
    </div>
  );
}

function MetricChip({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-2xl bg-white px-4 py-3 shadow-theme-xs dark:bg-white/[0.04]">
      <p className="text-xs font-medium tracking-[0.18em] text-gray-400 uppercase">
        {label}
      </p>
      <p className="mt-2 text-sm font-semibold text-gray-800 dark:text-white">
        {value}
      </p>
    </div>
  );
}

function FieldGroup({
  label,
  children,
}: {
  label: string;
  children: React.ReactNode;
}) {
  return (
    <div className="space-y-2">
      <span className="text-sm font-medium text-gray-700 dark:text-gray-300">
        {label}
      </span>
      {children}
    </div>
  );
}

function EndpointField({
  label,
  value,
  onChange,
  placeholder,
  type = "text",
  error,
}: {
  label: string;
  value: string;
  onChange: (value: string) => void;
  placeholder: string;
  type?: string;
  error?: string;
}) {
  return (
    <label className="block space-y-2">
      <span className="text-sm font-medium text-gray-700 dark:text-gray-300">
        {label}
      </span>
      <input
        type={type}
        value={value}
        onChange={(event) => onChange(event.target.value)}
        placeholder={placeholder}
        className={`w-full rounded-2xl border bg-white px-4 py-3 text-sm text-gray-800 outline-none transition dark:bg-gray-900/40 dark:text-white ${
          error
            ? "border-error-400 focus:border-error-500 dark:border-error-500/70"
            : "border-gray-200 focus:border-gray-400 dark:border-gray-700"
        }`}
      />
      {error ? <span className="text-xs text-error-600 dark:text-error-300">{error}</span> : null}
    </label>
  );
}

function ChoiceCard({
  active,
  title,
  onClick,
}: {
  active: boolean;
  title: string;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={`rounded-2xl border px-4 py-4 text-left transition ${
        active
          ? "border-gray-900 bg-white shadow-theme-sm dark:border-white dark:bg-gray-900"
          : "border-gray-200 bg-gray-50 hover:border-gray-300 dark:border-gray-800 dark:bg-gray-900/40 dark:hover:border-gray-700"
      }`}
    >
      <p className="text-sm font-semibold text-gray-900 dark:text-white">{title}</p>
    </button>
  );
}

function ChoicePill({
  active,
  onClick,
  children,
}: {
  active: boolean;
  onClick: () => void;
  children: React.ReactNode;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={`rounded-full border px-3 py-2 text-xs font-semibold transition ${
        active
          ? "border-gray-900 bg-gray-900 text-white dark:border-white dark:bg-white dark:text-gray-900"
          : "border-gray-200 bg-white text-gray-600 hover:border-gray-300 dark:border-gray-700 dark:bg-gray-900/40 dark:text-gray-300"
      }`}
    >
      {children}
    </button>
  );
}

function CertTextarea({
  label,
  value,
  onChange,
  placeholder,
  error,
}: {
  label: string;
  value: string;
  onChange: (value: string) => void;
  placeholder: string;
  error?: string;
}) {
  return (
    <div className="space-y-2">
      <span className="text-xs font-medium tracking-[0.12em] text-gray-400 uppercase">
        {label}
      </span>
      <textarea
        rows={5}
        value={value}
        onChange={(event) => onChange(event.target.value)}
        placeholder={placeholder}
        className={`w-full rounded-2xl border bg-white px-4 py-3 text-sm text-gray-800 outline-none transition dark:bg-gray-900/40 dark:text-white ${
          error
            ? "border-error-400 focus:border-error-500 dark:border-error-500/70"
            : "border-gray-200 focus:border-gray-400 dark:border-gray-700"
        }`}
      />
      {error ? <p className="text-xs text-error-600 dark:text-error-300">{error}</p> : null}
    </div>
  );
}

function CertDropzone({
  label,
  file,
  onFileChange,
  error,
}: {
  label: string;
  file: File | null;
  onFileChange: (file: File | null) => void;
  error?: string;
}) {
  const { getRootProps, getInputProps, isDragActive } = useDropzone({
    onDrop: (acceptedFiles) => onFileChange(acceptedFiles[0] ?? null),
    multiple: false,
  });

  return (
    <div className="space-y-2">
      <span className="text-xs font-medium tracking-[0.12em] text-gray-400 uppercase">
        {label}
      </span>
      <div
        {...getRootProps()}
        className={`rounded-2xl border border-dashed p-5 text-center transition ${
          isDragActive
            ? "border-gray-900 bg-gray-100 dark:border-white dark:bg-gray-800"
            : error
              ? "border-error-400 bg-white hover:border-error-500 dark:border-error-500/70 dark:bg-gray-900/40 dark:hover:border-error-400"
              : "border-gray-300 bg-white hover:border-gray-400 dark:border-gray-700 dark:bg-gray-900/40 dark:hover:border-gray-500"
        }`}
      >
        <input {...getInputProps()} />
        <p className="text-sm font-medium text-gray-700 dark:text-gray-300">
          Drag and drop a cert file here
        </p>
        <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
          or click to browse PEM / CRT / KEY files
        </p>
        {file != null && (
          <div className="mt-3 rounded-xl bg-gray-50 px-3 py-2 text-sm text-gray-700 dark:bg-gray-800/80 dark:text-gray-200">
            {file.name}
          </div>
        )}
      </div>
      {error ? <p className="text-xs text-error-600 dark:text-error-300">{error}</p> : null}
    </div>
  );
}

function endpointFieldErrorKey(field: keyof EndpointFormState): EndpointFieldErrorKey | null {
  switch (field) {
    case "name":
      return "name";
    case "host":
      return "host";
    case "port":
      return "port";
    case "clientCertContent":
    case "clientCertFile":
      return "clientCert";
    case "clientKeyContent":
    case "clientKeyFile":
      return "clientKey";
    default:
      return null;
  }
}

function validateEndpointForm(
  form: EndpointFormState,
  options: { requireName: boolean; existingDetail?: DeliveryEndpoint | null },
): EndpointFieldErrors {
  const errors: EndpointFieldErrors = {};

  if (options.requireName && form.name.trim() === "") {
    errors.name = "Name is required.";
  }
  if (form.host.trim() === "") {
    errors.host = "Host is required.";
  }

  const port = Number(form.port.trim());
  if (form.port.trim() === "") {
    errors.port = "Port is required.";
  } else if (!Number.isFinite(port) || port <= 0) {
    errors.port = "Port must be a positive number.";
  }

  if (form.tlsMode === "mtls") {
    const hasClientCert =
      form.certInputMode === "manual"
        ? form.clientCertContent.trim() !== ""
        : form.clientCertFile != null;
    const hasClientKey =
      form.certInputMode === "manual"
        ? form.clientKeyContent.trim() !== ""
        : form.clientKeyFile != null;

    if (!hasClientCert && !options.existingDetail?.hasClientCert) {
      errors.clientCert = "Client certificate is required for mTLS.";
    }
    if (!hasClientKey && !options.existingDetail?.hasClientKey) {
      errors.clientKey = "Client private key is required for mTLS.";
    }
  }

  return errors;
}

function StatusDot({
  status,
}: {
  status: EndpointStatus;
}) {
  const tone =
    status === "active"
      ? "bg-success-500"
      : status === "draining"
        ? "bg-warning-500"
        : "bg-error-500";

  return <span className={`inline-flex h-2.5 w-2.5 rounded-full ${tone}`} />;
}

function emptyEndpointForm(): EndpointFormState {
  return {
    name: "",
    host: "",
    port: "587",
    username: "",
    password: "",
    priority: "100",
    weight: "1",
    tlsMode: "starttls",
    status: "active",
    certInputMode: "manual",
    caCertContent: "",
    clientCertContent: "",
    clientKeyContent: "",
    caCertFile: null,
    clientCertFile: null,
    clientKeyFile: null,
  };
}

async function buildEndpointPayload(form: EndpointFormState): Promise<Record<string, unknown>> {
  const caCertPEM =
    form.certInputMode === "manual"
      ? form.caCertContent.trim()
      : await readOptionalFile(form.caCertFile);
  const clientCertPEM =
    form.certInputMode === "manual"
      ? form.clientCertContent.trim()
      : await readOptionalFile(form.clientCertFile);
  const clientKeyPEM =
    form.certInputMode === "manual"
      ? form.clientKeyContent.trim()
      : await readOptionalFile(form.clientKeyFile);

  return {
    name: form.name.trim(),
    provider_kind: "smtp",
    host: form.host.trim(),
    port: parseInteger(form.port, 587),
    username: form.username.trim(),
    password: form.password,
    priority: parseInteger(form.priority, 100),
    weight: parseInteger(form.weight, 1),
    max_connections: 10,
    max_parallel_sends: 20,
    max_messages_per_second: 0,
    burst: 0,
    warmup_state: "stable",
    status: form.status,
    tls_mode: form.tlsMode,
    ca_cert_pem: form.tlsMode === "none" || form.tlsMode === "starttls" ? "" : caCertPEM,
    client_cert_pem: form.tlsMode === "mtls" ? clientCertPEM : "",
    client_key_pem: form.tlsMode === "mtls" ? clientKeyPEM : "",
  };
}

function endpointToForm(endpoint: DeliveryEndpoint): EndpointFormState {
  return {
    name: endpoint.name,
    host: endpoint.host,
    port: String(endpoint.port),
    username: endpoint.username,
    password: "",
    priority: String(endpoint.priority),
    weight: String(endpoint.weight),
    tlsMode: endpoint.tlsMode,
    status: endpoint.status,
    certInputMode: "manual",
    caCertContent: "",
    clientCertContent: "",
    clientKeyContent: "",
    caCertFile: null,
    clientCertFile: null,
    clientKeyFile: null,
  };
}

async function readOptionalFile(file: File | null): Promise<string> {
  if (file == null) {
    return "";
  }
  return file.text();
}

function parseInteger(value: string, fallback: number): number {
  const parsed = Number.parseInt(value, 10);
  if (Number.isFinite(parsed)) {
    return parsed;
  }
  return fallback;
}
