"use client";

import { useWorkspaceScope } from "@/context/WorkspaceScopeContext";

export default function WorkspaceScopeSelect() {
  const { workspaceID, workspaces, isLoading, error, setWorkspaceID } = useWorkspaceScope();

  return (
    <div className="w-[220px] xl:w-[260px]">
      <label htmlFor="global-workspace-scope" className="sr-only">
        Workspace scope
      </label>
      <select
        id="global-workspace-scope"
        value={workspaceID}
        onChange={(event) => setWorkspaceID(event.target.value)}
        disabled={isLoading || workspaces.length === 0}
        className="h-11 w-full rounded-lg border border-gray-200 bg-gray-50 px-4 text-sm text-gray-800 shadow-theme-xs outline-none transition focus:border-brand-300 focus:ring-3 focus:ring-brand-500/10 disabled:cursor-not-allowed disabled:opacity-70 dark:border-gray-800 dark:bg-gray-900 dark:bg-white/[0.03] dark:text-white/90 dark:focus:border-brand-800"
      >
        {error !== "" ? <option value="">Workspace unavailable</option> : null}
        {error === "" && workspaces.length === 0 ? (
          <option value="">{isLoading ? "Loading workspaces..." : "No workspace available"}</option>
        ) : null}
        {workspaces.map((item) => (
          <option key={item.id} value={item.id}>
            {item.slug}
          </option>
        ))}
      </select>
    </div>
  );
}

