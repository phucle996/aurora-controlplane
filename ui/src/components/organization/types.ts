"use client";

export type Pagination = {
  page: number;
  limit: number;
  total: number;
  total_pages: number;
};

export type TenantStatus = "active" | "suspended" | "archived";
export type WorkspaceStatus = "active" | "disabled" | "archived";

export type TenantItem = {
  id: string;
  name: string;
  slug: string;
  status: TenantStatus;
  created_at: string;
  updated_at: string;
};

export type WorkspaceItem = {
  id: string;
  tenant_id: string;
  tenant_name: string;
  name: string;
  slug: string;
  status: WorkspaceStatus;
  created_at: string;
  updated_at: string;
};

export type PaginatedResult<T> = {
  items: T[];
  pagination: Pagination;
};

export type TenantMutationInput = {
  name?: string;
  status?: TenantStatus;
};

export type WorkspaceMutationInput = {
  name?: string;
  status?: WorkspaceStatus;
  tenant_id?: string;
};
