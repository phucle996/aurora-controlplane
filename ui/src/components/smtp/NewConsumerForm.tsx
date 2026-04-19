"use client";

import React, { useMemo, useState } from "react";
import { useRouter } from "next/navigation";
import { useDropzone } from "react-dropzone";
import { parseAPIError } from "@/components/auth/auth-utils";
import ComponentCard from "@/components/common/ComponentCard";
import PageBreadcrumb from "@/components/common/PageBreadCrumb";

type TransportID = "redis-stream" | "rabbitmq" | "kafka" | "nats";
type TLSMode = "disabled" | "tls" | "mtls";
type CertMode = "manual" | "upload";
type ConsumerStatus = "active" | "disabled";

type ConsumerFormState = {
  name: string;
  batchSize: string;
  status: ConsumerStatus;
  note: string;
  transportType: TransportID;
  tlsMode: TLSMode;
  certMode: CertMode;
  host: string;
  port: string;
  source: string;
  consumerGroup: string;
  username: string;
  password: string;
  brokers: string;
  saslMechanism: string;
  serverURL: string;
  authToken: string;
  exchange: string;
  caCertContent: string;
  clientCertContent: string;
  clientKeyContent: string;
  caCertFile: File | null;
  clientCertFile: File | null;
  clientKeyFile: File | null;
};

type SubmitTone = "success" | "error";

const transportOptions: {
  id: TransportID;
  name: string;
  logo: string;
  accent: string;
}[] = [
  { id: "redis-stream", name: "Redis Stream", logo: "RS", accent: "from-red-500 to-orange-400" },
  { id: "rabbitmq", name: "RabbitMQ", logo: "RQ", accent: "from-orange-500 to-amber-400" },
  { id: "kafka", name: "Kafka", logo: "KF", accent: "from-gray-900 to-gray-700" },
  { id: "nats", name: "NATS", logo: "NT", accent: "from-sky-500 to-cyan-400" },
];

const transportFieldMap: Record<
  TransportID,
  Array<{ key: keyof ConsumerFormState; label: string; placeholder: string; secret?: boolean }>
> = {
  "redis-stream": [
    { key: "host", label: "Host", placeholder: "localhost" },
    { key: "port", label: "Port", placeholder: "6379" },
    { key: "source", label: "Stream key", placeholder: "smtp.outbound" },
    { key: "consumerGroup", label: "Consumer group", placeholder: "smtp-delivery-workers" },
    { key: "username", label: "Username", placeholder: "default", secret: true },
    { key: "password", label: "Password", placeholder: "••••••••", secret: true },
  ],
  rabbitmq: [
    { key: "host", label: "Host", placeholder: "localhost" },
    { key: "port", label: "Port", placeholder: "5672" },
    { key: "source", label: "Queue", placeholder: "smtp.outbound" },
    { key: "exchange", label: "Exchange", placeholder: "smtp.exchange" },
    { key: "username", label: "Username", placeholder: "smtp_user", secret: true },
    { key: "password", label: "Password", placeholder: "••••••••", secret: true },
  ],
  kafka: [
    { key: "brokers", label: "Brokers", placeholder: "kafka-1:9092,kafka-2:9092" },
    { key: "source", label: "Topic", placeholder: "smtp.outbound" },
    { key: "consumerGroup", label: "Consumer group", placeholder: "smtp-consumers" },
    { key: "saslMechanism", label: "SASL mechanism", placeholder: "SCRAM-SHA-256" },
    { key: "username", label: "Username", placeholder: "smtp_user", secret: true },
    { key: "password", label: "Password", placeholder: "••••••••", secret: true },
  ],
  nats: [
    { key: "serverURL", label: "Server URL", placeholder: "nats://localhost:4222" },
    { key: "source", label: "Subject", placeholder: "smtp.outbound" },
    { key: "consumerGroup", label: "Queue group", placeholder: "smtp-workers" },
    { key: "authToken", label: "Auth token", placeholder: "token-or-jwt", secret: true },
  ],
};

const requiredFieldsByTransport: Record<TransportID, Array<keyof ConsumerFormState>> = {
  "redis-stream": ["host", "port", "source", "consumerGroup"],
  rabbitmq: ["host", "port", "source", "username", "password"],
  kafka: ["brokers", "source", "consumerGroup"],
  nats: ["serverURL", "source"],
};

function emptyConsumerForm(): ConsumerFormState {
  return {
    name: "",
    batchSize: "128",
    status: "active",
    note: "",
    transportType: "redis-stream",
    tlsMode: "disabled",
    certMode: "manual",
    host: "",
    port: "",
    source: "",
    consumerGroup: "",
    username: "",
    password: "",
    brokers: "",
    saslMechanism: "",
    serverURL: "",
    authToken: "",
    exchange: "",
    caCertContent: "",
    clientCertContent: "",
    clientKeyContent: "",
    caCertFile: null,
    clientCertFile: null,
    clientKeyFile: null,
  };
}

export default function NewConsumerForm() {
  const router = useRouter();
  const [form, setForm] = useState<ConsumerFormState>(emptyConsumerForm());
  const [fieldErrors, setFieldErrors] = useState<Partial<Record<keyof ConsumerFormState, string>>>(
    {},
  );
  const [submitResult, setSubmitResult] = useState<{
    tone: SubmitTone;
    message: string;
  } | null>(null);
  const [isSaving, setIsSaving] = useState(false);
  const [isTryingConnect, setIsTryingConnect] = useState(false);

  const connectionFields = useMemo(
    () => transportFieldMap[form.transportType],
    [form.transportType],
  );

  function updateField<K extends keyof ConsumerFormState>(
    key: K,
    value: ConsumerFormState[K],
  ) {
    setForm((current) => ({ ...current, [key]: value }));
    setSubmitResult(null);
    setFieldErrors((current) => {
      if (current[key] == null) {
        return current;
      }
      const next = { ...current };
      delete next[key];
      return next;
    });
  }

  async function validateAndBuildPayload() {
    const nextErrors: Partial<Record<keyof ConsumerFormState, string>> = {};

    if (form.name.trim() === "") {
      nextErrors.name = "Consumer name is required";
    }

    for (const key of requiredFieldsByTransport[form.transportType]) {
      if (String(form[key] ?? "").trim() === "") {
        nextErrors[key] = "This field is required";
      }
    }

    if (Number.parseInt(form.batchSize, 10) <= 0) {
      nextErrors.batchSize = "Batch size must be greater than 0";
    }

    if (Object.keys(nextErrors).length > 0) {
      setFieldErrors(nextErrors);
      return null;
    }

    const caCert = await resolveCertContent(form.caCertContent, form.caCertFile);
    const clientCert = await resolveCertContent(form.clientCertContent, form.clientCertFile);
    const clientKey = await resolveCertContent(form.clientKeyContent, form.clientKeyFile);

    const connectionConfig: Record<string, unknown> = {};
    const secretConfig: Record<string, unknown> = {};

    switch (form.transportType) {
      case "redis-stream":
        connectionConfig.host = form.host.trim();
        connectionConfig.port = form.port.trim();
        connectionConfig.addr = `${form.host.trim()}:${form.port.trim()}`;
        secretConfig.username = form.username.trim();
        secretConfig.password = form.password;
        break;
      case "rabbitmq":
        connectionConfig.host = form.host.trim();
        connectionConfig.port = form.port.trim();
        connectionConfig.exchange = form.exchange.trim();
        secretConfig.username = form.username.trim();
        secretConfig.password = form.password;
        break;
      case "kafka":
        connectionConfig.brokers = form.brokers.trim();
        connectionConfig.sasl_mechanism = form.saslMechanism.trim();
        secretConfig.username = form.username.trim();
        secretConfig.password = form.password;
        break;
      case "nats":
        connectionConfig.server_url = form.serverURL.trim();
        secretConfig.auth_token = form.authToken.trim();
        break;
    }

    connectionConfig.tls_mode = form.tlsMode;
    if (caCert.trim() !== "") {
      secretConfig.ca_cert_pem = caCert.trim();
    }
    if (form.tlsMode === "mtls") {
      if (clientCert.trim() !== "") {
        secretConfig.client_cert_pem = clientCert.trim();
      }
      if (clientKey.trim() !== "") {
        secretConfig.client_key_pem = clientKey.trim();
      }
    }

    return {
      name: form.name.trim(),
      transport_type: form.transportType,
      source: form.source.trim(),
      consumer_group: form.consumerGroup.trim(),
      batch_size: Number.parseInt(form.batchSize, 10),
      status: form.status,
      note: form.note.trim(),
      connection_config: connectionConfig,
      secret_config: secretConfig,
    };
  }

  async function handleSave() {
    const payload = await validateAndBuildPayload();
    if (payload == null) {
      setSubmitResult(null);
      return;
    }

    setIsSaving(true);
    setSubmitResult(null);

    try {
      const response = await fetch("/api/v1/smtp/consumers", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify(payload),
      });

      if (!response.ok) {
        throw new Error(await parseAPIError(response));
      }

      setSubmitResult({
        tone: "success",
        message: "Consumer saved successfully.",
      });
      router.push("/smtp/consumers");
    } catch (err) {
      setSubmitResult({
        tone: "error",
        message: err instanceof Error ? err.message : "Failed to save consumer.",
      });
    } finally {
      setIsSaving(false);
    }
  }

  async function handleTryConnect() {
    const payload = await validateAndBuildPayload();
    if (payload == null) {
      setSubmitResult(null);
      return;
    }

    setIsTryingConnect(true);
    setSubmitResult(null);

    try {
      const response = await fetch("/api/v1/smtp/consumers/try-connect", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify(payload),
      });

      if (!response.ok) {
        throw new Error(await parseAPIError(response));
      }

      const result = (await response.json().catch(() => null)) as { message?: string } | null;
      setSubmitResult({
        tone: "success",
        message:
          typeof result?.message === "string" && result.message.trim() !== ""
            ? result.message
            : "Consumer connection succeeded.",
      });
    } catch (err) {
      setSubmitResult({
        tone: "error",
        message: err instanceof Error ? err.message : "Consumer connection failed.",
      });
    } finally {
      setIsTryingConnect(false);
    }
  }

  return (
    <div className="space-y-6">
      <PageBreadcrumb pageTitle="New Consumer" />

      <ComponentCard
        title="Create Consumer"
        desc="Create a live SMTP consumer and verify the transport connection before enabling runtime delivery."
      >
        <div className="grid gap-6 xl:grid-cols-[1.05fr_0.95fr]">
          <div className="space-y-5">
            <Field label="Consumer name">
              <TextInput
                placeholder="smtp.redis.primary"
                value={form.name}
                onChange={(value) => updateField("name", value)}
                error={fieldErrors.name}
              />
            </Field>

            <Field label="Prefetch / batch size">
              <TextInput
                type="number"
                placeholder="128"
                value={form.batchSize}
                onChange={(value) => updateField("batchSize", value)}
                error={fieldErrors.batchSize}
              />
            </Field>

            <Field label="Status">
              <SelectInput
                value={form.status}
                onChange={(value) => updateField("status", value as ConsumerStatus)}
                options={[
                  { value: "active", label: "Active" },
                  { value: "disabled", label: "Disabled" },
                ]}
              />
            </Field>

            <Field label="Notes">
              <TextArea
                rows={8}
                placeholder="Primary ingress for transactional email jobs."
                value={form.note}
                onChange={(value) => updateField("note", value)}
              />
            </Field>
          </div>

          <div className="space-y-5">
            <Field label="Transport type">
              <div className="grid gap-3 md:grid-cols-2">
                {transportOptions.map((option) => {
                  const active = option.id === form.transportType;
                  return (
                    <button
                      key={option.id}
                      type="button"
                      onClick={() => updateField("transportType", option.id)}
                      className={`rounded-2xl border bg-white p-4 text-left transition dark:bg-gray-900/40 ${
                        active
                          ? "border-gray-900 shadow-theme-sm dark:border-white"
                          : "border-gray-200 hover:border-gray-300 dark:border-gray-700 dark:hover:border-gray-500"
                      }`}
                    >
                      <div className="flex items-center gap-4">
                        <span
                          className={`inline-flex h-11 w-11 shrink-0 items-center justify-center rounded-2xl bg-gradient-to-br ${option.accent} text-sm font-semibold text-white`}
                        >
                          {option.logo}
                        </span>
                        <p className="text-sm font-semibold text-gray-900 dark:text-white">
                          {option.name}
                        </p>
                      </div>
                    </button>
                  );
                })}
              </div>
            </Field>

            <Field label="Connection settings">
              <div className="grid gap-4 md:grid-cols-2">
                {connectionFields.map((field) => (
                  <div key={field.label} className="space-y-2">
                    <span className="text-xs font-medium tracking-[0.12em] text-gray-400 uppercase">
                      {field.label}
                    </span>
                    <TextInput
                      type={field.secret ? "password" : field.key === "port" ? "number" : "text"}
                      placeholder={field.placeholder}
                      value={String(form[field.key] ?? "")}
                      onChange={(value) =>
                        updateField(field.key, value as ConsumerFormState[typeof field.key])
                      }
                      error={fieldErrors[field.key]}
                    />
                  </div>
                ))}
              </div>
            </Field>

            <Field label="Security">
              <div className="space-y-4 rounded-2xl border border-gray-200 bg-gray-50 p-4 dark:border-gray-800 dark:bg-gray-900/40">
                <div className="flex flex-wrap gap-2">
                  <ChoicePill
                    active={form.tlsMode === "disabled"}
                    onClick={() => updateField("tlsMode", "disabled")}
                  >
                    No TLS
                  </ChoicePill>
                  <ChoicePill
                    active={form.tlsMode === "tls"}
                    onClick={() => updateField("tlsMode", "tls")}
                  >
                    TLS
                  </ChoicePill>
                  <ChoicePill
                    active={form.tlsMode === "mtls"}
                    onClick={() => updateField("tlsMode", "mtls")}
                  >
                    mTLS
                  </ChoicePill>
                </div>

                {form.tlsMode !== "disabled" ? (
                  <>
                    <div className="flex flex-wrap gap-2">
                      <ChoicePill
                        active={form.certMode === "manual"}
                        onClick={() => updateField("certMode", "manual")}
                      >
                        Manual cert input
                      </ChoicePill>
                      <ChoicePill
                        active={form.certMode === "upload"}
                        onClick={() => updateField("certMode", "upload")}
                      >
                        Drag / drop cert files
                      </ChoicePill>
                    </div>

                    {form.certMode === "manual" ? (
                      <div className="space-y-4">
                        <CertTextarea
                          label="CA certificate"
                          placeholder="-----BEGIN CERTIFICATE-----"
                          value={form.caCertContent}
                          onChange={(value) => updateField("caCertContent", value)}
                        />
                        {form.tlsMode === "mtls" ? (
                          <>
                            <CertTextarea
                              label="Client certificate"
                              placeholder="-----BEGIN CERTIFICATE-----"
                              value={form.clientCertContent}
                              onChange={(value) => updateField("clientCertContent", value)}
                            />
                            <CertTextarea
                              label="Client private key"
                              placeholder="-----BEGIN PRIVATE KEY-----"
                              value={form.clientKeyContent}
                              onChange={(value) => updateField("clientKeyContent", value)}
                            />
                          </>
                        ) : null}
                      </div>
                    ) : (
                      <div className="space-y-4">
                        <CertDropzone
                          label="CA certificate file"
                          file={form.caCertFile}
                          onChange={(file) => updateField("caCertFile", file)}
                        />
                        {form.tlsMode === "mtls" ? (
                          <>
                            <CertDropzone
                              label="Client certificate file"
                              file={form.clientCertFile}
                              onChange={(file) => updateField("clientCertFile", file)}
                            />
                            <CertDropzone
                              label="Client private key file"
                              file={form.clientKeyFile}
                              onChange={(file) => updateField("clientKeyFile", file)}
                            />
                          </>
                        ) : null}
                      </div>
                    )}
                  </>
                ) : null}
              </div>
            </Field>
          </div>

          <div className="xl:col-span-2">
            {submitResult != null ? (
              <div
                className={`mb-4 rounded-2xl border px-4 py-3 text-sm ${
                  submitResult.tone === "success"
                    ? "border-success-200 bg-success-50 text-success-700"
                    : "border-error-200 bg-error-50 text-error-700"
                }`}
              >
                {submitResult.message}
              </div>
            ) : null}

            <div className="flex flex-wrap gap-3 pt-2">
              <button
                type="button"
                disabled={isSaving}
                onClick={() => void handleSave()}
                className="inline-flex items-center rounded-xl bg-gray-900 px-4 py-2.5 text-sm font-semibold text-white transition hover:bg-gray-800 disabled:cursor-not-allowed disabled:opacity-60 dark:bg-white dark:text-gray-900 dark:hover:bg-gray-200"
              >
                {isSaving ? "Saving..." : "Save Consumer"}
              </button>
              <button
                type="button"
                disabled={isTryingConnect}
                onClick={() => void handleTryConnect()}
                className="inline-flex items-center rounded-xl border border-gray-200 bg-white px-4 py-2.5 text-sm font-semibold text-gray-700 transition hover:bg-gray-50 disabled:cursor-not-allowed disabled:opacity-60 dark:border-gray-700 dark:bg-gray-900/40 dark:text-gray-200 dark:hover:bg-gray-900/80"
              >
                {isTryingConnect ? "Trying..." : "Try Connect"}
              </button>
            </div>
          </div>
        </div>
      </ComponentCard>
    </div>
  );
}

function Field({
  label,
  children,
}: {
  label: string;
  children: React.ReactNode;
}) {
  return (
    <label className="block space-y-2">
      <span className="text-sm font-medium text-gray-700 dark:text-gray-300">
        {label}
      </span>
      {children}
    </label>
  );
}

function TextInput({
  placeholder,
  type = "text",
  value,
  onChange,
  error,
}: {
  placeholder: string;
  type?: string;
  value: string;
  onChange: (value: string) => void;
  error?: string;
}) {
  return (
    <div className="space-y-2">
      <input
        type={type}
        placeholder={placeholder}
        value={value}
        onChange={(event) => onChange(event.target.value)}
        className="w-full rounded-2xl border border-gray-200 bg-white px-4 py-3 text-sm text-gray-800 outline-none transition focus:border-gray-400 dark:border-gray-700 dark:bg-gray-900/40 dark:text-white"
      />
      {error != null ? <p className="text-xs text-error-600">{error}</p> : null}
    </div>
  );
}

function TextArea({
  placeholder,
  rows,
  value,
  onChange,
}: {
  placeholder: string;
  rows: number;
  value: string;
  onChange: (value: string) => void;
}) {
  return (
    <textarea
      rows={rows}
      placeholder={placeholder}
      value={value}
      onChange={(event) => onChange(event.target.value)}
      className="w-full rounded-2xl border border-gray-200 bg-white px-4 py-3 text-sm text-gray-800 outline-none transition focus:border-gray-400 dark:border-gray-700 dark:bg-gray-900/40 dark:text-white"
    />
  );
}

function SelectInput({
  value,
  onChange,
  options,
}: {
  value: string;
  onChange: (value: string) => void;
  options: Array<{ value: string; label: string }>;
}) {
  return (
    <select
      value={value}
      onChange={(event) => onChange(event.target.value)}
      className="w-full rounded-2xl border border-gray-200 bg-white px-4 py-3 text-sm text-gray-800 outline-none transition focus:border-gray-400 dark:border-gray-700 dark:bg-gray-900/40 dark:text-white"
    >
      {options.map((option) => (
        <option key={option.value} value={option.value}>
          {option.label}
        </option>
      ))}
    </select>
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
  placeholder,
  value,
  onChange,
}: {
  label: string;
  placeholder: string;
  value: string;
  onChange: (value: string) => void;
}) {
  return (
    <div className="space-y-2">
      <span className="text-xs font-medium tracking-[0.12em] text-gray-400 uppercase">
        {label}
      </span>
      <textarea
        rows={5}
        placeholder={placeholder}
        value={value}
        onChange={(event) => onChange(event.target.value)}
        className="w-full rounded-2xl border border-gray-200 bg-white px-4 py-3 text-sm text-gray-800 outline-none transition focus:border-gray-400 dark:border-gray-700 dark:bg-gray-900/40 dark:text-white"
      />
    </div>
  );
}

function CertDropzone({
  label,
  file,
  onChange,
}: {
  label: string;
  file: File | null;
  onChange: (file: File | null) => void;
}) {
  const { getRootProps, getInputProps, isDragActive } = useDropzone({
    onDrop: (acceptedFiles) => onChange(acceptedFiles[0] ?? null),
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
        {file != null ? (
          <div className="mt-3 rounded-xl bg-gray-50 px-3 py-2 text-sm text-gray-700 dark:bg-gray-800/80 dark:text-gray-200">
            {file.name}
          </div>
        ) : null}
      </div>
    </div>
  );
}

async function resolveCertContent(manual: string, file: File | null): Promise<string> {
  if (manual.trim() !== "") {
    return manual;
  }
  if (file == null) {
    return "";
  }
  return file.text();
}
