"use client";

import { useEffect, useState } from "react";
import PageBreadcrumb from "@/components/common/PageBreadCrumb";
import ComponentCard from "@/components/common/ComponentCard";
import Label from "@/components/form/Label";
import Input from "@/components/form/input/InputField";
import Pagination from "@/components/tables/Pagination";
import Button from "@/components/ui/button/Button";
import Badge from "@/components/ui/badge/Badge";
import { Modal } from "@/components/ui/modal";
import { useToast } from "@/components/ui/toast/ToastProvider";
import {
  createWorkspace,
  deleteWorkspace,
  listTenantOptions,
  listWorkspaces,
  updateWorkspace,
} from "@/components/organization/api";
import type {
  Pagination as PaginationMeta,
  TenantItem,
  WorkspaceItem,
  WorkspaceStatus,
} from "@/components/organization/types";

const emptyPagination: PaginationMeta = {
  page: 1,
  limit: 20,
  total: 0,
  total_pages: 0,
};

function statusColor(status: WorkspaceStatus) {
  if (status === "active") {
    return "success";
  }
  if (status === "disabled") {
    return "warning";
  }
  return "light";
}

function formatDateTime(value: string) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return "-";
  }
  return date.toLocaleString();
}

export default function WorkspaceManagementPage() {
  const { pushToast } = useToast();
  const [items, setItems] = useState<WorkspaceItem[]>([]);
  const [tenantOptions, setTenantOptions] = useState<TenantItem[]>([]);
  const [pagination, setPagination] = useState<PaginationMeta>(emptyPagination);
  const [isLoading, setIsLoading] = useState(true);
  const [queryInput, setQueryInput] = useState("");
  const [query, setQuery] = useState("");
  const [statusInput, setStatusInput] = useState("");
  const [status, setStatus] = useState("");
  const [tenantFilterInput, setTenantFilterInput] = useState("");
  const [tenantFilter, setTenantFilter] = useState("");
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [editingItem, setEditingItem] = useState<WorkspaceItem | null>(null);
  const [formName, setFormName] = useState("");
  const [formStatus, setFormStatus] = useState<WorkspaceStatus>("active");
  const [formTenantID, setFormTenantID] = useState("");
  const [isSaving, setIsSaving] = useState(false);

  useEffect(() => {
    let cancelled = false;

    async function loadTenantOptions() {
      try {
        const payload = await listTenantOptions();
        if (cancelled) {
          return;
        }
        setTenantOptions(payload.items);
      } catch (error) {
        if (cancelled) {
          return;
        }
        pushToast({
          kind: "error",
          message: error instanceof Error ? error.message : "Failed to load tenant options.",
        });
      }
    }

    void loadTenantOptions();
    return () => {
      cancelled = true;
    };
  }, [pushToast]);

  useEffect(() => {
    let cancelled = false;

    async function load() {
      setIsLoading(true);
      try {
        const payload = await listWorkspaces({
          page: pagination.page,
          limit: pagination.limit,
          q: query,
          status,
          tenant_id: tenantFilter,
        });
        if (cancelled) {
          return;
        }
        setItems(payload.items);
        setPagination(payload.pagination);
      } catch (error) {
        if (cancelled) {
          return;
        }
        pushToast({
          kind: "error",
          message: error instanceof Error ? error.message : "Failed to load workspaces.",
        });
      } finally {
        if (!cancelled) {
          setIsLoading(false);
        }
      }
    }

    void load();
    return () => {
      cancelled = true;
    };
  }, [pagination.page, pagination.limit, pushToast, query, status, tenantFilter]);

  function openCreateModal() {
    setEditingItem(null);
    setFormName("");
    setFormStatus("active");
    setFormTenantID("");
    setIsModalOpen(true);
  }

  function openEditModal(item: WorkspaceItem) {
    setEditingItem(item);
    setFormName(item.name);
    setFormStatus(item.status);
    setFormTenantID(item.tenant_id);
    setIsModalOpen(true);
  }

  function closeModal() {
    setIsModalOpen(false);
    setEditingItem(null);
  }

  async function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setIsSaving(true);
    try {
      if (editingItem == null) {
        await createWorkspace({
          name: formName,
          status: formStatus,
          tenant_id: formTenantID,
        });
        pushToast({ kind: "success", message: "Workspace created." });
        setPagination((current) => ({ ...current, page: 1 }));
      } else {
        await updateWorkspace(editingItem.id, {
          name: formName,
          status: formStatus,
        });
        pushToast({ kind: "success", message: "Workspace updated." });
      }

      closeModal();
      const payload = await listWorkspaces({
        page: editingItem == null ? 1 : pagination.page,
        limit: pagination.limit,
        q: query,
        status,
        tenant_id: tenantFilter,
      });
      setItems(payload.items);
      setPagination(payload.pagination);
    } catch (error) {
      pushToast({
        kind: "error",
        message: error instanceof Error ? error.message : "Failed to save workspace.",
      });
    } finally {
      setIsSaving(false);
    }
  }

  async function handleDelete(item: WorkspaceItem) {
    if (!window.confirm(`Delete workspace "${item.name}"?`)) {
      return;
    }

    try {
      await deleteWorkspace(item.id);
      pushToast({ kind: "success", message: "Workspace deleted." });

      const nextPage =
        items.length === 1 && pagination.page > 1 ? pagination.page - 1 : pagination.page;
      const payload = await listWorkspaces({
        page: nextPage,
        limit: pagination.limit,
        q: query,
        status,
        tenant_id: tenantFilter,
      });
      setItems(payload.items);
      setPagination(payload.pagination);
    } catch (error) {
      pushToast({
        kind: "error",
        message: error instanceof Error ? error.message : "Failed to delete workspace.",
      });
    }
  }

  function handleApplyFilters() {
    setPagination((current) => ({ ...current, page: 1 }));
    setQuery(queryInput.trim());
    setStatus(statusInput);
    setTenantFilter(tenantFilterInput);
  }

  function handleResetFilters() {
    setQueryInput("");
    setQuery("");
    setStatusInput("");
    setStatus("");
    setTenantFilterInput("");
    setTenantFilter("");
    setPagination((current) => ({ ...current, page: 1 }));
  }

  return (
    <div className="space-y-6">
      <PageBreadcrumb pageTitle="Workspace" />

      <section className="rounded-3xl border border-gray-200 bg-white p-6 dark:border-gray-800 dark:bg-white/[0.03] lg:p-8">
        <div className="grid gap-5 lg:grid-cols-[minmax(0,1.3fr)_repeat(2,minmax(0,1fr))]">
          <div className="space-y-3">
            <div className="inline-flex items-center rounded-full bg-brand-500/10 px-3 py-1 text-xs font-semibold uppercase tracking-[0.24em] text-brand-600 dark:text-brand-400">
              Organization
            </div>
            <h1 className="text-3xl font-semibold text-gray-900 dark:text-white">
              Manage workspaces and tenant attachment cleanly
            </h1>
            <p className="text-sm leading-7 text-gray-500 dark:text-gray-400">
              Workspaces can stand alone or belong to a tenant. Creation sets the
              tenant relation, and later edits keep that relation read-only.
            </p>
          </div>

          <div className="rounded-2xl border border-gray-200 p-5 dark:border-gray-800">
            <p className="text-sm text-gray-500 dark:text-gray-400">Visible workspaces</p>
            <p className="mt-2 text-3xl font-semibold text-gray-900 dark:text-white">
              {pagination.total}
            </p>
          </div>

          <div className="rounded-2xl border border-gray-200 p-5 dark:border-gray-800">
            <p className="text-sm text-gray-500 dark:text-gray-400">Active tenant options</p>
            <p className="mt-2 text-3xl font-semibold text-gray-900 dark:text-white">
              {tenantOptions.length}
            </p>
          </div>
        </div>
      </section>

      <ComponentCard
        title="Workspace Management"
        desc="Create and maintain organization workspaces using the real core API."
        headerAction={<Button onClick={openCreateModal}>New Workspace</Button>}
      >
        <div className="grid gap-4 xl:grid-cols-[minmax(0,1fr)_220px_260px_auto_auto]">
          <Input
            type="text"
            value={queryInput}
            onChange={(event) => setQueryInput(event.target.value)}
            placeholder="Search by workspace, slug, or tenant..."
          />

          <select
            value={statusInput}
            onChange={(event) => setStatusInput(event.target.value)}
            className="h-11 rounded-lg border border-gray-300 bg-transparent px-4 text-sm text-gray-800 shadow-theme-xs outline-none transition focus:border-brand-300 focus:ring-3 focus:ring-brand-500/10 dark:border-gray-700 dark:bg-gray-900 dark:text-white/90 dark:focus:border-brand-800"
          >
            <option value="">All statuses</option>
            <option value="active">Active</option>
            <option value="disabled">Disabled</option>
            <option value="archived">Archived</option>
          </select>

          <select
            value={tenantFilterInput}
            onChange={(event) => setTenantFilterInput(event.target.value)}
            className="h-11 rounded-lg border border-gray-300 bg-transparent px-4 text-sm text-gray-800 shadow-theme-xs outline-none transition focus:border-brand-300 focus:ring-3 focus:ring-brand-500/10 dark:border-gray-700 dark:bg-gray-900 dark:text-white/90 dark:focus:border-brand-800"
          >
            <option value="">All tenants</option>
            {tenantOptions.map((item) => (
              <option key={item.id} value={item.id}>
                {item.name}
              </option>
            ))}
          </select>

          <Button variant="outline" onClick={handleResetFilters}>
            Reset
          </Button>
          <Button onClick={handleApplyFilters}>Apply</Button>
        </div>

        <div className="overflow-x-auto">
          <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-800">
            <thead>
              <tr className="text-left text-xs uppercase tracking-[0.18em] text-gray-400 dark:text-gray-500">
                <th className="px-4 py-3 font-medium">Workspace</th>
                <th className="px-4 py-3 font-medium">Slug</th>
                <th className="px-4 py-3 font-medium">Tenant</th>
                <th className="px-4 py-3 font-medium">Status</th>
                <th className="px-4 py-3 font-medium">Updated</th>
                <th className="px-4 py-3 font-medium">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-200 dark:divide-gray-800">
              {isLoading ? (
                <tr>
                  <td colSpan={6} className="px-4 py-12 text-center text-sm text-gray-500 dark:text-gray-400">
                    Loading workspaces...
                  </td>
                </tr>
              ) : null}

              {!isLoading && items.length === 0 ? (
                <tr>
                  <td colSpan={6} className="px-4 py-12 text-center text-sm text-gray-500 dark:text-gray-400">
                    No workspaces match the current filters.
                  </td>
                </tr>
              ) : null}

              {!isLoading
                ? items.map((item) => (
                    <tr key={item.id}>
                      <td className="px-4 py-4">
                        <div className="space-y-1">
                          <p className="text-sm font-semibold text-gray-900 dark:text-white">
                            {item.name}
                          </p>
                          <p className="text-xs text-gray-500 dark:text-gray-400">{item.id}</p>
                        </div>
                      </td>
                      <td className="px-4 py-4 text-sm text-gray-700 dark:text-gray-300">
                        {item.slug}
                      </td>
                      <td className="px-4 py-4 text-sm text-gray-700 dark:text-gray-300">
                        {item.tenant_name.trim() !== "" ? item.tenant_name : "Standalone"}
                      </td>
                      <td className="px-4 py-4">
                        <Badge color={statusColor(item.status)}>{item.status}</Badge>
                      </td>
                      <td className="px-4 py-4 text-sm text-gray-700 dark:text-gray-300">
                        {formatDateTime(item.updated_at)}
                      </td>
                      <td className="px-4 py-4">
                        <div className="flex flex-wrap gap-2">
                          <Button size="sm" variant="outline" onClick={() => openEditModal(item)}>
                            Edit
                          </Button>
                          <Button
                            size="sm"
                            variant="outline"
                            className="text-error-600"
                            onClick={() => void handleDelete(item)}
                          >
                            Delete
                          </Button>
                        </div>
                      </td>
                    </tr>
                  ))
                : null}
            </tbody>
          </table>
        </div>

        <div className="flex items-center justify-between gap-3 border-t border-gray-100 pt-4 dark:border-gray-800">
          <p className="text-sm text-gray-500 dark:text-gray-400">
            Showing {items.length} of {pagination.total} workspaces
          </p>
          <Pagination
            currentPage={pagination.page}
            totalPages={Math.max(pagination.total_pages, 1)}
            onPageChange={(nextPage) => {
              if (nextPage < 1 || nextPage === pagination.page) {
                return;
              }
              setPagination((current) => ({ ...current, page: nextPage }));
            }}
          />
        </div>
      </ComponentCard>

      <Modal isOpen={isModalOpen} onClose={closeModal} className="max-w-[720px] p-5 lg:p-8">
        <form onSubmit={handleSubmit}>
          <h3 className="mb-6 text-lg font-medium text-gray-800 dark:text-white/90">
            {editingItem == null ? "Create Workspace" : "Update Workspace"}
          </h3>

          <div className="grid gap-5 sm:grid-cols-2">
            <div className="sm:col-span-2">
              <Label htmlFor="workspace-name">Name</Label>
              <Input
                id="workspace-name"
                type="text"
                value={formName}
                onChange={(event) => setFormName(event.target.value)}
                placeholder="Production Workspace"
                required
              />
            </div>

            <div>
              <Label htmlFor="workspace-status">Status</Label>
              <select
                id="workspace-status"
                value={formStatus}
                onChange={(event) => setFormStatus(event.target.value as WorkspaceStatus)}
                className="h-11 w-full rounded-lg border border-gray-300 bg-transparent px-4 text-sm text-gray-800 shadow-theme-xs outline-none transition focus:border-brand-300 focus:ring-3 focus:ring-brand-500/10 dark:border-gray-700 dark:bg-gray-900 dark:text-white/90 dark:focus:border-brand-800"
              >
                <option value="active">Active</option>
                <option value="disabled">Disabled</option>
                <option value="archived">Archived</option>
              </select>
            </div>

            <div>
              <Label htmlFor="workspace-tenant">Tenant</Label>
              {editingItem == null ? (
                <select
                  id="workspace-tenant"
                  value={formTenantID}
                  onChange={(event) => setFormTenantID(event.target.value)}
                  className="h-11 w-full rounded-lg border border-gray-300 bg-transparent px-4 text-sm text-gray-800 shadow-theme-xs outline-none transition focus:border-brand-300 focus:ring-3 focus:ring-brand-500/10 dark:border-gray-700 dark:bg-gray-900 dark:text-white/90 dark:focus:border-brand-800"
                >
                  <option value="">Standalone workspace</option>
                  {tenantOptions.map((item) => (
                    <option key={item.id} value={item.id}>
                      {item.name}
                    </option>
                  ))}
                </select>
              ) : (
                <div className="rounded-lg border border-gray-200 bg-gray-50 px-4 py-3 text-sm text-gray-600 dark:border-gray-800 dark:bg-gray-900 dark:text-gray-300">
                  {editingItem.tenant_name.trim() !== "" ? editingItem.tenant_name : "Standalone workspace"}
                  <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
                    Tenant attachment is immutable after create.
                  </p>
                </div>
              )}
            </div>
          </div>

          <div className="mt-6 flex items-center justify-end gap-3">
            <Button type="button" variant="outline" onClick={closeModal} disabled={isSaving}>
              Cancel
            </Button>
            <Button type="submit" disabled={isSaving}>
              {isSaving ? "Saving..." : editingItem == null ? "Create Workspace" : "Save Changes"}
            </Button>
          </div>
        </form>
      </Modal>
    </div>
  );
}
