"use client";

import { useEffect, useMemo, useState } from "react";
import { useRouter } from "next/navigation";
import PageBreadcrumb from "@/components/common/PageBreadCrumb";
import Button from "@/components/ui/button/Button";
import Input from "@/components/form/input/InputField";
import Badge from "@/components/ui/badge/Badge";
import { PlusIcon } from "@/icons";
import {
  listWorkspaceMarketplaceCatalog,
  type WorkspaceMarketplaceCatalogItem,
} from "@/components/workspace/api";

function buildDeployHref(item: WorkspaceMarketplaceCatalogItem) {
  return `/marketplace/deploy?resource=${encodeURIComponent(item.resource_model)}`;
}

function MarketplaceCard(props: { item: WorkspaceMarketplaceCatalogItem }) {
  const router = useRouter();
  const versionCount = props.item.versions.length;
  const versionPreview = props.item.versions.slice(0, 3).map((version) => version.resource_version);

  return (
    <article className="flex h-full flex-col rounded-3xl border border-gray-200 bg-white p-5 shadow-theme-xs transition hover:-translate-y-0.5 hover:shadow-lg dark:border-gray-800 dark:bg-white/[0.03]">
      <div className="flex items-start justify-between gap-4">
        <div className="flex min-w-0 items-start gap-3">
          <div className="flex h-12 w-12 shrink-0 items-center justify-center rounded-2xl bg-brand-500 text-sm font-semibold text-white shadow-sm">
            {props.item.name.slice(0, 2).toUpperCase()}
          </div>
          <div className="min-w-0">
            <p className="truncate text-base font-semibold text-gray-900 dark:text-white">
              {props.item.name}
            </p>
            <p className="mt-1 line-clamp-2 text-sm leading-6 text-gray-500 dark:text-gray-400">
              {props.item.summary}
            </p>
          </div>
        </div>

        <Badge color="primary">{props.item.resource_model}</Badge>
      </div>

      <div className="mt-5 space-y-4">
        <div className="grid gap-3 rounded-2xl bg-gray-50 p-4 dark:bg-white/[0.04]">
          <Row label="Resource type" value={props.item.resource_type} />
          <Row label="Template" value={props.item.template_name} />
          <Row label="Default version" value={props.item.default_version} />
        </div>

        <div>
          <p className="text-xs font-semibold uppercase tracking-[0.18em] text-gray-400 dark:text-gray-500">
            Supported versions
          </p>
          <div className="mt-3 flex flex-wrap gap-2">
            {versionPreview.map((version) => (
              <span
                key={version}
                className="inline-flex rounded-full border border-gray-200 bg-white px-3 py-1 text-xs font-medium text-gray-700 dark:border-gray-800 dark:bg-white/[0.03] dark:text-gray-300"
              >
                {version}
              </span>
            ))}
            {versionCount > versionPreview.length ? (
              <span className="inline-flex rounded-full border border-dashed border-gray-300 px-3 py-1 text-xs font-medium text-gray-500 dark:border-gray-700 dark:text-gray-400">
                +{versionCount - versionPreview.length} more
              </span>
            ) : null}
          </div>
        </div>
      </div>

      <div className="mt-6 flex items-center justify-between gap-3 border-t border-gray-200 pt-5 dark:border-gray-800">
        <Button
          className="rounded-xl px-4"
          startIcon={<PlusIcon className="size-4" />}
          onClick={() => router.push(buildDeployHref(props.item))}
        >
          Deploy
        </Button>

        <div className="text-right">
          <p className="text-xs font-medium uppercase tracking-[0.18em] text-gray-400 dark:text-gray-500">
            Package
          </p>
          <p className="mt-1 text-sm font-semibold text-gray-900 dark:text-white">
            {props.item.slug}
          </p>
        </div>
      </div>
    </article>
  );
}

function Row(props: { label: string; value: string }) {
  return (
    <div className="flex items-start justify-between gap-3">
      <span className="text-xs font-medium uppercase tracking-[0.16em] text-gray-400 dark:text-gray-500">
        {props.label}
      </span>
      <span className="max-w-[55%] text-right text-sm font-medium text-gray-900 dark:text-white">
        {props.value}
      </span>
    </div>
  );
}

export default function MarketplacePage() {
  const [templates, setTemplates] = useState<WorkspaceMarketplaceCatalogItem[]>([]);
  const [search, setSearch] = useState("");
  const [resourceTypeFilter, setResourceTypeFilter] = useState("all");

  useEffect(() => {
    void listWorkspaceMarketplaceCatalog()
      .then(setTemplates)
      .catch(() => setTemplates([]));
  }, []);

  const filteredItems = useMemo(() => {
    const query = search.trim().toLowerCase();

    return templates.filter((item) => {
      const matchesSearch =
        query === "" ||
        item.name.toLowerCase().includes(query) ||
        item.summary.toLowerCase().includes(query) ||
        item.resource_model.toLowerCase().includes(query) ||
        item.resource_type.toLowerCase().includes(query) ||
        item.template_name.toLowerCase().includes(query);
      const matchesResourceType =
        resourceTypeFilter === "all" || item.resource_type === resourceTypeFilter;
      return matchesSearch && matchesResourceType;
    });
  }, [resourceTypeFilter, search, templates]);

  const resourceTypes = useMemo(
    () => Array.from(new Set(templates.map((item) => item.resource_type))).sort(),
    [templates],
  );

  return (
    <div className="space-y-6">
      <PageBreadcrumb pageTitle="Marketplace" />

      <section className="rounded-3xl border border-gray-200 bg-white dark:border-gray-800 dark:bg-white/[0.03]">
        <div className="flex flex-col gap-4 border-b border-gray-200 px-6 py-5 dark:border-gray-800 xl:flex-row xl:items-end xl:justify-between">
          <div className="max-w-3xl space-y-2">
            <h1 className="text-2xl font-semibold text-gray-900 dark:text-white">
              Marketplace catalog
            </h1>
            <p className="text-sm leading-7 text-gray-500 dark:text-gray-400">
              Browse the packages that admins have published, then open the deployment
              flow with the selected package already filled in.
            </p>
          </div>

          <div className="inline-flex items-center rounded-full bg-brand-500/10 px-3 py-1 text-xs font-semibold uppercase tracking-[0.24em] text-brand-600 dark:text-brand-400">
            {filteredItems.length} packages
          </div>
        </div>

        <div className="border-b border-gray-200 px-6 py-5 dark:border-gray-800">
          <div className="grid gap-4 lg:grid-cols-[minmax(0,1fr)_240px]">
            <Input
              type="text"
              value={search}
              onChange={(event) => setSearch(event.target.value)}
              placeholder="Search package name, summary, resource type, or template..."
            />

            <select
              value={resourceTypeFilter}
              onChange={(event) => setResourceTypeFilter(event.target.value)}
              className="h-11 rounded-lg border border-gray-300 bg-transparent px-4 text-sm text-gray-800 shadow-theme-xs outline-none transition focus:border-brand-300 focus:ring-3 focus:ring-brand-500/10 dark:border-gray-700 dark:bg-gray-900 dark:text-white/90 dark:focus:border-brand-800"
            >
              <option value="all">All resource types</option>
              {resourceTypes.map((item) => (
                <option key={item} value={item}>
                  {item}
                </option>
              ))}
            </select>
          </div>
        </div>

        <div className="px-6 py-6">
          {filteredItems.length === 0 ? (
            <div className="rounded-3xl border border-dashed border-gray-200 px-6 py-16 text-center dark:border-gray-800">
              <p className="text-lg font-semibold text-gray-900 dark:text-white">
                No marketplace packages are available yet.
              </p>
              <p className="mx-auto mt-2 max-w-xl text-sm leading-7 text-gray-500 dark:text-gray-400">
                Try another search or wait until admins publish a package to the catalog.
              </p>
            </div>
          ) : (
            <div className="grid gap-5 md:grid-cols-2 xl:grid-cols-3">
              {filteredItems.map((item) => (
                <MarketplaceCard key={item.slug} item={item} />
              ))}
            </div>
          )}
        </div>
      </section>
    </div>
  );
}
