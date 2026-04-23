"use client";

import { parseAPIError } from "@/components/auth/auth-utils";
import type {
  PaginatedResult,
  TenantItem,
  TenantMutationInput,
  WorkspaceItem,
  WorkspaceMutationInput,
} from "@/components/organization/types";

type APIEnvelope<T> = {
  data?: T;
  message?: string;
};

type ListQuery = {
  page?: number;
  limit?: number;
  q?: string;
  status?: string;
  tenant_id?: string;
};

function buildListPath(path: string, query: ListQuery) {
  const params = new URLSearchParams();
  if (query.page && query.page > 0) {
    params.set("page", String(query.page));
  }
  if (query.limit && query.limit > 0) {
    params.set("limit", String(query.limit));
  }
  if (query.q && query.q.trim() !== "") {
    params.set("q", query.q.trim());
  }
  if (query.status && query.status.trim() !== "") {
    params.set("status", query.status.trim());
  }
  if (query.tenant_id && query.tenant_id.trim() !== "") {
    params.set("tenant_id", query.tenant_id.trim());
  }
  const suffix = params.toString();
  return suffix === "" ? path : `${path}?${suffix}`;
}

async function readData<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(path, {
    cache: "no-store",
    credentials: "include",
    ...init,
  });
  if (!response.ok) {
    throw new Error(await parseAPIError(response));
  }

  const payload = (await response.json()) as APIEnvelope<T>;
  if (payload.data === undefined) {
    throw new Error("Missing API data payload.");
  }
  return payload.data;
}

async function mutate<T>(path: string, init?: RequestInit): Promise<T | null> {
  const response = await fetch(path, {
    cache: "no-store",
    credentials: "include",
    ...init,
  });
  if (!response.ok) {
    throw new Error(await parseAPIError(response));
  }

  const payload = (await response.json()) as APIEnvelope<T>;
  return payload.data ?? null;
}

export function listTenants(query: ListQuery = {}) {
  return readData<PaginatedResult<TenantItem>>(buildListPath("/api/v1/core/tenants", query));
}

export function getTenant(id: string) {
  return readData<TenantItem>(`/api/v1/core/tenants/${id}`);
}

export function createTenant(input: TenantMutationInput) {
  return mutate<TenantItem>("/api/v1/core/tenants", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify(input),
  });
}

export function updateTenant(id: string, input: TenantMutationInput) {
  return mutate<TenantItem>(`/api/v1/core/tenants/${id}`, {
    method: "PATCH",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify(input),
  });
}

export function deleteTenant(id: string) {
  return mutate<null>(`/api/v1/core/tenants/${id}`, {
    method: "DELETE",
  });
}

export function listTenantOptions() {
  return listTenants({ page: 1, limit: 100, status: "active" });
}

export function listWorkspaces(query: ListQuery = {}) {
  return readData<PaginatedResult<WorkspaceItem>>(buildListPath("/api/v1/core/workspaces", query));
}

export function getWorkspace(id: string) {
  return readData<WorkspaceItem>(`/api/v1/core/workspaces/${id}`);
}

export function createWorkspace(input: WorkspaceMutationInput) {
  return mutate<WorkspaceItem>("/api/v1/core/workspaces", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify(input),
  });
}

export function updateWorkspace(id: string, input: WorkspaceMutationInput) {
  return mutate<WorkspaceItem>(`/api/v1/core/workspaces/${id}`, {
    method: "PATCH",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify(input),
  });
}

export function deleteWorkspace(id: string) {
  return mutate<null>(`/api/v1/core/workspaces/${id}`, {
    method: "DELETE",
  });
}
