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
  createTenant,
  deleteTenant,
  listTenants,
  updateTenant,
} from "@/components/organization/api";
import type {
  Pagination as PaginationMeta,
  TenantItem,
  TenantStatus,
} from "@/components/organization/types";

const emptyPagination: PaginationMeta = {
  page: 1,
  limit: 20,
  total: 0,
  total_pages: 0,
};

function statusColor(status: TenantStatus) {
  if (status === "active") {
    return "success";
  }
  if (status === "suspended") {
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

export default function TenantManagementPage() {
  const { pushToast } = useToast();
  const [items, setItems] = useState<TenantItem[]>([]);
  const [pagination, setPagination] = useState<PaginationMeta>(emptyPagination);
  const [isLoading, setIsLoading] = useState(true);
  const [queryInput, setQueryInput] = useState("");
  const [query, setQuery] = useState("");
  const [statusInput, setStatusInput] = useState("");
  const [status, setStatus] = useState("");
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [editingItem, setEditingItem] = useState<TenantItem | null>(null);
  const [formName, setFormName] = useState("");
  const [formStatus, setFormStatus] = useState<TenantStatus>("active");
  const [isSaving, setIsSaving] = useState(false);

  useEffect(() => {
    let cancelled = false;

    async function load() {
      setIsLoading(true);
      try {
        const payload = await listTenants({
          page: pagination.page,
          limit: pagination.limit,
          q: query,
          status,
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
          message: error instanceof Error ? error.message : "Failed to load tenants.",
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
  }, [pagination.page, pagination.limit, pushToast, query, status]);

  function openCreateModal() {
    setEditingItem(null);
    setFormName("");
    setFormStatus("active");
    setIsModalOpen(true);
  }

  function openEditModal(item: TenantItem) {
    setEditingItem(item);
    setFormName(item.name);
    setFormStatus(item.status);
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
        await createTenant({
          name: formName,
          status: formStatus,
        });
        pushToast({ kind: "success", message: "Tenant created." });
        setPagination((current) => ({ ...current, page: 1 }));
      } else {
        await updateTenant(editingItem.id, {
          name: formName,
          status: formStatus,
        });
        pushToast({ kind: "success", message: "Tenant updated." });
      }

      closeModal();
      const payload = await listTenants({
        page: editingItem == null ? 1 : pagination.page,
        limit: pagination.limit,
        q: query,
        status,
      });
      setItems(payload.items);
      setPagination(payload.pagination);
    } catch (error) {
      pushToast({
        kind: "error",
        message: error instanceof Error ? error.message : "Failed to save tenant.",
      });
    } finally {
      setIsSaving(false);
    }
  }

  async function handleDelete(item: TenantItem) {
    if (!window.confirm(`Delete tenant "${item.name}"? Related workspaces will be detached.`)) {
      return;
    }

    try {
      await deleteTenant(item.id);
      pushToast({ kind: "success", message: "Tenant deleted." });

      const nextPage =
        items.length === 1 && pagination.page > 1 ? pagination.page - 1 : pagination.page;
      const payload = await listTenants({
        page: nextPage,
        limit: pagination.limit,
        q: query,
        status,
      });
      setItems(payload.items);
      setPagination(payload.pagination);
    } catch (error) {
      pushToast({
        kind: "error",
        message: error instanceof Error ? error.message : "Failed to delete tenant.",
      });
    }
  }

  function handleApplyFilters() {
    setPagination((current) => ({ ...current, page: 1 }));
    setQuery(queryInput.trim());
    setStatus(statusInput);
  }

  function handleResetFilters() {
    setQueryInput("");
    setQuery("");
    setStatusInput("");
    setStatus("");
    setPagination((current) => ({ ...current, page: 1 }));
  }

  return (
    <div className="space-y-6">
      <PageBreadcrumb pageTitle="Tenants" />

      <section className="rounded-3xl border border-gray-200 bg-white p-6 dark:border-gray-800 dark:bg-white/[0.03] lg:p-8">
        <div className="grid gap-5 lg:grid-cols-[minmax(0,1.3fr)_repeat(2,minmax(0,1fr))]">
          <div className="space-y-3">
            <div className="inline-flex items-center rounded-full bg-brand-500/10 px-3 py-1 text-xs font-semibold uppercase tracking-[0.24em] text-brand-600 dark:text-brand-400">
              Organization
            </div>
            <h1 className="text-3xl font-semibold text-gray-900 dark:text-white">
              Manage tenants without leaving the control plane
            </h1>
            <p className="text-sm leading-7 text-gray-500 dark:text-gray-400">
              Create organization boundaries, suspend them when needed, and keep
              workspace ownership clean without exposing internal fields.
            </p>
          </div>

          <div className="rounded-2xl border border-gray-200 p-5 dark:border-gray-800">
            <p className="text-sm text-gray-500 dark:text-gray-400">Visible tenants</p>
            <p className="mt-2 text-3xl font-semibold text-gray-900 dark:text-white">
              {pagination.total}
            </p>
          </div>

          <div className="rounded-2xl border border-gray-200 p-5 dark:border-gray-800">
            <p className="text-sm text-gray-500 dark:text-gray-400">Current page</p>
            <p className="mt-2 text-3xl font-semibold text-gray-900 dark:text-white">
              {pagination.page}
            </p>
          </div>
        </div>
      </section>

      <ComponentCard
        title="Tenant Management"
        desc="Search, create, update, and delete tenants with the real core API."
        headerAction={<Button onClick={openCreateModal}>New Tenant</Button>}
      >
        <div className="grid gap-4 xl:grid-cols-[minmax(0,1fr)_240px_auto_auto]">
          <Input
            type="text"
            value={queryInput}
            onChange={(event) => setQueryInput(event.target.value)}
            placeholder="Search by tenant name or slug..."
          />

          <select
            value={statusInput}
            onChange={(event) => setStatusInput(event.target.value)}
            className="h-11 rounded-lg border border-gray-300 bg-transparent px-4 text-sm text-gray-800 shadow-theme-xs outline-none transition focus:border-brand-300 focus:ring-3 focus:ring-brand-500/10 dark:border-gray-700 dark:bg-gray-900 dark:text-white/90 dark:focus:border-brand-800"
          >
            <option value="">All statuses</option>
            <option value="active">Active</option>
            <option value="suspended">Suspended</option>
            <option value="archived">Archived</option>
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
                <th className="px-4 py-3 font-medium">Tenant</th>
                <th className="px-4 py-3 font-medium">Slug</th>
                <th className="px-4 py-3 font-medium">Status</th>
                <th className="px-4 py-3 font-medium">Updated</th>
                <th className="px-4 py-3 font-medium">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-200 dark:divide-gray-800">
              {isLoading ? (
                <tr>
                  <td colSpan={5} className="px-4 py-12 text-center text-sm text-gray-500 dark:text-gray-400">
                    Loading tenants...
                  </td>
                </tr>
              ) : null}

              {!isLoading && items.length === 0 ? (
                <tr>
                  <td colSpan={5} className="px-4 py-12 text-center text-sm text-gray-500 dark:text-gray-400">
                    No tenants match the current filters.
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
            Showing {items.length} of {pagination.total} tenants
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

      <Modal isOpen={isModalOpen} onClose={closeModal} className="max-w-[620px] p-5 lg:p-8">
        <form onSubmit={handleSubmit}>
          <h3 className="mb-6 text-lg font-medium text-gray-800 dark:text-white/90">
            {editingItem == null ? "Create Tenant" : "Update Tenant"}
          </h3>

          <div className="space-y-5">
            <div>
              <Label htmlFor="tenant-name">Name</Label>
              <Input
                id="tenant-name"
                type="text"
                value={formName}
                onChange={(event) => setFormName(event.target.value)}
                placeholder="Aurora Organization"
                required
              />
            </div>

            <div>
              <Label htmlFor="tenant-status">Status</Label>
              <select
                id="tenant-status"
                value={formStatus}
                onChange={(event) => setFormStatus(event.target.value as TenantStatus)}
                className="h-11 w-full rounded-lg border border-gray-300 bg-transparent px-4 text-sm text-gray-800 shadow-theme-xs outline-none transition focus:border-brand-300 focus:ring-3 focus:ring-brand-500/10 dark:border-gray-700 dark:bg-gray-900 dark:text-white/90 dark:focus:border-brand-800"
              >
                <option value="active">Active</option>
                <option value="suspended">Suspended</option>
                <option value="archived">Archived</option>
              </select>
            </div>
          </div>

          <div className="mt-6 flex items-center justify-end gap-3">
            <Button type="button" variant="outline" onClick={closeModal} disabled={isSaving}>
              Cancel
            </Button>
            <Button type="submit" disabled={isSaving}>
              {isSaving ? "Saving..." : editingItem == null ? "Create Tenant" : "Save Changes"}
            </Button>
          </div>
        </form>
      </Modal>
    </div>
  );
}
