"use client";

import { useState } from "react";
import { SMTPPageShell } from "@/components/smtp/SMTPPageShell";
import { GeneralTab } from "@/components/smtp/GeneralTab";
import { ConsumerTab } from "@/components/smtp/ConsumerTab";
import { LaneTab } from "@/components/smtp/LaneTab";
import { EndpointTab } from "@/components/smtp/EndpointTab";
import { TemplateTab } from "@/components/smtp/TemplateTab";

export default function SMTPGeneralPage() {
  return (
    <SMTPPageShell>
      <GeneralTab />
    </SMTPPageShell>
  );
}

export function SMTPConsumersPage() {
  const [consumerSearch, setConsumerSearch] = useState("");

  return (
    <SMTPPageShell>
      <ConsumerTab
        search={consumerSearch}
        onSearchChange={setConsumerSearch}
      />
    </SMTPPageShell>
  );
}

export function SMTPTemplatesPage() {
  const [templateSearch, setTemplateSearch] = useState("");

  return (
    <SMTPPageShell>
      <TemplateTab
        search={templateSearch}
        onSearchChange={setTemplateSearch}
      />
    </SMTPPageShell>
  );
}

export function SMTPLanesPage() {
  return (
    <SMTPPageShell>
      <LaneTab />
    </SMTPPageShell>
  );
}

export function SMTPEndpointsPage() {
  const [endpointSearch, setEndpointSearch] = useState("");
  const [endpointMode, setEndpointMode] = useState<"view" | "create" | "edit">("view");
  const [selectedEndpointId, setSelectedEndpointId] = useState<string>("");

  return (
    <SMTPPageShell>
      <EndpointTab
        search={endpointSearch}
        onSearchChange={setEndpointSearch}
        mode={endpointMode}
        onCreate={() => setEndpointMode("create")}
        onEdit={() => setEndpointMode("edit")}
        onView={() => setEndpointMode("view")}
        selectedEndpointId={selectedEndpointId}
        onSelectEndpoint={setSelectedEndpointId}
      />
    </SMTPPageShell>
  );
}
