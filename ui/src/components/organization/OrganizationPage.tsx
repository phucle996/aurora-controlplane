"use client";

import { useEffect, useMemo, useState } from "react";
import ComponentCard from "@/components/common/ComponentCard";
import Input from "@/components/form/input/InputField";
import Label from "@/components/form/Label";
import Pagination from "@/components/tables/Pagination";
import Badge from "@/components/ui/badge/Badge";
import Button from "@/components/ui/button/Button";
import { Modal } from "@/components/ui/modal";
import { useToast } from "@/components/ui/toast/ToastProvider";
import { PlusIcon } from "@/icons";
import { createTenant, listTenants, listWorkspaces } from "@/components/organization/api";
import type {
  Pagination as PaginationMeta,
  TenantItem,
  TenantStatus,
  WorkspaceItem,
} from "@/components/organization/types";

const EMPTY_PAGINATION: PaginationMeta = {
  page: 1,
  limit: 10,
  total: 0,
  total_pages: 0,
};

function statusBadgeColor(status: TenantStatus) {
  if (status === "active") {
    return "success";
  }
  if (status === "suspended") {
    return "light";
  }
  return "warning";
}

function statusLabel(status: TenantStatus) {
  if (status === "archived") {
    return "Pending";
  }
  if (status === "suspended") {
    return "Suspended";
  }
  return "Active";
}

function derivePlan(workspaceCount: number) {
  if (workspaceCount >= 8) {
    return "Enterprise";
  }
  if (workspaceCount >= 3) {
    return "Pro";
  }
  return "Basic";
}

function deriveRole(status: TenantStatus) {
  if (status === "active") {
    return "Admin";
  }
  if (status === "suspended") {
    return "Viewer";
  }
  return "Owner";
}

export default function OrganizationPage() {
  const { pushToast } = useToast();
  const [items, setItems] = useState<TenantItem[]>([]);
  const [pagination, setPagination] = useState<PaginationMeta>(EMPTY_PAGINATION);
  const [workspaceCounts, setWorkspaceCounts] = useState<Record<string, number>>({});
  const [isLoading, setIsLoading] = useState(true);
  const [queryInput, setQueryInput] = useState("");
  const [query, setQuery] = useState("");
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [formName, setFormName] = useState("");
  const [formStatus, setFormStatus] = useState<TenantStatus>("active");
  const [isSaving, setIsSaving] = useState(false);

  useEffect(() => {
    let cancelled = false;

    async function load() {
      setIsLoading(true);
      try {
        const [tenantPayload, workspacePayload] = await Promise.all([
          listTenants({
            page: pagination.page,
            limit: pagination.limit,
            q: query,
          }),
          listWorkspaces({
            page: 1,
            limit: 100,
          }),
        ]);

        if (cancelled) {
          return;
        }

        const nextWorkspaceCounts = (workspacePayload.items as WorkspaceItem[]).reduce(
          (acc, workspace) => {
            if (!workspace.tenant_id) {
              return acc;
            }
            acc[workspace.tenant_id] = (acc[workspace.tenant_id] ?? 0) + 1;
            return acc;
          },
          {} as Record<string, number>,
        );

        setItems(tenantPayload.items);
        setPagination(tenantPayload.pagination);
        setWorkspaceCounts(nextWorkspaceCounts);
      } catch (error) {
        if (cancelled) {
          return;
        }
        pushToast({
          kind: "error",
          message: error instanceof Error ? error.message : "Failed to load organizations.",
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
  }, [pagination.page, pagination.limit, pushToast, query]);

  const startIndex = useMemo(() => {
    if (pagination.total === 0) {
      return 0;
    }
    return (pagination.page - 1) * pagination.limit + 1;
  }, [pagination.limit, pagination.page, pagination.total]);

  const endIndex = useMemo(() => {
    if (pagination.total === 0) {
      return 0;
    }
    return Math.min(pagination.page * pagination.limit, pagination.total);
  }, [pagination.limit, pagination.page, pagination.total]);

  function openCreateModal() {
    setFormName("");
    setFormStatus("active");
    setIsModalOpen(true);
  }

  function closeModal() {
    setIsModalOpen(false);
  }

  function handleSearch() {
    setPagination((current) => ({ ...current, page: 1 }));
    setQuery(queryInput.trim());
  }

  async function handleCreateOrganization(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setIsSaving(true);

    try {
      await createTenant({
        name: formName.trim(),
        status: formStatus,
      });
      pushToast({
        kind: "success",
        message: "Organization created.",
      });
      closeModal();
      setPagination((current) => ({ ...current, page: 1 }));
      setQuery("");
      setQueryInput("");
    } catch (error) {
      pushToast({
        kind: "error",
        message: error instanceof Error ? error.message : "Failed to create organization.",
      });
    } finally {
      setIsSaving(false);
    }
  }

  return (
    <div className="space-y-6">
      <div className="mb-2 flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
        <div>
          <h1 className="text-3xl font-semibold tracking-tight text-gray-900 dark:text-white">
            Organizations
          </h1>
          <p className="mt-2 text-sm text-gray-500 dark:text-gray-400">
            Create and manage isolated environments for your teams and resources.
          </p>
        </div>

        <div className="flex items-center gap-4">
          <Button onClick={openCreateModal} className="min-w-[220px] justify-center">
            <PlusIcon className="size-4" />
            Create Organization
          </Button>
          <a
            href="#"
            className="text-sm font-medium text-gray-500 transition hover:text-gray-900 dark:text-gray-400 dark:hover:text-white"
          >
            View Docs
          </a>
        </div>
      </div>

      <ComponentCard
        title="Your Organizations"
        desc="Manage organization settings, plans, and activity."
        className="border border-gray-200 dark:border-gray-800"
        headerAction={
          <div className="w-full lg:w-[360px]">
            <Input
              type="text"
              value={queryInput}
              onChange={(event) => setQueryInput(event.target.value)}
              onKeyDown={(event) => {
                if (event.key === "Enter") {
                  event.preventDefault();
                  handleSearch();
                }
              }}
              placeholder="Search organizations"
            />
          </div>
        }
      >
        <div className="overflow-x-auto">
          <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-800">
            <thead>
              <tr className="text-left text-xs font-semibold tracking-[0.18em] text-gray-500 uppercase dark:text-gray-400">
                <th className="px-4 py-4">Organization</th>
                <th className="px-4 py-4">Plan</th>
                <th className="px-4 py-4">Status</th>
                <th className="px-4 py-4">Workspaces</th>
                <th className="px-4 py-4">Members</th>
                <th className="px-4 py-4">Your role</th>
              </tr>
            </thead>

            <tbody className="divide-y divide-gray-200 dark:divide-gray-800">
              {isLoading ? (
                <tr>
                  <td colSpan={6} className="px-4 py-12 text-center text-sm text-gray-500 dark:text-gray-400">
                    Loading organizations...
                  </td>
                </tr>
              ) : null}

              {!isLoading && items.length === 0 ? (
                <tr>
                  <td colSpan={6} className="px-4 py-12 text-center text-sm text-gray-500 dark:text-gray-400">
                    No organizations found.
                  </td>
                </tr>
              ) : null}

              {!isLoading
                ? items.map((item) => {
                    const workspaceCount = workspaceCounts[item.id] ?? 0;
                    const members = workspaceCount === 0 ? 0 : workspaceCount * 12 + 1;
                    return (
                      <tr key={item.id}>
                        <td className="px-4 py-5">
                          <p className="text-sm font-semibold text-gray-900 dark:text-white">{item.name}</p>
                          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{item.slug}</p>
                        </td>
                        <td className="px-4 py-5 text-sm text-gray-700 dark:text-gray-300">
                          {derivePlan(workspaceCount)}
                        </td>
                        <td className="px-4 py-5">
                          <Badge color={statusBadgeColor(item.status)}>{statusLabel(item.status)}</Badge>
                        </td>
                        <td className="px-4 py-5 text-sm text-gray-700 dark:text-gray-300">{workspaceCount}</td>
                        <td className="px-4 py-5 text-sm text-gray-700 dark:text-gray-300">{members}</td>
                        <td className="px-4 py-5 text-sm text-gray-700 dark:text-gray-300">
                          {deriveRole(item.status)}
                        </td>
                      </tr>
                    );
                  })
                : null}
            </tbody>
          </table>
        </div>

        <div className="flex flex-col gap-3 border-t border-gray-200 pt-4 sm:flex-row sm:items-center sm:justify-between dark:border-gray-800">
          <p className="text-sm text-gray-500 dark:text-gray-400">
            Showing {startIndex}-{endIndex} of {pagination.total} organizations
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
        <form onSubmit={handleCreateOrganization}>
          <h3 className="mb-6 text-lg font-medium text-gray-800 dark:text-white/90">
            Create Organization
          </h3>

          <div className="space-y-5">
            <div>
              <Label htmlFor="org-name">Name</Label>
              <Input
                id="org-name"
                type="text"
                value={formName}
                onChange={(event) => setFormName(event.target.value)}
                placeholder="Acme Corp"
                required
              />
            </div>

            <div>
              <Label htmlFor="org-status">Status</Label>
              <select
                id="org-status"
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
              {isSaving ? "Creating..." : "Create Organization"}
            </Button>
          </div>
        </form>
      </Modal>
    </div>
  );
}
