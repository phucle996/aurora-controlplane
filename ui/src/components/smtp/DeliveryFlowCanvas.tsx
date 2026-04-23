"use client";

import { useEffect, useMemo, useState } from "react";
import { createPortal } from "react-dom";
import {
  Background,
  Handle,
  MarkerType,
  Position,
  ReactFlow,
  type Edge,
  type Node,
  type NodeProps,
} from "@xyflow/react";
import "@xyflow/react/dist/style.css";
import type { DeliveryEndpoint, TemplateItem } from "@/components/smtp/types";

type DeliveryFlowCanvasProps = {
  gatewayName: string;
  gatewayStatus: string;
  trafficClass: string;
  fallbackGatewayName?: string;
  fallbackGatewayStatus?: string;
  selectedTemplates: TemplateItem[];
  selectedEndpoints: DeliveryEndpoint[];
  availableTemplates: TemplateItem[];
  availableEndpoints: DeliveryEndpoint[];
  onSaveTemplates: (templateIDs: string[]) => Promise<void>;
  onSaveEndpoints: (endpointIDs: string[]) => Promise<void>;
  isSavingTemplates: boolean;
  isSavingEndpoints: boolean;
};

type SelectionNodeData = {
  eyebrow: string;
  subtitle: string;
  emptyText: string;
  side: "left" | "right";
  items: Array<{ id: string; title: string; meta: string }>;
};

type GatewayNodeData = {
  gatewayName: string;
  gatewayStatus: string;
  trafficClass: string;
  kind: "primary" | "fallback";
};

const nodeTypes = {
  selection: SelectionFlowNode,
  gateway: GatewayFlowNode,
};

export function DeliveryFlowCanvas({
  gatewayName,
  gatewayStatus,
  trafficClass,
  fallbackGatewayName,
  fallbackGatewayStatus,
  selectedTemplates,
  selectedEndpoints,
  availableTemplates,
  availableEndpoints,
  onSaveTemplates,
  onSaveEndpoints,
  isSavingTemplates,
  isSavingEndpoints,
}: DeliveryFlowCanvasProps) {
  const [isTemplateDrawerOpen, setIsTemplateDrawerOpen] = useState(false);
  const [isEndpointDrawerOpen, setIsEndpointDrawerOpen] = useState(false);
  const [templateDraftIDs, setTemplateDraftIDs] = useState<string[]>([]);
  const [endpointDraftIDs, setEndpointDraftIDs] = useState<string[]>([]);

  const selectedTemplateSet = useMemo(() => new Set(templateDraftIDs), [templateDraftIDs]);
  const selectedEndpointSet = useMemo(() => new Set(endpointDraftIDs), [endpointDraftIDs]);

  const templateItems = useMemo(
    () =>
      selectedTemplates.map((item) => ({
        id: item.id,
        title: item.name,
        meta: item.consumer === "none" ? item.status : `${item.consumer} · ${item.status}`,
      })),
    [selectedTemplates],
  );

  const endpointItems = useMemo(
    () =>
      selectedEndpoints.map((item) => ({
        id: item.id,
        title: item.name,
        meta: `${item.host}:${item.port} · ${item.status}`,
      })),
    [selectedEndpoints],
  );

  const isFallbackServing = Boolean(
    fallbackGatewayName &&
      normalizeStatus(fallbackGatewayStatus) === "active" &&
      normalizeStatus(gatewayStatus) !== "active",
  );

  const nodes = useMemo<Node[]>(
    () => {
      const baseNodes: Node[] = [
        {
          id: "templates",
          type: "selection",
          position: { x: 30, y: 125 },
          draggable: false,
          selectable: false,
          data: {
            eyebrow: "Templates",
            subtitle: `${selectedTemplates.length} selected`,
            emptyText: "No template is routed into this gateway yet.",
            side: "left",
            items: templateItems,
          } satisfies SelectionNodeData,
        },
        {
          id: "lane",
          type: "gateway",
          position: { x: 455, y: 120 },
          draggable: false,
          selectable: false,
          data: {
            gatewayName,
            gatewayStatus,
            trafficClass,
            kind: "primary",
          } satisfies GatewayNodeData,
        },
        {
          id: "endpoints",
          type: "selection",
          position: { x: 860, y: 125 },
          draggable: false,
          selectable: false,
          data: {
            eyebrow: "Endpoints",
            subtitle: `${selectedEndpoints.length} selected`,
            emptyText: "No endpoint is attached to this gateway yet.",
            side: "right",
            items: endpointItems,
          } satisfies SelectionNodeData,
        },
      ];

      if (fallbackGatewayName) {
        baseNodes.push({
          id: "fallback",
          type: "gateway",
          position: { x: 480, y: 360 },
          draggable: false,
          selectable: false,
          data: {
            gatewayName: fallbackGatewayName,
            gatewayStatus: fallbackGatewayStatus || "disabled",
            trafficClass,
            kind: "fallback",
          } satisfies GatewayNodeData,
        });
      }

      return baseNodes;
    },
    [
      endpointItems,
      fallbackGatewayName,
      fallbackGatewayStatus,
      gatewayName,
      gatewayStatus,
      selectedEndpoints.length,
      selectedTemplates.length,
      templateItems,
      trafficClass,
    ],
  );

  const edges = useMemo<Edge[]>(
    () => {
      const flowEdges: Edge[] = [
        {
          id: "templates-to-lane",
          source: "templates",
          target: "lane",
          sourceHandle: "out",
          targetHandle: "in-left",
          animated: selectedTemplates.length > 0,
          markerEnd: { type: MarkerType.ArrowClosed, width: 20, height: 20 },
          style: { stroke: "#94a3b8", strokeWidth: 2.4 },
        },
      ];

      if (fallbackGatewayName) {
        flowEdges.push({
          id: "gateway-to-fallback",
          source: "lane",
          target: "fallback",
          sourceHandle: "out-bottom",
          targetHandle: "in-top",
          animated: isFallbackServing,
          markerEnd: { type: MarkerType.ArrowClosed, width: 20, height: 20 },
          style: {
            stroke: isFallbackServing ? "#0f766e" : "#cbd5e1",
            strokeWidth: isFallbackServing ? 3 : 2.2,
          },
        });
      }

      flowEdges.push({
        id: "lane-to-endpoints",
        source: isFallbackServing ? "fallback" : "lane",
        target: "endpoints",
        sourceHandle: isFallbackServing ? "out-right" : "out-right",
        targetHandle: "in",
        animated: selectedEndpoints.length > 0,
        markerEnd: { type: MarkerType.ArrowClosed, width: 20, height: 20 },
        style: {
          stroke: isFallbackServing ? "#0f766e" : "#94a3b8",
          strokeWidth: isFallbackServing ? 3 : 2.4,
        },
      });

      return flowEdges;
    },
    [fallbackGatewayName, isFallbackServing, selectedEndpoints.length, selectedTemplates.length],
  );

  async function handleTemplateSave() {
    try {
      await onSaveTemplates(orderSelectedIDs(availableTemplates.map((item) => item.id), selectedTemplateSet));
      setIsTemplateDrawerOpen(false);
    } catch {}
  }

  async function handleEndpointSave() {
    try {
      await onSaveEndpoints(orderSelectedIDs(availableEndpoints.map((item) => item.id), selectedEndpointSet));
      setIsEndpointDrawerOpen(false);
    } catch {}
  }

  return (
    <>
      <div className="overflow-hidden rounded-[28px] border border-gray-200 bg-[radial-gradient(circle_at_top,_rgba(59,130,246,0.08),_transparent_32%),linear-gradient(180deg,rgba(255,255,255,0.96),rgba(248,250,252,0.98))] dark:border-gray-700 dark:bg-[radial-gradient(circle_at_top,_rgba(56,189,248,0.08),_transparent_32%),linear-gradient(180deg,rgba(17,24,39,0.94),rgba(17,24,39,0.98))]">
        <div className="h-[560px] w-full">
          <ReactFlow
            nodes={nodes}
            edges={edges}
            nodeTypes={nodeTypes}
            onNodeClick={(_, node) => {
              if (node.id === "templates") {
                setTemplateDraftIDs(selectedTemplates.map((item) => item.id));
                setIsTemplateDrawerOpen(true);
                return;
              }
              if (node.id === "endpoints") {
                setEndpointDraftIDs(selectedEndpoints.map((item) => item.id));
                setIsEndpointDrawerOpen(true);
              }
            }}
            fitView
            fitViewOptions={{ padding: 0.12, minZoom: 0.9, maxZoom: 1 }}
            nodesDraggable={false}
            nodesConnectable={false}
            elementsSelectable={false}
            panOnDrag={false}
            zoomOnScroll={false}
            zoomOnPinch={false}
            zoomOnDoubleClick={false}
            preventScrolling={false}
            proOptions={{ hideAttribution: true }}
            className="bg-transparent"
          >
            <Background gap={26} size={1.2} color="#dbe4f0" />
          </ReactFlow>
        </div>
      </div>

      <SelectionDrawer
        open={isTemplateDrawerOpen}
        title="Select Templates"
        subtitle="Choose which templates route into this gateway. Changes are only saved when you press Save."
        items={availableTemplates.map((item) => ({
          id: item.id,
          title: item.name,
          description: `${item.category} · ${item.status}${item.consumer === "none" ? "" : ` · ${item.consumer}`}`,
        }))}
        selectedIDs={selectedTemplateSet}
        isSaving={isSavingTemplates}
        onClose={() => setIsTemplateDrawerOpen(false)}
        onToggle={(id) => setTemplateDraftIDs((current) => toggleSelection(current, id))}
        onSave={() => void handleTemplateSave()}
      />

      <SelectionDrawer
        open={isEndpointDrawerOpen}
        title="Select Endpoints"
        subtitle="Choose which endpoints this gateway can send through. Changes are only saved when you press Save."
        items={availableEndpoints.map((item) => ({
          id: item.id,
          title: item.name,
          description: `${item.host}:${item.port} · ${item.status}${item.username ? ` · ${item.username}` : ""}`,
        }))}
        selectedIDs={selectedEndpointSet}
        isSaving={isSavingEndpoints}
        onClose={() => setIsEndpointDrawerOpen(false)}
        onToggle={(id) => setEndpointDraftIDs((current) => toggleSelection(current, id))}
        onSave={() => void handleEndpointSave()}
      />
    </>
  );
}

function SelectionFlowNode({ data }: NodeProps<Node<SelectionNodeData>>) {
  const isLeft = data.side === "left";

  return (
    <div className="nodrag nopan relative block w-[330px] cursor-pointer rounded-[26px] border border-gray-200/90 bg-white/95 p-5 text-left shadow-[0_20px_60px_-30px_rgba(15,23,42,0.45)] backdrop-blur transition hover:-translate-y-0.5 hover:shadow-[0_24px_70px_-34px_rgba(15,23,42,0.55)] dark:border-gray-700/80 dark:bg-gray-900/90">
      {isLeft ? (
        <Handle type="source" id="out" position={Position.Right} className="!h-3 !w-3 !border-2 !border-white !bg-slate-400" />
      ) : (
        <Handle type="target" id="in" position={Position.Left} className="!h-3 !w-3 !border-2 !border-white !bg-slate-400" />
      )}

      <div className={`flex items-start justify-between gap-4 ${isLeft ? "" : "text-right"}`}>
        <div className="min-w-0">
          <p className="text-[11px] font-medium tracking-[0.22em] text-gray-400 uppercase">{data.eyebrow}</p>
          <p className="mt-3 text-sm font-semibold text-gray-900 dark:text-white">{data.subtitle}</p>
        </div>
        <span className="inline-flex h-10 w-10 shrink-0 items-center justify-center rounded-2xl border border-gray-200 bg-white text-lg text-gray-500 dark:border-gray-700 dark:bg-gray-950 dark:text-gray-300">
          {isLeft ? "T" : "E"}
        </span>
      </div>

      <div className="mt-5 space-y-3">
        {data.items.length === 0 ? (
          <div className="rounded-2xl border border-dashed border-gray-300 px-4 py-5 text-sm text-gray-500 dark:border-gray-700 dark:text-gray-400">
            {data.emptyText}
          </div>
        ) : (
          data.items.slice(0, 3).map((item) => (
            <div
              key={item.id}
              className="rounded-2xl border border-gray-200 bg-gray-50 px-4 py-3 dark:border-gray-800 dark:bg-gray-950/60"
            >
              <div className={`flex items-start gap-3 ${isLeft ? "justify-between" : "justify-between flex-row-reverse"}`}>
                <div className="min-w-0">
                  <p className="truncate text-sm font-semibold text-gray-900 dark:text-white">{item.title}</p>
                  <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">{item.meta}</p>
                </div>
                <span className="mt-0.5 inline-flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-emerald-500 text-[11px] font-bold text-white">
                  ✓
                </span>
              </div>
            </div>
          ))
        )}
        {data.items.length > 3 ? (
          <div className="rounded-2xl border border-dashed border-gray-300 px-4 py-3 text-xs font-medium tracking-[0.18em] text-gray-400 uppercase dark:border-gray-700">
            +{data.items.length - 3} more
          </div>
        ) : null}
        <p className="text-xs font-medium text-blue-600 dark:text-blue-300">{`Click to manage ${data.eyebrow.toLowerCase()}`}</p>
      </div>
    </div>
  );
}

function GatewayFlowNode({ data }: NodeProps<Node<GatewayNodeData>>) {
  const isFallback = data.kind === "fallback";
  return (
    <div
      className={`relative w-[280px] rounded-[28px] border px-6 py-6 text-center shadow-[0_20px_60px_-32px_rgba(15,23,42,0.55)] ${
        isFallback
          ? "border-teal-200 bg-teal-50/90 dark:border-teal-500/30 dark:bg-teal-500/10"
          : "border-gray-200 bg-white dark:border-gray-700 dark:bg-gray-900"
      }`}
    >
      <Handle type="target" id="in-left" position={Position.Left} className="!h-3 !w-3 !border-2 !border-white !bg-slate-400" />
      <Handle type="source" id="out-right" position={Position.Right} className="!h-3 !w-3 !border-2 !border-white !bg-slate-400" />
      {isFallback ? (
        <Handle type="target" id="in-top" position={Position.Top} className="!h-3 !w-3 !border-2 !border-white !bg-teal-500" />
      ) : (
        <Handle type="source" id="out-bottom" position={Position.Bottom} className="!h-3 !w-3 !border-2 !border-white !bg-slate-400" />
      )}
      <p className="text-[11px] font-medium tracking-[0.22em] text-gray-400 uppercase">{isFallback ? "Fallback Gateway" : "Gateway"}</p>
      <h4 className="mt-3 text-2xl font-semibold tracking-tight text-gray-900 dark:text-white">{data.gatewayName}</h4>
      <div className="mt-4 flex justify-center">
        <StatusPill status={data.gatewayStatus} />
      </div>
      <div className="mt-5 rounded-2xl border border-gray-200 bg-gray-50 px-4 py-4 text-left dark:border-gray-800 dark:bg-gray-950/60">
        <GatewayInfoRow label="Traffic Class" value={data.trafficClass} />
      </div>
    </div>
  );
}

function SelectionDrawer({
  open,
  title,
  subtitle,
  items,
  selectedIDs,
  isSaving,
  onClose,
  onToggle,
  onSave,
}: {
  open: boolean;
  title: string;
  subtitle: string;
  items: Array<{ id: string; title: string; description: string }>;
  selectedIDs: Set<string>;
  isSaving: boolean;
  onClose: () => void;
  onToggle: (id: string) => void;
  onSave: () => void;
}) {
  useEffect(() => {
    if (!open) {
      return;
    }
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        onClose();
      }
    };
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [onClose, open]);

  if (!open || typeof document === "undefined") {
    return null;
  }

  return createPortal(
    <div className="fixed inset-0 z-[100100] flex">
      <button
        type="button"
        aria-label="Close drawer"
        onClick={onClose}
        className="h-full flex-1 bg-gray-950/50 backdrop-blur-sm"
      />
      <div
        className="flex h-full w-full max-w-[430px] flex-col border-l border-gray-200 bg-white shadow-2xl dark:border-gray-800 dark:bg-gray-950"
      >
        <div className="border-b border-gray-200 px-6 py-5 dark:border-gray-800">
          <div className="flex items-start justify-between gap-4">
            <div>
              <p className="text-xs font-medium tracking-[0.18em] text-gray-400 uppercase">{title}</p>
              <p className="mt-3 text-sm text-gray-500 dark:text-gray-400">{subtitle}</p>
            </div>
            <button
              type="button"
              onClick={onClose}
              className="inline-flex h-10 w-10 items-center justify-center rounded-xl border border-gray-200 text-lg text-gray-500 transition hover:bg-gray-50 dark:border-gray-700 dark:text-gray-300 dark:hover:bg-gray-900"
            >
              ×
            </button>
          </div>
        </div>

        <div className="flex-1 overflow-y-auto px-4 py-4">
          <div className="space-y-3">
            {items.map((item) => {
              const selected = selectedIDs.has(item.id);
              return (
                <button
                  key={item.id}
                  type="button"
                  onClick={() => onToggle(item.id)}
                  className={`flex w-full items-start gap-3 rounded-2xl border px-4 py-4 text-left transition ${
                    selected
                      ? "border-emerald-200 bg-emerald-50 dark:border-emerald-500/30 dark:bg-emerald-500/10"
                      : "border-gray-200 bg-white hover:bg-gray-50 dark:border-gray-800 dark:bg-gray-900 dark:hover:bg-gray-900/80"
                  }`}
                >
                  <span
                    className={`mt-0.5 inline-flex h-5 w-5 shrink-0 items-center justify-center rounded-full border text-[11px] font-bold ${
                      selected
                        ? "border-emerald-500 bg-emerald-500 text-white"
                        : "border-gray-300 text-transparent dark:border-gray-700"
                    }`}
                  >
                    ✓
                  </span>
                  <span className="min-w-0">
                    <span className="block truncate text-sm font-semibold text-gray-900 dark:text-white">{item.title}</span>
                    <span className="mt-1 block text-xs text-gray-500 dark:text-gray-400">{item.description}</span>
                  </span>
                </button>
              );
            })}
          </div>
        </div>

        <div className="border-t border-gray-200 px-6 py-5 dark:border-gray-800">
          <div className="flex items-center justify-end gap-3">
            <button
              type="button"
              onClick={onClose}
              className="inline-flex items-center rounded-xl border border-gray-200 bg-white px-4 py-2.5 text-sm font-semibold text-gray-700 transition hover:bg-gray-50 dark:border-gray-700 dark:bg-gray-900 dark:text-gray-200 dark:hover:bg-gray-800"
            >
              Cancel
            </button>
            <button
              type="button"
              onClick={onSave}
              disabled={isSaving}
              className="inline-flex items-center rounded-xl bg-gray-900 px-4 py-2.5 text-sm font-semibold text-white transition hover:bg-gray-800 disabled:cursor-not-allowed disabled:opacity-60 dark:bg-white dark:text-gray-900 dark:hover:bg-gray-200"
            >
              {isSaving ? "Saving..." : "Save"}
            </button>
          </div>
        </div>
      </div>
    </div>,
    document.body,
  );
}

function StatusPill({ status }: { status: string }) {
  const normalized = status.trim().toLowerCase();
  const className =
    normalized === "active" || normalized === "ready" || normalized === "healthy"
      ? "border-emerald-200 bg-emerald-50 text-emerald-700 dark:border-emerald-500/30 dark:bg-emerald-500/10 dark:text-emerald-300"
      : normalized === "disabled" || normalized === "maintenance" || normalized === "unhealthy" || normalized === "failed" || normalized === "error"
        ? "border-rose-200 bg-rose-50 text-rose-700 dark:border-rose-500/30 dark:bg-rose-500/10 dark:text-rose-300"
        : "border-amber-200 bg-amber-50 text-amber-700 dark:border-amber-500/30 dark:bg-amber-500/10 dark:text-amber-300";

  return (
    <span className={`rounded-full border px-3 py-1 text-xs font-semibold capitalize ${className}`}>
      {status}
    </span>
  );
}

function GatewayInfoRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-center justify-between gap-3">
      <span className="text-xs font-medium tracking-[0.18em] text-gray-400 uppercase">{label}</span>
      <span className="text-sm font-semibold text-gray-900 dark:text-white">{value}</span>
    </div>
  );
}

function normalizeStatus(status?: string) {
  return (status || "").trim().toLowerCase();
}

function toggleSelection(current: string[], id: string) {
  return current.includes(id) ? current.filter((item) => item !== id) : [...current, id];
}

function orderSelectedIDs(order: string[], selectedSet: Set<string>) {
  return order.filter((id) => selectedSet.has(id));
}
